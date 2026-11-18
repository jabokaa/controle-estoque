// Package assistant é o chat com IA: um orquestrador que entende o pedido do usuário e
// delega para agentes especialistas (cadastro, estoque, nota fiscal), que agem chamando
// os casos de uso do sistema.
//
// Este pacote não conhece nenhum provedor de IA. Ele fala com a interface LanguageModel;
// o Claude é só uma implementação dela (internal/infrastructure/llm). Para trocar de
// provedor, basta escrever outro adaptador que implemente LanguageModel.
package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// LanguageModel é a porta para qualquer LLM: recebe a conversa e devolve a próxima mensagem.
type LanguageModel interface {
	Complete(ctx context.Context, request CompletionRequest) (CompletionResponse, error)
}

// Erros neutros que os adaptadores devolvem, para a tela mostrar uma mensagem útil.
var (
	ErrModelAuthentication = errors.New("credencial do provedor de IA inválida ou ausente")
	ErrModelRateLimited    = errors.New("limite de uso do provedor de IA atingido; tente em instantes")
	ErrModelUnavailable    = errors.New("provedor de IA indisponível no momento")
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type ContentKind string

const (
	ContentText       ContentKind = "text"
	ContentImage      ContentKind = "image"    // MediaType + Data
	ContentDocument   ContentKind = "document" // PDF: MediaType + Data
	ContentToolCall   ContentKind = "tool_call"
	ContentToolResult ContentKind = "tool_result"
)

// Content é um pedaço de uma mensagem: texto, arquivo, chamada de ferramenta ou resultado dela.
type Content struct {
	Kind ContentKind

	Text string // ContentText e ContentToolResult

	MediaType string // ContentImage e ContentDocument
	Data      []byte

	ToolCallID string          // ContentToolCall e ContentToolResult
	ToolName   string          // ContentToolCall
	ToolInput  json.RawMessage // ContentToolCall
	IsError    bool            // ContentToolResult
}

func TextContent(text string) Content {
	return Content{Kind: ContentText, Text: text}
}

func ToolResultContent(toolCallID, text string, isError bool) Content {
	return Content{Kind: ContentToolResult, ToolCallID: toolCallID, Text: text, IsError: isError}
}

// Message é uma mensagem da conversa.
type Message struct {
	Role    Role
	Content []Content

	// ProviderState guarda a resposta original do provedor (ex.: os blocos de raciocínio do
	// Claude), que precisa ser reenviada sem alteração nas próximas chamadas. Só o adaptador
	// do provedor lê esse campo; o resto do sistema usa Content.
	ProviderState any
}

// Text junta os textos da mensagem.
func (message Message) Text() string {
	var parts []string
	for _, content := range message.Content {
		if content.Kind == ContentText && strings.TrimSpace(content.Text) != "" {
			parts = append(parts, strings.TrimSpace(content.Text))
		}
	}
	return strings.Join(parts, "\n\n")
}

// ToolCalls devolve as chamadas de ferramenta que o modelo pediu nesta mensagem.
func (message Message) ToolCalls() []Content {
	var calls []Content
	for _, content := range message.Content {
		if content.Kind == ContentToolCall {
			calls = append(calls, content)
		}
	}
	return calls
}

// ToolDefinition descreve uma ferramenta para o modelo (nome, para que serve e parâmetros).
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any // propriedades do JSON Schema
	Required    []string
}

type CompletionRequest struct {
	System   string
	Messages []Message
	Tools    []ToolDefinition
}

type StopReason string

const (
	StopEndTurn   StopReason = "end_turn"   // terminou de responder
	StopToolUse   StopReason = "tool_use"   // quer chamar ferramentas
	StopMaxTokens StopReason = "max_tokens" // a resposta foi cortada
	StopRefusal   StopReason = "refusal"    // o modelo recusou o pedido
)

type CompletionResponse struct {
	Message    Message
	StopReason StopReason
}
