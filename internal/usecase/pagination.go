package usecase

import "controle-estoque/internal/domain/repository"

// Regras de paginação compartilhadas por todos os casos de uso de listagem.

const (
	defaultPageSize = 10
	maxPageSize     = 50
)

// normalizePageRequest garante página mínima 1 e tamanho de página entre 1 e maxPageSize,
// venha a consulta de onde vier (tela, API, teste).
func normalizePageRequest(pageRequest repository.PageRequest) repository.PageRequest {
	if pageRequest.Number < 1 {
		pageRequest.Number = 1
	}
	if pageRequest.Size < 1 {
		pageRequest.Size = defaultPageSize
	}
	if pageRequest.Size > maxPageSize {
		pageRequest.Size = maxPageSize
	}
	return pageRequest
}

// normalizeSortDirection devolve SortDescending só quando pedido explicitamente; qualquer outro valor vira crescente.
func normalizeSortDirection(direction repository.SortDirection) repository.SortDirection {
	if direction == repository.SortDescending {
		return repository.SortDescending
	}
	return repository.SortAscending
}

// paginate busca a página pedida usando fetch. Se ela não existir mais
// (ex.: página 5 de um filtro que só tem 2 páginas), busca a última página.
//
// Uso:
//
//	return paginate(query.Page, func(pageRequest repository.PageRequest) (repository.Page[entity.Category], error) {
//	    query.Page = pageRequest
//	    return repo.List(ctx, query)
//	})
func paginate[Item any](
	pageRequest repository.PageRequest,
	fetch func(repository.PageRequest) (repository.Page[Item], error),
) (repository.Page[Item], error) {
	page, err := fetch(pageRequest)
	if err != nil {
		return page, err
	}

	lastPage := page.TotalPages()
	if lastPage > 0 && pageRequest.Number > lastPage {
		pageRequest.Number = lastPage
		return fetch(pageRequest)
	}
	return page, nil
}
