package usecase

import (
	"context"
	"errors"

	"controle-estoque/internal/domain/entity"
	"controle-estoque/internal/domain/repository"
)

// ErrCategoryNameTaken: já existe outra categoria com o mesmo nome.
// É uma regra que depende do banco, por isso fica no caso de uso e não na entidade.
var ErrCategoryNameTaken = errors.New("já existe uma categoria com esse nome")

// CategoryInput são os dados que o usuário informa ao cadastrar ou editar uma categoria.
type CategoryInput struct {
	Name        string
	Description string
}

// ensureCategoryNameIsAvailable devolve ErrCategoryNameTaken se o nome já estiver em uso
// por outra categoria (ignoreID é a própria categoria, na edição).
func ensureCategoryNameIsAvailable(ctx context.Context, repo repository.CategoryRepository, name string, ignoreID int64) error {
	exists, err := repo.ExistsByName(ctx, name, ignoreID)
	if err != nil {
		return err
	}
	if exists {
		return ErrCategoryNameTaken
	}
	return nil
}

// CreateCategory cadastra uma categoria nova.
type CreateCategory struct {
	repo repository.CategoryRepository
}

func NewCreateCategory(repo repository.CategoryRepository) *CreateCategory {
	return &CreateCategory{repo: repo}
}

func (createCategory *CreateCategory) Execute(ctx context.Context, input CategoryInput) (entity.Category, error) {
	category, err := entity.NewCategory(input.Name, input.Description)
	if err != nil {
		return category, err
	}
	if err := ensureCategoryNameIsAvailable(ctx, createCategory.repo, category.Name, 0); err != nil {
		return category, err
	}
	err = createCategory.repo.Create(ctx, &category)
	return category, err
}

// UpdateCategory altera uma categoria existente.
type UpdateCategory struct {
	repo repository.CategoryRepository
}

func NewUpdateCategory(repo repository.CategoryRepository) *UpdateCategory {
	return &UpdateCategory{repo: repo}
}

func (updateCategory *UpdateCategory) Execute(ctx context.Context, id int64, input CategoryInput) (entity.Category, error) {
	category, err := updateCategory.repo.FindByID(ctx, id)
	if err != nil {
		return category, err
	}
	if err := category.Change(input.Name, input.Description); err != nil {
		return category, err
	}
	if err := ensureCategoryNameIsAvailable(ctx, updateCategory.repo, category.Name, category.ID); err != nil {
		return category, err
	}
	err = updateCategory.repo.Update(ctx, &category)
	return category, err
}
