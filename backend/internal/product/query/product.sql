-- name: CreateProduct :exec
INSERT INTO products (id, name, slug, description, short_description, status, brand_id, is_featured, created_at, updated_at, deleted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW(), NULL);

-- name: GetAllProducts :many
SELECT id, name, slug, description, short_description, brand_id, status, version, is_featured, created_at, updated_at, deleted_at
FROM products
WHERE deleted_at IS NULL
ORDER BY created_at DESC;


-- name: UpdateProduct :one
UPDATE products
SET name = $2,
    slug = $3,
    description = $4,
    short_description = $5,
    status = $6,
    brand_id = $7,
    is_featured = $8,
    version = version + 1,
    updated_at = NOW()
WHERE id = $1 AND version = $9 AND deleted_at IS NULL
RETURNING id, name, slug, description, short_description, brand_id, status, version, is_featured, created_at, updated_at, deleted_at;

-- name: DeleteProduct :execresult
UPDATE products
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetProductDetailBySlug :many
SELECT
    p.id,
    p.name,
    p.slug,
    p.description,
    p.short_description,
    p.brand_id,
    p.status,
    p.version,
    p.is_featured,
    p.created_at,
    p.updated_at,
    p.deleted_at,
    po.id          AS option_id,
    po.name        AS option_name,
    po.position    AS option_position,
    po.is_active   AS option_is_active,
    po.created_at  AS option_created_at,
    po.updated_at  AS option_updated_at,
    pov.id         AS value_id,
    pov.value      AS value_text,
    pov.position   AS value_position,
    pov.is_active  AS value_is_active,
    pov.created_at AS value_created_at,
    pov.updated_at AS value_updated_at
FROM products p
LEFT JOIN product_options po
    ON po.product_id = p.id
    AND po.deleted_at IS NULL
    AND po.is_active = TRUE
LEFT JOIN product_option_values pov
    ON pov.product_option_id = po.id
    AND pov.deleted_at IS NULL
    AND pov.is_active = TRUE
WHERE p.slug = $1
  AND p.deleted_at IS NULL
ORDER BY po.position ASC, po.created_at ASC, pov.position ASC, pov.created_at ASC;

-- name: GetProductDetailByID :many
SELECT
    p.id,
    p.name,
    p.slug,
    p.description,
    p.short_description,
    p.brand_id,
    p.status,
    p.version,
    p.is_featured,
    p.created_at,
    p.updated_at,
    p.deleted_at,
    po.id          AS option_id,
    po.name        AS option_name,
    po.position    AS option_position,
    po.is_active   AS option_is_active,
    po.created_at  AS option_created_at,
    po.updated_at  AS option_updated_at,
    pov.id         AS value_id,
    pov.value      AS value_text,
    pov.position   AS value_position,
    pov.is_active  AS value_is_active,
    pov.created_at AS value_created_at,
    pov.updated_at AS value_updated_at
FROM products p
LEFT JOIN product_options po
    ON po.product_id = p.id
    AND po.deleted_at IS NULL
    AND po.is_active = TRUE
LEFT JOIN product_option_values pov
    ON pov.product_option_id = po.id
    AND pov.deleted_at IS NULL
    AND pov.is_active = TRUE
WHERE p.id = $1
  AND p.deleted_at IS NULL
ORDER BY po.position ASC, po.created_at ASC, pov.position ASC, pov.created_at ASC;
