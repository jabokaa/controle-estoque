// Package handler recebe as requisições HTTP, chama os casos de uso e renderiza o front (web/templates).
package handler

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

// templates guarda todas as views de web/templates, lidas uma vez ao iniciar:
//   - layouts/    estrutura das páginas (cabeçalho, menu lateral, scripts)
//   - components/ peças reutilizáveis (ícones, cabeçalho de página, estado vazio...)
//   - pages/      as telas em si, renderizadas pelos handlers pelo nome do arquivo
var templates = loadTemplates()

func loadTemplates() *template.Template {
	root := template.New("").Funcs(template.FuncMap{"dict": dict})
	for _, pattern := range []string{
		"web/templates/layouts/*.html",
		"web/templates/components/*.html",
		"web/templates/pages/*.html",
	} {
		root = template.Must(root.ParseGlob(pattern))
	}
	return root
}

// dict monta um mapa a partir de pares chave/valor, para passar vários parâmetros a um componente:
//
//	{{template "components/page_header" (dict "Title" "Categorias" "Subtitle" "...")}}
func dict(keysAndValues ...any) (map[string]any, error) {
	if len(keysAndValues)%2 != 0 {
		return nil, errors.New("dict: número ímpar de argumentos")
	}
	values := make(map[string]any, len(keysAndValues)/2)
	for index := 0; index < len(keysAndValues); index += 2 {
		key, isString := keysAndValues[index].(string)
		if !isString {
			return nil, fmt.Errorf("dict: a chave %v não é texto", keysAndValues[index])
		}
		values[key] = keysAndValues[index+1]
	}
	return values, nil
}

// render desenha a página com o status HTTP informado.
// Monta o HTML num buffer antes de enviar: se o template falhar no meio,
// o usuário recebe um erro 500 em vez de meia página.
func render(writer http.ResponseWriter, status int, page string, data any) {
	var buffer bytes.Buffer
	if err := templates.ExecuteTemplate(&buffer, page, data); err != nil {
		serverError(writer, "renderizar "+page, err)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(status)
	_, _ = buffer.WriteTo(writer)
}

// serverError registra o erro no log e responde 500 sem expor detalhes ao usuário.
func serverError(writer http.ResponseWriter, action string, err error) {
	log.Printf("erro ao %s: %v", action, err)
	http.Error(writer, "Ocorreu um erro inesperado. Tente novamente.", http.StatusInternalServerError)
}
