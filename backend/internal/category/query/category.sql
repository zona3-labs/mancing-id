-- name: CreateCategory :exec
INSERT INTO categories (id, parent_id,name, slug, is_active, created_at, updated_at, deleted_at)
VALUES ($1, $2, $3,$4,TRUE, NOW(), NOW(), NULL);

-- name: GetAllCategories :many
SELECT id, parent_id, name, slug, is_active, created_at, updated_at, deleted_at
FROM categories
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetCategoryBySlug :one
SELECT id, parent_id, name, slug, is_active, created_at, updated_at, deleted_at
FROM categories
WHERE slug = $1 AND deleted_at IS NULL;

-- name: GetCategoryTreeBySlug :many
WITH RECURSIVE category_tree AS (
    SELECT categories.id, categories.parent_id, categories.name, categories.slug, categories.is_active, categories.created_at, categories.updated_at, categories.deleted_at
    FROM categories
    WHERE categories.slug = $1 AND categories.deleted_at IS NULL

    UNION ALL

    SELECT c.id, c.parent_id, c.name, c.slug, c.is_active, c.created_at, c.updated_at, c.deleted_at
    FROM categories c
    INNER JOIN category_tree ct ON c.parent_id = ct.id
    WHERE c.deleted_at IS NULL
)
SELECT id, parent_id, name, slug, is_active, created_at, updated_at, deleted_at FROM category_tree;

-- name: UpdateCategory :exec
UPDATE categories
SET parent_id = $2,
    name = $3,
    slug = $4,
    is_active = $5,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteCategory :execresult
UPDATE categories
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: CheckActiveCategoriesByParentId :one
SELECT COUNT(*) FROM categories
WHERE parent_id = $1 AND deleted_at IS NULL;