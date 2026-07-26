package category

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID        uuid.UUID   `json:"id"`
	ParentID  *uuid.UUID  `json:"parent_id"`
	Name      string      `json:"name"`
	Slug      string      `json:"slug"`
	IsActive  bool        `json:"is_active"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	DeletedAt *time.Time  `json:"deleted_at,omitempty"`
	Children  []*Category `json:"children,omitempty"`
}
