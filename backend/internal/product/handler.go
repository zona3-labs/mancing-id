package product

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zona3-labs/mancing-id/internal/brand"
	"github.com/zona3-labs/mancing-id/internal/response"
)

type ProductHandler struct {
	usecase   ProductUsecase
	authoring ProductCatalogService
}

func NewProductHandler(usecase ProductUsecase, authoring ProductCatalogService) *ProductHandler {
	return &ProductHandler{usecase: usecase, authoring: authoring}
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
	Version          *int64        `json:"version"`
}

// CreateProduct godoc
// @Summary      Create a new product
// @Description  Create a new product with basic details
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        request body createProductRequest true "Product details"
// @Success      201  {object}  response.Envelope{data=Product}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products [post]
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

	if err := h.authoring.CreateProduct(c.Request.Context(), product); err != nil {
		h.writeError(c, err)
		return
	}

	response.Created(c, product)
}

// GetAllProducts godoc
// @Summary      Get all products
// @Description  Retrieve all products
// @Tags         products
// @Produce      json
// @Success      200  {object}  response.Envelope{data=[]Product}
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products [get]
func (h *ProductHandler) GetAllProducts(c *gin.Context) {
	products, err := h.authoring.GetAllProducts(c.Request.Context())
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, "products retrieved successfully", products)
}

// GetProductBySlug godoc
// @Summary      Get product by slug
// @Description  Get product details by unique slug
// @Tags         products
// @Produce      json
// @Param        slug  path      string  true  "Product Slug"
// @Success      200  {object}  response.Envelope{data=ProductDetail}
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/slug/{slug} [get]
func (h *ProductHandler) GetProductBySlug(c *gin.Context) {
	slug := c.Param("slug")

	detail, err := h.authoring.GetProductBySlug(c.Request.Context(), slug)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.OK(c, "product retrieved successfully", detail)
}

// GetProductByID godoc
// @Summary      Get product by ID
// @Description  Get full product details including options, images, and variants by product ID
// @Tags         products
// @Produce      json
// @Param        id  path      string  true  "Product ID (UUID)"
// @Success      200  {object}  response.Envelope{data=ProductDetail}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id} [get]
func (h *ProductHandler) GetProductByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid product id")
		return
	}

	detail, err := h.authoring.GetProductDetailByID(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.OK(c, "product retrieved successfully", detail)
}

// UpdateProduct godoc
// @Summary      Update a product
// @Description  Update details of an existing product
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        id    path      string                 true  "Product ID (UUID)"
// @Param        request body updateProductRequest true  "Updated product details"
// @Success      200  {object}  response.Envelope{data=Product}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id} [put]
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid product id")
		return
	}

	var req updateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Version == nil {
		response.BadRequest(c, "product version is required")
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
		Version:          *req.Version,
	}

	if err := h.authoring.UpdateProduct(c.Request.Context(), product); err != nil {
		h.writeError(c, err)
		return
	}

	response.OK(c, "product updated successfully", product)
}

// DeleteProduct godoc
// @Summary      Delete a product
// @Description  Soft delete an existing product by its ID
// @Tags         products
// @Produce      json
// @Param        id    path      string  true  "Product ID (UUID)"
// @Success      204  "No Content"
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id} [delete]
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid product id")
		return
	}

	var req struct {
		Version *int64 `json:"version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Version == nil {
		response.Problem(c, http.StatusBadRequest, "product_version_required", "Version Required", "product version is required")
		return
	}

	if err := h.authoring.DeleteProduct(c.Request.Context(), id, *req.Version); err != nil {
		h.writeError(c, err)
		return
	}

	response.NoContent(c)
}

func (h *ProductHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrProductNotFound):
		response.Problem(c, http.StatusNotFound, "product_not_found", "Product Not Found", "the product does not exist")
	case errors.Is(err, ErrProductVersionConflict):
		response.Problem(c, http.StatusConflict, "product_version_conflict", "Product Version Conflict", "the product was changed by another request")
	case errors.Is(err, ErrProductNotEditable):
		response.Problem(c, http.StatusConflict, "product_not_editable", "Product Not Editable", "only a never-published draft product can be changed or deleted")
	case errors.Is(err, ErrProductDraftRequired):
		response.Problem(c, http.StatusBadRequest, "product_draft_required", "Draft Product Required", "product authoring operations require a draft product")
	case errors.Is(err, ErrProductBrandRequired):
		response.Problem(c, http.StatusBadRequest, "product_brand_required", "Brand Required", "a product must reference an active brand")
	case errors.Is(err, ErrProductBrandNotActive):
		response.Problem(c, http.StatusConflict, "brand_not_active", "Brand Not Active", "a product can reference only an active brand")
	case errors.Is(err, brand.ErrBrandNotFound):
		response.Problem(c, http.StatusNotFound, "brand_not_found", "Brand Not Found", "the brand does not exist")
	case errors.Is(err, ErrInvalidProduct):
		response.Problem(c, http.StatusBadRequest, "invalid_product", "Invalid Product", "product name or slug must contain letters or numbers")
	default:
		response.InternalError(c, err)
	}
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

// CreateProductOption godoc
// @Summary      Create product option
// @Description  Create a new option (e.g. Size, Color) with values for a product
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        id    path      string                     true  "Product ID (UUID)"
// @Param        request body createProductOptionRequest true  "Product option details"
// @Success      201  {object}  response.Envelope{data=ProductOptionWithValues}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id}/options [post]
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

// GetProductOptions godoc
// @Summary      Get product options
// @Description  Get all options and their values for a specific product
// @Tags         products
// @Produce      json
// @Param        id  path      string  true  "Product ID (UUID)"
// @Success      200  {object}  response.Envelope{data=[]ProductOptionWithValues}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id}/options [get]
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

// UpdateProductOption godoc
// @Summary      Update product option
// @Description  Update details of an existing product option
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        id        path      string                     true  "Product ID (UUID)"
// @Param        optionId  path      string                     true  "Option ID (UUID)"
// @Param        request   body      updateProductOptionRequest true  "Updated option details"
// @Success      200  {object}  response.Envelope{data=ProductOption}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id}/options/{optionId} [put]
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

// DeleteProductOption godoc
// @Summary      Delete product option
// @Description  Soft delete a product option
// @Tags         products
// @Produce      json
// @Param        id        path      string  true  "Product ID (UUID)"
// @Param        optionId  path      string  true  "Option ID (UUID)"
// @Success      204  "No Content"
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id}/options/{optionId} [delete]
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

// CreateProductOptionValue godoc
// @Summary      Create product option value
// @Description  Create a new value for a specific product option
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        id        path      string                          true  "Product ID (UUID)"
// @Param        optionId  path      string                          true  "Option ID (UUID)"
// @Param        request   body      createProductOptionValueRequest true  "Option value details"
// @Success      201  {object}  response.Envelope{data=ProductOptionValue}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id}/options/{optionId}/values [post]
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

// GetProductOptionValues godoc
// @Summary      Get product option values
// @Description  Retrieve all values for a specific product option
// @Tags         products
// @Produce      json
// @Param        id        path      string  true  "Product ID (UUID)"
// @Param        optionId  path      string  true  "Option ID (UUID)"
// @Success      200  {object}  response.Envelope{data=[]ProductOptionValue}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id}/options/{optionId}/values [get]
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

// DeleteProductOptionValue godoc
// @Summary      Delete product option value
// @Description  Soft delete a product option value
// @Tags         products
// @Produce      json
// @Param        id        path      string  true  "Product ID (UUID)"
// @Param        optionId  path      string  true  "Option ID (UUID)"
// @Param        valueId   path      string  true  "Option Value ID (UUID)"
// @Success      204  "No Content"
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id}/options/{optionId}/values/{valueId} [delete]
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

// CreateProductImage godoc
// @Summary      Create product image
// @Description  Add a new image associated with a product
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        id       path      string                     true  "Product ID (UUID)"
// @Param        request  body      createProductImageRequest  true  "Product image details"
// @Success      201  {object}  response.Envelope{data=ProductImage}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id}/images [post]
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

// GetProductImages godoc
// @Summary      Get product images
// @Description  Retrieve all images associated with a specific product
// @Tags         products
// @Produce      json
// @Param        id  path      string  true  "Product ID (UUID)"
// @Success      200  {object}  response.Envelope{data=[]ProductImage}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id}/images [get]
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

// SetPrimaryImage godoc
// @Summary      Set primary product image
// @Description  Set a specific image as the primary image for a product
// @Tags         products
// @Produce      json
// @Param        id       path      string  true  "Product ID (UUID)"
// @Param        imageId  path      string  true  "Image ID (UUID)"
// @Success      204  "No Content"
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id}/images/{imageId}/primary [put]
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

// DeleteProductImage godoc
// @Summary      Delete product image
// @Description  Delete an image association for a product
// @Tags         products
// @Produce      json
// @Param        id       path      string  true  "Product ID (UUID)"
// @Param        imageId  path      string  true  "Image ID (UUID)"
// @Success      204  "No Content"
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id}/images/{imageId} [delete]
func (h *ProductHandler) DeleteProductImage(c *gin.Context) {
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

	err = h.usecase.DeleteProductImage(c.Request.Context(), productID, imageID)
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

// CreateProductVariant godoc
// @Summary      Create product variant
// @Description  Create a new variant of a product (SKU, price, stock, weight, status, image) and bind option values
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        id       path      string                       true  "Product ID (UUID)"
// @Param        request  body      createProductVariantRequest  true  "Product variant details"
// @Success      201  {object}  response.Envelope{data=ProductVariantDetail}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id}/variants [post]
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

// GetProductVariants godoc
// @Summary      Get product variants
// @Description  Retrieve all variants associated with a specific product
// @Tags         products
// @Produce      json
// @Param        id  path      string  true  "Product ID (UUID)"
// @Success      200  {object}  response.Envelope{data=[]ProductVariantDetail}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/{id}/variants [get]
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

// GetProductVariantByID godoc
// @Summary      Get product variant by ID
// @Description  Retrieve details of a product variant by its unique ID
// @Tags         products
// @Produce      json
// @Param        variantId  path      string  true  "Variant ID (UUID)"
// @Success      200  {object}  response.Envelope{data=ProductVariantDetail}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/variants/{variantId} [get]
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

// UpdateProductVariant godoc
// @Summary      Update product variant
// @Description  Update details of an existing product variant
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        variantId  path      string                       true  "Variant ID (UUID)"
// @Param        request    body      updateProductVariantRequest  true  "Updated variant details"
// @Success      200  {object}  response.Envelope{data=ProductVariant}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/variants/{variantId} [put]
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

// DeleteProductVariant godoc
// @Summary      Delete product variant
// @Description  Soft delete an existing product variant
// @Tags         products
// @Produce      json
// @Param        variantId  path      string  true  "Variant ID (UUID)"
// @Success      204  "No Content"
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /products/variants/{variantId} [delete]
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
