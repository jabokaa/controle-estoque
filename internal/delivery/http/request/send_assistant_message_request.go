package request

// SendAssistantMessageRequest valida o texto enviado no chat: POST /assistant/messages
// Os arquivos anexados são validados no handler (tipo real do arquivo e tamanho).
type SendAssistantMessageRequest struct {
	Message string `form:"message" label:"Mensagem" validate:"max=4000"`
}
