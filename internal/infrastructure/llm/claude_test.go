package llm

// Testes do adaptador do Claude sem chamar a API: conferem o JSON que seria enviado
// e a leitura de uma resposta de exemplo.

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/v2"

	"controle-estoque/internal/usecase/assistant"
)

func sampleRequest() assistant.CompletionRequest {
	return assistant.CompletionRequest{
		System: "Você é um agente de teste.",
		Tools: []assistant.ToolDefinition{{
			Name:        "buscar_itens",
			Description: "Busca itens.",
			Parameters:  map[string]any{"texto": map[string]any{"type": "string"}},
			Required:    []string{"texto"},
		}},
		Messages: []assistant.Message{
			{Role: assistant.RoleUser, Content: []assistant.Content{
				{Kind: assistant.ContentImage, MediaType: "image/png", Data: []byte("png")},
				assistant.TextContent("Leia a nota"),
			}},
			{Role: assistant.RoleAssistant, Content: []assistant.Content{{
				Kind: assistant.ContentToolCall, ToolCallID: "toolu_1", ToolName: "buscar_itens", ToolInput: json.RawMessage(`{"texto":"caneta"}`),
			}}},
			{Role: assistant.RoleUser, Content: []assistant.Content{assistant.ToolResultContent("toolu_1", "#1 | Caneta", false)}},
		},
	}
}

func TestBuildParamsProducesTheExpectedRequest(t *testing.T) {
	claude, _ := NewClaude("chave-de-teste", "claude-opus-5-5", "medium")
	params, err := claude.buildParams(sampleRequest())
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)

	for _, expected := range []string{
		`"model":"claude-opus-5-5"`,
		`"strict":true`,                        // parâmetros sempre seguem o schema
		`"additionalProperties":false`,         // exigido pelo strict
		`"cache_control":{"type":"ephemeral"}`, // cache de instruções e histórico
		`"effort":"medium"`,                    // esforço explícito (padrão do Opus 5.5)
		`"fallbacks":"default"`,                // fallback no servidor para recusas
		`"type":"image"`,                       // imagem da nota
		`"media_type":"image/png"`,
		`"data":"cG5n"`, // "png" em base64
		`"type":"tool_use"`,
		`"type":"tool_result"`,
		`"tool_use_id":"toolu_1"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("a requisição não contém %s\n%s", expected, body)
		}
	}
	if !slices.Contains(params.Betas, anthropic.AnthropicBetaServerSideFallback2026_07_01) {
		t.Errorf("faltou o cabeçalho beta do fallback: %v", params.Betas)
	}
	if strings.Contains(body, `"thinking"`) {
		t.Errorf("thinking não deve ser enviado (o padrão do modelo é adaptativo): %s", body)
	}
}

func TestModelsWithoutServerSideFallbackDoNotSendIt(t *testing.T) {
	claude, _ := NewClaude("chave-de-teste", "claude-haiku-5-5", "low")
	params, err := claude.buildParams(sampleRequest())
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(params)
	if strings.Contains(string(encoded), "fallbacks") || len(params.Betas) > 0 {
		t.Fatalf("haiku não deveria pedir fallback: %s", encoded)
	}
}

func TestResponseIsConvertedAndThinkingIsPreserved(t *testing.T) {
	var response anthropic.BetaMessage
	err := json.Unmarshal([]byte(`{
		"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-opus-5-5",
		"content": [
			{"type": "thinking", "thinking": "", "signature": "assinatura-do-raciocinio"},
			{"type": "text", "text": "Vou procurar."},
			{"type": "tool_use", "id": "toolu_9", "name": "buscar_itens", "input": {"texto": "caneta"}}
		],
		"stop_reason": "tool_use", "stop_sequence": null,
		"usage": {"input_tokens": 10, "output_tokens": 5}
	}`), &response)
	if err != nil {
		t.Fatal(err)
	}

	converted := fromClaudeResponse(&response)
	if converted.StopReason != assistant.StopToolUse {
		t.Fatalf("stop reason = %s", converted.StopReason)
	}
	if converted.Message.Text() != "Vou procurar." {
		t.Fatalf("texto = %q", converted.Message.Text())
	}
	calls := converted.Message.ToolCalls()
	if len(calls) != 1 || calls[0].ToolCallID != "toolu_9" || calls[0].ToolName != "buscar_itens" {
		t.Fatalf("chamadas = %+v", calls)
	}
	var input map[string]string
	if err := json.Unmarshal(calls[0].ToolInput, &input); err != nil || input["texto"] != "caneta" {
		t.Fatalf("parâmetros = %s (%v)", calls[0].ToolInput, err)
	}

	// Na próxima chamada, a mensagem volta exatamente como veio, com o bloco de raciocínio.
	messages, err := toClaudeMessages([]assistant.Message{converted.Message})
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(messages)
	if !strings.Contains(string(encoded), `"signature":"assinatura-do-raciocinio"`) {
		t.Fatalf("o bloco de raciocínio não foi preservado: %s", encoded)
	}
}

func TestSonnetGetsFallbackAndConfiguredEffort(t *testing.T) {
	claude, err := NewClaude("chave-de-teste", "claude-sonnet-5-5", "high")
	if err != nil {
		t.Fatal(err)
	}
	params, err := claude.buildParams(sampleRequest())
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(params)
	for _, expected := range []string{`"model":"claude-sonnet-5-5"`, `"effort":"high"`, `"fallbacks":"default"`} {
		if !strings.Contains(string(encoded), expected) {
			t.Errorf("a requisição não contém %s", expected)
		}
	}
}

func TestInvalidEffortIsRejected(t *testing.T) {
	if _, err := NewClaude("chave-de-teste", "claude-sonnet-5-5", "medio"); err == nil {
		t.Fatal("esforço inválido deveria dar erro")
	}
}
