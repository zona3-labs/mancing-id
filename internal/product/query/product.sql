-- name: CreateProduct :exec
INSERT INTO products (id, name, slug, description, short_description, status, brand_id, is_featured, created_at, updated_at, deleted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW(), NULL);

-- name: GetAllProducts :many
SELECT id, name, slug, description, short_description, brand_id, status, is_featured, created_at, updated_at, deleted_at
FROM products
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetProductBySlug :one
SELECT id, name, slug, description, short_description, brand_id, status, is_featured, created_at, updated_at, deleted_at
FROM products
WHERE slug = $1 AND deleted_at IS NULL;

-- name: UpdateProduct :exec
UPDATE products
SET name = $2,
    slug = $3,
    description = $4,
    short_description = $5,
    status = $6,
    brand_id = $7,
    is_featured = $8,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteProduct :execresult
UPDATE products
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;
