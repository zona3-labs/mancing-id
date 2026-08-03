-- name: CreateBrand :one
INSERT INTO brands (id, name, slug, logo_path, status, version, created_at, updated_at, deleted_at)
VALUES ($1, $2, $3, $4, 'draft', 1, NOW(), NOW(), NULL)
RETURNING id, name, slug, logo_path, status, version, created_at, updated_at, deleted_at;

-- name: GetBrandByID :one
SELECT id, name, slug, logo_path, status, version, created_at, updated_at, deleted_at
FROM brands
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetAllBrands :many
SELECT id, name, slug, logo_path, status, version, created_at, updated_at, deleted_at
FROM brands
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetPublicBrands :many
SELECT id, name, slug, logo_path, status, version, created_at, updated_at, deleted_at
FROM brands
WHERE status IN ('active', 'inactive') AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetBrandBySlug :one
SELECT id, name, slug, logo_path, status, version, created_at, updated_at, deleted_at
FROM brands
WHERE slug = $1 AND deleted_at IS NULL;

-- name: GetPublicBrandBySlug :one
SELECT id, name, slug, logo_path, status, version, created_at, updated_at, deleted_at
FROM brands
WHERE slug = $1 AND status IN ('active', 'inactive') AND deleted_at IS NULL;

-- name: UpdateBrand :one
UPDATE brands
SET name = $2,
    slug = $3,
    logo_path = $4,
    version = version + 1,
    updated_at = NOW()
WHERE brands.id = $1
  AND status = 'draft'
  AND version = $5
  AND deleted_at IS NULL
RETURNING id, name, slug, logo_path, status, version, created_at, updated_at, deleted_at;

-- name: ActivateBrand :execresult
UPDATE brands
SET status = 'active', version = version + 1, updated_at = NOW()
WHERE id = $1 AND status = 'draft' AND version = $2 AND deleted_at IS NULL;

-- name: DeactivateBrand :execresult
UPDATE brands
SET status = 'inactive', version = version + 1, updated_at = NOW()
WHERE id = $1 AND status = 'active' AND version = $2 AND deleted_at IS NULL;

-- name: ReactivateBrand :execresult
UPDATE brands
SET status = 'active', version = version + 1, updated_at = NOW()
WHERE id = $1 AND status = 'inactive' AND version = $2 AND deleted_at IS NULL;

-- name: CountProductsByBrandID :one
SELECT COUNT(*)
FROM products
WHERE brand_id = $1;

-- name: DeleteBrand :execresult
UPDATE brands
SET deleted_at = NOW(), version = version + 1, updated_at = NOW()
WHERE brands.id = $1
  AND brands.status = 'draft'
  AND brands.version = $2
  AND brands.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM products WHERE products.brand_id = brands.id);
