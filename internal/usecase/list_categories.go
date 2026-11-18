// Package usecase contém as regras de aplicação (casos de uso).
// Depende apenas do domínio (entidades e interfaces de repositório).
package usecase

import (
	"context"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

type ListCategories struct {
	repo repository.CategoryRepository
}

func NewListCategories(repo repository.CategoryRepository) *ListCategories {
	return &ListCategories{repo: repo}
}

func (listCategories *ListCategories) Execute(ctx context.Context) ([]entity.Category, error) {
	return listCategories.repo.List(ctx)
}
