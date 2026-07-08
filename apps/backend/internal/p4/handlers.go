package p4

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the p4 workspace-listing endpoint used by the
// workspace↔P4 binding UI. pcraft performs no p4 mutations, so there are no
// checkout/submit routes.
func RegisterRoutes(api *gin.RouterGroup, svc *Service) {
	api.GET("/p4/workspaces", func(c *gin.Context) {
		workspaces, err := svc.ListWorkspaces(c.Request.Context(), c.Query("p4user"))
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"workspaces": workspaces, "total": len(workspaces)})
	})
}
