package app

import (
	"gorm.io/gorm"

	"controle-estoque/internal/config"
	"controle-estoque/internal/delivery/http/handler"
	"controle-estoque/internal/infrastructure/llm"
	"controle-estoque/internal/infrastructure/memory"
	"controle-estoque/internal/infrastructure/repository"
	"controle-estoque/internal/usecase"
	"controle-estoque/internal/usecase/assistant"
)

type App struct {
	HomeHandler          *handler.HomeHandler
	CategoryHandler      *handler.CategoryHandler
	ItemHandler          *handler.ItemHandler
	StockMovementHandler *handler.StockMovementHandler
	AssistantHandler     *handler.AssistantHandler
}

func NewApp(db *gorm.DB, cfg config.Config) (*App, error) {
	// Repositórios (infraestrutura)
	categoryRepo := repository.NewCategoryRepositoryGorm(db)
	itemRepo := repository.NewItemRepositoryGorm(db)
	movementRepo := repository.NewStockMovementRepositoryGorm(db)

	// Casos de uso compartilhados entre as telas e o assistente
	stockSummary := usecase.NewGetStockSummary(itemRepo)
	listAllCategories := usecase.NewListAllCategories(categoryRepo)
	createCategory := usecase.NewCreateCategory(categoryRepo)
	listItems := usecase.NewListItems(itemRepo)
	createItem := usecase.NewCreateItem(itemRepo, categoryRepo, movementRepo)
	registerMovement := usecase.NewRegisterStockMovement(movementRepo)

	homeHandler := handler.NewHomeHandler(stockSummary)

	categoryHandler := handler.NewCategoryHandler(handler.CategoryUseCases{
		List:   usecase.NewListCategories(categoryRepo),
		Get:    usecase.NewGetCategory(categoryRepo),
		Create: createCategory,
		Update: usecase.NewUpdateCategory(categoryRepo),
		Delete: usecase.NewDeleteCategory(categoryRepo, itemRepo),
	})

	itemHandler := handler.NewItemHandler(handler.ItemUseCases{
		List:       listItems,
		Get:        usecase.NewGetItem(itemRepo),
		Create:     createItem,
		Update:     usecase.NewUpdateItem(itemRepo, categoryRepo),
		Delete:     usecase.NewDeleteItem(itemRepo, movementRepo),
		Categories: listAllCategories,
		Summary:    stockSummary,
	})

	stockMovementHandler := handler.NewStockMovementHandler(handler.StockMovementUseCases{
		List:     usecase.NewListStockMovements(movementRepo),
		Register: registerMovement,
		Summary:  stockSummary,
		Items:    usecase.NewListAllItems(itemRepo),
	})

	// Assistente de IA: o LLM é escolhido aqui. Para trocar de provedor, troque esta linha
	// por outro adaptador que implemente assistant.LanguageModel.
	claude, err := llm.NewClaude(cfg.AnthropicAPIKey, cfg.ClaudeModel, cfg.ClaudeEffort)
	if err != nil {
		return nil, err
	}
	var languageModel assistant.LanguageModel = claude
	orchestrator := assistant.NewOrchestrator(languageModel, assistant.SystemUseCases{
		ListAllCategories: listAllCategories,
		CreateCategory:    createCategory,
		ListItems:         listItems,
		CreateItem:        createItem,
		RegisterMovement:  registerMovement,
		StockSummary:      stockSummary,
	}, memory.NewConversationStore())
	assistantHandler := handler.NewAssistantHandler(orchestrator, cfg.AnthropicAPIKey != "", cfg.ClaudeModel)

	app := &App{}
	app.HomeHandler = homeHandler
	app.CategoryHandler = categoryHandler
	app.ItemHandler = itemHandler
	app.StockMovementHandler = stockMovementHandler
	app.AssistantHandler = assistantHandler
	return app, nil
}
