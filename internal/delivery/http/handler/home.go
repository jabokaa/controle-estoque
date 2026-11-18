package handler

import "net/http"

// Home responde GET /.
func Home(writer http.ResponseWriter, request *http.Request) {
	if err := templates.ExecuteTemplate(writer, "index.html", nil); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}
