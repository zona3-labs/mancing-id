-- +goose Up

CREATE TABLE categories (
    id          UUID        PRIMARY KEY,
    parent_id   UUID        REFERENCES categories(id) ON DELETE SET NULL,
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) NOT NULL UNIQUE,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_categories_parent_id  ON categories(parent_id);
CREATE INDEX idx_categories_is_active  ON categories(is_active);
CREATE INDEX idx_categories_deleted_at ON categories(deleted_at);

-- +goose Down
DROP INDEX IF EXISTS idx_categories_deleted_at;
DROP INDEX IF EXISTS idx_categories_is_active;
DROP INDEX IF EXISTS idx_categories_parent_id;
DROP TABLE IF EXISTS categories;
