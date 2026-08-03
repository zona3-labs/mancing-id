package brand

import (
	"context"
	"errors"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zona3-labs/mancing-id/internal/response"
	"github.com/zona3-labs/mancing-id/internal/upload"
)

type BrandHandler struct {
	usecase     BrandUsecase
	logoService BrandLogoService
	deletion    BrandDeletionService
}

type BrandLogoService interface {
	UploadBrandLogo(context.Context, uuid.UUID, multipart.File, *multipart.FileHeader) (string, error)
}

type BrandDeletionService interface {
	DeleteBrand(context.Context, uuid.UUID, int64) error
}

func NewBrandHandler(usecase BrandUsecase, logoService BrandLogoService, deletion BrandDeletionService) *BrandHandler {
	return &BrandHandler{usecase: usecase, logoService: logoService, deletion: deletion}
}

func (h *BrandHandler) RegisterRoutes(rg *gin.RouterGroup) {
	brands := rg.Group("/brands")
	brands.GET("", h.GetPublicBrands)
	brands.GET("/:slug", h.GetPublicBrandBySlug)
}

func (h *BrandHandler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	brands := rg.Group("/admin/brands")
	brands.POST("", h.CreateBrand)
	brands.GET("", h.GetAllBrands)
	brands.GET("/:identifier", h.GetAdminBrand)
	brands.PUT("/:id", h.UpdateBrand)
	brands.POST("/:id/activate", h.ActivateBrand)
	brands.POST("/:id/deactivate", h.DeactivateBrand)
	brands.POST("/:id/reactivate", h.ReactivateBrand)
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
	Version  *int64  `json:"version"`
}

type brandVersionRequest struct {
	Version *int64 `json:"version"`
}

func (h *BrandHandler) CreateBrand(c *gin.Context) {
	var req createBrandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Problem(c, http.StatusBadRequest, "invalid_request", "Invalid Request", "brand name is required")
		return
	}
	if req.LogoPath != nil {
		response.Problem(c, http.StatusBadRequest, "managed_logo_required", "Managed Logo Required", "brand logos must be uploaded as managed images")
		return
	}

	brand := &Brand{Name: req.Name, Slug: req.Slug, LogoPath: req.LogoPath}
	if err := h.usecase.CreateBrand(c.Request.Context(), brand); err != nil {
		h.writeError(c, err)
		return
	}
	response.Created(c, brand)
}

func (h *BrandHandler) GetAllBrands(c *gin.Context) {
	brands, err := h.usecase.GetAllBrands(c.Request.Context())
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, "brands retrieved successfully", brands)
}

func (h *BrandHandler) GetPublicBrands(c *gin.Context) {
	brands, err := h.usecase.GetPublicBrands(c.Request.Context())
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, "brands retrieved successfully", brands)
}

func (h *BrandHandler) GetAdminBrand(c *gin.Context) {
	identifier := c.Param("identifier")
	var brand *Brand
	var err error
	if id, parseErr := uuid.Parse(identifier); parseErr == nil {
		brand, err = h.usecase.GetBrandByID(c.Request.Context(), id)
	} else {
		brand, err = h.usecase.GetBrandBySlug(c.Request.Context(), identifier)
	}
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.OK(c, "brand retrieved successfully", brand)
}

func (h *BrandHandler) GetPublicBrandBySlug(c *gin.Context) {
	brand, err := h.usecase.GetPublicBrandBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.OK(c, "brand retrieved successfully", brand)
}

func (h *BrandHandler) UpdateBrand(c *gin.Context) {
	id, err := parseBrandID(c)
	if err != nil {
		return
	}
	var req updateBrandRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Version == nil {
		response.Problem(c, http.StatusBadRequest, "brand_version_required", "Version Required", "brand version is required")
		return
	}
	if req.LogoPath != nil {
		response.Problem(c, http.StatusBadRequest, "managed_logo_required", "Managed Logo Required", "brand logos must be uploaded as managed images")
		return
	}

	brand := &Brand{ID: id, Name: req.Name, Slug: req.Slug, LogoPath: req.LogoPath}
	if err := h.usecase.UpdateBrand(c.Request.Context(), brand, *req.Version); err != nil {
		h.writeError(c, err)
		return
	}
	response.OK(c, "brand updated successfully", brand)
}

func (h *BrandHandler) ActivateBrand(c *gin.Context) {
	h.transitionBrand(c, h.usecase.ActivateBrand)
}

func (h *BrandHandler) DeactivateBrand(c *gin.Context) {
	h.transitionBrand(c, h.usecase.DeactivateBrand)
}

func (h *BrandHandler) ReactivateBrand(c *gin.Context) {
	h.transitionBrand(c, h.usecase.ReactivateBrand)
}

func (h *BrandHandler) transitionBrand(c *gin.Context, transition func(context.Context, uuid.UUID, int64) error) {
	id, err := parseBrandID(c)
	if err != nil {
		return
	}
	var req brandVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Version == nil {
		response.Problem(c, http.StatusBadRequest, "brand_version_required", "Version Required", "brand version is required")
		return
	}
	if err := transition(c.Request.Context(), id, *req.Version); err != nil {
		h.writeError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *BrandHandler) DeleteBrand(c *gin.Context) {
	id, err := parseBrandID(c)
	if err != nil {
		return
	}
	var req brandVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Version == nil {
		response.Problem(c, http.StatusBadRequest, "brand_version_required", "Version Required", "brand version is required")
		return
	}
	if err := h.deletion.DeleteBrand(c.Request.Context(), id, *req.Version); err != nil {
		h.writeError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *BrandHandler) UploadBrandLogo(c *gin.Context) {
	if h.logoService == nil {
		response.InternalError(c, errors.New("brand logo service is not configured"))
		return
	}
	id, err := parseBrandID(c)
	if err != nil {
		return
	}
	file, header, err := c.Request.FormFile("logo")
	if err != nil {
		response.Problem(c, http.StatusBadRequest, "logo_required", "Logo Required", "logo field is required")
		return
	}
	defer file.Close()

	url, err := h.logoService.UploadBrandLogo(c.Request.Context(), id, file, header)
	if err != nil {
		switch {
		case errors.Is(err, upload.ErrInvalidFileType):
			response.Problem(c, http.StatusBadRequest, "invalid_logo_type", "Invalid Logo Type", "logo must be a JPEG, PNG, or WebP image")
		case errors.Is(err, upload.ErrFileTooLarge):
			response.Problem(c, http.StatusBadRequest, "logo_too_large", "Logo Too Large", "logo exceeds the maximum allowed size")
		case errors.Is(err, upload.ErrImageDecode):
			response.Problem(c, http.StatusBadRequest, "invalid_logo_image", "Invalid Logo Image", "logo image could not be decoded")
		default:
			h.writeError(c, err)
		}
		return
	}
	response.OK(c, "logo uploaded successfully", gin.H{"logo_path": url})
}

func parseBrandID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Problem(c, http.StatusBadRequest, "invalid_brand_id", "Invalid Brand ID", "brand id must be a UUID")
	}
	return id, err
}

func (h *BrandHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrBrandNotFound):
		response.Problem(c, http.StatusNotFound, "brand_not_found", "Brand Not Found", "the brand does not exist")
	case errors.Is(err, ErrBrandVersionConflict):
		response.Problem(c, http.StatusConflict, "brand_version_conflict", "Brand Version Conflict", "the brand was changed by another request")
	case errors.Is(err, ErrBrandHasProducts):
		response.Problem(c, http.StatusConflict, "brand_has_products", "Brand Has Products", "only an unreferenced draft brand can be deleted")
	case errors.Is(err, ErrBrandNotEditable):
		response.Problem(c, http.StatusConflict, "brand_not_editable", "Brand Not Editable", "only a draft brand can be changed or deleted")
	case errors.Is(err, ErrInvalidBrand):
		response.Problem(c, http.StatusBadRequest, "invalid_brand", "Invalid Brand", "brand name must contain letters or numbers")
	case errors.Is(err, ErrManagedLogoRequired):
		response.Problem(c, http.StatusBadRequest, "managed_logo_required", "Managed Logo Required", "brand logos must be uploaded as managed images")
	default:
		response.InternalError(c, err)
	}
}
