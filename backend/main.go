package main

import (
	"evo-image-api/api"
	"evo-image-api/config"
	"evo-image-api/core"
	"evo-image-api/database"
	"evo-image-api/middleware"
	"evo-image-api/models"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Core Initialization
	config.LoadConfig()
	database.InitSQLite()

	// 2. Database Migration (Ensuring no schema bugs)
	err := database.DB.AutoMigrate(
		&models.User{}, 
		&models.APIKey{}, 
		&models.Account{},
	)
	if err != nil {
		log.Fatalf("Migration bug detected: %v", err)
	}

	// 3. Router Setup
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Global Middleware
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	// Public Routes
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "Singularity Kernel Online", "time": time.Now().Format(time.RFC3339)})
	})

	// V1 API (Standard OpenAI Compatibility)
	v1 := r.Group("/v1")
	v1.Use(middleware.AuthMiddleware())
	{
		v1.POST("/images/generations", api.GenerateImageHandler)
	}

	// Image Proxy Route (Public, but requires encoded params)
	r.GET("/v1/p/img", api.ProxyImageHandler)

	// Admin API (Management Dashboard)
	admin := r.Group("/v1/admin")
	{
		admin.GET("/stats", api.GetStats)
		admin.GET("/accounts", api.GetAccounts)
		admin.POST("/accounts/import", api.ImportAccounts)
		admin.GET("/accounts/export", api.ExportAccounts)
	}

	// 4. Background Evolver
	// Detector scans for capability and updates has_img2 status in real-time
	detector := core.NewDetector(2 * time.Hour)
	detector.Start()

	log.Printf("🚀 Evo-ImageAPI v4.0 starting on port %s...", config.AppConfig.Port)
	if err := r.Run(":" + config.AppConfig.Port); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
