package usecase

import (
	"context"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// ListAllItems devolve todos os itens, por nome, sem paginação
// (ex.: para o campo "Item" do formulário de movimentação e o filtro do histórico).
type ListAllItems struct {
	repo repository.ItemRepository
}

func NewListAllItems(repo repository.ItemRepository) *ListAllItems {
	return &ListAllItems{repo: repo}
}

func (listAllItems *ListAllItems) Execute(ctx context.Context) ([]entity.Item, error) {
	return listAllItems.repo.ListAll(ctx)
}
