-- +goose Up
CREATE TYPE product_status AS ENUM ('draft', 'active', 'archived');

CREATE TABLE products (
    id          UUID        PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) NOT NULL UNIQUE,
    description TEXT NULL,
    short_description VARCHAR(255) NULL,
    brand_id    UUID REFERENCES brands(id) ON DELETE SET NULL,
    status      product_status NOT NULL DEFAULT 'draft',
    version     BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    is_featured BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_products_brand_id    ON products(brand_id);
CREATE INDEX idx_products_status      ON products(status);
CREATE INDEX idx_products_is_featured ON products(is_featured);
CREATE INDEX idx_products_deleted_at  ON products(deleted_at);

-- +goose Down
DROP INDEX IF EXISTS idx_products_deleted_at;
DROP INDEX IF EXISTS idx_products_is_featured;
DROP INDEX IF EXISTS idx_products_status;
DROP INDEX IF EXISTS idx_products_brand_id;
DROP TABLE IF EXISTS products;
DROP TYPE IF EXISTS product_status;
