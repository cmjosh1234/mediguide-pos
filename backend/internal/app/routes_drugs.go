package app

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/handlers"
	"mediguide/internal/middleware"
)

func registerDrugsRoutes(protected *gin.RouterGroup, drugH handlers.DrugHandler) {
	protected.GET("/drugs", middleware.RequireAnyPermission("drug.read", "guideline.read"), drugH.List)
	protected.GET("/drugs/:id", middleware.RequireAnyPermission("drug.read", "guideline.read"), drugH.Get)
	protected.POST("/drugs", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugH.Create)
	protected.PATCH("/drugs/:id", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugH.Update)
	protected.DELETE("/drugs/:id", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugH.Delete)
	protected.POST("/drugs/:id/usage", drugH.RecordUsage)
}
