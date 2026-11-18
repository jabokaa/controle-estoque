package request

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Conteúdos mínimos que o Go reconhece pelo começo do arquivo (assinatura).
var (
	pngContent = []byte("\x89PNG\r\n\x1a\n" + "resto da imagem")
	pdfContent = []byte("%PDF-1.4\n" + "resto do pdf")
)

type testFile struct {
	name    string
	content []byte
}

// newMultipartRequest monta um POST multipart como o navegador envia.
func newMultipartRequest(t *testing.T, message string, files ...testFile) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("message", message); err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		part, err := writer.CreateFormFile("attachments", file.name)
		if err != nil {
			t.Fatal(err)
		}
		part.Write(file.content)
	}
	writer.Close()

	httpRequest := httptest.NewRequest(http.MethodPost, "/assistant/messages", &body)
	httpRequest.Header.Set("Content-Type", writer.FormDataContentType())
	return httpRequest
}

func TestValidMessageWithAttachments(t *testing.T) {
	var form SendAssistantMessageRequest
	errors := Parse(newMultipartRequest(t, "Dê entrada nesta nota",
		testFile{"nota.png", pngContent},
		testFile{"nota.pdf", pdfContent},
	), &form)
	if errors != nil {
		t.Fatalf("não deveria ter erros: %v", errors)
	}
	if form.Message != "Dê entrada nesta nota" || len(form.Attachments) != 2 {
		t.Fatalf("form = %+v", form)
	}
	if form.Attachments[0].MediaType != "image/png" || form.Attachments[1].MediaType != "application/pdf" {
		t.Fatalf("tipos detectados: %s, %s", form.Attachments[0].MediaType, form.Attachments[1].MediaType)
	}

	input := form.ToInput("conversa-1")
	if input.ConversationID != "conversa-1" || len(input.Attachments) != 2 || input.Attachments[0].FileName != "nota.png" {
		t.Fatalf("input = %+v", input)
	}
}

func TestMessageWithoutAttachments(t *testing.T) {
	var form SendAssistantMessageRequest
	if errors := Parse(newMultipartRequest(t, "Qual o total em estoque?"), &form); errors != nil {
		t.Fatalf("não deveria ter erros: %v", errors)
	}
	if len(form.Attachments) != 0 {
		t.Fatalf("anexos = %d", len(form.Attachments))
	}
}

func TestFileTypeIsCheckedByContentNotByName(t *testing.T) {
	var form SendAssistantMessageRequest
	errors := Parse(newMultipartRequest(t, "oi", testFile{"disfarcado.png", []byte("só texto, não é imagem")}), &form)
	expected := "O arquivo “disfarcado.png” não é de um tipo aceito (JPG, PNG, GIF, WEBP, PDF)."
	if got := errors.Get("attachments"); got != expected {
		t.Fatalf("erro = %q, esperado %q", got, expected)
	}
}

func TestFileLargerThanTheLimitIsRejected(t *testing.T) {
	big := append([]byte(nil), pngContent...)
	big = append(big, bytes.Repeat([]byte("x"), 5<<20)...)

	var form SendAssistantMessageRequest
	errors := Parse(newMultipartRequest(t, "oi", testFile{"grande.png", big}), &form)
	if got := errors.Get("attachments"); got != "O arquivo “grande.png” passa de 5 MB." {
		t.Fatalf("erro = %q", got)
	}
}

func TestTooManyFilesAreRejected(t *testing.T) {
	var form SendAssistantMessageRequest
	errors := Parse(newMultipartRequest(t, "oi",
		testFile{"1.png", pngContent}, testFile{"2.png", pngContent},
		testFile{"3.png", pngContent}, testFile{"4.png", pngContent},
	), &form)
	if got := errors.Get("attachments"); !strings.Contains(got, "Anexos") || !strings.Contains(got, "3") {
		t.Fatalf("erro = %q", got)
	}
}

func TestLongMessageAndBadFileAreReportedTogether(t *testing.T) {
	var form SendAssistantMessageRequest
	errors := Parse(newMultipartRequest(t, strings.Repeat("a", 4001), testFile{"x.txt", []byte("texto")}), &form)
	if !errors.Has("message") || !errors.Has("attachments") {
		t.Fatalf("esperado erro nos dois campos: %v", errors)
	}
	summary := errors.Summary()
	if !strings.Contains(summary, "x.txt") || !strings.Contains(summary, "Mensagem") {
		t.Fatalf("resumo = %q", summary)
	}
}
