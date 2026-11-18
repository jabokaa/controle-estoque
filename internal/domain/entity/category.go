// Package entity contém as entidades de negócio do estoque.
// Camada mais interna: não depende de nenhuma outra camada nem de bibliotecas externas.
package entity

import "time"

type Category struct {
	ID          int64
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
