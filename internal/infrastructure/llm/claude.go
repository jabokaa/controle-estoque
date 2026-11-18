// Package llm contém os adaptadores de provedores de IA para a porta assistant.LanguageModel.
// Para usar outro provedor, crie outro arquivo aqui implementando Complete.
package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"

	"controle-estoque/internal/usecase/assistant"
)

// claudeMaxTokens é o teto de tokens por resposta. Generoso para não cortar o raciocínio
// no meio; as respostas reais do chat são bem menores.
const claudeMaxTokens = 16000

// modelsWithServerSideFallback são os modelos que aceitam o fallback no servidor:
// se o modelo recusar um pedido por engano, outro modelo responde na mesma chamada.
var modelsWithServerSideFallback = map[string]bool{
	"claude-opus-5-5":   true,
	"claude-opus-5":     true,
	"claude-sonnet-5-5": true,
	"claude-fable-5-1":  true,
	"claude-fable-5-2":  true,
}

// validEfforts são os níveis de esforço aceitos pela API.
var validEfforts = map[string]anthropic.BetaOutputConfigEffort{
	"low":    anthropic.BetaOutputConfigEffortLow,
	"medium": anthropic.BetaOutputConfigEffortMedium,
	"high":   anthropic.BetaOutputConfigEffortHigh,
	"xhigh":  anthropic.BetaOutputConfigEffortXhigh,
	"max":    anthropic.BetaOutputConfigEffortMax,
}

// Claude implementa assistant.LanguageModel com a API de Mensagens da Anthropic.
type Claude struct {
	client anthropic.Client
	model  string
	effort anthropic.BetaOutputConfigEffort
}

// NewClaude cria o adaptador com os valores do .env:
// ANTHROPIC_API_KEY, CLAUDE_MODELO e CLAUDE_ESFORCO (low, medium, high, xhigh ou max).
func NewClaude(apiKey, model, effort string) (*Claude, error) {
	validEffort, isValid := validEfforts[effort]
	if !isValid {
		return nil, fmt.Errorf("CLAUDE_ESFORCO inválido: %q (use low, medium, high, xhigh ou max)", effort)
	}
	return &Claude{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
		effort: validEffort,
	}, nil
}

func (claude *Claude) Complete(ctx context.Context, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
	params, err := claude.buildParams(request)
	if err != nil {
		return assistant.CompletionResponse{}, err
	}

	response, err := claude.client.Beta.Messages.New(ctx, params)
	if err != nil {
		return assistant.CompletionResponse{}, translateError(err)
	}
	return fromClaudeResponse(response), nil
}

// buildParams monta a requisição para a API a partir do pedido neutro.
func (claude *Claude) buildParams(request assistant.CompletionRequest) (anthropic.BetaMessageNewParams, error) {
	messages, err := toClaudeMessages(request.Messages)
	if err != nil {
		return anthropic.BetaMessageNewParams{}, err
	}

	params := anthropic.BetaMessageNewParams{
		Model:     claude.model,
		MaxTokens: claudeMaxTokens,
		// Instruções e ferramentas mudam pouco: ficam em cache entre as chamadas do mesmo agente.
		System: []anthropic.BetaTextBlockParam{{
			Text:         request.System,
			CacheControl: anthropic.NewBetaCacheControlEphemeralParam(),
		}},
		Messages: messages,
		Tools:    toClaudeTools(request.Tools),
		// Cache automático do histórico: cada passo do loop reaproveita o anterior.
		CacheControl: anthropic.NewBetaCacheControlEphemeralParam(),
		// Raciocínio adaptativo é o padrão do modelo; effort (CLAUDE_ESFORCO) controla o quanto ele pensa.
		OutputConfig: anthropic.BetaOutputConfigParam{Effort: claude.effort},
	}
	if modelsWithServerSideFallback[claude.model] {
		params.Fallbacks = anthropic.BetaFallbacksParamOfDefault()
		params.Betas = []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01}
	}
	return params, nil
}

// fromClaudeResponse converte a resposta da API na mensagem neutra.
func fromClaudeResponse(response *anthropic.BetaMessage) assistant.CompletionResponse {
	// ProviderState guarda a resposta original (com os blocos de raciocínio) para
	// ela voltar sem alteração na próxima chamada.
	message := assistant.Message{Role: assistant.RoleAssistant, ProviderState: response.ToParam()}
	for _, block := range response.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.BetaTextBlock:
			message.Content = append(message.Content, assistant.TextContent(variant.Text))
		case anthropic.BetaToolUseBlock:
			message.Content = append(message.Content, assistant.Content{
				Kind:       assistant.ContentToolCall,
				ToolCallID: variant.ID,
				ToolName:   variant.Name,
				ToolInput:  json.RawMessage(variant.JSON.Input.Raw()),
			})
		}
	}

	return assistant.CompletionResponse{Message: message, StopReason: toStopReason(response.StopReason)}
}

func toStopReason(stopReason anthropic.BetaStopReason) assistant.StopReason {
	switch stopReason {
	case anthropic.BetaStopReasonToolUse:
		return assistant.StopToolUse
	case anthropic.BetaStopReasonMaxTokens:
		return assistant.StopMaxTokens
	case anthropic.BetaStopReasonRefusal:
		return assistant.StopRefusal
	default:
		return assistant.StopEndTurn
	}
}

// toClaudeMessages converte a conversa neutra no formato da API.
func toClaudeMessages(messages []assistant.Message) ([]anthropic.BetaMessageParam, error) {
	converted := make([]anthropic.BetaMessageParam, 0, len(messages))
	for _, message := range messages {
		// Mensagem que veio do próprio Claude: reenvia exatamente como chegou.
		if original, isClaudeMessage := message.ProviderState.(anthropic.BetaMessageParam); isClaudeMessage {
			converted = append(converted, original)
			continue
		}

		blocks := make([]anthropic.BetaContentBlockParamUnion, 0, len(message.Content))
		for _, content := range message.Content {
			switch content.Kind {
			case assistant.ContentText:
				blocks = append(blocks, anthropic.NewBetaTextBlock(content.Text))
			case assistant.ContentImage:
				blocks = append(blocks, anthropic.NewBetaImageBlock(anthropic.BetaBase64ImageSourceParam{
					Data:      base64.StdEncoding.EncodeToString(content.Data),
					MediaType: anthropic.BetaBase64ImageSourceMediaType(content.MediaType),
				}))
			case assistant.ContentDocument:
				blocks = append(blocks, anthropic.NewBetaDocumentBlock(anthropic.BetaBase64PDFSourceParam{
					Data: base64.StdEncoding.EncodeToString(content.Data),
				}))
			case assistant.ContentToolCall:
				var input any
				if err := json.Unmarshal(content.ToolInput, &input); err != nil {
					return nil, fmt.Errorf("parâmetros da ferramenta %s: %w", content.ToolName, err)
				}
				blocks = append(blocks, anthropic.NewBetaToolUseBlock(content.ToolCallID, input, content.ToolName))
			case assistant.ContentToolResult:
				blocks = append(blocks, anthropic.NewBetaToolResultBlock(content.ToolCallID, content.Text, content.IsError))
			}
		}

		if message.Role == assistant.RoleAssistant {
			converted = append(converted, anthropic.BetaMessageParam{OfAssistant: &anthropic.BetaAssistantMessageParam{Content: blocks}})
		} else {
			converted = append(converted, anthropic.NewBetaUserMessage(blocks...))
		}
	}
	return converted, nil
}

// toClaudeTools converte as ferramentas. strict garante que os parâmetros que o modelo
// manda seguem o schema (todos os campos obrigatórios, nenhum campo a mais).
func toClaudeTools(definitions []assistant.ToolDefinition) []anthropic.BetaToolUnionParam {
	tools := make([]anthropic.BetaToolUnionParam, 0, len(definitions))
	for _, definition := range definitions {
		tool := anthropic.BetaToolParam{
			Name:        definition.Name,
			Description: anthropic.String(definition.Description),
			InputSchema: anthropic.BetaToolInputSchemaParam{
				Properties:  definition.Parameters,
				Required:    definition.Required,
				ExtraFields: map[string]any{"additionalProperties": false},
			},
			Strict: anthropic.Bool(true),
		}
		tools = append(tools, anthropic.BetaToolUnionParam{OfTool: &tool})
	}
	return tools
}

// translateError converte os erros da API nos erros neutros do assistente.
func translateError(err error) error {
	var apiError *anthropic.Error
	if errors.As(err, &apiError) {
		switch {
		case apiError.StatusCode == http.StatusUnauthorized || apiError.StatusCode == http.StatusForbidden:
			return fmt.Errorf("%w: %v", assistant.ErrModelAuthentication, err)
		case apiError.StatusCode == http.StatusTooManyRequests:
			return fmt.Errorf("%w: %v", assistant.ErrModelRateLimited, err)
		case apiError.StatusCode >= http.StatusInternalServerError:
			return fmt.Errorf("%w: %v", assistant.ErrModelUnavailable, err)
		}
		return err
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: %v", assistant.ErrModelUnavailable, err)
}
