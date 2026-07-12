package product

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zone3-labs/mancing-id/internal/response"
)

type ProductHandler struct {
	usecase ProductUsecase
}

func NewProductHandler(usecase ProductUsecase) *ProductHandler {
	return &ProductHandler{usecase: usecase}
}

func (h *ProductHandler) RegisterRoutes(rg *gin.RouterGroup) {
	products := rg.Group("/products")
	products.POST("", h.CreateProduct)
	products.GET("", h.GetAllProducts)
	products.GET("/:slug", h.GetProductBySlug)
	products.PUT("/:id", h.UpdateProduct)
	products.DELETE("/:id", h.DeleteProduct)
}

type createProductRequest struct {
	Name             string        `json:"name" binding:"required"`
	Slug             string        `json:"slug"`
	Description      *string       `json:"description"`
	ShortDescription *string       `json:"short_description"`
	Status           ProductStatus `json:"status" binding:"required,oneof=draft active archived"`
	BrandID          *uuid.UUID    `json:"brand_id"`
	IsFeature        bool          `json:"is_feature"`
}

type updateProductRequest struct {
	Name             string        `json:"name" binding:"required"`
	Slug             string        `json:"slug"`
	Description      *string       `json:"description"`
	ShortDescription *string       `json:"short_description"`
	Status           ProductStatus `json:"status" binding:"required,oneof=draft active archived"`
	BrandID          *uuid.UUID    `json:"brand_id"`
	IsFeature        bool          `json:"is_feature"`
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req createProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	product := &Product{
		Name:             req.Name,
		Slug:             req.Slug,
		Description:      req.Description,
		ShortDescription: req.ShortDescription,
		Status:           req.Status,
		BrandID:          req.BrandID,
		IsFeature:        req.IsFeature,
	}

	if err := h.usecase.CreateProduct(c.Request.Context(), product); err != nil {
		response.InternalError(c, err)
		return
	}

	response.Created(c, product)
}

func (h *ProductHandler) GetAllProducts(c *gin.Context) {
	products, err := h.usecase.GetAllProducts(c.Request.Context())
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, "products retrieved successfully", products)
}

func (h *ProductHandler) GetProductBySlug(c *gin.Context) {
	slug := c.Param("slug")

	product, err := h.usecase.GetProductBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.NotFound(c, "product not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.OK(c, "product retrieved successfully", product)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid product id")
		return
	}

	var req updateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	product := &Product{
		ID:               id,
		Name:             req.Name,
		Slug:             req.Slug,
		Description:      req.Description,
		ShortDescription: req.ShortDescription,
		Status:           req.Status,
		BrandID:          req.BrandID,
		IsFeature:        req.IsFeature,
	}

	if err := h.usecase.UpdateProduct(c.Request.Context(), product); err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.NotFound(c, "product not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.OK(c, "product updated successfully", product)
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid product id")
		return
	}

	if err := h.usecase.DeleteProduct(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.NotFound(c, "product not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.NoContent(c)
}
