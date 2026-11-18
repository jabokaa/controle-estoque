package usecase

import (
	"context"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// StockMovementInput são os dados que o usuário informa ao dar entrada ou baixa no estoque.
type StockMovementInput struct {
	ItemID   int64
	Type     entity.MovementType
	Quantity int
	Note     string
}

// RegisterStockMovement dá entrada ou baixa no estoque de um item.
type RegisterStockMovement struct {
	movementRepo repository.StockMovementRepository
}

func NewRegisterStockMovement(movementRepo repository.StockMovementRepository) *RegisterStockMovement {
	return &RegisterStockMovement{movementRepo: movementRepo}
}

// Execute registra a movimentação e atualiza o estoque do item, numa única transação.
// A movimentação devolvida traz o Item com a quantidade já atualizada.
//
// Erros: repository.ErrNotFound (item não existe), entity.ErrMovementTypeInvalid,
// entity.ErrMovementQuantityInvalid e entity.ErrInsufficientStock (saída maior que o estoque).
func (registerStockMovement *RegisterStockMovement) Execute(ctx context.Context, input StockMovementInput) (entity.StockMovement, error) {
	return registerStockMovement.movementRepo.Register(ctx, input.ItemID, func(item *entity.Item) (entity.StockMovement, error) {
		// A regra de negócio fica na entidade; o repositório só garante a transação.
		return entity.NewStockMovement(item, input.Type, input.Quantity, input.Note)
	})
}
