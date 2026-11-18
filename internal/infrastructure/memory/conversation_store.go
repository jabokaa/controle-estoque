// Package memory guarda dados na memória do processo (somem quando o servidor reinicia).
package memory

import (
	"context"
	"maps"
	"sync"

	"controle-estoque/internal/usecase/assistant"
)

// ConversationStore implementa assistant.ConversationStore em memória.
// Simples para começar; para não perder as conversas ao reiniciar, troque por uma
// implementação em banco sem mexer no caso de uso.
type ConversationStore struct {
	mutex         sync.RWMutex
	conversations map[string]assistant.Conversation
}

func NewConversationStore() *ConversationStore {
	return &ConversationStore{conversations: map[string]assistant.Conversation{}}
}

func (store *ConversationStore) Find(ctx context.Context, id string) (assistant.Conversation, bool) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	conversation, exists := store.conversations[id]
	return copyConversation(conversation), exists
}

func (store *ConversationStore) Save(ctx context.Context, conversation assistant.Conversation) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.conversations[conversation.ID] = copyConversation(conversation)
}

// copyConversation copia as listas e o mapa, para quem lê não alterar o que está guardado.
func copyConversation(conversation assistant.Conversation) assistant.Conversation {
	conversation.Messages = append([]assistant.Message(nil), conversation.Messages...)
	conversation.Turns = append([]assistant.Turn(nil), conversation.Turns...)
	conversation.Attachments = maps.Clone(conversation.Attachments)
	if conversation.Attachments == nil {
		conversation.Attachments = map[string]assistant.Attachment{}
	}
	return conversation
}
