-- -------------------------------------------------------
-- product_options
-- -------------------------------------------------------

-- name: CreateProductOption :one
INSERT INTO product_options (id, product_id, name, position, created_at, updated_at)
VALUES ($1, $2, $3, $4, NOW(), NOW())
RETURNING *;

-- name: GetProductOptionsByProductID :many
SELECT id, product_id, name, position, created_at, updated_at
FROM product_options
WHERE product_id = $1
ORDER BY position ASC, created_at ASC;

-- name: GetProductOptionByID :one
SELECT id, product_id, name, position, created_at, updated_at
FROM product_options
WHERE id = $1;

-- name: UpdateProductOption :one
UPDATE product_options
SET name       = $2,
    position   = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteProductOption :execresult
DELETE FROM product_options
WHERE id = $1;

-- -------------------------------------------------------
-- product_option_values
-- -------------------------------------------------------

-- name: CreateProductOptionValue :one
INSERT INTO product_option_values (id, product_option_id, value, position, created_at, updated_at)
VALUES ($1, $2, $3, $4, NOW(), NOW())
RETURNING *;

-- name: GetProductOptionValuesByOptionID :many
SELECT id, product_option_id, value, position, created_at, updated_at
FROM product_option_values
WHERE product_option_id = $1
ORDER BY position ASC, created_at ASC;

-- name: GetProductOptionValueByID :one
SELECT id, product_option_id, value, position, created_at, updated_at
FROM product_option_values
WHERE id = $1;

-- name: DeleteProductOptionValue :execresult
DELETE FROM product_option_values
WHERE id = $1;