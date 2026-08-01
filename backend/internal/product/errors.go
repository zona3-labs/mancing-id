package product

import "errors"

var (
	ErrProductNotFound        = errors.New("product not found")
	ErrProductVersionConflict = errors.New("product version conflict")
	ErrOptionNotFound         = errors.New("option not found")
	ErrOptionValueNotFound    = errors.New("option value not found")
	ErrProductImageNotFound   = errors.New("product image not found")
	ErrVariantNotFound        = errors.New("variant not found")
	ErrSkuAlreadyExists       = errors.New("sku already exists")
)
