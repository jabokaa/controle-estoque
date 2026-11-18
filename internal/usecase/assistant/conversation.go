package assistant

import (
	"context"
	"sync"
	"time"
)

// Attachment é um arquivo enviado no chat (foto ou PDF de nota fiscal).
type Attachment struct {
	ID        string // "anexo-1", "anexo-2"... é como o orquestrador se refere a ele
	FileName  string
	MediaType string
	Data      []byte
}

// ActionKind é o tipo de alteração que um agente fez no sistema.
type ActionKind string

const (
	ActionCategoryCreated ActionKind = "category_created"
	ActionItemCreated     ActionKind = "item_created"
	ActionStockIn         ActionKind = "stock_in"
	ActionStockOut        ActionKind = "stock_out"
)

// Action é uma alteração feita durante a resposta, para a tela mostrar ("Item X criado").
type Action struct {
	Kind        ActionKind
	Description string
	Link        string
}

// Turn é uma troca do chat (pergunta e resposta), no formato que a tela mostra.
type Turn struct {
	UserText        string
	AttachmentNames []string
	AssistantText   string
	Actions         []Action
	Failed          bool
	CreatedAt       time.Time
}

// Conversation é uma conversa com o assistente.
type Conversation struct {
	ID          string
	Messages    []Message // histórico do orquestrador (o que vai para o modelo)
	Attachments map[string]Attachment
	Turns       []Turn // histórico para a tela
}

// ConversationStore guarda as conversas (hoje em memória; poderia ser banco ou Redis).
type ConversationStore interface {
	Find(ctx context.Context, id string) (Conversation, bool)
	Save(ctx context.Context, conversation Conversation)
}

// actionLog acumula as ações de uma resposta. Vai no context para as ferramentas registrarem.
type actionLog struct {
	mutex   sync.Mutex
	actions []Action
}

type actionLogKey struct{}

func withActionLog(ctx context.Context, registry *actionLog) context.Context {
	return context.WithValue(ctx, actionLogKey{}, registry)
}

// recordAction registra uma ação feita por uma ferramenta.
func recordAction(ctx context.Context, action Action) {
	if registry, exists := ctx.Value(actionLogKey{}).(*actionLog); exists {
		registry.mutex.Lock()
		defer registry.mutex.Unlock()
		registry.actions = append(registry.actions, action)
	}
}

func (registry *actionLog) list() []Action {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	return append([]Action(nil), registry.actions...)
}
