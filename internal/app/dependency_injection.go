package app

import (
	"gorm.io/gorm"

	"controle-estoque/internal/delivery/http/handler"
	"controle-estoque/internal/infrastructure/repository"
	"controle-estoque/internal/usecase"
)

type App struct {
	CategoryHandler *handler.CategoryHandler
}

func NewApp(db *gorm.DB) *App {
	categoryRepo := repository.NewCategoryRepositoryGorm(db)
	categoryHandler := handler.NewCategoryHandler(handler.CategoryUseCases{
		List:   usecase.NewListCategories(categoryRepo),
		Get:    usecase.NewGetCategory(categoryRepo),
		Create: usecase.NewCreateCategory(categoryRepo),
		Update: usecase.NewUpdateCategory(categoryRepo),
		Delete: usecase.NewDeleteCategory(categoryRepo),
	})

	app := &App{}
	app.CategoryHandler = categoryHandler
	return app
}
