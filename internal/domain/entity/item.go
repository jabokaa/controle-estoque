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
	Quantity    int // quantidade em estoque
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ItemData são os dados que podem ser informados ao criar ou alterar um item.
type ItemData struct {
	CategoryID  int64
	Name        string
	Description string
	Quantity    int
}

// NewItem cria um item novo, já aplicando as regras de negócio.
func NewItem(data ItemData) (Item, error) {
	var item Item
	err := item.Change(data)
	return item, err
}

// Change altera os dados do item, aplicando as regras de negócio.
// Vale para qualquer origem (tela, API, importação), por isso fica na entidade.
func (item *Item) Change(data ItemData) error {
	name := strings.TrimSpace(data.Name)
	if name == "" {
		return ErrItemNameRequired
	}
	if data.CategoryID <= 0 {
		return ErrItemCategoryRequired
	}
	if data.Quantity < 0 {
		return ErrItemQuantityNegative
	}

	if item.CategoryID != data.CategoryID {
		item.Category = Category{} // trocou de categoria: a carregada não vale mais
	}
	item.CategoryID = data.CategoryID
	item.Name = name
	item.Description = strings.TrimSpace(data.Description)
	item.Quantity = data.Quantity
	return nil
}

// InStock diz se há unidades do item em estoque.
func (item Item) InStock() bool {
	return item.Quantity > 0
}
