package assistant

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Limites do chat.
const (
	maxMessageLength              = 4000
	maxAttachmentsPerMessage      = 3
	maxAttachmentsPerConversation = 10
)

var (
	ErrEmptyMessage       = errors.New("escreva uma mensagem ou anexe um arquivo")
	ErrMessageTooLong     = fmt.Errorf("a mensagem pode ter no máximo %d caracteres", maxMessageLength)
	ErrTooManyAttachments = fmt.Errorf("envie no máximo %d arquivos por mensagem e %d por conversa", maxAttachmentsPerMessage, maxAttachmentsPerConversation)
)

// AttachmentUpload é um arquivo enviado pelo usuário junto com a mensagem.
type AttachmentUpload struct {
	FileName  string
	MediaType string
	Data      []byte
}

// SendMessageInput é uma mensagem do usuário para o assistente.
type SendMessageInput struct {
	ConversationID string // vazio: começa uma conversa nova
	Text           string
	Attachments    []AttachmentUpload
}

// Orchestrator é o caso de uso do chat: recebe a mensagem do usuário, conversa com o modelo
// (o orquestrador) e executa as delegações para os agentes especialistas.
type Orchestrator struct {
	model    LanguageModel
	useCases SystemUseCases
	store    ConversationStore

	// conversationLocks garante uma resposta por vez em cada conversa
	// (duas mensagens simultâneas embaralhariam o histórico).
	conversationLocks sync.Map
}

func NewOrchestrator(model LanguageModel, useCases SystemUseCases, store ConversationStore) *Orchestrator {
	return &Orchestrator{model: model, useCases: useCases, store: store}
}

// FindConversation devolve uma conversa existente (para a tela mostrar o histórico).
func (orchestrator *Orchestrator) FindConversation(ctx context.Context, id string) (Conversation, bool) {
	if id == "" {
		return Conversation{}, false
	}
	return orchestrator.store.Find(ctx, id)
}

// SendMessage processa uma mensagem do usuário e devolve a conversa atualizada e a nova troca.
// Erros do modelo não são devolvidos como erro: viram uma resposta com Failed = true, para a
// conversa continuar (e para as ações já feitas aparecerem). Só erros de entrada são devolvidos.
func (orchestrator *Orchestrator) SendMessage(ctx context.Context, input SendMessageInput) (Conversation, Turn, error) {
	// remove espaços em branco do início e fim da mensagem
	text := strings.TrimSpace(input.Text)
	if text == "" && len(input.Attachments) == 0 {
		return Conversation{}, Turn{}, ErrEmptyMessage
	}
	if len([]rune(text)) > maxMessageLength {
		return Conversation{}, Turn{}, ErrMessageTooLong
	}
	if len(input.Attachments) > maxAttachmentsPerMessage {
		return Conversation{}, Turn{}, ErrTooManyAttachments
	}

	conversation, found := orchestrator.FindConversation(ctx, input.ConversationID)
	if !found {
		conversation = Conversation{ID: newConversationID(), Attachments: map[string]Attachment{}}
	}
	if len(conversation.Attachments)+len(input.Attachments) > maxAttachmentsPerConversation {
		return conversation, Turn{}, ErrTooManyAttachments
	}

	lock, _ := orchestrator.conversationLocks.LoadOrStore(conversation.ID, &sync.Mutex{})
	lock.(*sync.Mutex).Lock()
	//Libera o lock ao final da função para isso serve o defer abaixo.
	defer lock.(*sync.Mutex).Unlock()

	// Anexos: ficam guardados na conversa; o orquestrador só vê os IDs (o agente de nota
	// fiscal é quem recebe o arquivo). Assim a imagem não é reenviada a cada mensagem do chat.
	turn := Turn{UserText: text, CreatedAt: time.Now()}
	var attachmentNotes []string
	for _, upload := range input.Attachments {
		attachment := Attachment{
			ID:        fmt.Sprintf("anexo-%d", len(conversation.Attachments)+1),
			FileName:  upload.FileName,
			MediaType: upload.MediaType,
			Data:      upload.Data,
		}
		conversation.Attachments[attachment.ID] = attachment
		turn.AttachmentNames = append(turn.AttachmentNames, attachment.FileName)
		attachmentNotes = append(attachmentNotes, fmt.Sprintf("%s (%s, %s)", attachment.ID, attachment.FileName, attachment.MediaType))
	}

	userText := text
	if userText == "" {
		userText = "(sem texto)"
	}
	if len(attachmentNotes) > 0 {
		userText += "\n\n[Arquivos anexados nesta mensagem: " + strings.Join(attachmentNotes, "; ") + "]"
	}
	messages := append(append([]Message(nil), conversation.Messages...), Message{Role: RoleUser, Content: []Content{TextContent(userText)}})

	// Resposta: o orquestrador roda como um agente cujas ferramentas são os outros agentes.
	actions := &actionLog{}
	runContext := withActionLog(ctx, actions)
	run, err := runAgent(runContext, orchestrator.model, orchestrator.orchestratorAgent(conversation), messages)

	turn.Actions = actions.list()
	if err != nil {
		turn.Failed = true
		turn.AssistantText = failureMessage(err, len(turn.Actions) > 0)
		// Fecha a vez do assistente no histórico, para o modelo saber na próxima mensagem
		// que esta resposta falhou (e o que já tinha sido feito).
		run.Messages = append(run.Messages, Message{Role: RoleAssistant, Content: []Content{TextContent(turn.AssistantText)}})
	} else {
		turn.AssistantText = run.Answer
	}

	conversation.Messages = run.Messages
	conversation.Turns = append(conversation.Turns, turn)
	orchestrator.store.Save(ctx, conversation)
	return conversation, turn, nil
}

// orchestratorAgent monta o orquestrador da conversa: suas ferramentas delegam para os agentes.
func (orchestrator *Orchestrator) orchestratorAgent(conversation Conversation) Agent {
	return Agent{
		Name:   "orquestrador",
		System: orchestratorSystem,
		Tools: []Tool{
			orchestrator.delegationTool("delegar_cadastro",
				"Delega ao agente de cadastro: consultar e criar categorias e itens.",
				catalogAgent(orchestrator.useCases), conversation, false),
			orchestrator.delegationTool("delegar_estoque",
				"Delega ao agente de estoque: consultar estoque e total em estoque, registrar entradas e baixas.",
				stockAgent(orchestrator.useCases), conversation, false),
			orchestrator.delegationTool("delegar_nota_fiscal",
				"Delega ao agente de nota fiscal: ler notas fiscais anexadas e extrair os itens, comparando com o cadastro. Não grava nada.",
				invoiceAgent(orchestrator.useCases), conversation, true),
		},
	}
}

// delegationTool cria uma ferramenta do orquestrador que roda um agente especialista numa
// conversa nova (só com a tarefa) e devolve a resposta final dele.
func (orchestrator *Orchestrator) delegationTool(name, description string, agent Agent, conversation Conversation, withAttachments bool) Tool {
	parameters := map[string]any{
		"tarefa": map[string]any{"type": "string", "description": "Tarefa completa e autossuficiente para o agente."},
	}
	required := []string{"tarefa"}
	if withAttachments {
		parameters["anexos"] = map[string]any{
			"type":        "array",
			"items":       map[string]any{"type": "string"},
			"description": "IDs dos anexos a analisar (ex.: [\"anexo-1\"]).",
		}
		required = append(required, "anexos")
	}

	return Tool{
		Definition: ToolDefinition{Name: name, Description: description, Parameters: parameters, Required: required},
		Run: func(ctx context.Context, input json.RawMessage) (string, error) {
			var params struct {
				Task          string   `json:"tarefa"`
				AttachmentIDs []string `json:"anexos"`
			}
			if err := decodeToolInput(input, &params); err != nil {
				return "", err
			}
			if strings.TrimSpace(params.Task) == "" {
				return "", errors.New("a tarefa está vazia")
			}

			content := []Content{}
			for _, attachmentID := range params.AttachmentIDs {
				attachment, exists := conversation.Attachments[attachmentID]
				if !exists {
					return "", fmt.Errorf("anexo %q não existe nesta conversa", attachmentID)
				}
				kind := ContentImage
				if attachment.MediaType == "application/pdf" {
					kind = ContentDocument
				}
				content = append(content, Content{Kind: kind, MediaType: attachment.MediaType, Data: attachment.Data})
			}
			content = append(content, TextContent(params.Task))

			run, err := runAgent(ctx, orchestrator.model, agent, []Message{{Role: RoleUser, Content: content}})
			if err != nil {
				return "", fmt.Errorf("%s não concluiu: %w", agent.Name, err)
			}
			if strings.TrimSpace(run.Answer) == "" {
				return "(o agente não devolveu resposta)", nil
			}
			return run.Answer, nil
		},
	}
}

// failureMessage explica a falha ao usuário, avisando se algo já tinha sido alterado.
func failureMessage(err error, hasActions bool) string {
	reason := "Ocorreu um erro ao falar com a IA."
	switch {
	case errors.Is(err, ErrModelAuthentication):
		reason = "A IA não está configurada: confira a variável ANTHROPIC_API_KEY no arquivo .env e reinicie o servidor."
	case errors.Is(err, ErrModelRateLimited):
		reason = "O limite de uso da IA foi atingido. Tente de novo em instantes."
	case errors.Is(err, ErrModelUnavailable):
		reason = "A IA está indisponível no momento. Tente de novo em instantes."
	case errors.Is(err, ErrTooManySteps):
		reason = "A tarefa ficou longa demais e foi interrompida."
	case errors.Is(err, context.Canceled):
		reason = "A resposta foi cancelada."
	}
	if hasActions {
		reason += " Atenção: algumas alterações já foram feitas (veja abaixo)."
	}
	return reason
}

func newConversationID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
