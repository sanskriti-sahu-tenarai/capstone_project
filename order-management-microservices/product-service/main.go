package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type Product struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

type ProductCatalog struct {
	mu       sync.RWMutex
	products map[int64]Product
}

func newProductCatalog() *ProductCatalog {
	return &ProductCatalog{
		products: map[int64]Product{
			501: {ID: 501, Name: "Wireless Noise-Canceling Headphones", Price: 120.00, Stock: 50},
			502: {ID: 502, Name: "Mechanical Gaming Keyboard", Price: 85.50, Stock: 40},
			503: {ID: 503, Name: "Ultra-wide Ergonomic Monitor 34-inch", Price: 499.99, Stock: 15},
			504: {ID: 504, Name: "USB-C Multi-port Hub", Price: 35.00, Stock: 100},
		},
	}
}

func (c *ProductCatalog) Get(id int64) (Product, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.products[id]
	return p, ok
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	catalog := newProductCatalog()

	router := gin.Default()

	// Microservice Health Probes
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive", "service": "product-microservice"})
	})

	router.GET("/health/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready", "service": "product-microservice"})
	})

	// Product Catalog APIs
	v1 := router.Group("/api/v1")
	{
		// Returns price directly for Order Service integration
		v1.GET("/products/:id/price", func(c *gin.Context) {
			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
				return
			}

			p, ok := catalog.Get(id)
			if !ok {
				// Default dynamic pricing if not in catalog
				c.JSON(http.StatusOK, gin.H{
					"productId": id,
					"price":     75.00,
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"productId": p.ID,
				"price":     p.Price,
			})
		})

		// Full product details
		v1.GET("/products/:id", func(c *gin.Context) {
			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
				return
			}

			p, ok := catalog.Get(id)
			if !ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
				return
			}

			c.JSON(http.StatusOK, p)
		})
	}

	port := os.Getenv("PRODUCT_PORT")
	if port == "" {
		port = "8081"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		slog.Info("Product Microservice running", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Product Microservice error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	slog.Info("Product Microservice stopped cleanly")
}
