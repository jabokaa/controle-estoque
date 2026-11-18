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

// itemModel é o formato da tabela items para o GORM.
// Fica aqui na infraestrutura para que a entidade do domínio não dependa do GORM.
type itemModel struct {
	ID          int64
	CategoryID  int64
	Category    categoryModel `gorm:"foreignKey:CategoryID"` // usado no Joins("Category")
	Name        string
	Description *string // ponteiro porque a coluna aceita NULL
	Quantity    int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (itemModel) TableName() string {
	return "items"
}

// toEntity converte o model do banco na entidade do domínio.
// As datas ficam gravadas em UTC no banco e são convertidas para o fuso local aqui.
func (model itemModel) toEntity() entity.Item {
	item := entity.Item{
		ID:         model.ID,
		CategoryID: model.CategoryID,
		Name:       model.Name,
		Quantity:   model.Quantity,
		CreatedAt:  model.CreatedAt.Local(),
		UpdatedAt:  model.UpdatedAt.Local(),
	}
	if model.Description != nil {
		item.Description = *model.Description
	}
	if model.Category.ID != 0 {
		item.Category = model.Category.toEntity()
	}
	return item
}

// newItemModel converte a entidade do domínio no model do banco.
// Descrição vazia vira NULL.
func newItemModel(item entity.Item) itemModel {
	model := itemModel{
		ID:         item.ID,
		CategoryID: item.CategoryID,
		Name:       item.Name,
		Quantity:   item.Quantity,
		CreatedAt:  item.CreatedAt,
		UpdatedAt:  item.UpdatedAt,
	}
	if item.Description != "" {
		model.Description = &item.Description
	}
	return model
}

// itemSortColumns liga cada campo de ordenação do domínio a uma coluna.
// Só colunas desta lista chegam ao ORDER BY, então ninguém consegue injetar SQL pela ordenação.
// "Category" é o apelido que o GORM dá à tabela categories no Joins("Category").
var itemSortColumns = map[repository.ItemSortField]clause.Column{
	repository.ItemSortByName:      {Table: "items", Name: "name"},
	repository.ItemSortByCategory:  {Table: "Category", Name: "name"},
	repository.ItemSortByQuantity:  {Table: "items", Name: "quantity"},
	repository.ItemSortByCreatedAt: {Table: "items", Name: "created_at"},
}

type ItemRepositoryGorm struct {
	db *gorm.DB
}

func NewItemRepositoryGorm(db *gorm.DB) *ItemRepositoryGorm {
	return &ItemRepositoryGorm{db: db}
}

func (itemRepository *ItemRepositoryGorm) List(ctx context.Context, query repository.ItemListQuery) (repository.Page[entity.Item], error) {
	page := repository.Page[entity.Item]{Number: query.Page.Number, Size: query.Page.Size}

	// SELECT COUNT(*) FROM items WHERE items.name LIKE ? AND items.category_id = ?
	if err := itemRepository.filtered(ctx, query).Count(&page.Total).Error; err != nil {
		return page, err
	}

	// SELECT items.*, Category.* FROM items LEFT JOIN categories Category ON ...
	// WHERE ... ORDER BY <coluna> <direção>, items.id LIMIT ? OFFSET ?
	sortColumn, isKnownColumn := itemSortColumns[query.SortBy]
	if !isKnownColumn {
		sortColumn = itemSortColumns[repository.ItemSortByName]
	}
	var models []itemModel
	err := itemRepository.filtered(ctx, query).
		Joins("Category"). // traz o nome da categoria na mesma consulta (sem N+1)
		Order(clause.OrderByColumn{
			Column: sortColumn,
			Desc:   query.SortDirection == repository.SortDescending,
		}).
		Order("items.id"). // desempate: mantém a ordem estável entre as páginas
		Limit(query.Page.Size).
		Offset(query.Page.Offset()).
		Find(&models).Error
	if err != nil {
		return page, err
	}

	page.Items = toEntities(models, itemModel.toEntity)
	return page, nil
}

// filtered monta a consulta base com os filtros da listagem.
// Cada chamada cria uma consulta nova, porque o GORM não deve reaproveitar a mesma para COUNT e SELECT.
// As colunas levam o prefixo "items." porque, com o JOIN, categories também tem name e id.
func (itemRepository *ItemRepositoryGorm) filtered(ctx context.Context, query repository.ItemListQuery) *gorm.DB {
	statement := itemRepository.db.WithContext(ctx).Model(&itemModel{})
	if query.Name != "" {
		statement = statement.Where(`items.name LIKE ? ESCAPE '\'`, "%"+escapeLike(query.Name)+"%")
	}
	if query.CategoryID > 0 {
		statement = statement.Where("items.category_id = ?", query.CategoryID)
	}
	return statement
}

// FindByID: SELECT items.*, Category.* FROM items LEFT JOIN categories Category ... WHERE items.id = ?
func (itemRepository *ItemRepositoryGorm) FindByID(ctx context.Context, id int64) (entity.Item, error) {
	var model itemModel
	err := itemRepository.db.WithContext(ctx).
		Joins("Category").
		Where("items.id = ?", id).
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.Item{}, repository.ErrNotFound
	}
	if err != nil {
		return entity.Item{}, err
	}
	return model.toEntity(), nil
}

// ExistsByName: SELECT count(*) FROM items WHERE name = ? COLLATE NOCASE AND id <> ?
func (itemRepository *ItemRepositoryGorm) ExistsByName(ctx context.Context, name string, ignoreID int64) (bool, error) {
	var count int64
	err := itemRepository.db.WithContext(ctx).
		Model(&itemModel{}).
		Where("name = ? COLLATE NOCASE", name).
		Where("id <> ?", ignoreID).
		Count(&count).Error
	return count > 0, err
}

// CountByCategory: SELECT count(*) FROM items WHERE category_id = ?
func (itemRepository *ItemRepositoryGorm) CountByCategory(ctx context.Context, categoryID int64) (int64, error) {
	var count int64
	err := itemRepository.db.WithContext(ctx).
		Model(&itemModel{}).
		Where("category_id = ?", categoryID).
		Count(&count).Error
	return count, err
}

// Create: INSERT INTO items (category_id, name, description, quantity, created_at, updated_at) VALUES (...)
func (itemRepository *ItemRepositoryGorm) Create(ctx context.Context, item *entity.Item) error {
	model := newItemModel(*item)
	// Omit(clause.Associations): grava só o item, nunca tenta inserir/alterar a categoria junto.
	if err := itemRepository.db.WithContext(ctx).Omit(clause.Associations).Create(&model).Error; err != nil {
		return err
	}
	category := item.Category
	*item = model.toEntity() // devolve o ID e as datas geradas
	item.Category = category
	return nil
}

// Update: UPDATE items SET category_id = ?, name = ?, description = ?, updated_at = ? WHERE id = ?
// A quantidade não entra: ela só muda por movimentação (StockMovementRepositoryGorm.Register).
func (itemRepository *ItemRepositoryGorm) Update(ctx context.Context, item *entity.Item) error {
	model := newItemModel(*item)
	model.UpdatedAt = time.Now().UTC()

	result := itemRepository.db.WithContext(ctx).
		Model(&itemModel{ID: item.ID}).
		Omit(clause.Associations).
		Select("category_id", "name", "description", "updated_at"). // grava inclusive description NULL
		Updates(&model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	item.UpdatedAt = model.UpdatedAt.Local()
	return nil
}

// Delete: DELETE FROM items WHERE id = ?
func (itemRepository *ItemRepositoryGorm) Delete(ctx context.Context, id int64) error {
	result := itemRepository.db.WithContext(ctx).Delete(&itemModel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// ListAll: SELECT items.*, Category.* FROM items LEFT JOIN categories Category ... ORDER BY items.name
func (itemRepository *ItemRepositoryGorm) ListAll(ctx context.Context) ([]entity.Item, error) {
	var models []itemModel
	if err := itemRepository.db.WithContext(ctx).Joins("Category").Order("items.name").Find(&models).Error; err != nil {
		return nil, err
	}
	return toEntities(models, itemModel.toEntity), nil
}

// Summary: SELECT SUM(quantity), COUNT(*), SUM(CASE WHEN quantity = 0 ...) FROM items
func (itemRepository *ItemRepositoryGorm) Summary(ctx context.Context) (repository.StockSummary, error) {
	var summary repository.StockSummary
	err := itemRepository.db.WithContext(ctx).
		Model(&itemModel{}).
		Select(`COALESCE(SUM(quantity), 0) AS total_units,
		        COUNT(*) AS item_count,
		        COALESCE(SUM(CASE WHEN quantity = 0 THEN 1 ELSE 0 END), 0) AS out_of_stock_count`).
		Scan(&summary).Error
	return summary, err
}
