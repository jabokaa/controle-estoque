package handler

import (
	"net/http"
	"net/url"
	"strings"
)

// Mensagens "flash": aparecem uma única vez, na próxima página depois de um redirect
// (equivalente ao session()->flash() / ->with('success', ...) do Laravel).
// Ficam num cookie que é apagado assim que a mensagem é lida.

const flashCookieName = "flash"

type flashMessage struct {
	Kind string // "success" ou "danger"
	Text string
}

func setFlash(writer http.ResponseWriter, kind, text string) {
	http.SetCookie(writer, &http.Cookie{
		Name:     flashCookieName,
		Value:    url.QueryEscape(kind + "|" + text),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// safeReturnPath devolve path se ele for a própria listagem (listPath) ou ela com filtros
// (listPath?...). Qualquer outra coisa vira listPath, para ninguém usar o return_to
// para mandar o usuário a outro site.
func safeReturnPath(path, listPath string) string {
	if path == listPath || strings.HasPrefix(path, listPath+"?") {
		return path
	}
	return listPath
}

// popFlash lê a mensagem (se houver) e apaga o cookie.
func popFlash(writer http.ResponseWriter, httpRequest *http.Request) *flashMessage {
	cookie, err := httpRequest.Cookie(flashCookieName)
	if err != nil {
		return nil
	}
	http.SetCookie(writer, &http.Cookie{Name: flashCookieName, Path: "/", MaxAge: -1})

	decoded, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return nil
	}
	kind, text, found := strings.Cut(decoded, "|")
	if !found {
		return nil
	}
	return &flashMessage{Kind: kind, Text: text}
}
