package repository

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
