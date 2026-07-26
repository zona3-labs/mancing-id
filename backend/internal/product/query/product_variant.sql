-- name: CreateProductVariant :one
INSERT INTO product_variants (id, product_id, image_id, sku, price, stock, weight, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
RETURNING *;

-- name: BulkCreateProductVariantOptionValues :exec
INSERT INTO product_variant_option_values (variant_id, product_option_value_id)
SELECT $1::uuid, UNNEST($2::uuid[]);

-- name: GetProductVariantByID :many
SELECT
    pv.id, pv.product_id, pv.image_id, pv.sku, pv.price, pv.stock, pv.weight, pv.status, pv.created_at, pv.updated_at, pv.deleted_at,
    pov.id AS value_id, pov.product_option_id, pov.value AS value_text, pov.position AS value_position, pov.is_active AS value_is_active, pov.created_at AS value_created_at, pov.updated_at AS value_updated_at
FROM product_variants pv
LEFT JOIN product_variant_option_values pvov ON pvov.variant_id = pv.id
LEFT JOIN product_option_values pov ON pov.id = pvov.product_option_value_id AND pov.deleted_at IS NULL
WHERE pv.id = $1 AND pv.deleted_at IS NULL;

-- name: GetProductVariantsByProductID :many
SELECT
    pv.id, pv.product_id, pv.image_id, pv.sku, pv.price, pv.stock, pv.weight, pv.status, pv.created_at, pv.updated_at, pv.deleted_at,
    pov.id AS value_id, pov.product_option_id, pov.value AS value_text, pov.position AS value_position, pov.is_active AS value_is_active, pov.created_at AS value_created_at, pov.updated_at AS value_updated_at
FROM product_variants pv
LEFT JOIN product_variant_option_values pvov ON pvov.variant_id = pv.id
LEFT JOIN product_option_values pov ON pov.id = pvov.product_option_value_id AND pov.deleted_at IS NULL
WHERE pv.product_id = $1 AND pv.deleted_at IS NULL
ORDER BY pv.created_at DESC;

-- name: UpdateProductVariant :one
UPDATE product_variants
SET image_id = $2,
    sku = $3,
    price = $4,
    stock = $5,
    weight = $6,
    status = $7,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteProductVariant :execresult
UPDATE product_variants
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;
