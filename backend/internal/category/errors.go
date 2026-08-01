package category

import "errors"

var (
	ErrCategoryNotFound        = errors.New("category not found")
	ErrCategoryVersionConflict = errors.New("category version conflict")
	ErrCategoryNotEditable     = errors.New("category is not editable")
	ErrCategoryHasChildren     = errors.New("category has children")
	ErrInvalidCategory         = errors.New("invalid category")
	ErrParentCategoryNotFound  = errors.New("parent category not found")
)
