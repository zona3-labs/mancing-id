package brand

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zone3-labs/mancing-id/internal/response"
	"github.com/zone3-labs/mancing-id/internal/upload"
)

type BrandHandler struct {
	usecase BrandUsecase
}

func NewBrandHandler(usecase BrandUsecase) *BrandHandler {
	return &BrandHandler{usecase: usecase}
}

func (h *BrandHandler) RegisterRoutes(rg *gin.RouterGroup) {
	brands := rg.Group("/brands")
	brands.POST("", h.CreateBrand)
	brands.GET("", h.GetAllBrand)
	brands.GET("/:slug", h.GetBrandBySlug)
	brands.PUT("/:id", h.UpdateBrand)
	brands.POST("/:id/logo", h.UploadBrandLogo)
	brands.DELETE("/:id", h.DeleteBrand)
}

type createBrandRequest struct {
	Name     string  `json:"name" binding:"required"`
	Slug     string  `json:"slug"`
	LogoPath *string `json:"logo_path"`
}

type updateBrandRequest struct {
	Name     string  `json:"name" binding:"required"`
	Slug     string  `json:"slug"`
	LogoPath *string `json:"logo_path"`
	IsActive bool    `json:"is_active"`
}

// CreateBrand godoc
// @Summary      Create a new brand
// @Description  Create a new brand with name, slug, and logo path
// @Tags         brands
// @Accept       json
// @Produce      json
// @Param        request body createBrandRequest true "Brand details"
// @Success      201  {object}  response.Envelope{data=Brand}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /brands [post]
func (h *BrandHandler) CreateBrand(c *gin.Context) {
	var req createBrandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	brand := &Brand{
		Name:     req.Name,
		Slug:     req.Slug,
		LogoPath: req.LogoPath,
	}

	if err := h.usecase.CreateBrand(c.Request.Context(), brand); err != nil {
		response.InternalError(c, err)
		return
	}

	response.Created(c, brand)
}

// GetAllBrand godoc
// @Summary      Get all brands
// @Description  Retrieve all registered brands
// @Tags         brands
// @Produce      json
// @Success      200  {object}  response.Envelope{data=[]Brand}
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /brands [get]
func (h *BrandHandler) GetAllBrand(c *gin.Context) {
	brands, err := h.usecase.GetAllBrand(c.Request.Context())
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, "brands retrieved successfully", brands)
}

// GetBrandBySlug godoc
// @Summary      Get brand by slug
// @Description  Get brand details by its unique URL slug
// @Tags         brands
// @Produce      json
// @Param        slug  path      string  true  "Brand Slug"
// @Success      200  {object}  response.Envelope{data=Brand}
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /brands/{slug} [get]
func (h *BrandHandler) GetBrandBySlug(c *gin.Context) {
	slug := c.Param("slug")

	brand, err := h.usecase.GetBrandBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, ErrBrandNotFound) {
			response.NotFound(c, "brand not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.OK(c, "brand retrieved successfully", brand)
}

// UpdateBrand godoc
// @Summary      Update a brand
// @Description  Update details of an existing brand by its ID
// @Tags         brands
// @Accept       json
// @Produce      json
// @Param        id    path      string                 true  "Brand ID (UUID)"
// @Param        request body updateBrandRequest true  "Updated brand details"
// @Success      200  {object}  response.Envelope{data=Brand}
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /brands/{id} [put]
func (h *BrandHandler) UpdateBrand(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid brand id")
		return
	}

	var req updateBrandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	brand := &Brand{
		ID:       id,
		Name:     req.Name,
		Slug:     req.Slug,
		LogoPath: req.LogoPath,
		IsActive: req.IsActive,
	}

	if err := h.usecase.UpdateBrand(c.Request.Context(), brand); err != nil {
		if errors.Is(err, ErrBrandNotFound) {
			response.NotFound(c, "brand not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.OK(c, "brand updated successfully", brand)
}

// DeleteBrand godoc
// @Summary      Delete a brand
// @Description  Soft delete an existing brand by its ID
// @Tags         brands
// @Produce      json
// @Param        id    path      string  true  "Brand ID (UUID)"
// @Success      204  "No Content"
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /brands/{id} [delete]
func (h *BrandHandler) DeleteBrand(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid brand id")
		return
	}

	if err := h.usecase.DeleteBrand(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrBrandNotFound) {
			response.NotFound(c, "brand not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.NoContent(c)
}

// UploadBrandLogo godoc
// @Summary      Upload brand logo
// @Description  Upload logo image for an existing brand
// @Tags         brands
// @Accept       multipart/form-data
// @Produce      json
// @Param        id    path      string  true  "Brand ID (UUID)"
// @Param        logo  formData  file    true  "Logo image file"
// @Success      200  {object}  response.Envelope{data=map[string]string} "logo_path response"
// @Failure      400  {object}  response.ErrorEnvelope
// @Failure      404  {object}  response.ErrorEnvelope
// @Failure      500  {object}  response.ErrorEnvelope
// @Router       /brands/{id}/logo [post]
func (h *BrandHandler) UploadBrandLogo(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid brand id")
		return
	}

	file, header, err := c.Request.FormFile("logo")
	if err != nil {
		response.BadRequest(c, "logo field is required")
		return
	}
	defer file.Close()

	url, err := h.usecase.UploadBrandLogo(c.Request.Context(), id, file, header)
	if err != nil {
		switch {
		case errors.Is(err, ErrBrandNotFound):
			response.NotFound(c, "brand not found")
		case errors.Is(err, upload.ErrInvalidFileType):
			response.BadRequest(c, err.Error())
		case errors.Is(err, upload.ErrFileTooLarge):
			response.BadRequest(c, err.Error())
		default:
			response.InternalError(c, err)
		}
		return
	}

	response.OK(c, "logo uploaded successfully", gin.H{"logo_path": url})
}
