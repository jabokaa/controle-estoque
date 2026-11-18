package repository

// SortDirection é a direção da ordenação de uma listagem.
type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

// PageRequest diz qual página de uma listagem buscar.
type PageRequest struct {
	Number int // começa em 1
	Size   int // itens por página
}

// Offset é quantos registros pular para chegar na página (usado no OFFSET do SQL).
func (pageRequest PageRequest) Offset() int {
	return (pageRequest.Number - 1) * pageRequest.Size
}

// Page é uma página de resultados, com o total de registros encontrados
// (equivalente ao LengthAwarePaginator do Laravel).
type Page[Item any] struct {
	Items  []Item
	Number int
	Size   int
	Total  int64
}

// TotalPages é quantas páginas existem no total.
func (page Page[Item]) TotalPages() int {
	if page.Size <= 0 {
		return 0
	}
	return int((page.Total + int64(page.Size) - 1) / int64(page.Size))
}
