package entity

import (
	"errors"
	"strings"
	"time"
)

// Regras de negócio de um item.
var (
	ErrItemNameRequired     = errors.New("o nome do item é obrigatório")
	ErrItemCategoryRequired = errors.New("o item precisa de uma categoria")
	ErrItemQuantityNegative = errors.New("a quantidade do item não pode ser negativa")
)

type Item struct {
	ID          int64
	CategoryID  int64
	Category    Category // preenchida nas consultas (nome da categoria para exibir)
	Name        string
	Description string
	// Quantity é a quantidade em estoque. Só muda por movimentação (NewStockMovement),
	// para o saldo sempre bater com o histórico de entradas e saídas.
	Quantity  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ItemData são os dados cadastrais de um item (a quantidade não entra aqui: ela muda por movimentação).
type ItemData struct {
	CategoryID  int64
	Name        string
	Description string
}

// NewItem cria um item novo, com estoque zero, já aplicando as regras de negócio.
// O estoque inicial é lançado depois, como uma movimentação de entrada.
func NewItem(data ItemData) (Item, error) {
	var item Item
	err := item.Change(data)
	return item, err
}

// Change altera os dados cadastrais do item, aplicando as regras de negócio.
// Vale para qualquer origem (tela, API, importação), por isso fica na entidade.
func (item *Item) Change(data ItemData) error {
	name := strings.TrimSpace(data.Name)
	if name == "" {
		return ErrItemNameRequired
	}
	if data.CategoryID <= 0 {
		return ErrItemCategoryRequired
	}

	if item.CategoryID != data.CategoryID {
		item.Category = Category{} // trocou de categoria: a carregada não vale mais
	}
	item.CategoryID = data.CategoryID
	item.Name = name
	item.Description = strings.TrimSpace(data.Description)
	return nil
}

// InStock diz se há unidades do item em estoque.
func (item Item) InStock() bool {
	return item.Quantity > 0
}
