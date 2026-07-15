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

	// Options
	options := products.Group("/:id/options")
	options.POST("", h.CreateProductOption)
	options.GET("", h.GetProductOptions)
	options.PUT("/:optionId", h.UpdateProductOption)
	options.DELETE("/:optionId", h.DeleteProductOption)

	// Option Values
	options.POST("/:optionId/values", h.CreateProductOptionValue)
	options.GET("/:optionId/values", h.GetProductOptionValues)
	options.DELETE("/:optionId/values/:valueId", h.DeleteProductOptionValue)
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

// -------------------------------------------------------
// Options
// -------------------------------------------------------

type createProductOptionRequest struct {
	Name     string `json:"name" binding:"required"`
	Position int16  `json:"position"`
}

type updateProductOptionRequest struct {
	Name     string `json:"name" binding:"required"`
	Position int16  `json:"position"`
}

func (h *ProductHandler) CreateProductOption(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid product id")
		return
	}

	var req createProductOptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	option := &ProductOption{
		ProductID: productID,
		Name:      req.Name,
		Position:  req.Position,
	}

	result, err := h.usecase.CreateProductOption(c.Request.Context(), option)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.NotFound(c, "product not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.Created(c, result)
}

func (h *ProductHandler) GetProductOptions(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid product id")
		return
	}

	options, err := h.usecase.GetProductOptions(c.Request.Context(), productID)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, "options retrieved successfully", options)
}

func (h *ProductHandler) UpdateProductOption(c *gin.Context) {
	optionID, err := uuid.Parse(c.Param("optionId"))
	if err != nil {
		response.BadRequest(c, "invalid option id")
		return
	}

	var req updateProductOptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	option := &ProductOption{
		ID:       optionID,
		Name:     req.Name,
		Position: req.Position,
	}

	result, err := h.usecase.UpdateProductOption(c.Request.Context(), option)
	if err != nil {
		if errors.Is(err, ErrOptionNotFound) {
			response.NotFound(c, "option not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.OK(c, "option updated successfully", result)
}

func (h *ProductHandler) DeleteProductOption(c *gin.Context) {
	optionID, err := uuid.Parse(c.Param("optionId"))
	if err != nil {
		response.BadRequest(c, "invalid option id")
		return
	}

	if err := h.usecase.DeleteProductOption(c.Request.Context(), optionID); err != nil {
		if errors.Is(err, ErrOptionNotFound) {
			response.NotFound(c, "option not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.NoContent(c)
}

// -------------------------------------------------------
// Option Values
// -------------------------------------------------------

type createProductOptionValueRequest struct {
	Value    string `json:"value" binding:"required"`
	Position int16  `json:"position"`
}

func (h *ProductHandler) CreateProductOptionValue(c *gin.Context) {
	optionID, err := uuid.Parse(c.Param("optionId"))
	if err != nil {
		response.BadRequest(c, "invalid option id")
		return
	}

	var req createProductOptionValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	value := &ProductOptionValue{
		ProductOptionID: optionID,
		Value:           req.Value,
		Position:        req.Position,
	}

	result, err := h.usecase.CreateProductOptionValue(c.Request.Context(), value)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.Created(c, result)
}

func (h *ProductHandler) GetProductOptionValues(c *gin.Context) {
	optionID, err := uuid.Parse(c.Param("optionId"))
	if err != nil {
		response.BadRequest(c, "invalid option id")
		return
	}

	values, err := h.usecase.GetProductOptionValues(c.Request.Context(), optionID)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, "option values retrieved successfully", values)
}

func (h *ProductHandler) DeleteProductOptionValue(c *gin.Context) {
	valueID, err := uuid.Parse(c.Param("valueId"))
	if err != nil {
		response.BadRequest(c, "invalid option value id")
		return
	}

	if err := h.usecase.DeleteProductOptionValue(c.Request.Context(), valueID); err != nil {
		if errors.Is(err, ErrOptionValueNotFound) {
			response.NotFound(c, "option value not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.NoContent(c)
}
