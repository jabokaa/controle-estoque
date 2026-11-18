// Package repository implementa as interfaces de internal/domain/repository usando o GORM.
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
// As datas ficam gravadas em UTC no banco e são convertidas para o fuso local aqui.
func (model categoryModel) toEntity() entity.Category {
	category := entity.Category{
		ID:        model.ID,
		Name:      model.Name,
		CreatedAt: model.CreatedAt.Local(),
		UpdatedAt: model.UpdatedAt.Local(),
	}
	if model.Description != nil {
		category.Description = *model.Description
	}
	return category
}

// newCategoryModel converte a entidade do domínio no model do banco.
// Descrição vazia vira NULL.
func newCategoryModel(category entity.Category) categoryModel {
	model := categoryModel{
		ID:        category.ID,
		Name:      category.Name,
		CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt,
	}
	if category.Description != "" {
		model.Description = &category.Description
	}
	return model
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

// FindByID: SELECT * FROM categories WHERE id = ? LIMIT 1
func (categoryRepository *CategoryRepositoryGorm) FindByID(ctx context.Context, id int64) (entity.Category, error) {
	var model categoryModel
	err := categoryRepository.db.WithContext(ctx).First(&model, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.Category{}, repository.ErrNotFound
	}
	if err != nil {
		return entity.Category{}, err
	}
	return model.toEntity(), nil
}

// ExistsByName: SELECT count(*) FROM categories WHERE name = ? COLLATE NOCASE AND id <> ?
// COLLATE NOCASE faz "papelaria" e "Papelaria" serem considerados o mesmo nome.
func (categoryRepository *CategoryRepositoryGorm) ExistsByName(ctx context.Context, name string, ignoreID int64) (bool, error) {
	var count int64
	err := categoryRepository.db.WithContext(ctx).
		Model(&categoryModel{}).
		Where("name = ? COLLATE NOCASE", name).
		Where("id <> ?", ignoreID).
		Count(&count).Error
	return count > 0, err
}

// Create: INSERT INTO categories (name, description, created_at, updated_at) VALUES (...)
func (categoryRepository *CategoryRepositoryGorm) Create(ctx context.Context, category *entity.Category) error {
	model := newCategoryModel(*category)
	if err := categoryRepository.db.WithContext(ctx).Create(&model).Error; err != nil {
		return err
	}
	*category = model.toEntity() // devolve o ID e as datas geradas
	return nil
}

// Update: UPDATE categories SET name = ?, description = ?, updated_at = ? WHERE id = ?
func (categoryRepository *CategoryRepositoryGorm) Update(ctx context.Context, category *entity.Category) error {
	model := newCategoryModel(*category)
	model.UpdatedAt = time.Now().UTC()

	result := categoryRepository.db.WithContext(ctx).
		Model(&categoryModel{ID: category.ID}).
		Select("name", "description", "updated_at"). // grava inclusive description NULL
		Updates(&model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	category.UpdatedAt = model.UpdatedAt.Local()
	return nil
}

// Delete: DELETE FROM categories WHERE id = ?
func (categoryRepository *CategoryRepositoryGorm) Delete(ctx context.Context, id int64) error {
	result := categoryRepository.db.WithContext(ctx).Delete(&categoryModel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}
