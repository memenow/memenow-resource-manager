// Package main is the entry point for the memenow-resource-manager service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"memenow.ai/memenow-resource-manager/operator"
)

// Version information - set via ldflags at build time
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// installHelm is the function used to install Helm charts. Replaced in tests.
var installHelm = operator.InstallHelmWithContext

const (
	// Default server configuration
	defaultPort            = "8080"
	defaultShutdownTimeout = 30 * time.Second
	defaultRequestTimeout  = 5 * time.Minute
)

func main() {
	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting memenow-resource-manager version=%s commit=%s buildTime=%s", Version, GitCommit, BuildTime)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Create router
	router := setupRouter()

	// Create HTTP server
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

func setupRouter() *gin.Engine {
	// Set Gin to release mode in production
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/ok", healthCheckHandler)
	router.GET("/health", healthCheckHandler)
	router.GET("/version", versionHandler)

	// API routes
	v1 := router.Group("/v1")
	{
		v1.POST("/create", createHelmReleaseHandler)
	}

	return router
}

func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "Service is running",
	})
}

func versionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":   Version,
		"gitCommit": GitCommit,
		"buildTime": BuildTime,
	})
}

func createHelmReleaseHandler(c *gin.Context) {
	// Extract query parameters
	chartName := c.Query("chart")
	namespaceName := c.Query("namespace")
	releaseName := c.Query("release")

	// Validate required parameters
	if chartName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required parameter: chart",
		})
		return
	}

	if namespaceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required parameter: namespace",
		})
		return
	}

	if releaseName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required parameter: release",
		})
		return
	}

	// Create context with timeout for the operation
	ctx, cancel := context.WithTimeout(c.Request.Context(), defaultRequestTimeout)
	defer cancel()

	// Log the installation request
	log.Printf("Installing Helm chart: chart=%s, namespace=%s, release=%s",
		chartName, namespaceName, releaseName)

	// Install Helm chart with context support
	err := installHelm(
		ctx,
		[]string{chartName},
		[]string{namespaceName},
		[]string{releaseName},
		nil, // Could accept values from request body in the future
	)

	if err != nil {
		// Check if context was canceled or timed out
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("Helm installation timed out: %v", err)
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"error":   "Installation timed out",
				"message": err.Error(),
			})
			return
		}

		if errors.Is(err, context.Canceled) {
			log.Printf("Helm installation canceled: %v", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "Installation canceled",
				"message": err.Error(),
			})
			return
		}

		log.Printf("Error installing Helm chart: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to install Helm chart",
			"message": err.Error(),
		})
		return
	}

	log.Printf("Successfully installed Helm chart: %s", releaseName)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Helm chart installed successfully",
		"release": releaseName,
	})
}
