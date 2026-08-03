-- +goose Up

CREATE TYPE brand_status AS ENUM ('draft', 'active', 'inactive');

CREATE TABLE brands (
    id          UUID        PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) NOT NULL UNIQUE,
    logo_path   VARCHAR(255),
    status      brand_status NOT NULL DEFAULT 'draft',
    version     BIGINT      NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_brands_status     ON brands(status);
CREATE INDEX idx_brands_deleted_at ON brands(deleted_at);

-- +goose Down
DROP INDEX IF EXISTS idx_brands_deleted_at;
DROP INDEX IF EXISTS idx_brands_status;
DROP TABLE IF EXISTS brands;
DROP TYPE IF EXISTS brand_status;
