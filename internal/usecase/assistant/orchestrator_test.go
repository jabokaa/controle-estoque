package assistant_test

// Testes do orquestrador + agentes sem chamar nenhuma IA: um modelo roteirizado faz o papel
// do LLM, e as ferramentas rodam de verdade contra um banco SQLite temporário (com as
// migrations reais). Assim o teste é grátis, rápido e determinístico.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/infrastructure/database"
	"controle-estoque/internal/infrastructure/memory"
	infrarepository "controle-estoque/internal/infrastructure/repository"
	"controle-estoque/internal/usecase"
	"controle-estoque/internal/usecase/assistant"
)

// ---------- modelo roteirizado ----------

type scriptedStep func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error)

type scriptedModel struct {
	t        *testing.T
	steps    []scriptedStep
	requests []assistant.CompletionRequest
}

func (model *scriptedModel) Complete(ctx context.Context, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
	model.requests = append(model.requests, request)
	if len(model.steps) == 0 {
		model.t.Fatalf("o modelo foi chamado mais vezes que o roteiro previa (chamada %d)", len(model.requests))
	}
	step := model.steps[0]
	model.steps = model.steps[1:]
	return step(model.t, request)
}

func callTool(name string, input any) assistant.CompletionResponse {
	encoded, _ := json.Marshal(input)
	return assistant.CompletionResponse{
		StopReason: assistant.StopToolUse,
		Message: assistant.Message{Role: assistant.RoleAssistant, Content: []assistant.Content{{
			Kind: assistant.ContentToolCall, ToolCallID: "call-" + name, ToolName: name, ToolInput: encoded,
		}}},
	}
}

func answer(text string) assistant.CompletionResponse {
	return assistant.CompletionResponse{
		StopReason: assistant.StopEndTurn,
		Message:    assistant.Message{Role: assistant.RoleAssistant, Content: []assistant.Content{assistant.TextContent(text)}},
	}
}

func toolNames(request assistant.CompletionRequest) []string {
	var names []string
	for _, tool := range request.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	return names
}

func expectTools(t *testing.T, request assistant.CompletionRequest, expected ...string) {
	t.Helper()
	slices.Sort(expected)
	if got := toolNames(request); !slices.Equal(got, expected) {
		t.Fatalf("ferramentas oferecidas = %v, esperado %v", got, expected)
	}
}

// lastToolResult devolve o resultado de ferramenta da última mensagem da conversa.
func lastToolResult(t *testing.T, request assistant.CompletionRequest) assistant.Content {
	t.Helper()
	last := request.Messages[len(request.Messages)-1]
	for _, content := range last.Content {
		if content.Kind == assistant.ContentToolResult {
			return content
		}
	}
	t.Fatalf("a última mensagem não tem resultado de ferramenta: %+v", last)
	return assistant.Content{}
}

// ---------- sistema de teste ----------

type testSystem struct {
	useCases   assistant.SystemUseCases
	getItem    *usecase.GetItem
	pen        entity.Item // "Caneta azul", estoque 5
	stationery entity.Category
}

func newTestSystem(t *testing.T) testSystem {
	t.Helper()
	ctx := context.Background()

	db, err := database.Connect("sqlite", filepath.Join(t.TempDir(), "teste.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close(db) })
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}

	categoryRepo := infrarepository.NewCategoryRepositoryGorm(db)
	itemRepo := infrarepository.NewItemRepositoryGorm(db)
	movementRepo := infrarepository.NewStockMovementRepositoryGorm(db)

	system := testSystem{
		useCases: assistant.SystemUseCases{
			ListAllCategories: usecase.NewListAllCategories(categoryRepo),
			CreateCategory:    usecase.NewCreateCategory(categoryRepo),
			ListItems:         usecase.NewListItems(itemRepo),
			CreateItem:        usecase.NewCreateItem(itemRepo, categoryRepo, movementRepo),
			RegisterMovement:  usecase.NewRegisterStockMovement(movementRepo),
			StockSummary:      usecase.NewGetStockSummary(itemRepo),
		},
		getItem: usecase.NewGetItem(itemRepo),
	}

	system.stationery, err = system.useCases.CreateCategory.Execute(ctx, usecase.CategoryInput{Name: "Papelaria"})
	if err != nil {
		t.Fatal(err)
	}
	system.pen, err = system.useCases.CreateItem.Execute(ctx, usecase.ItemInput{
		CategoryID: system.stationery.ID, Name: "Caneta azul", InitialQuantity: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	return system
}

func (system testSystem) quantity(t *testing.T, id int64) int {
	t.Helper()
	item, err := system.getItem.Execute(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return item.Quantity
}

// ---------- testes ----------

func TestStockEntryIsDelegatedToStockAgent(t *testing.T) {
	system := newTestSystem(t)
	task := fmt.Sprintf("Dar entrada de 10 unidades no item #%d Caneta azul.", system.pen.ID)

	model := &scriptedModel{t: t, steps: []scriptedStep{
		// 1. Orquestrador: só tem as ferramentas de delegação.
		func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
			expectTools(t, request, "delegar_cadastro", "delegar_estoque", "delegar_nota_fiscal")
			return callTool("delegar_estoque", map[string]any{"tarefa": task}), nil
		},
		// 2. Agente de estoque: começa numa conversa nova, só com a tarefa.
		func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
			expectTools(t, request, "buscar_itens", "resumo_estoque", "dar_entrada", "dar_baixa")
			if len(request.Messages) != 1 || request.Messages[0].Text() != task {
				t.Fatalf("o agente deveria receber só a tarefa, recebeu %+v", request.Messages)
			}
			return callTool("dar_entrada", map[string]any{"item_id": system.pen.ID, "quantidade": 10, "observacao": "Compra"}), nil
		},
		// 3. Agente de estoque lê o resultado da ferramenta e responde.
		func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
			result := lastToolResult(t, request)
			if result.IsError || !strings.Contains(result.Text, "Estoque atual: 15") {
				t.Fatalf("resultado inesperado de dar_entrada: %+v", result)
			}
			return answer("Entrada de 10 un. registrada. Estoque atual: 15."), nil
		},
		// 4. Orquestrador recebe a resposta do agente e responde ao usuário.
		func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
			if result := lastToolResult(t, request); !strings.Contains(result.Text, "Estoque atual: 15") {
				t.Fatalf("o orquestrador não recebeu a resposta do agente: %+v", result)
			}
			return answer("Pronto! A Caneta azul agora tem 15 unidades."), nil
		},
	}}

	orchestrator := assistant.NewOrchestrator(model, system.useCases, memory.NewConversationStore())
	conversation, turn, err := orchestrator.SendMessage(context.Background(), assistant.SendMessageInput{Text: "Chegaram 10 canetas azuis"})
	if err != nil {
		t.Fatal(err)
	}

	if turn.Failed || turn.AssistantText != "Pronto! A Caneta azul agora tem 15 unidades." {
		t.Fatalf("resposta inesperada: %+v", turn)
	}
	if len(turn.Actions) != 1 || turn.Actions[0].Kind != assistant.ActionStockIn {
		t.Fatalf("ações registradas = %+v, esperado uma entrada", turn.Actions)
	}
	if got := system.quantity(t, system.pen.ID); got != 15 {
		t.Fatalf("estoque no banco = %d, esperado 15", got)
	}

	// A próxima mensagem da mesma conversa leva o histórico inteiro ao orquestrador.
	model.steps = []scriptedStep{func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
		// usuário, chamada, resultado, resposta + nova mensagem do usuário
		if len(request.Messages) != 5 {
			t.Fatalf("histórico com %d mensagens, esperado 5", len(request.Messages))
		}
		return answer("De nada!"), nil
	}}
	if _, _, err := orchestrator.SendMessage(context.Background(), assistant.SendMessageInput{
		ConversationID: conversation.ID, Text: "Obrigado",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestInvoiceAgentReceivesTheAttachmentAndCannotWrite(t *testing.T) {
	system := newTestSystem(t)
	invoiceImage := []byte("\x89PNG imagem de nota fiscal")

	model := &scriptedModel{t: t, steps: []scriptedStep{
		// Orquestrador: vê o ID do anexo, mas não recebe a imagem.
		func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
			userMessage := request.Messages[len(request.Messages)-1]
			if !strings.Contains(userMessage.Text(), "anexo-1 (nota.png, image/png)") {
				t.Fatalf("o orquestrador deveria ver o ID do anexo: %q", userMessage.Text())
			}
			for _, content := range userMessage.Content {
				if content.Kind == assistant.ContentImage {
					t.Fatal("a imagem não deveria ir para o orquestrador")
				}
			}
			return callTool("delegar_nota_fiscal", map[string]any{"tarefa": "Extraia os itens da nota.", "anexos": []string{"anexo-1"}}), nil
		},
		// Agente de nota fiscal: recebe a imagem e só tem ferramentas de leitura.
		func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
			expectTools(t, request, "listar_categorias", "buscar_itens")
			first := request.Messages[0].Content[0]
			if first.Kind != assistant.ContentImage || first.MediaType != "image/png" || string(first.Data) != string(invoiceImage) {
				t.Fatalf("o agente deveria receber a imagem, recebeu %+v", first.Kind)
			}
			return answer("Fornecedor: Papelaria Central | NF: 123\nItens:\n- Caneta azul | qtd: 20 un | cadastro: EXISTE"), nil
		},
		func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
			return answer("Encontrei 20 Canetas azuis na NF 123. Confirma a entrada?"), nil
		},
	}}

	orchestrator := assistant.NewOrchestrator(model, system.useCases, memory.NewConversationStore())
	_, turn, err := orchestrator.SendMessage(context.Background(), assistant.SendMessageInput{
		Text:        "Dê entrada nesta nota",
		Attachments: []assistant.AttachmentUpload{{FileName: "nota.png", MediaType: "image/png", Data: invoiceImage}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(turn.Actions) != 0 {
		t.Fatalf("a extração não deveria alterar nada, ações: %+v", turn.Actions)
	}
	if !slices.Equal(turn.AttachmentNames, []string{"nota.png"}) {
		t.Fatalf("anexos da troca = %v", turn.AttachmentNames)
	}
	if got := system.quantity(t, system.pen.ID); got != 5 {
		t.Fatalf("o estoque não deveria mudar: %d", got)
	}
}

func TestInsufficientStockComesBackAsToolError(t *testing.T) {
	system := newTestSystem(t)

	model := &scriptedModel{t: t, steps: []scriptedStep{
		func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
			return callTool("delegar_estoque", map[string]any{"tarefa": "Baixa de 999 Caneta azul"}), nil
		},
		func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
			return callTool("dar_baixa", map[string]any{"item_id": system.pen.ID, "quantidade": 999, "observacao": ""}), nil
		},
		func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
			result := lastToolResult(t, request)
			if !result.IsError || !strings.Contains(result.Text, "há apenas 5") {
				t.Fatalf("esperado erro de estoque insuficiente, veio %+v", result)
			}
			return answer("Estoque insuficiente: só há 5."), nil
		},
		func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
			return answer("Não foi possível: só há 5 unidades."), nil
		},
	}}

	orchestrator := assistant.NewOrchestrator(model, system.useCases, memory.NewConversationStore())
	_, turn, err := orchestrator.SendMessage(context.Background(), assistant.SendMessageInput{Text: "Baixa de 999 canetas"})
	if err != nil {
		t.Fatal(err)
	}
	if len(turn.Actions) != 0 || system.quantity(t, system.pen.ID) != 5 {
		t.Fatalf("nada deveria ter mudado: ações %+v", turn.Actions)
	}
}

func TestUnknownAttachmentIsReportedToTheOrchestrator(t *testing.T) {
	system := newTestSystem(t)

	model := &scriptedModel{t: t, steps: []scriptedStep{
		func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
			return callTool("delegar_nota_fiscal", map[string]any{"tarefa": "Leia", "anexos": []string{"anexo-9"}}), nil
		},
		func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
			if result := lastToolResult(t, request); !result.IsError || !strings.Contains(result.Text, "anexo-9") {
				t.Fatalf("esperado erro de anexo inexistente, veio %+v", result)
			}
			return answer("Não encontrei esse anexo. Pode enviar de novo?"), nil
		},
	}}

	orchestrator := assistant.NewOrchestrator(model, system.useCases, memory.NewConversationStore())
	if _, _, err := orchestrator.SendMessage(context.Background(), assistant.SendMessageInput{Text: "Leia a nota"}); err != nil {
		t.Fatal(err)
	}
}

func TestModelFailureClosesTheTurnAndExplains(t *testing.T) {
	system := newTestSystem(t)

	model := &scriptedModel{t: t, steps: []scriptedStep{
		func(t *testing.T, request assistant.CompletionRequest) (assistant.CompletionResponse, error) {
			return assistant.CompletionResponse{}, fmt.Errorf("%w: 401", assistant.ErrModelAuthentication)
		},
	}}

	orchestrator := assistant.NewOrchestrator(model, system.useCases, memory.NewConversationStore())
	conversation, turn, err := orchestrator.SendMessage(context.Background(), assistant.SendMessageInput{Text: "Oi"})
	if err != nil {
		t.Fatal(err)
	}
	if !turn.Failed || !strings.Contains(turn.AssistantText, "ANTHROPIC_API_KEY") {
		t.Fatalf("esperada falha explicando a configuração, veio %+v", turn)
	}
	// O histórico termina com uma mensagem do assistente, para a próxima pergunta seguir normal.
	last := conversation.Messages[len(conversation.Messages)-1]
	if last.Role != assistant.RoleAssistant {
		t.Fatalf("o histórico deveria terminar com o assistente, terminou com %s", last.Role)
	}
}

func TestEmptyMessageIsRejected(t *testing.T) {
	system := newTestSystem(t)
	orchestrator := assistant.NewOrchestrator(&scriptedModel{t: t}, system.useCases, memory.NewConversationStore())
	_, _, err := orchestrator.SendMessage(context.Background(), assistant.SendMessageInput{Text: "   "})
	if !errors.Is(err, assistant.ErrEmptyMessage) {
		t.Fatalf("esperado ErrEmptyMessage, veio %v", err)
	}
}
