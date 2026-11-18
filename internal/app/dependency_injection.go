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
	listCategories := usecase.NewListCategories(categoryRepo)
	categoryHandler := handler.NewCategoryHandler(listCategories)
	app := &App{}
	app.CategoryHandler = categoryHandler
	return app
}
