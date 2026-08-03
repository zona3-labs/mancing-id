package brand

import "errors"

var (
	ErrBrandNotFound        = errors.New("brand not found")
	ErrBrandVersionConflict = errors.New("brand version conflict")
	ErrBrandNotEditable     = errors.New("brand is not editable")
	ErrBrandHasProducts     = errors.New("brand has products")
	ErrInvalidBrand         = errors.New("invalid brand")
)
