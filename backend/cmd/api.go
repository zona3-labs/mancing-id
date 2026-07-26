package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/zona3-labs/mancing-id/docs"
	"github.com/zona3-labs/mancing-id/internal/brand"
	"github.com/zona3-labs/mancing-id/internal/category"
	"github.com/zona3-labs/mancing-id/internal/config"
	"github.com/zona3-labs/mancing-id/internal/product"
	"github.com/zona3-labs/mancing-id/internal/response"
	"github.com/zona3-labs/mancing-id/internal/upload"
)

type application struct {
	config   *config.Config
	db       *sql.DB
	uploader upload.FileUploader
}

func (app *application) mount() http.Handler {
	router := gin.New()

	router.Use(
		gin.Logger(),
		gin.Recovery(),
		requestid.New(),
	)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := router.Group("/api")

	v1 := api.Group("/v1")
	app.registerV1Routes(v1)

	return router
}

func (app *application) registerV1Routes(v1 *gin.RouterGroup) {
	v1.GET("/healthz", func(c *gin.Context) {
		response.OK(c, "ok", nil)
	})
	v1.GET("/ping", func(c *gin.Context) {
		response.OK(c, "pong", nil)
	})

	//	Category
	categoryRepo := category.NewCategoryRepository(app.db)
	categoryUsecase := category.NewCategoryUsecase(categoryRepo)
	categoryHandler := category.NewCategoryHandler(categoryUsecase)
	categoryHandler.RegisterRoutes(v1)

	// Brand
	brandRepo := brand.NewBrandRepository(app.db)
	brandUsecase := brand.NewBrandUsecase(brandRepo, app.uploader)
	brandHandler := brand.NewBrandHandler(brandUsecase)
	brandHandler.RegisterRoutes(v1)

	// Product
	productRepo := product.NewProductRepository(app.db)
	productUsecase := product.NewProductUsecase(productRepo)
	productHandler := product.NewProductHandler(productUsecase)
	productHandler.RegisterRoutes(v1)
}

func (app *application) run(h http.Handler) error {
	srv := &http.Server{
		Addr:         ":" + strconv.Itoa(app.config.HttpServer.Port),
		Handler:      h,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	log.Printf("server started on %v", app.config.HttpServer.Port)

	return srv.ListenAndServe()
}
