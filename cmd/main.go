package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"memenow.ai/memenow-resource-manager/operator"
	"net/http"
)

func main() {
	router := gin.Default()

	router.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Service is running",
		})
	})

	operatorGroup := router.Group("/v1")
	{
		operatorGroup.POST("/create", func(c *gin.Context) {
			chartName := c.Query("chart")
			namespaceName := c.Query("namespace")
			releaseName := c.Query("release")
			err := operator.InstallHelm([]string{chartName}, []string{namespaceName}, []string{releaseName})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": fmt.Sprintf("Error installing helm chart: %s", err.Error()),
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "Install OK",
			})
		})
	}

	err := router.Run(":8080")
	if err != nil {
		log.Fatalf("Failed to run server: %s", err.Error())
	}
}
