-- name: CreateBrand :exec
INSERT INTO brands (id, name, slug, logo_path, is_active, created_at, updated_at, deleted_at)
VALUES ($1, $2, $3, $4, TRUE, NOW(), NOW(), NULL);

-- name: GetAllBrands :many
SELECT id, name, slug, logo_path, is_active, created_at, updated_at, deleted_at
FROM brands
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetBrandBySlug :one
SELECT id, name, slug, logo_path, is_active, created_at, updated_at, deleted_at
FROM brands
WHERE slug = $1 AND deleted_at IS NULL;

-- name: UpdateBrand :execresult
UPDATE brands
SET name = $2,
    slug = $3,
    logo_path = $4,
    is_active = $5,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteBrand :execresult
UPDATE brands
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateBrandLogo :execresult
UPDATE brands
SET logo_path  = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;
