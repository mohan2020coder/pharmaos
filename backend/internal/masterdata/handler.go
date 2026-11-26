package masterdata

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/auth"
	"github.com/pharmaos/pharmaos/backend/internal/middleware"
)

type Handler struct {
	doctors    *Service[Doctor]
	chemists   *Service[Chemist]
	stockists  *Service[Stockist]
	products   *Service[Product]
	categories *Service[ProductCategory]
	db         *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		doctors:    NewService[Doctor](db),
		chemists:   NewService[Chemist](db),
		stockists:  NewService[Stockist](db),
		products:   NewService[Product](db),
		categories: NewService[ProductCategory](db),
		db:         db,
	}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	doctors := router.Group("/doctors")
	doctors.GET("", middleware.RequirePermission("doctor.view"), h.listDoctors)
	doctors.POST("", middleware.RequirePermission("doctor.create"), h.createDoctor)
	doctors.GET("/:id", middleware.RequirePermission("doctor.view"), h.getDoctor)
	doctors.PATCH("/:id", middleware.RequirePermission("doctor.update"), h.updateDoctor)
	doctors.DELETE("/:id", middleware.RequirePermission("doctor.update"), h.deleteDoctor)
	doctors.GET("/:id/territories", middleware.RequirePermission("doctor.view"), h.doctorTerritories)
	doctors.POST("/:id/territories", middleware.RequirePermission("doctor.update"), h.assignDoctorTerritory)

	chemists := router.Group("/chemists")
	chemists.GET("", middleware.RequirePermission("chemist.view"), h.listChemists)
	chemists.POST("", middleware.RequirePermission("chemist.manage"), h.createChemist)
	chemists.GET("/:id", middleware.RequirePermission("chemist.view"), h.getChemist)
	chemists.PATCH("/:id", middleware.RequirePermission("chemist.manage"), h.updateChemist)
	chemists.DELETE("/:id", middleware.RequirePermission("chemist.manage"), h.deleteChemist)

	stockists := router.Group("/stockists")
	stockists.GET("", middleware.RequirePermission("stockist.view"), h.listStockists)
	stockists.POST("", middleware.RequirePermission("stockist.manage"), h.createStockist)
	stockists.GET("/:id", middleware.RequirePermission("stockist.view"), h.getStockist)
	stockists.PATCH("/:id", middleware.RequirePermission("stockist.manage"), h.updateStockist)
	stockists.DELETE("/:id", middleware.RequirePermission("stockist.manage"), h.deleteStockist)

	products := router.Group("/products")
	products.GET("", middleware.RequirePermission("product.view"), h.listProducts)
	products.POST("", middleware.RequirePermission("product.manage"), h.createProduct)
	products.GET("/:id", middleware.RequirePermission("product.view"), h.getProduct)
	products.PATCH("/:id", middleware.RequirePermission("product.manage"), h.updateProduct)
	products.DELETE("/:id", middleware.RequirePermission("product.manage"), h.deleteProduct)

	categories := router.Group("/product-categories")
	categories.GET("", middleware.RequirePermission("product.view"), h.listCategories)
	categories.POST("", middleware.RequirePermission("product.manage"), h.createCategory)
}

func activeFilter(c *gin.Context) (*bool, bool) {
	raw := c.Query("is_active")
	if raw == "" {
		return nil, true
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, false
	}
	return &value, true
}

func tenant(c *gin.Context) uuid.UUID {
	return auth.MustPrincipal(c.Request.Context()).TenantID
}
