-- name: CreateCategory :one
INSERT INTO categories (id, parent_id, name, slug, status, version, created_at, updated_at, deleted_at)
VALUES ($1, $2, $3, $4, 'draft', 1, NOW(), NOW(), NULL)
RETURNING id, parent_id, name, slug, status, version, created_at, updated_at, deleted_at;

-- name: GetAllCategories :many
SELECT id, parent_id, name, slug, status, version, created_at, updated_at, deleted_at
FROM categories
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetActiveCategories :many
WITH RECURSIVE active_categories AS (
    SELECT id, parent_id, name, slug, status, version, created_at, updated_at, deleted_at, ARRAY[id] AS path
    FROM categories
    WHERE parent_id IS NULL AND status = 'active' AND deleted_at IS NULL

    UNION ALL

    SELECT c.id, c.parent_id, c.name, c.slug, c.status, c.version, c.created_at, c.updated_at, c.deleted_at, parent.path || c.id
    FROM categories c
    INNER JOIN active_categories parent ON c.parent_id = parent.id
    WHERE c.status = 'active' AND c.deleted_at IS NULL AND NOT c.id = ANY(parent.path)
)
SELECT id, parent_id, name, slug, status, version, created_at, updated_at, deleted_at
FROM active_categories
ORDER BY created_at;

-- name: GetCategoryByID :one
SELECT id, parent_id, name, slug, status, version, created_at, updated_at, deleted_at
FROM categories
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetCategoryBySlug :one
SELECT id, parent_id, name, slug, status, version, created_at, updated_at, deleted_at
FROM categories
WHERE slug = $1 AND deleted_at IS NULL;

-- name: GetCategoryTreeBySlug :many
WITH RECURSIVE category_tree AS (
    SELECT categories.id, categories.parent_id, categories.name, categories.slug, categories.status, categories.version, categories.created_at, categories.updated_at, categories.deleted_at, 0 AS depth, ARRAY[categories.id] AS path
    FROM categories
    WHERE categories.slug = $1 AND categories.deleted_at IS NULL

    UNION ALL

    SELECT c.id, c.parent_id, c.name, c.slug, c.status, c.version, c.created_at, c.updated_at, c.deleted_at, ct.depth + 1, ct.path || c.id
    FROM categories c
    INNER JOIN category_tree ct ON c.parent_id = ct.id
    WHERE c.deleted_at IS NULL AND NOT c.id = ANY(ct.path)
)
SELECT id, parent_id, name, slug, status, version, created_at, updated_at, deleted_at
FROM category_tree
ORDER BY depth, created_at;

-- name: GetActiveCategoryTreeBySlug :many
WITH RECURSIVE category_tree AS (
    SELECT categories.id, categories.parent_id, categories.name, categories.slug, categories.status, categories.version, categories.created_at, categories.updated_at, categories.deleted_at, 0 AS depth, ARRAY[categories.id] AS path
    FROM categories
    WHERE categories.slug = $1 AND categories.status = 'active' AND categories.deleted_at IS NULL

    UNION ALL

    SELECT c.id, c.parent_id, c.name, c.slug, c.status, c.version, c.created_at, c.updated_at, c.deleted_at, ct.depth + 1, ct.path || c.id
    FROM categories c
    INNER JOIN category_tree ct ON c.parent_id = ct.id
    WHERE c.status = 'active' AND c.deleted_at IS NULL AND NOT c.id = ANY(ct.path)
)
SELECT id, parent_id, name, slug, status, version, created_at, updated_at, deleted_at
FROM category_tree
ORDER BY depth, created_at;

-- name: UpdateCategory :one
WITH RECURSIVE ancestors AS (
    SELECT id, parent_id, ARRAY[id] AS path
    FROM categories
    WHERE id = $2 AND $2 IS NOT NULL AND deleted_at IS NULL

    UNION ALL

    SELECT parent.id, parent.parent_id, child.path || parent.id
    FROM categories parent
    INNER JOIN ancestors child ON parent.id = child.parent_id
    WHERE parent.deleted_at IS NULL AND NOT parent.id = ANY(child.path)
)
UPDATE categories AS category
SET parent_id = $2,
    name = $3,
    slug = $4,
    version = version + 1,
    updated_at = NOW()
WHERE category.id = $1
  AND category.status = 'draft'
  AND category.version = $5
  AND category.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM ancestors WHERE ancestors.id = $1)
RETURNING category.id, category.parent_id, category.name, category.slug, category.status, category.version, category.created_at, category.updated_at, category.deleted_at;

-- name: DeleteCategory :execresult
UPDATE categories AS category
SET deleted_at = NOW(),
    version = version + 1,
    updated_at = NOW()
WHERE category.id = $1
  AND category.status = 'draft'
  AND category.version = $2
  AND category.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1
      FROM categories child
      WHERE child.parent_id = category.id
        AND child.deleted_at IS NULL
  );

-- name: ActivateCategory :execresult
WITH RECURSIVE ancestors AS (
    SELECT parent.id, parent.parent_id, parent.status, parent.deleted_at, ARRAY[parent.id] AS path
    FROM categories category
    INNER JOIN categories parent ON parent.id = category.parent_id
    WHERE category.id = $1

    UNION ALL

    SELECT parent.id, parent.parent_id, parent.status, parent.deleted_at, child.path || parent.id
    FROM categories parent
    INNER JOIN ancestors child ON parent.id = child.parent_id
    WHERE NOT parent.id = ANY(child.path)
)
UPDATE categories AS category
SET status = 'active',
    version = version + 1,
    updated_at = NOW()
WHERE category.id = $1
  AND category.status = 'draft'
  AND category.version = $2
  AND category.deleted_at IS NULL
  AND (
      category.parent_id IS NULL
      OR NOT EXISTS (SELECT 1 FROM ancestors WHERE ancestors.status <> 'active' OR ancestors.deleted_at IS NOT NULL)
  );

-- name: RetireCategory :execresult
WITH RECURSIVE descendants AS (
    SELECT child.id, child.status, child.deleted_at, ARRAY[child.id] AS path
    FROM categories child
    WHERE child.parent_id = $1

    UNION ALL

    SELECT child.id, child.status, child.deleted_at, parent.path || child.id
    FROM categories child
    INNER JOIN descendants parent ON child.parent_id = parent.id
    WHERE NOT child.id = ANY(parent.path)
)
UPDATE categories AS category
SET status = 'retired',
    version = version + 1,
    updated_at = NOW()
WHERE category.id = $1
  AND category.status = 'active'
  AND category.version = $2
  AND category.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM descendants WHERE descendants.deleted_at IS NULL AND descendants.status <> 'retired');


-- name: CheckCategoriesByParentID :one
SELECT COUNT(*) FROM categories
WHERE parent_id = $1 AND deleted_at IS NULL;

-- name: CheckNonRetiredCategoriesByParentID :one
WITH RECURSIVE descendants AS (
    SELECT category.id, category.status, category.deleted_at, ARRAY[category.id] AS path
    FROM categories category
    WHERE category.parent_id = $1

    UNION ALL

    SELECT child.id, child.status, child.deleted_at, parent.path || child.id
    FROM categories child
    INNER JOIN descendants parent ON child.parent_id = parent.id
    WHERE NOT child.id = ANY(parent.path)
)
SELECT COUNT(*) FROM descendants
WHERE deleted_at IS NULL AND status <> 'retired';
