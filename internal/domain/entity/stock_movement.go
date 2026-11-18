package entity

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// MovementType é o tipo de movimentação de estoque.
type MovementType string

const (
	MovementIn  MovementType = "in"  // entrada: soma ao estoque
	MovementOut MovementType = "out" // saída (baixa): subtrai do estoque
)

// Regras de negócio de uma movimentação.
var (
	ErrMovementTypeInvalid     = errors.New("o tipo da movimentação deve ser entrada ou saída")
	ErrMovementQuantityInvalid = errors.New("a quantidade da movimentação deve ser maior que zero")
	ErrInsufficientStock       = errors.New("estoque insuficiente")
)

// InsufficientStockError é devolvido quando uma saída é maior que o estoque do item.
// errors.Is(err, ErrInsufficientStock) funciona; Available diz quanto havia.
type InsufficientStockError struct {
	Available int
}

func (insufficientStock InsufficientStockError) Error() string {
	return fmt.Sprintf("estoque insuficiente: há apenas %d unidade(s)", insufficientStock.Available)
}

func (insufficientStock InsufficientStockError) Is(target error) bool {
	return target == ErrInsufficientStock
}

// StockMovement é uma entrada ou saída de estoque de um item.
type StockMovement struct {
	ID           int64
	ItemID       int64
	Item         Item // preenchido nas consultas (nome do item para exibir)
	Type         MovementType
	Quantity     int // sempre positiva; o Type diz se soma ou subtrai
	BalanceAfter int // saldo do item depois desta movimentação
	Note         string
	CreatedAt    time.Time
}

// IsEntry diz se a movimentação é uma entrada.
func (movement StockMovement) IsEntry() bool {
	return movement.Type == MovementIn
}

// NewStockMovement aplica uma movimentação ao item: altera item.Quantity e devolve o registro
// da movimentação. Se alguma regra for violada, o item não é alterado.
//
// Regras: tipo entrada ou saída; quantidade maior que zero; uma saída não pode deixar o estoque negativo.
func NewStockMovement(item *Item, movementType MovementType, quantity int, note string) (StockMovement, error) {
	if movementType != MovementIn && movementType != MovementOut {
		return StockMovement{}, ErrMovementTypeInvalid
	}
	if quantity <= 0 {
		return StockMovement{}, ErrMovementQuantityInvalid
	}

	newQuantity := item.Quantity + quantity
	if movementType == MovementOut {
		if quantity > item.Quantity {
			return StockMovement{}, InsufficientStockError{Available: item.Quantity}
		}
		newQuantity = item.Quantity - quantity
	}

	item.Quantity = newQuantity
	return StockMovement{
		ItemID:       item.ID,
		Item:         *item,
		Type:         movementType,
		Quantity:     quantity,
		BalanceAfter: newQuantity,
		Note:         strings.TrimSpace(note),
	}, nil
}
