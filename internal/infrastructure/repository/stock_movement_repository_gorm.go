package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// stockMovementModel é o formato da tabela stock_movements para o GORM.
type stockMovementModel struct {
	ID           int64
	ItemID       int64
	Item         itemModel `gorm:"foreignKey:ItemID"` // usado no Joins("Item")
	Type         string
	Quantity     int
	BalanceAfter int
	Note         *string // ponteiro porque a coluna aceita NULL
	CreatedAt    time.Time
}

func (stockMovementModel) TableName() string {
	return "stock_movements"
}

// toEntity converte o model do banco na entidade do domínio.
func (model stockMovementModel) toEntity() entity.StockMovement {
	movement := entity.StockMovement{
		ID:           model.ID,
		ItemID:       model.ItemID,
		Type:         entity.MovementType(model.Type),
		Quantity:     model.Quantity,
		BalanceAfter: model.BalanceAfter,
		CreatedAt:    model.CreatedAt.Local(),
	}
	if model.Note != nil {
		movement.Note = *model.Note
	}
	if model.Item.ID != 0 {
		movement.Item = model.Item.toEntity()
	}
	return movement
}

// newStockMovementModel converte a entidade do domínio no model do banco. Observação vazia vira NULL.
func newStockMovementModel(movement entity.StockMovement) stockMovementModel {
	model := stockMovementModel{
		ItemID:       movement.ItemID,
		Type:         string(movement.Type),
		Quantity:     movement.Quantity,
		BalanceAfter: movement.BalanceAfter,
	}
	if movement.Note != "" {
		model.Note = &movement.Note
	}
	return model
}

type StockMovementRepositoryGorm struct {
	db *gorm.DB
}

func NewStockMovementRepositoryGorm(db *gorm.DB) *StockMovementRepositoryGorm {
	return &StockMovementRepositoryGorm{db: db}
}

func (movementRepository *StockMovementRepositoryGorm) List(ctx context.Context, query repository.StockMovementListQuery) (repository.Page[entity.StockMovement], error) {
	page := repository.Page[entity.StockMovement]{Number: query.Page.Number, Size: query.Page.Size}

	// SELECT COUNT(*) FROM stock_movements WHERE item_id = ? AND type = ?
	if err := movementRepository.filtered(ctx, query).Count(&page.Total).Error; err != nil {
		return page, err
	}

	// SELECT stock_movements.*, Item.* FROM stock_movements LEFT JOIN items Item ...
	// ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?
	var models []stockMovementModel
	err := movementRepository.filtered(ctx, query).
		Joins("Item").
		Order("stock_movements.created_at DESC").
		Order("stock_movements.id DESC"). // desempate: movimentações no mesmo segundo
		Limit(query.Page.Size).
		Offset(query.Page.Offset()).
		Find(&models).Error
	if err != nil {
		return page, err
	}

	page.Items = toEntities(models, stockMovementModel.toEntity)
	return page, nil
}

// filtered monta a consulta base com os filtros do histórico.
func (movementRepository *StockMovementRepositoryGorm) filtered(ctx context.Context, query repository.StockMovementListQuery) *gorm.DB {
	statement := movementRepository.db.WithContext(ctx).Model(&stockMovementModel{})
	if query.ItemID > 0 {
		statement = statement.Where("stock_movements.item_id = ?", query.ItemID)
	}
	if query.Type != "" {
		statement = statement.Where("stock_movements.type = ?", string(query.Type))
	}
	return statement
}

// CountByItem: SELECT count(*) FROM stock_movements WHERE item_id = ?
func (movementRepository *StockMovementRepositoryGorm) CountByItem(ctx context.Context, itemID int64) (int64, error) {
	var count int64
	err := movementRepository.db.WithContext(ctx).
		Model(&stockMovementModel{}).
		Where("item_id = ?", itemID).
		Count(&count).Error
	return count, err
}

// Register faz tudo numa transação (BEGIN ... COMMIT; em caso de erro, ROLLBACK):
//
//	SELECT ... FROM items WHERE id = ?                       -- lê o saldo atual
//	(apply: regra de negócio calcula o novo saldo)
//	UPDATE items SET quantity = ?, updated_at = ? WHERE id = ?
//	INSERT INTO stock_movements (...) VALUES (...)
func (movementRepository *StockMovementRepositoryGorm) Register(ctx context.Context, itemID int64, apply repository.ApplyMovementFunc) (entity.StockMovement, error) {
	var registered entity.StockMovement

	err := movementRepository.db.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		// 1. Lê o item dentro da transação.
		var stored itemModel
		err := transaction.Joins("Category").Where("items.id = ?", itemID).First(&stored).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repository.ErrNotFound
		}
		if err != nil {
			return err
		}
		item := stored.toEntity()

		// 2. Regra de negócio (entidade): calcula o novo saldo ou recusa.
		movement, err := apply(&item)
		if err != nil {
			return err // devolver erro aqui desfaz a transação
		}

		// 3. Grava o novo saldo do item.
		err = transaction.Model(&itemModel{ID: item.ID}).
			Updates(map[string]any{"quantity": item.Quantity, "updated_at": time.Now().UTC()}).Error
		if err != nil {
			return err
		}

		// 4. Grava a movimentação.
		model := newStockMovementModel(movement)
		if err := transaction.Omit(clause.Associations).Create(&model).Error; err != nil {
			return err
		}

		registered = model.toEntity()
		registered.Item = item
		return nil
	})
	return registered, err
}
