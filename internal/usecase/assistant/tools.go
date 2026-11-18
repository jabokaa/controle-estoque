package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
	"controle-estoque/internal/usecase"
)

// SystemUseCases são os casos de uso que as ferramentas dos agentes chamam.
// Os agentes passam pelas mesmas regras e validações da tela: não existe atalho para o banco.
type SystemUseCases struct {
	ListAllCategories *usecase.ListAllCategories
	CreateCategory    *usecase.CreateCategory
	ListItems         *usecase.ListItems
	CreateItem        *usecase.CreateItem
	RegisterMovement  *usecase.RegisterStockMovement
	StockSummary      *usecase.GetStockSummary
}

// searchResultLimit é quantos itens a busca devolve ao modelo de uma vez.
const searchResultLimit = 50

// ---------- Consultas ----------

func listCategoriesTool(useCases SystemUseCases) Tool {
	return Tool{
		Definition: ToolDefinition{
			Name:        "listar_categorias",
			Description: "Lista todas as categorias cadastradas, com ID e nome.",
			Parameters:  map[string]any{},
		},
		Run: func(ctx context.Context, input json.RawMessage) (string, error) {
			categories, err := useCases.ListAllCategories.Execute(ctx)
			if err != nil {
				return "", err
			}
			if len(categories) == 0 {
				return "Nenhuma categoria cadastrada.", nil
			}
			var lines []string
			for _, category := range categories {
				lines = append(lines, fmt.Sprintf("#%d | %s", category.ID, category.Name))
			}
			return strings.Join(lines, "\n"), nil
		},
	}
}

func searchItemsTool(useCases SystemUseCases) Tool {
	return Tool{
		Definition: ToolDefinition{
			Name: "buscar_itens",
			Description: "Busca itens cadastrados cujo nome contém o texto informado (sem diferenciar maiúsculas). " +
				"Devolve ID, nome, categoria e estoque atual. Texto vazio lista todos (até 50). " +
				"Para achar um item, prefira uma palavra-chave curta (ex.: \"caneta\" em vez de \"caneta esferográfica azul 1.0\").",
			Parameters: map[string]any{
				"texto": map[string]any{"type": "string", "description": "Parte do nome do item. Vazio para listar todos."},
			},
			Required: []string{"texto"},
		},
		Run: func(ctx context.Context, input json.RawMessage) (string, error) {
			var params struct {
				Text string `json:"texto"`
			}
			if err := decodeToolInput(input, &params); err != nil {
				return "", err
			}
			page, err := useCases.ListItems.Execute(ctx, repository.ItemListQuery{
				Name: strings.TrimSpace(params.Text),
				Page: repository.PageRequest{Number: 1, Size: searchResultLimit},
			})
			if err != nil {
				return "", err
			}
			if len(page.Items) == 0 {
				return fmt.Sprintf("Nenhum item encontrado com %q.", params.Text), nil
			}
			var lines []string
			for _, item := range page.Items {
				lines = append(lines, fmt.Sprintf("#%d | %s | categoria: %s (#%d) | estoque: %d",
					item.ID, item.Name, item.Category.Name, item.CategoryID, item.Quantity))
			}
			if page.Total > int64(len(page.Items)) {
				lines = append(lines, fmt.Sprintf("(mostrando %d de %d; refine a busca)", len(page.Items), page.Total))
			}
			return strings.Join(lines, "\n"), nil
		},
	}
}

func stockSummaryTool(useCases SystemUseCases) Tool {
	return Tool{
		Definition: ToolDefinition{
			Name:        "resumo_estoque",
			Description: "Devolve o total de unidades em estoque, quantos itens estão cadastrados e quantos estão sem estoque.",
			Parameters:  map[string]any{},
		},
		Run: func(ctx context.Context, input json.RawMessage) (string, error) {
			summary, err := useCases.StockSummary.Execute(ctx)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Total em estoque: %d unidades. Itens cadastrados: %d. Itens sem estoque: %d.",
				summary.TotalUnits, summary.ItemCount, summary.OutOfStockCount), nil
		},
	}
}

// ---------- Cadastro ----------

func createCategoryTool(useCases SystemUseCases) Tool {
	return Tool{
		Definition: ToolDefinition{
			Name:        "criar_categoria",
			Description: "Cria uma categoria. Use só quando nenhuma categoria existente servir.",
			Parameters: map[string]any{
				"nome":      map[string]any{"type": "string", "description": "Nome da categoria (até 100 caracteres)."},
				"descricao": map[string]any{"type": "string", "description": "Descrição curta, ou vazio."},
			},
			Required: []string{"nome", "descricao"},
		},
		Run: func(ctx context.Context, input json.RawMessage) (string, error) {
			var params struct {
				Name        string `json:"nome"`
				Description string `json:"descricao"`
			}
			if err := decodeToolInput(input, &params); err != nil {
				return "", err
			}
			category, err := useCases.CreateCategory.Execute(ctx, usecase.CategoryInput{Name: params.Name, Description: params.Description})
			if err != nil {
				return "", err
			}
			recordAction(ctx, Action{
				Kind:        ActionCategoryCreated,
				Description: fmt.Sprintf("Categoria “%s” criada", category.Name),
				Link:        fmt.Sprintf("/categories/%d/edit", category.ID),
			})
			return fmt.Sprintf("Categoria criada: #%d | %s", category.ID, category.Name), nil
		},
	}
}

func createItemTool(useCases SystemUseCases) Tool {
	return Tool{
		Definition: ToolDefinition{
			Name: "criar_item",
			Description: "Cadastra um item novo com estoque zero. Antes, use buscar_itens para não duplicar " +
				"(o nome precisa ser único). O estoque inicial é lançado depois, como entrada, pelo agente de estoque.",
			Parameters: map[string]any{
				"nome":         map[string]any{"type": "string", "description": "Nome do item (até 100 caracteres)."},
				"categoria_id": map[string]any{"type": "integer", "description": "ID de uma categoria existente."},
				"descricao":    map[string]any{"type": "string", "description": "Descrição curta, ou vazio."},
			},
			Required: []string{"nome", "categoria_id", "descricao"},
		},
		Run: func(ctx context.Context, input json.RawMessage) (string, error) {
			var params struct {
				Name        string `json:"nome"`
				CategoryID  int64  `json:"categoria_id"`
				Description string `json:"descricao"`
			}
			if err := decodeToolInput(input, &params); err != nil {
				return "", err
			}
			item, err := useCases.CreateItem.Execute(ctx, usecase.ItemInput{
				CategoryID:  params.CategoryID,
				Name:        params.Name,
				Description: params.Description,
			})
			if err != nil {
				return "", err
			}
			recordAction(ctx, Action{
				Kind:        ActionItemCreated,
				Description: fmt.Sprintf("Item “%s” criado em %s", item.Name, item.Category.Name),
				Link:        fmt.Sprintf("/items/%d/edit", item.ID),
			})
			return fmt.Sprintf("Item criado: #%d | %s | categoria: %s | estoque: 0", item.ID, item.Name, item.Category.Name), nil
		},
	}
}

// ---------- Estoque ----------

// stockMovementTool cria a ferramenta de entrada (MovementIn) ou de baixa (MovementOut).
func stockMovementTool(useCases SystemUseCases, movementType entity.MovementType) Tool {
	name, description, actionKind, verb := "dar_entrada",
		"Dá entrada (soma) de unidades no estoque de um item.", ActionStockIn, "Entrada"
	if movementType == entity.MovementOut {
		name, description, actionKind, verb = "dar_baixa",
			"Dá baixa (subtrai) de unidades do estoque de um item. Falha se o estoque for insuficiente.", ActionStockOut, "Baixa"
	}

	return Tool{
		Definition: ToolDefinition{
			Name:        name,
			Description: description,
			Parameters: map[string]any{
				"item_id":    map[string]any{"type": "integer", "description": "ID do item (veja buscar_itens)."},
				"quantidade": map[string]any{"type": "integer", "description": "Quantidade de unidades, maior que zero."},
				"observacao": map[string]any{"type": "string", "description": "Motivo ou origem (ex.: \"NF 1234 - Fornecedor X\", \"venda\"), ou vazio."},
			},
			Required: []string{"item_id", "quantidade", "observacao"},
		},
		Run: func(ctx context.Context, input json.RawMessage) (string, error) {
			var params struct {
				ItemID   int64  `json:"item_id"`
				Quantity int    `json:"quantidade"`
				Note     string `json:"observacao"`
			}
			if err := decodeToolInput(input, &params); err != nil {
				return "", err
			}
			movement, err := useCases.RegisterMovement.Execute(ctx, usecase.StockMovementInput{
				ItemID:   params.ItemID,
				Type:     movementType,
				Quantity: params.Quantity,
				Note:     params.Note,
			})
			if err != nil {
				return "", err
			}
			recordAction(ctx, Action{
				Kind:        actionKind,
				Description: fmt.Sprintf("%s de %d un. em “%s” (estoque: %d)", verb, movement.Quantity, movement.Item.Name, movement.BalanceAfter),
				Link:        fmt.Sprintf("/movements?item_id=%d", movement.ItemID),
			})
			return fmt.Sprintf("%s registrada: %d un. em #%d %s. Estoque atual: %d.",
				verb, movement.Quantity, movement.ItemID, movement.Item.Name, movement.BalanceAfter), nil
		},
	}
}
