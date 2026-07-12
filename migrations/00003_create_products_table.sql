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
    is_featured BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

-- +goose Down
DROP TABLE IF EXISTS products;
DROP TYPE IF EXISTS product_status;
