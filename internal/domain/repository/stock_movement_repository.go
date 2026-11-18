package repository

import (
	"context"

	"controle-estoque/internal/domain/entity"
)

// StockMovementListQuery reúne os filtros e a página do histórico de movimentações.
// O histórico é sempre do mais recente para o mais antigo.
type StockMovementListQuery struct {
	ItemID int64               // 0 = todos os itens
	Type   entity.MovementType // "" = entradas e saídas
	Page   PageRequest
}

// ApplyMovementFunc recebe o item (lido dentro da transação) e aplica a regra de negócio:
// altera item.Quantity e devolve a movimentação a gravar, ou um erro para desfazer tudo.
type ApplyMovementFunc func(item *entity.Item) (entity.StockMovement, error)

// StockMovementRepository é o contrato de acesso às movimentações de estoque.
type StockMovementRepository interface {
	// List devolve as movimentações com o Item preenchido.
	List(ctx context.Context, query StockMovementListQuery) (Page[entity.StockMovement], error)

	// CountByItem diz quantas movimentações o item tem.
	CountByItem(ctx context.Context, itemID int64) (int64, error)

	// Register executa, numa única transação: lê o item, chama apply e grava
	// a nova quantidade do item e a movimentação. Se qualquer passo falhar, nada é gravado.
	// Devolve ErrNotFound se o item não existir.
	Register(ctx context.Context, itemID int64, apply ApplyMovementFunc) (entity.StockMovement, error)
}
