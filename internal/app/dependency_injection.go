package app

import (
	"gorm.io/gorm"

	"controle-estoque/internal/delivery/http/handler"
	"controle-estoque/internal/infrastructure/repository"
	"controle-estoque/internal/usecase"
)

type App struct {
	HomeHandler          *handler.HomeHandler
	CategoryHandler      *handler.CategoryHandler
	ItemHandler          *handler.ItemHandler
	StockMovementHandler *handler.StockMovementHandler
}

func NewApp(db *gorm.DB) *App {
	categoryRepo := repository.NewCategoryRepositoryGorm(db)
	itemRepo := repository.NewItemRepositoryGorm(db)
	movementRepo := repository.NewStockMovementRepositoryGorm(db)

	stockSummary := usecase.NewGetStockSummary(itemRepo)

	homeHandler := handler.NewHomeHandler(stockSummary)

	categoryHandler := handler.NewCategoryHandler(handler.CategoryUseCases{
		List:   usecase.NewListCategories(categoryRepo),
		Get:    usecase.NewGetCategory(categoryRepo),
		Create: usecase.NewCreateCategory(categoryRepo),
		Update: usecase.NewUpdateCategory(categoryRepo),
		Delete: usecase.NewDeleteCategory(categoryRepo, itemRepo),
	})

	itemHandler := handler.NewItemHandler(handler.ItemUseCases{
		List:       usecase.NewListItems(itemRepo),
		Get:        usecase.NewGetItem(itemRepo),
		Create:     usecase.NewCreateItem(itemRepo, categoryRepo, movementRepo),
		Update:     usecase.NewUpdateItem(itemRepo, categoryRepo),
		Delete:     usecase.NewDeleteItem(itemRepo, movementRepo),
		Categories: usecase.NewListAllCategories(categoryRepo),
		Summary:    stockSummary,
	})

	stockMovementHandler := handler.NewStockMovementHandler(handler.StockMovementUseCases{
		List:     usecase.NewListStockMovements(movementRepo),
		Register: usecase.NewRegisterStockMovement(movementRepo),
		Summary:  stockSummary,
		Items:    usecase.NewListAllItems(itemRepo),
	})

	app := &App{}
	app.HomeHandler = homeHandler
	app.CategoryHandler = categoryHandler
	app.ItemHandler = itemHandler
	app.StockMovementHandler = stockMovementHandler
	return app
}
