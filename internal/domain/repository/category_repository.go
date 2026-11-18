// Package repository define as interfaces (contratos) de persistência das entidades.
// As implementações concretas ficam em internal/infrastructure/repository.
package repository

import (
	"context"

	"controle-estoque/internal/domain/entity"
)

type CategoryRepository interface {
	List(ctx context.Context) ([]entity.Category, error)
}
