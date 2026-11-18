package request

import "controle-estoque/internal/usecase/assistant"

// SendAssistantMessageRequest valida a mensagem enviada no chat: POST /assistant/messages
// (multipart: texto + até 3 arquivos de nota fiscal).
//
// Equivale, no Laravel, a:
//
//	'message'       => 'max:4000',
//	'attachments'   => 'max:3',
//	'attachments.*' => 'file|mimes:jpg,png,gif,webp,pdf|max:5120',
type SendAssistantMessageRequest struct {
	Message     string         `form:"message"     label:"Mensagem" validate:"max=4000"`
	Attachments []UploadedFile `form:"attachments" label:"Anexos"   validate:"max=3,dive,file_types=image/jpeg image/png image/gif image/webp application/pdf,file_max_size=5242880"`
}

// ToInput converte o request (dados da tela) na entrada que o caso de uso entende.
func (sendRequest SendAssistantMessageRequest) ToInput(conversationID string) assistant.SendMessageInput {
	uploads := make([]assistant.AttachmentUpload, 0, len(sendRequest.Attachments))
	for _, file := range sendRequest.Attachments {
		uploads = append(uploads, assistant.AttachmentUpload{FileName: file.Name, MediaType: file.MediaType, Data: file.Data})
	}
	return assistant.SendMessageInput{
		ConversationID: conversationID,
		Text:           sendRequest.Message,
		Attachments:    uploads,
	}
}
