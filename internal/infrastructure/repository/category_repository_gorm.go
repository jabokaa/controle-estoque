// Package repository implementa as interfaces de internal/domain/repository usando o GORM.
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"controle-estoque/internal/domain/entity"
)

// categoryModel é o formato da tabela categories para o GORM.
// Fica aqui na infraestrutura para que a entidade do domínio não dependa do GORM.
type categoryModel struct {
	ID          int64
	Name        string
	Description *string // ponteiro porque a coluna aceita NULL
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (categoryModel) TableName() string {
	return "categories"
}

// toEntity converte o model do banco na entidade do domínio.
func (model categoryModel) toEntity() entity.Category {
	category := entity.Category{
		ID:        model.ID,
		Name:      model.Name,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
	if model.Description != nil {
		category.Description = *model.Description
	}
	return category
}

type CategoryRepositoryGorm struct {
	db *gorm.DB
}

func NewCategoryRepositoryGorm(db *gorm.DB) *CategoryRepositoryGorm {
	return &CategoryRepositoryGorm{db: db}
}

func (categoryRepository *CategoryRepositoryGorm) List(ctx context.Context) ([]entity.Category, error) {
	var models []categoryModel
	err := categoryRepository.db.WithContext(ctx).Order("name").Find(&models).Error
	if err != nil {
		return nil, err
	}

	return toEntities(models, categoryModel.toEntity), nil
}
