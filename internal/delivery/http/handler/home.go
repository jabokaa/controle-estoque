package handler

import (
	"net/http"

	"controle-estoque/internal/domain/repository"
	"controle-estoque/internal/usecase"
)

type HomeHandler struct {
	summary *usecase.GetStockSummary
}

func NewHomeHandler(summary *usecase.GetStockSummary) *HomeHandler {
	return &HomeHandler{summary: summary}
}

// homeView é tudo o que o template pages/index.html precisa.
type homeView struct {
	Summary repository.StockSummary
}

// Show responde GET /: início com o resumo do estoque.
func (homeHandler *HomeHandler) Show(writer http.ResponseWriter, httpRequest *http.Request) {
	summary, err := homeHandler.summary.Execute(httpRequest.Context())
	if err != nil {
		serverError(writer, "resumir estoque", err)
		return
	}
	render(writer, http.StatusOK, "index.html", homeView{Summary: summary})
}
