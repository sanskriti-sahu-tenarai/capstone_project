package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"order-management/config"
	"order-management/controller"
	"order-management/database"
	_ "order-management/docs"
	"order-management/middleware"
	"order-management/repository"
	"order-management/service"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Order Management Microservice API
// @version         1.0
// @description     Production-ready Order Management Microservice with JWT Auth, PostgreSQL, and Resilient Product Microservice client.
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your token with format: Bearer <token>
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.LoadConfig()
	middleware.SetSecretKey(cfg.JWTSecret)

	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		slog.Error("Database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	orderRepo := repository.NewOrderRepository(db)
	productClient := service.NewProductClient(cfg.ProductServiceURL)
	orderService := service.NewOrderService(orderRepo, productClient)
	orderCtrl := controller.NewOrderController(orderService)

	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.StructuredLogger())
	router.Use(middleware.SecurityHeaders())

	// Swagger Interactive Testing UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health and observability probes
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive"})
	})
	router.GET("/health/ready", func(c *gin.Context) {
		if err := db.PingContext(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": "db ping failure"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Authentication endpoint
	router.POST("/api/v1/auth/login", orderCtrl.Login)

	// Protected business endpoints
	v1 := router.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware())
	{
		v1.POST("/orders", middleware.RoleMiddleware("user", "admin"), orderCtrl.CreateOrder)
		v1.GET("/orders/:id", middleware.RoleMiddleware("user", "admin"), orderCtrl.GetOrder)
		v1.GET("/orders", middleware.RoleMiddleware("user", "admin"), orderCtrl.ListOrders)
		v1.PUT("/orders/:id/status", middleware.RoleMiddleware("admin"), orderCtrl.UpdateOrderStatus)
	}

	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router,
	}

	go func() {
		slog.Info("Order Management Microservice started", "port", cfg.AppPort, "swagger_ui", "http://localhost:"+cfg.AppPort+"/swagger/index.html")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Listen failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Gracefully shutting down Order Microservice...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Forced shutdown required", "error", err)
	}
	slog.Info("Order Microservice stopped cleanly")
}
