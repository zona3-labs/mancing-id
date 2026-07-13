-- +goose Up

-- -------------------------------------------------------
-- product_options
-- Defines the option "types" that belong to a product.
-- e.g. product A has options: "Size", "Color"
-- -------------------------------------------------------
CREATE TABLE product_options (
    id          UUID         PRIMARY KEY,
    product_id  UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,           -- e.g. "Size", "Color", "Material"
    position    SMALLINT     NOT NULL DEFAULT 0, -- display order
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- -------------------------------------------------------
-- product_option_values
-- Defines the possible values for each option.
-- e.g. "Size" → "S", "M", "L", "XL"
--      "Color" → "Red", "Blue", "Black"
-- -------------------------------------------------------
CREATE TABLE product_option_values (
    id                UUID         PRIMARY KEY,
    product_option_id UUID         NOT NULL REFERENCES product_options(id) ON DELETE CASCADE,
    value             VARCHAR(100) NOT NULL, -- e.g. "S", "Red", "Cotton"
    position          SMALLINT     NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    UNIQUE (product_option_id, value)
);

-- -------------------------------------------------------
-- product_images
-- Stores images that belong to a product.
-- One product can have many images.
-- is_primary marks the hero/thumbnail image.
-- -------------------------------------------------------
CREATE TABLE product_images (
    id          UUID         PRIMARY KEY,
    product_id  UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url         TEXT         NOT NULL,
    alt_text    VARCHAR(255) NULL,
    position    SMALLINT     NOT NULL DEFAULT 0,  -- display order
    is_primary  BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Only one image per product may be marked as primary
CREATE UNIQUE INDEX idx_product_images_primary
    ON product_images(product_id)
    WHERE is_primary = TRUE;

-- -------------------------------------------------------
-- product_variants
-- A concrete, purchasable variant of a product.
-- Each row represents a specific combination of option
-- values (e.g. Size=M + Color=Red).
-- image_id optionally links to one of the product's images
-- as a variant-specific swatch.
-- -------------------------------------------------------
CREATE TYPE variant_status AS ENUM ('draft', 'active', 'archived');

CREATE TABLE product_variants (
    id          UUID            PRIMARY KEY,
    product_id  UUID            NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    image_id    UUID            NULL REFERENCES product_images(id) ON DELETE SET NULL,
    sku         VARCHAR(100)    NOT NULL UNIQUE,
    price       NUMERIC(12, 2)  NOT NULL,
    stock       INT             NOT NULL DEFAULT 0,
    weight      NUMERIC(8, 2)   NULL,        -- in grams, optional
    status      variant_status  NOT NULL DEFAULT 'draft',
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

-- -------------------------------------------------------
-- product_variant_option_values
-- Junction table that links each variant to its selected
-- option values.
-- e.g. variant X → (Size=M), (Color=Red)
-- -------------------------------------------------------
CREATE TABLE product_variant_option_values (
    variant_id              UUID NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    product_option_value_id UUID NOT NULL REFERENCES product_option_values(id) ON DELETE CASCADE,

    PRIMARY KEY (variant_id, product_option_value_id)
);

-- Indexes for common query patterns
CREATE INDEX idx_product_options_product_id      ON product_options(product_id);
CREATE INDEX idx_product_option_values_option_id ON product_option_values(product_option_id);
CREATE INDEX idx_product_images_product_id       ON product_images(product_id);
CREATE INDEX idx_product_variants_product_id     ON product_variants(product_id);
CREATE INDEX idx_product_variants_sku            ON product_variants(sku);
CREATE INDEX idx_product_variants_image_id       ON product_variants(image_id);
CREATE INDEX idx_pvov_variant_id                 ON product_variant_option_values(variant_id);
CREATE INDEX idx_pvov_option_value_id            ON product_variant_option_values(product_option_value_id);

-- +goose Down
DROP TABLE IF EXISTS product_variant_option_values;
DROP TABLE IF EXISTS product_variants;
DROP TABLE IF EXISTS product_images;
DROP TABLE IF EXISTS product_option_values;
DROP TABLE IF EXISTS product_options;
DROP TYPE  IF EXISTS variant_status;
