package category

import "errors"

var (
	ErrCategoryNotFound       = errors.New("category not found")
	ErrCategoryHasChildren    = errors.New("category has active subcategories")
	ErrParentCategoryNotFound = errors.New("parent category not found")
)
