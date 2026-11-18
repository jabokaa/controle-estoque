package request

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"

	universalTranslator "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

// UploadedFile é um arquivo enviado num formulário multipart. Um campo []UploadedFile
// com a tag form é preenchido pelo Parse, como os campos de texto:
//
//	Attachments []UploadedFile `form:"attachments" label:"Anexos" validate:"max=3,dive,file_types=image/png application/pdf,file_max_size=5242880"`
//
// Regras para cada arquivo (depois do dive):
//
//	file_types=<tipos separados por espaço>  tipo real do arquivo, detectado pelo conteúdo
//	file_max_size=<bytes>                     tamanho máximo
//
// Equivale ao 'attachments.*' => 'file|mimes:...|max:...' do Laravel.
type UploadedFile struct {
	Name      string
	MediaType string // detectado pelo conteúdo, não pela extensão nem pelo que o navegador informa
	Size      int64
	Data      []byte
}

var uploadedFilesType = reflect.TypeOf([]UploadedFile(nil))

// maxMultipartMemory é quanto do formulário multipart fica em memória; o resto vai para disco temporário.
const maxMultipartMemory = 32 << 20

// mediaTypeLabels são os nomes amigáveis dos tipos, para as mensagens de erro.
var mediaTypeLabels = map[string]string{
	"image/jpeg":      "JPG",
	"image/png":       "PNG",
	"image/gif":       "GIF",
	"image/webp":      "WEBP",
	"application/pdf": "PDF",
}

// readUploadedFiles lê os arquivos do campo e detecta o tipo real de cada um.
func readUploadedFiles(httpRequest *http.Request, formName string) ([]UploadedFile, error) {
	if httpRequest.MultipartForm == nil {
		return nil, nil
	}
	headers := httpRequest.MultipartForm.File[formName]
	files := make([]UploadedFile, 0, len(headers))
	for _, header := range headers {
		opened, err := header.Open()
		if err != nil {
			return nil, fmt.Errorf("não foi possível ler o arquivo “%s”", header.Filename)
		}
		data, err := io.ReadAll(opened)
		opened.Close()
		if err != nil {
			return nil, fmt.Errorf("não foi possível ler o arquivo “%s”", header.Filename)
		}
		files = append(files, UploadedFile{
			Name:      header.Filename,
			MediaType: http.DetectContentType(data),
			Size:      header.Size,
			Data:      data,
		})
	}
	return files, nil
}

// registerFileRules registra as regras file_types e file_max_size e as mensagens delas.
func registerFileRules() error {
	if err := validate.RegisterValidation("file_types", func(field validator.FieldLevel) bool {
		file, isFile := field.Field().Interface().(UploadedFile)
		return isFile && slices.Contains(strings.Fields(field.Param()), file.MediaType)
	}); err != nil {
		return err
	}

	if err := validate.RegisterValidation("file_max_size", func(field validator.FieldLevel) bool {
		file, isFile := field.Field().Interface().(UploadedFile)
		maxSize, err := strconv.ParseInt(field.Param(), 10, 64)
		return isFile && err == nil && file.Size <= maxSize
	}); err != nil {
		return err
	}

	// As mensagens citam o nome do arquivo, que vem do próprio valor validado.
	noTemplate := func(universalTranslator.Translator) error { return nil }

	if err := validate.RegisterTranslation("file_types", translator, noTemplate,
		func(_ universalTranslator.Translator, fieldError validator.FieldError) string {
			var accepted []string
			for _, mediaType := range strings.Fields(fieldError.Param()) {
				if label, known := mediaTypeLabels[mediaType]; known {
					mediaType = label
				}
				accepted = append(accepted, mediaType)
			}
			return fmt.Sprintf("O arquivo “%s” não é de um tipo aceito (%s)", fileName(fieldError), strings.Join(accepted, ", "))
		}); err != nil {
		return err
	}

	return validate.RegisterTranslation("file_max_size", translator, noTemplate,
		func(_ universalTranslator.Translator, fieldError validator.FieldError) string {
			maxSize, _ := strconv.ParseInt(fieldError.Param(), 10, 64)
			return fmt.Sprintf("O arquivo “%s” passa de %d MB", fileName(fieldError), maxSize>>20)
		})
}

func fileName(fieldError validator.FieldError) string {
	if file, isFile := fieldError.Value().(UploadedFile); isFile {
		return file.Name
	}
	return "enviado"
}
