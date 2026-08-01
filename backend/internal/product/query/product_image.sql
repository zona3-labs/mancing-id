-- name: CreateProductImage :one
INSERT INTO product_images (id, product_id, url, alt_text, position, is_primary, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
RETURNING *;

-- name: GetProductImagesByProductID :many
SELECT id, product_id, url, alt_text, position, is_primary, created_at, updated_at
FROM product_images
WHERE product_id = $1
ORDER BY is_primary DESC, position ASC, created_at ASC;

-- name: GetProductImageByID :one
SELECT id, product_id, url, alt_text, position, is_primary, created_at, updated_at
FROM product_images
WHERE id = $1;

-- name: SetPrimaryImage :execresult
UPDATE product_images AS image
SET is_primary = CASE WHEN image.id = $2 THEN TRUE ELSE FALSE END,
    updated_at = NOW()
WHERE image.product_id = $1
  AND EXISTS (
      SELECT 1
      FROM product_images target
      WHERE target.id = $2 AND target.product_id = $1
  );

-- name: DeleteProductImage :execresult
DELETE FROM product_images
WHERE product_id = $1 AND id = $2;
