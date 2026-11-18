package repository

import "strings"

// toEntities converte uma lista de models do banco numa lista de entidades do domínio,
// aplicando a função convert em cada item.
//
// Exemplo: toEntities(models, categoryModel.toEntity)
func toEntities[Model any, Entity any](models []Model, convert func(Model) Entity) []Entity {
	entities := make([]Entity, 0, len(models))
	for _, model := range models {
		entities = append(entities, convert(model))
	}
	return entities
}

// likeEscaper escapa os curingas do LIKE (% e _) para que sejam buscados como texto comum.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// escapeLike prepara um texto digitado pelo usuário para ir dentro de um LIKE ... ESCAPE '\'.
func escapeLike(text string) string {
	return likeEscaper.Replace(text)
}
