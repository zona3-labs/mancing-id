package category

import (
	"errors"

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
	categories.POST("", h.CreateCategory)
	categories.GET("", h.GetAllCategories)
	categories.GET("/:slug", h.GetCategoryBySlug)
	categories.PUT("/:id", h.UpdateCategory)
	categories.DELETE("/:id", h.DeleteCategory)
}

type createCategoryRequest struct {
	Name     string     `json:"name" binding:"required"`
	Slug     string     `json:"slug"`
	ParentID *uuid.UUID `json:"parent_id"`
}

type updateCategoryRequest struct {
	Name     string     `json:"name" binding:"required"`
	Slug     string     `json:"slug"`
	IsActive bool       `json:"is_active"`
	ParentID *uuid.UUID `json:"parent_id"`
}

// CreateCategory godoc
// @Summary      Create a new category
// @Description  Create a new category with optional parent category
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        request body createCategoryRequest true "Category details"
// @Success      201  {object}  response.Envelope{data=Category}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /categories [post]
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	category := &Category{
		Name:     req.Name,
		Slug:     req.Slug,
		ParentID: req.ParentID,
	}

	if err := h.usecase.CreateCategory(c.Request.Context(), category); err != nil {
		if errors.Is(err, ErrParentCategoryNotFound) {
			response.BadRequest(c, "parent category not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.Created(c, category)
}

// GetAllCategories godoc
// @Summary      Get all categories
// @Description  Retrieve all categories in a hierarchical tree structure
// @Tags         categories
// @Produce      json
// @Success      200  {object}  response.Envelope{data=[]Category}
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /categories [get]
func (h *CategoryHandler) GetAllCategories(c *gin.Context) {
	categories, err := h.usecase.GetAllCategories(c.Request.Context())
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, "categories retrieved successfully", categories)
}

// GetCategoryBySlug godoc
// @Summary      Get category by slug
// @Description  Get category details by its unique URL slug
// @Tags         categories
// @Produce      json
// @Param        slug  path      string  true  "Category Slug"
// @Success      200  {object}  response.Envelope{data=Category}
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /categories/{slug} [get]
func (h *CategoryHandler) GetCategoryBySlug(c *gin.Context) {
	slug := c.Param("slug")

	category, err := h.usecase.GetCategoryBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			response.NotFound(c, "category not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.OK(c, "category retrieved successfully", category)
}

// UpdateCategory godoc
// @Summary      Update a category
// @Description  Update details of an existing category by its ID
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        id    path      string                 true  "Category ID (UUID)"
// @Param        request body updateCategoryRequest true  "Updated category details"
// @Success      200  {object}  response.Envelope{data=Category}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid category id")
		return
	}

	var req updateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	category := &Category{
		ID:       id,
		Name:     req.Name,
		Slug:     req.Slug,
		IsActive: req.IsActive,
		ParentID: req.ParentID,
	}

	if err := h.usecase.UpdateCategory(c.Request.Context(), category); err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			response.NotFound(c, "category not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.OK(c, "category updated successfully", category)
}

// DeleteCategory godoc
// @Summary      Delete a category
// @Description  Soft delete an existing category by its ID
// @Tags         categories
// @Produce      json
// @Param        id    path      string  true  "Category ID (UUID)"
// @Success      204  "No Content"
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid category id")
		return
	}

	if err := h.usecase.DeleteCategory(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			response.NotFound(c, "category not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.NoContent(c)
}
