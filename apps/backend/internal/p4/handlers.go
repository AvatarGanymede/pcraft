package p4

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type checkoutRequest struct {
	Files []string `json:"files"`
}

func RegisterRoutes(api *gin.RouterGroup, svc *Service) {
	api.GET("/p4/workspaces", func(c *gin.Context) {
		workspaces, err := svc.ListWorkspaces(c.Request.Context(), c.Query("p4user"))
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"workspaces": workspaces, "total": len(workspaces)})
	})

	api.POST("/tasks/:id/p4/checkout", func(c *gin.Context) {
		var req checkoutRequest
		if err := c.ShouldBindJSON(&req); err != nil || len(req.Files) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "files is required"})
			return
		}
		result, err := svc.Checkout(c.Request.Context(), c.Param("id"), req.Files)
		if err != nil && result != nil && !result.Allowed {
			c.JSON(http.StatusConflict, result)
			return
		}
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})
}
