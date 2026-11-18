package handler

import (
	"net/http"
	"strings"
	"time"

	"controle-estoque/internal/delivery/http/request"
	"controle-estoque/internal/usecase/assistant"
)

const (
	// assistantCookieName guarda o ID da conversa atual no navegador.
	assistantCookieName = "assistant_conversation"

	// maxRequestSize corta envios gigantes antes de lê-los (proteção do servidor).
	// As regras de cada arquivo (tipo, tamanho, quantidade) ficam no SendAssistantMessageRequest.
	// 16 * 2²⁰ = 16 MB
	maxRequestSize = 16 << 20
)

// assistantSuggestions são os exemplos clicáveis da conversa vazia.
var assistantSuggestions = []string{
	"Qual é o total de itens em estoque?",
	"Cadastre o item Caneta vermelha na categoria Papelaria",
	"Dê entrada de 10 unidades em Caneta azul",
	"Dê baixa de 2 unidades do Teclado USB, foram vendidas",
}

type AssistantHandler struct {
	orchestrator *assistant.Orchestrator
	enabled      bool   // false quando não há chave de API configurada
	modelName    string // exibido na tela
}

func NewAssistantHandler(orchestrator *assistant.Orchestrator, enabled bool, modelName string) *AssistantHandler {
	return &AssistantHandler{orchestrator: orchestrator, enabled: enabled, modelName: modelName}
}

// assistantPageView é tudo o que o template pages/assistant.html precisa.
type assistantPageView struct {
	Enabled     bool
	ModelName   string
	Turns       []assistant.Turn
	Suggestions []string
}

// Show responde GET /assistant: a tela do chat com o histórico da conversa atual.
func (assistantHandler *AssistantHandler) Show(writer http.ResponseWriter, httpRequest *http.Request) {
	view := assistantPageView{
		Enabled:     assistantHandler.enabled,
		ModelName:   assistantHandler.modelName,
		Suggestions: assistantSuggestions,
	}
	if cookie, err := httpRequest.Cookie(assistantCookieName); err == nil {
		if conversation, found := assistantHandler.orchestrator.FindConversation(httpRequest.Context(), cookie.Value); found {
			view.Turns = conversation.Turns
		}
	}
	render(writer, http.StatusOK, "assistant.html", view)
}

// Send responde POST /assistant/messages (multipart: message + attachments).
// Com HTMX, devolve só o trecho da nova troca para ser adicionado ao chat.
func (assistantHandler *AssistantHandler) Send(writer http.ResponseWriter, httpRequest *http.Request) {
	httpRequest.Body = http.MaxBytesReader(writer, httpRequest.Body, maxRequestSize)

	// 1. FormRequest: texto e arquivos (quantidade, tipo real e tamanho)
	var form request.SendAssistantMessageRequest
	if validationErrors := request.Parse(httpRequest, &form); validationErrors != nil {
		assistantHandler.respondError(writer, httpRequest, form.Message, validationErrors.Summary())
		return
	}

	if !assistantHandler.enabled {
		assistantHandler.respondError(writer, httpRequest, form.Message,
			"O assistente não está configurado: adicione ANTHROPIC_API_KEY no arquivo .env e reinicie o servidor.")
		return
	}

	// 2. Caso de uso (orquestrador + agentes)
	conversationID := ""
	if cookie, err := httpRequest.Cookie(assistantCookieName); err == nil {
		conversationID = cookie.Value
	}
	conversation, turn, err := assistantHandler.orchestrator.SendMessage(httpRequest.Context(), form.ToInput(conversationID))
	if err != nil {
		assistantHandler.respondError(writer, httpRequest, form.Message, capitalize(err.Error())+".")
		return
	}

	http.SetCookie(writer, &http.Cookie{
		Name:     assistantCookieName,
		Value:    conversation.ID,
		Path:     "/assistant",
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	assistantHandler.respondTurn(writer, httpRequest, turn)
}

// Reset responde POST /assistant/reset: começa uma conversa nova.
func (assistantHandler *AssistantHandler) Reset(writer http.ResponseWriter, httpRequest *http.Request) {
	http.SetCookie(writer, &http.Cookie{Name: assistantCookieName, Path: "/assistant", MaxAge: -1})
	http.Redirect(writer, httpRequest, "/assistant", http.StatusSeeOther)
}

// respondTurn devolve a troca: trecho HTML para o HTMX, ou a página inteira sem JavaScript.
func (assistantHandler *AssistantHandler) respondTurn(writer http.ResponseWriter, httpRequest *http.Request, turn assistant.Turn) {
	if httpRequest.Header.Get("HX-Request") != "true" {
		http.Redirect(writer, httpRequest, "/assistant", http.StatusSeeOther)
		return
	}
	render(writer, http.StatusOK, "components/chat_turn", turn)
}

// respondError mostra o problema como uma resposta de erro no próprio chat.
func (assistantHandler *AssistantHandler) respondError(writer http.ResponseWriter, httpRequest *http.Request, userText, message string) {
	if httpRequest.Header.Get("HX-Request") != "true" {
		http.Error(writer, message, http.StatusUnprocessableEntity)
		return
	}
	render(writer, http.StatusOK, "components/chat_turn", assistant.Turn{
		UserText:      userText,
		AssistantText: message,
		Failed:        true,
		CreatedAt:     time.Now(),
	})
}

func capitalize(text string) string {
	if text == "" {
		return text
	}
	runes := []rune(text)
	return strings.ToUpper(string(runes[0])) + string(runes[1:])
}
