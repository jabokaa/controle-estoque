package handler

// Peças de tela reutilizáveis por qualquer listagem: cabeçalhos de coluna ordenáveis e paginação.
// Os templates components/sortable_header e components/pagination recebem estas structs.

// sortableColumn é um cabeçalho de coluna que ordena a tabela ao ser clicado.
type sortableColumn struct {
	Label      string
	URL        string // link para ordenar por esta coluna
	Active     bool   // a tabela está ordenada por esta coluna
	Descending bool   // direção atual, quando Active
}

// columnDefinition descreve uma coluna ordenável: o campo usado na URL e o texto do cabeçalho.
type columnDefinition struct {
	Field string
	Label string
}

// newSortableColumns monta os cabeçalhos. Clicar na coluna ativa inverte a direção;
// clicar em outra coluna ordena por ela em ordem crescente.
func newSortableColumns(
	definitions []columnDefinition,
	currentField, currentDirection string,
	urlForSort func(field, direction string) string,
) []sortableColumn {
	columns := make([]sortableColumn, 0, len(definitions))
	for _, definition := range definitions {
		active := definition.Field == currentField
		descending := active && currentDirection == "desc"

		nextDirection := "asc"
		if active && !descending {
			nextDirection = "desc"
		}

		columns = append(columns, sortableColumn{
			Label:      definition.Label,
			URL:        urlForSort(definition.Field, nextDirection),
			Active:     active,
			Descending: descending,
		})
	}
	return columns
}

// selectOption é uma opção de um campo de seleção (<select>), como a categoria de um item.
type selectOption struct {
	Value    int64
	Label    string
	Selected bool
}

// pageLink é um botão da paginação. IsGap marca as reticências entre números distantes.
type pageLink struct {
	Number int
	URL    string
	Active bool
	IsGap  bool
}

type paginationView struct {
	Total       int64
	From        int // primeiro item mostrado (ex.: 11)
	To          int // último item mostrado (ex.: 20)
	TotalPages  int
	PreviousURL string // vazio na primeira página
	NextURL     string // vazio na última página
	Links       []pageLink
}

// newPaginationView monta a paginação: primeira e última página, a atual e suas vizinhas,
// com reticências no meio. Ex.: 1 … 4 [5] 6 … 12
func newPaginationView(currentPage, pageSize, totalPages int, total int64, urlForPage func(page int) string) paginationView {
	view := paginationView{Total: total, TotalPages: totalPages}
	if total == 0 {
		return view
	}

	view.From = (currentPage-1)*pageSize + 1
	view.To = min(currentPage*pageSize, int(total))
	if currentPage > 1 {
		view.PreviousURL = urlForPage(currentPage - 1)
	}
	if currentPage < totalPages {
		view.NextURL = urlForPage(currentPage + 1)
	}

	previousNumber := 0
	for number := 1; number <= totalPages; number++ {
		isEdge := number == 1 || number == totalPages
		isNeighbor := number >= currentPage-1 && number <= currentPage+1
		if !isEdge && !isNeighbor {
			continue
		}
		if number-previousNumber > 1 {
			view.Links = append(view.Links, pageLink{IsGap: true})
		}
		view.Links = append(view.Links, pageLink{
			Number: number,
			URL:    urlForPage(number),
			Active: number == currentPage,
		})
		previousNumber = number
	}
	return view
}
