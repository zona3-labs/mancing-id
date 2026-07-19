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
	products.GET("/slug/:slug", h.GetProductBySlug)
	products.GET("/:id", h.GetProductByID)
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

	// Images
	images := products.Group("/:id/images")
	images.POST("", h.CreateProductImage)
	images.GET("", h.GetProductImages)
	images.PUT("/:imageId/primary", h.SetPrimaryImage)
	images.DELETE("/:imageId", h.DeleteProductImage)

	// Variants
	variants := products.Group("/:id/variants")
	variants.POST("", h.CreateProductVariant)
	variants.GET("", h.GetProductVariants)

	products.GET("/variants/:variantId", h.GetProductVariantByID)
	products.PUT("/variants/:variantId", h.UpdateProductVariant)
	products.DELETE("/variants/:variantId", h.DeleteProductVariant)
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

	detail, err := h.usecase.GetProductBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.NotFound(c, "product not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.OK(c, "product retrieved successfully", detail)
}

func (h *ProductHandler) GetProductByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid product id")
		return
	}

	detail, err := h.usecase.GetProductDetailByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.NotFound(c, "product not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.OK(c, "product retrieved successfully", detail)
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

type createProductOptionValueInOptionRequest struct {
	Value    string `json:"value" binding:"required"`
	Position int16  `json:"position"`
	IsActive bool   `json:"is_active"`
}

type createProductOptionRequest struct {
	Name     string                                    `json:"name" binding:"required"`
	Position int16                                     `json:"position"`
	IsActive bool                                      `json:"is_active"`
	Values   []createProductOptionValueInOptionRequest `json:"values"`
}

type updateProductOptionRequest struct {
	Name     string `json:"name" binding:"required"`
	Position int16  `json:"position"`
	IsActive bool   `json:"is_active"`
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
		IsActive:  req.IsActive,
	}

	values := make([]*ProductOptionValue, 0, len(req.Values))
	for _, v := range req.Values {
		values = append(values, &ProductOptionValue{
			Value:    v.Value,
			Position: v.Position,
			IsActive: v.IsActive,
		})
	}

	result, err := h.usecase.CreateProductOptionWithValues(c.Request.Context(), option, values)
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
		IsActive: req.IsActive,
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
	IsActive bool   `json:"is_active"`
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
		IsActive:        req.IsActive,
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

// -------------------------------------------------------
// Images
// -------------------------------------------------------

type createProductImageRequest struct {
	Url       string  `json:"url" binding:"required"`
	AltText   *string `json:"alt_text"`
	Position  int16   `json:"position"`
	IsPrimary bool    `json:"is_primary"`
}

func (h *ProductHandler) CreateProductImage(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid product id")
		return
	}

	var req createProductImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	image := &ProductImage{
		ProductID: productID,
		Url:       req.Url,
		AltText:   req.AltText,
		Position:  req.Position,
		IsPrimary: req.IsPrimary,
	}

	result, err := h.usecase.CreateProductImage(c.Request.Context(), image)
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

func (h *ProductHandler) GetProductImages(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid product id")
		return
	}

	images, err := h.usecase.GetProductImages(c.Request.Context(), productID)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, "product images retrieved successfully", images)
}

func (h *ProductHandler) SetPrimaryImage(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid product id")
		return
	}

	imageID, err := uuid.Parse(c.Param("imageId"))
	if err != nil {
		response.BadRequest(c, "invalid image id")
		return
	}

	err = h.usecase.SetPrimaryImage(c.Request.Context(), productID, imageID)
	if err != nil {
		if errors.Is(err, ErrProductImageNotFound) {
			response.NotFound(c, "product image not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.NoContent(c)
}

func (h *ProductHandler) DeleteProductImage(c *gin.Context) {
	imageID, err := uuid.Parse(c.Param("imageId"))
	if err != nil {
		response.BadRequest(c, "invalid image id")
		return
	}

	err = h.usecase.DeleteProductImage(c.Request.Context(), imageID)
	if err != nil {
		if errors.Is(err, ErrProductImageNotFound) {
			response.NotFound(c, "product image not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.NoContent(c)
}

// -------------------------------------------------------
// Variants
// -------------------------------------------------------

type createProductVariantRequest struct {
	Sku            string        `json:"sku" binding:"required"`
	Price          string        `json:"price" binding:"required"`
	Stock          int32         `json:"stock"`
	Weight         *string       `json:"weight"`
	Status         VariantStatus `json:"status" binding:"required"`
	ImageID        *uuid.UUID    `json:"image_id"`
	OptionValueIDs []uuid.UUID   `json:"option_value_ids"`
}

type updateProductVariantRequest struct {
	Sku     string        `json:"sku" binding:"required"`
	Price   string        `json:"price" binding:"required"`
	Stock   int32         `json:"stock"`
	Weight  *string       `json:"weight"`
	Status  VariantStatus `json:"status" binding:"required"`
	ImageID *uuid.UUID    `json:"image_id"`
}

func (h *ProductHandler) CreateProductVariant(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid product id")
		return
	}

	var req createProductVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if !req.Status.IsValid() {
		response.BadRequest(c, "invalid status value")
		return
	}

	variant := &ProductVariant{
		ProductID: productID,
		Sku:       req.Sku,
		Price:     req.Price,
		Stock:     req.Stock,
		Weight:    req.Weight,
		Status:    req.Status,
		ImageID:   req.ImageID,
	}

	result, err := h.usecase.CreateProductVariant(c.Request.Context(), variant, req.OptionValueIDs)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.NotFound(c, "product not found")
			return
		}
		if errors.Is(err, ErrOptionValueNotFound) {
			response.NotFound(c, "one or more option values not found")
			return
		}
		if errors.Is(err, ErrSkuAlreadyExists) {
			response.BadRequest(c, "sku already exists")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.Created(c, result)
}

func (h *ProductHandler) GetProductVariants(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid product id")
		return
	}

	variants, err := h.usecase.GetProductVariants(c.Request.Context(), productID)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, "product variants retrieved successfully", variants)
}

func (h *ProductHandler) GetProductVariantByID(c *gin.Context) {
	variantID, err := uuid.Parse(c.Param("variantId"))
	if err != nil {
		response.BadRequest(c, "invalid variant id")
		return
	}

	result, err := h.usecase.GetProductVariantByID(c.Request.Context(), variantID)
	if err != nil {
		if errors.Is(err, ErrVariantNotFound) {
			response.NotFound(c, "variant not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.OK(c, "product variant retrieved successfully", result)
}

func (h *ProductHandler) UpdateProductVariant(c *gin.Context) {
	variantID, err := uuid.Parse(c.Param("variantId"))
	if err != nil {
		response.BadRequest(c, "invalid variant id")
		return
	}

	var req updateProductVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if !req.Status.IsValid() {
		response.BadRequest(c, "invalid status value")
		return
	}

	variant := &ProductVariant{
		ID:      variantID,
		Sku:     req.Sku,
		Price:   req.Price,
		Stock:   req.Stock,
		Weight:  req.Weight,
		Status:  req.Status,
		ImageID: req.ImageID,
	}

	result, err := h.usecase.UpdateProductVariant(c.Request.Context(), variant)
	if err != nil {
		if errors.Is(err, ErrVariantNotFound) {
			response.NotFound(c, "variant not found")
			return
		}
		if errors.Is(err, ErrSkuAlreadyExists) {
			response.BadRequest(c, "sku already exists")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.OK(c, "product variant updated successfully", result)
}

func (h *ProductHandler) DeleteProductVariant(c *gin.Context) {
	variantID, err := uuid.Parse(c.Param("variantId"))
	if err != nil {
		response.BadRequest(c, "invalid variant id")
		return
	}

	err = h.usecase.DeleteProductVariant(c.Request.Context(), variantID)
	if err != nil {
		if errors.Is(err, ErrVariantNotFound) {
			response.NotFound(c, "variant not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.NoContent(c)
}
