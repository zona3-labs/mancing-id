-- -------------------------------------------------------
-- product_options
-- -------------------------------------------------------

-- name: CreateProductOption :one
INSERT INTO product_options (id, product_id, name, position, is_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
RETURNING *;

-- name: GetProductOptionsByProductID :many
SELECT id, product_id, name, position, is_active, created_at, updated_at, deleted_at
FROM product_options
WHERE product_id = $1
  AND deleted_at IS NULL
ORDER BY position ASC, created_at ASC;

-- name: GetProductOptionByID :one
SELECT id, product_id, name, position, is_active, created_at, updated_at, deleted_at
FROM product_options
WHERE id = $1
  AND deleted_at IS NULL;

-- name: UpdateProductOption :one
UPDATE product_options
SET name       = $2,
    position   = $3,
    is_active  = $4,
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING *;

-- name: DeleteProductOption :execresult
UPDATE product_options
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;

-- -------------------------------------------------------
-- product_option_values
-- -------------------------------------------------------

-- name: CreateProductOptionValue :one
INSERT INTO product_option_values (id, product_option_id, value, position, is_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
RETURNING *;

-- name: GetProductOptionValuesByOptionID :many
SELECT id, product_option_id, value, position, is_active, created_at, updated_at, deleted_at
FROM product_option_values
WHERE product_option_id = $1
  AND deleted_at IS NULL
ORDER BY position ASC, created_at ASC;

-- name: GetProductOptionValueByID :one
SELECT id, product_option_id, value, position, is_active, created_at, updated_at, deleted_at
FROM product_option_values
WHERE id = $1
  AND deleted_at IS NULL;

-- name: DeleteProductOptionValue :execresult
UPDATE product_option_values
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;

-- name: BulkCreateProductOptionValues :many
INSERT INTO product_option_values (id, product_option_id, value, position, is_active, created_at, updated_at)
SELECT
    UNNEST($1::uuid[]),
    $2::uuid,
    UNNEST($3::varchar[]),
    UNNEST($4::smallint[]),
    UNNEST($5::boolean[]),
    NOW(),
    NOW()
RETURNING *;