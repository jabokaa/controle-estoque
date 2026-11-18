package app

import (
	"gorm.io/gorm"

	"controle-estoque/internal/delivery/http/handler"
	"controle-estoque/internal/infrastructure/repository"
	"controle-estoque/internal/usecase"
)

type App struct {
	CategoryHandler *handler.CategoryHandler
	ItemHandler     *handler.ItemHandler
}

func NewApp(db *gorm.DB) *App {
	categoryRepo := repository.NewCategoryRepositoryGorm(db)
	itemRepo := repository.NewItemRepositoryGorm(db)

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
		Create:     usecase.NewCreateItem(itemRepo, categoryRepo),
		Update:     usecase.NewUpdateItem(itemRepo, categoryRepo),
		Delete:     usecase.NewDeleteItem(itemRepo),
		Categories: usecase.NewListAllCategories(categoryRepo),
	})

	app := &App{}
	app.CategoryHandler = categoryHandler
	app.ItemHandler = itemHandler
	return app
}
