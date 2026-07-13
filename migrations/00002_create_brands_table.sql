-- +goose Up

CREATE TABLE brands (
    id          UUID        PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) NOT NULL UNIQUE,
    logo_path   VARCHAR(255),
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_brands_is_active  ON brands(is_active);
CREATE INDEX idx_brands_deleted_at ON brands(deleted_at);

-- +goose Down
DROP INDEX IF EXISTS idx_brands_deleted_at;
DROP INDEX IF EXISTS idx_brands_is_active;
DROP TABLE IF EXISTS brands;
