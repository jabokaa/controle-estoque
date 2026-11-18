package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

// maxAgentSteps limita quantas idas ao modelo um agente faz por tarefa,
// para um erro de raciocínio não virar um loop infinito (e uma conta alta).
const maxAgentSteps = 12

var ErrTooManySteps = errors.New("o agente não concluiu a tarefa no limite de passos")

// Tool é uma ferramenta que um agente pode usar: a descrição para o modelo
// e a função Go que executa de fato (chamando um caso de uso).
type Tool struct {
	Definition ToolDefinition
	Run        func(ctx context.Context, input json.RawMessage) (string, error)
}

// Agent é um agente: instruções (system prompt) e as ferramentas que ele pode usar.
type Agent struct {
	Name   string
	System string
	Tools  []Tool
}

// agentRun é o resultado de um agente: a resposta final e a conversa completa.
type agentRun struct {
	Answer   string
	Messages []Message
}

// runAgent é o loop de qualquer agente:
//
//  1. manda a conversa para o modelo;
//  2. se ele pediu ferramentas, executa todas e devolve os resultados numa só mensagem;
//  3. repete até ele responder sem pedir ferramentas.
//
// A conversa só cresce (nada é editado), como os provedores exigem para reaproveitar
// o raciocínio e o cache entre as chamadas.
func runAgent(ctx context.Context, model LanguageModel, agent Agent, messages []Message) (agentRun, error) {
	definitions := make([]ToolDefinition, 0, len(agent.Tools))
	toolsByName := make(map[string]Tool, len(agent.Tools))
	for _, tool := range agent.Tools {
		definitions = append(definitions, tool.Definition)
		toolsByName[tool.Definition.Name] = tool
	}

	for step := 0; step < maxAgentSteps; step++ {
		response, err := model.Complete(ctx, CompletionRequest{
			System:   agent.System,
			Messages: messages,
			Tools:    definitions,
		})
		if err != nil {
			return agentRun{Messages: messages}, err
		}
		messages = append(messages, response.Message)

		switch response.StopReason {
		case StopToolUse:
			results := executeToolCalls(ctx, agent.Name, toolsByName, response.Message.ToolCalls())
			messages = append(messages, Message{Role: RoleUser, Content: results})
			continue
		case StopRefusal:
			return agentRun{Answer: "Não posso ajudar com esse pedido.", Messages: messages}, nil
		case StopMaxTokens:
			return agentRun{Answer: response.Message.Text() + "\n\n(A resposta foi interrompida por ser longa demais.)", Messages: messages}, nil
		default:
			return agentRun{Answer: response.Message.Text(), Messages: messages}, nil
		}
	}
	return agentRun{Messages: messages}, ErrTooManySteps
}

// executeToolCalls roda as ferramentas pedidas, uma por vez (são gravações no banco),
// e devolve um resultado para cada chamada. Erro de ferramenta não interrompe o agente:
// vira um resultado com IsError, para o modelo ler a mensagem e decidir o que fazer.
func executeToolCalls(ctx context.Context, agentName string, toolsByName map[string]Tool, calls []Content) []Content {
	results := make([]Content, 0, len(calls))
	for _, call := range calls {
		tool, exists := toolsByName[call.ToolName]
		if !exists {
			results = append(results, ToolResultContent(call.ToolCallID, fmt.Sprintf("Ferramenta desconhecida: %s.", call.ToolName), true))
			continue
		}

		output, err := tool.Run(ctx, call.ToolInput)
		if err != nil {
			log.Printf("assistente: %s/%s falhou: %v", agentName, call.ToolName, err)
			results = append(results, ToolResultContent(call.ToolCallID, "Erro: "+err.Error(), true))
			continue
		}
		results = append(results, ToolResultContent(call.ToolCallID, output, false))
	}
	return results
}

// decodeToolInput lê os parâmetros de uma chamada de ferramenta para dentro de target.
func decodeToolInput(input json.RawMessage, target any) error {
	if err := json.Unmarshal(input, target); err != nil {
		return fmt.Errorf("parâmetros inválidos: %w", err)
	}
	return nil
}
