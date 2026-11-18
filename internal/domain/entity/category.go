// Package entity contém as entidades de negócio do estoque.
// Camada mais interna: não depende de nenhuma outra camada nem de bibliotecas externas.
package entity

import (
	"errors"
	"strings"
	"time"
)

// ErrCategoryNameRequired: regra de negócio, toda categoria precisa de um nome.
var ErrCategoryNameRequired = errors.New("o nome da categoria é obrigatório")

type Category struct {
	ID          int64
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewCategory cria uma categoria nova, já aplicando as regras de negócio.
func NewCategory(name, description string) (Category, error) {
	var category Category
	err := category.Change(name, description)
	return category, err
}

// Change altera os dados da categoria, aplicando as regras de negócio.
// Vale para qualquer origem (tela, API, importação), por isso fica na entidade.
func (category *Category) Change(name, description string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrCategoryNameRequired
	}
	category.Name = name
	category.Description = strings.TrimSpace(description)
	return nil
}
