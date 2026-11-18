-- +goose Up
CREATE TABLE categorias (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    nome        TEXT      NOT NULL UNIQUE,
    descricao   TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE categorias;
