-- +goose Up
CREATE TABLE stock_movements (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    item_id       INTEGER NOT NULL REFERENCES items (id) ON DELETE RESTRICT,
    type          TEXT    NOT NULL CHECK (type IN ('in', 'out')), -- in = entrada, out = saída (baixa)
    quantity      INTEGER NOT NULL CHECK (quantity > 0),
    balance_after INTEGER NOT NULL CHECK (balance_after >= 0),    -- saldo do item depois da movimentação
    note          TEXT,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_stock_movements_item_id ON stock_movements (item_id, created_at);

-- +goose Down
DROP TABLE stock_movements;
