package product

import "errors"

var (
	ErrProductNotFound        = errors.New("product not found")
	ErrProductVersionConflict = errors.New("product version conflict")
	ErrProductNotEditable     = errors.New("product is not editable")
	ErrProductDraftRequired   = errors.New("product must be a draft")
	ErrProductBrandRequired   = errors.New("product brand is required")
	ErrProductBrandNotActive  = errors.New("product brand is not active")
	ErrInvalidProduct         = errors.New("invalid product")
	ErrOptionNotFound         = errors.New("option not found")
	ErrOptionValueNotFound    = errors.New("option value not found")
	ErrProductImageNotFound   = errors.New("product image not found")
	ErrVariantNotFound        = errors.New("variant not found")
	ErrSkuAlreadyExists       = errors.New("sku already exists")
)
