package category

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zona3-labs/mancing-id/internal/response"
)

type CategoryHandler struct {
	usecase CategoryUsecase
}

func NewCategoryHandler(usecase CategoryUsecase) *CategoryHandler {
	return &CategoryHandler{usecase: usecase}
}

func (h *CategoryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	categories := rg.Group("/categories")
	categories.GET("", h.GetPublicCategories)
	categories.GET("/:identifier", h.GetPublicCategory)
}

func (h *CategoryHandler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	categories := rg.Group("/admin/categories")
	categories.POST("", h.CreateCategory)
	categories.GET("", h.GetAllCategories)
	categories.GET("/:identifier", h.GetCategory)
	categories.PUT("/:id", h.UpdateCategory)
	categories.POST("/:id/activate", h.ActivateCategory)
	categories.POST("/:id/retire", h.RetireCategory)
	categories.DELETE("/:id", h.DeleteCategory)
}

type createCategoryRequest struct {
	Name     string     `json:"name" binding:"required"`
	Slug     string     `json:"slug"`
	ParentID *uuid.UUID `json:"parent_id"`
}

type updateCategoryRequest struct {
	Name             string     `json:"name" binding:"required"`
	Slug             string     `json:"slug"`
	ParentID         *uuid.UUID `json:"parent_id"`
	Version          *int64     `json:"version"`
	parentIDProvided bool
}

func (r *updateCategoryRequest) UnmarshalJSON(data []byte) error {
	type request updateCategoryRequest
	var decoded request
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*r = updateCategoryRequest(decoded)
	_, r.parentIDProvided = fields["parent_id"]
	return nil
}

type deleteCategoryRequest struct {
	Version *int64 `json:"version"`
}

func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Problem(c, http.StatusBadRequest, "invalid_request", "Invalid Request", "category name is required")
		return
	}

	category := &Category{Name: req.Name, Slug: req.Slug, ParentID: req.ParentID}
	if err := h.usecase.CreateCategory(c.Request.Context(), category); err != nil {
		h.writeError(c, err)
		return
	}
	response.Created(c, category)
}

func (h *CategoryHandler) GetAllCategories(c *gin.Context) {
	categories, err := h.usecase.GetAllCategories(c.Request.Context())
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, "categories retrieved successfully", categories)
}

func (h *CategoryHandler) GetPublicCategories(c *gin.Context) {
	categories, err := h.usecase.GetActiveCategories(c.Request.Context())
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, "categories retrieved successfully", categories)
}

func (h *CategoryHandler) GetCategory(c *gin.Context) {
	identifier := c.Param("identifier")
	var category *Category
	var err error
	if id, parseErr := uuid.Parse(identifier); parseErr == nil {
		category, err = h.usecase.GetCategoryByID(c.Request.Context(), id)
	} else {
		category, err = h.usecase.GetCategoryBySlug(c.Request.Context(), identifier)
	}
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.OK(c, "category retrieved successfully", category)
}

func (h *CategoryHandler) GetPublicCategory(c *gin.Context) {
	identifier := c.Param("identifier")
	if _, err := uuid.Parse(identifier); err == nil {
		h.writeError(c, ErrCategoryNotFound)
		return
	}

	category, err := h.usecase.GetActiveCategoryBySlug(c.Request.Context(), identifier)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.OK(c, "category retrieved successfully", category)
}

func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Problem(c, http.StatusBadRequest, "invalid_category_id", "Invalid Category ID", "category id must be a UUID")
		return
	}

	var req updateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Version == nil {
		response.Problem(c, http.StatusBadRequest, "category_version_required", "Version Required", "category version is required")
		return
	}

	parentID := req.ParentID
	if !req.parentIDProvided {
		current, getErr := h.usecase.GetCategoryByID(c.Request.Context(), id)
		if getErr != nil {
			h.writeError(c, getErr)
			return
		}
		parentID = current.ParentID
	}
	category := &Category{ID: id, Name: req.Name, Slug: req.Slug, ParentID: parentID}
	if err := h.usecase.UpdateCategory(c.Request.Context(), category, *req.Version); err != nil {
		h.writeError(c, err)
		return
	}
	response.OK(c, "category updated successfully", category)
}

func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Problem(c, http.StatusBadRequest, "invalid_category_id", "Invalid Category ID", "category id must be a UUID")
		return
	}

	var req deleteCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Version == nil {
		response.Problem(c, http.StatusBadRequest, "category_version_required", "Version Required", "category version is required")
		return
	}

	if err := h.usecase.DeleteCategory(c.Request.Context(), id, *req.Version); err != nil {
		h.writeError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *CategoryHandler) ActivateCategory(c *gin.Context) {
	h.transitionCategory(c, func(id uuid.UUID, version int64) error {
		return h.usecase.ActivateCategory(c.Request.Context(), id, version)
	})
}

func (h *CategoryHandler) RetireCategory(c *gin.Context) {
	h.transitionCategory(c, func(id uuid.UUID, version int64) error {
		return h.usecase.RetireCategory(c.Request.Context(), id, version)
	})
}

func (h *CategoryHandler) transitionCategory(c *gin.Context, transition func(uuid.UUID, int64) error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Problem(c, http.StatusBadRequest, "invalid_category_id", "Invalid Category ID", "category id must be a UUID")
		return
	}

	var req deleteCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Version == nil {
		response.Problem(c, http.StatusBadRequest, "category_version_required", "Version Required", "category version is required")
		return
	}
	if err := transition(id, *req.Version); err != nil {
		h.writeError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *CategoryHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrCategoryNotFound):
		response.Problem(c, http.StatusNotFound, "category_not_found", "Category Not Found", "the category does not exist")
	case errors.Is(err, ErrParentCategoryNotFound):
		response.Problem(c, http.StatusNotFound, "category_parent_not_found", "Parent Category Not Found", "the parent category does not exist")
	case errors.Is(err, ErrCategoryVersionConflict):
		response.Problem(c, http.StatusConflict, "category_version_conflict", "Category Version Conflict", "the category was changed by another request")
	case errors.Is(err, ErrCategoryHasChildren):
		response.Problem(c, http.StatusConflict, "category_has_children", "Category Has Children", "only an unused draft category can be deleted")
	case errors.Is(err, ErrCategoryCycle):
		response.Problem(c, http.StatusBadRequest, "category_cycle", "Category Cycle", "a category cannot be its own ancestor")
	case errors.Is(err, ErrCategoryAncestorNotActive):
		response.Problem(c, http.StatusConflict, "category_ancestor_not_active", "Category Ancestor Not Active", "all category ancestors must be active before activation")
	case errors.Is(err, ErrCategoryNotEditable):
		response.Problem(c, http.StatusConflict, "category_not_editable", "Category Not Editable", "only a draft category can be changed or deleted")
	case errors.Is(err, ErrInvalidCategory):
		response.Problem(c, http.StatusBadRequest, "invalid_category", "Invalid Category", "category name must contain letters or numbers")
	default:
		response.InternalError(c, err)
	}
}
