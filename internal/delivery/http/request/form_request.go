// Package request é o equivalente aos FormRequests do Laravel (app/Http/Requests):
// cada request é uma struct que declara os campos esperados e as regras de validação nas tags.
//
//	type ListCategoriesRequest struct {
//	    Name string `form:"name" label:"Nome" validate:"max=100"`
//	}
//
//	form     → nome do campo na query string ou no formulário
//	label    → nome do campo nas mensagens de erro
//	validate → regras (https://pkg.go.dev/github.com/go-playground/validator/v10)
//
// O handler chama Parse, que faz o papel do $request->validated() do Laravel.
package request

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/locales/pt_BR"
	universalTranslator "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	ptBRTranslations "github.com/go-playground/validator/v10/translations/pt_BR"
)

// Defaulter é implementado pelos requests que preenchem valores padrão antes de validar
// (como o prepareForValidation do Laravel).
type Defaulter interface {
	SetDefaults()
}

// InvalidFieldResetter é implementado pelos requests que, em vez de recusar um valor inválido,
// trocam esse valor pelo padrão (ex.: filtros de listagem: ?page=abc vira página 1).
// Formulários NÃO implementam: o valor digitado precisa voltar para a tela junto com o erro.
type InvalidFieldResetter interface {
	ResetsInvalidFields()
}

// ValidationErrors guarda a mensagem de erro de cada campo inválido, pelo nome do campo no formulário.
type ValidationErrors map[string]string

// Has diz se o campo tem erro. Uso no template: {{if .Errors.Has "name"}}
func (validationErrors ValidationErrors) Has(field string) bool {
	_, exists := validationErrors[field]
	return exists
}

// Get devolve a mensagem de erro do campo. Uso no template: {{.Errors.Get "name"}}
func (validationErrors ValidationErrors) Get(field string) string {
	return validationErrors[field]
}

var (
	validate   = validator.New(validator.WithRequiredStructEnabled())
	translator universalTranslator.Translator
)

func init() {
	portuguese := pt_BR.New()
	translator, _ = universalTranslator.New(portuguese, portuguese).GetTranslator("pt_BR")
	if err := ptBRTranslations.RegisterDefaultTranslations(validate, translator); err != nil {
		panic(err)
	}

	// Troca "deve ser um de [asc desc]" por "deve ser um destes valores: asc, desc".
	err := validate.RegisterTranslation("oneof", translator,
		func(translation universalTranslator.Translator) error {
			return translation.Add("oneof", "{0} deve ser um destes valores: {1}", true)
		},
		func(translation universalTranslator.Translator, fieldError validator.FieldError) string {
			message, _ := translation.T("oneof", fieldError.Field(), strings.ReplaceAll(fieldError.Param(), " ", ", "))
			return message
		},
	)
	if err != nil {
		panic(err)
	}

	// Nas mensagens, usa o label do campo ("Nome") em vez do nome Go ("Name").
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		if label := field.Tag.Get("label"); label != "" {
			return label
		}
		return field.Name
	})
}

// Parse preenche target (ponteiro para uma struct de request) com os dados da requisição,
// aplica os valores padrão e valida.
//
// Devolve nil quando está tudo certo. Quando há erros:
//   - por padrão, os valores digitados continuam na struct, para reexibir o formulário (como o old() do Laravel);
//   - se o request implementar InvalidFieldResetter, os campos inválidos voltam ao valor padrão,
//     e o request continua utilizável (ex.: listar com a ordenação padrão e avisar o usuário).
func Parse(httpRequest *http.Request, target any) ValidationErrors {
	_, resetsInvalidFields := target.(InvalidFieldResetter)

	// Trata o parsing do formulário.
	if err := httpRequest.ParseForm(); err != nil {
		return ValidationErrors{"_form": "Não foi possível ler os dados enviados."}
	}

	// Pega o tipo e valor da struct de destino.
	targetValue := reflect.ValueOf(target).Elem()
	targetType := targetValue.Type()
	validationErrors := ValidationErrors{}

	// 1. Bind: copia cada campo da requisição para a struct, convertendo o tipo.
	for index := 0; index < targetType.NumField(); index++ {
		// Pega o campo da struct e o valor enviado no formulário.
		field := targetType.Field(index)
		formName := field.Tag.Get("form")
		rawValue := strings.TrimSpace(httpRequest.Form.Get(formName))
		if formName == "" || rawValue == "" {
			continue
		}

		// Pega o tipo e valor do campo da struct.
		fieldValue := targetValue.Field(index)
		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(rawValue)
		case reflect.Int:
			number, err := strconv.Atoi(rawValue)
			if err != nil {
				validationErrors[formName] = fmt.Sprintf("%s deve ser um número.", labelOf(field))
				continue
			}
			fieldValue.SetInt(int64(number))
		}
	}

	// 2. Valores padrão para o que não veio.
	applyDefaults(target)

	// 3. Validação pelas tags validate.
	if err := validate.Struct(target); err != nil {
		var fieldErrors validator.ValidationErrors
		if !errors.As(err, &fieldErrors) {
			return ValidationErrors{"_form": err.Error()}
		}
		for _, fieldError := range fieldErrors {
			structField, _ := targetType.FieldByName(fieldError.StructField())
			validationErrors[structField.Tag.Get("form")] = fieldError.Translate(translator) + "."
			if resetsInvalidFields {
				targetValue.FieldByName(fieldError.StructField()).SetZero()
			}
		}
	}

	if len(validationErrors) == 0 {
		return nil
	}
	if resetsInvalidFields {
		applyDefaults(target) // devolve o valor padrão aos campos que foram zerados por serem inválidos
	}
	return validationErrors
}

func applyDefaults(target any) {
	if defaulter, hasDefaults := target.(Defaulter); hasDefaults {
		defaulter.SetDefaults()
	}
}

func labelOf(field reflect.StructField) string {
	if label := field.Tag.Get("label"); label != "" {
		return label
	}
	return field.Name
}
