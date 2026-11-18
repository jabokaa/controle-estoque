package usecase

import (
	"context"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// ListAllCategories devolve todas as categorias, por nome, sem paginação
// (ex.: para o campo "Categoria" do formulário de item e o filtro da listagem de itens).
type ListAllCategories struct {
	repo repository.CategoryRepository
}

func NewListAllCategories(repo repository.CategoryRepository) *ListAllCategories {
	return &ListAllCategories{repo: repo}
}

func (listAllCategories *ListAllCategories) Execute(ctx context.Context) ([]entity.Category, error) {
	return listAllCategories.repo.ListAll(ctx)
}
