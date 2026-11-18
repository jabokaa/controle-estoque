// Package handler recebe as requisições HTTP, chama os casos de uso e renderiza o front (web/templates).
package handler

import (
	"errors"
	"fmt"
	"html/template"
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
