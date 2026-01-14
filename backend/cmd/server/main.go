package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prediction-market/backend/internal/config"
	"github.com/prediction-market/backend/internal/handlers"
	"github.com/prediction-market/backend/internal/middleware"
	"github.com/prediction-market/backend/internal/models"
	"github.com/prediction-market/backend/internal/services/orderbook"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cfg := config.Load()

	db, err := models.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	obm := orderbook.NewOrderBookManager()

	marketHandler := handlers.NewMarketHandler(db)
	orderHandler := handlers.NewOrderHandler(db, obm)
	adminHandler := handlers.NewAdminHandler(db)

	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Wallet-Address")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Public API
	api := r.Group("/api")
	{
		api.GET("/markets", marketHandler.List)
		api.GET("/markets/:id", marketHandler.Get)
		api.GET("/markets/:id/trades", marketHandler.GetTrades)
		api.GET("/markets/:id/orderbook", orderHandler.GetOrderBook)
	}

	// User API (requires wallet)
	user := r.Group("/api")
	user.Use(middleware.WalletAuth())
	{
		user.POST("/orders", orderHandler.PlaceOrder)
		user.DELETE("/orders/:id", orderHandler.CancelOrder)
		user.GET("/user/orders", orderHandler.GetUserOrders)
	}

	// Admin API (requires JWT)
	admin := r.Group("/api/admin")
	admin.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		admin.POST("/markets", adminHandler.CreateMarket)
		admin.POST("/markets/:id/resolve", adminHandler.ResolveMarket)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
