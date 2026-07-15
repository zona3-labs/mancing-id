package product

import "errors"

var (
	ErrProductNotFound     = errors.New("product not found")
	ErrOptionNotFound      = errors.New("option not found")
	ErrOptionValueNotFound = errors.New("option value not found")
)
