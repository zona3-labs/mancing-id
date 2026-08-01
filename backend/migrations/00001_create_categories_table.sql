-- +goose Up

CREATE TYPE category_status AS ENUM ('draft', 'active', 'retired');

CREATE TABLE categories (
    id          UUID        PRIMARY KEY,
    parent_id   UUID        REFERENCES categories(id) ON DELETE SET NULL,
    CONSTRAINT categories_no_self_parent CHECK (parent_id IS NULL OR parent_id <> id),
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) NOT NULL UNIQUE,
    status      category_status NOT NULL DEFAULT 'draft',
    version     BIGINT      NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_categories_parent_id  ON categories(parent_id);
CREATE INDEX idx_categories_status     ON categories(status);
CREATE INDEX idx_categories_deleted_at ON categories(deleted_at);

-- +goose Down
DROP INDEX IF EXISTS idx_categories_deleted_at;
DROP INDEX IF EXISTS idx_categories_status;
DROP INDEX IF EXISTS idx_categories_parent_id;
DROP TABLE IF EXISTS categories;
DROP TYPE IF EXISTS category_status;
