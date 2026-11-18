// Package repository implementa as interfaces de internal/domain/repository usando o GORM.
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
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

// categorySortColumns liga cada campo de ordenação do domínio a uma coluna da tabela.
// Só colunas desta lista chegam ao ORDER BY, então ninguém consegue injetar SQL pela ordenação.
var categorySortColumns = map[repository.CategorySortField]string{
	repository.CategorySortByName:        "name",
	repository.CategorySortByDescription: "description",
	repository.CategorySortByCreatedAt:   "created_at",
}

type CategoryRepositoryGorm struct {
	db *gorm.DB
}

func NewCategoryRepositoryGorm(db *gorm.DB) *CategoryRepositoryGorm {
	return &CategoryRepositoryGorm{db: db}
}

func (categoryRepository *CategoryRepositoryGorm) List(ctx context.Context, query repository.CategoryListQuery) (repository.Page[entity.Category], error) {
	page := repository.Page[entity.Category]{Number: query.Page.Number, Size: query.Page.Size}

	// SELECT COUNT(*) ... WHERE name LIKE ?
	if err := categoryRepository.filtered(ctx, query).Count(&page.Total).Error; err != nil {
		return page, err
	}

	// SELECT * ... WHERE name LIKE ? ORDER BY <coluna> <direção>, id LIMIT ? OFFSET ?
	sortColumn, isKnownColumn := categorySortColumns[query.SortBy]
	if !isKnownColumn {
		sortColumn = "name"
	}
	var models []categoryModel
	err := categoryRepository.filtered(ctx, query).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: sortColumn},
			Desc:   query.SortDirection == repository.SortDescending,
		}).
		Order("id"). // desempate: mantém a ordem estável entre as páginas
		Limit(query.Page.Size).
		Offset(query.Page.Offset()).
		Find(&models).Error
	if err != nil {
		return page, err
	}

	page.Items = toEntities(models, categoryModel.toEntity)
	return page, nil
}

// filtered monta a consulta base com os filtros da listagem.
// Cada chamada cria uma consulta nova, porque o GORM não deve reaproveitar a mesma para COUNT e SELECT.
func (categoryRepository *CategoryRepositoryGorm) filtered(ctx context.Context, query repository.CategoryListQuery) *gorm.DB {
	statement := categoryRepository.db.WithContext(ctx).Model(&categoryModel{})
	if query.Name != "" {
		statement = statement.Where(`name LIKE ? ESCAPE '\'`, "%"+escapeLike(query.Name)+"%")
	}
	return statement
}
