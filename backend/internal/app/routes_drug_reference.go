package app

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/handlers"
	"mediguide/internal/middleware"
)

func registerDrugReferenceRoutes(protected *gin.RouterGroup, drugReferenceH handlers.DrugReferenceHandler) {
	protected.GET("/drug-categories", middleware.RequireAnyPermission("drug.read", "guideline.read"), drugReferenceH.ListCategories)
	protected.GET("/drug-categories/:id", middleware.RequireAnyPermission("drug.read", "guideline.read"), drugReferenceH.GetCategory)
	protected.POST("/drug-categories", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugReferenceH.CreateCategory)
	protected.PATCH("/drug-categories/:id", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugReferenceH.UpdateCategory)
	protected.DELETE("/drug-categories/:id", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugReferenceH.DeleteCategory)

	protected.GET("/drug-tags", middleware.RequireAnyPermission("drug.read", "guideline.read"), drugReferenceH.ListTags)
	protected.GET("/drug-tags/:id", middleware.RequireAnyPermission("drug.read", "guideline.read"), drugReferenceH.GetTag)
	protected.POST("/drug-tags", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugReferenceH.CreateTag)
	protected.PATCH("/drug-tags/:id", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugReferenceH.UpdateTag)
	protected.DELETE("/drug-tags/:id", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugReferenceH.DeleteTag)

	protected.GET("/drug-classes", middleware.RequireAnyPermission("drug.read", "guideline.read"), drugReferenceH.ListClasses)
	protected.GET("/drug-classes/:id", middleware.RequireAnyPermission("drug.read", "guideline.read"), drugReferenceH.GetClass)
	protected.POST("/drug-classes", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugReferenceH.CreateClass)
	protected.PATCH("/drug-classes/:id", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugReferenceH.UpdateClass)
	protected.DELETE("/drug-classes/:id", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugReferenceH.DeleteClass)

	protected.GET("/therapeutic-categories", middleware.RequireAnyPermission("drug.read", "guideline.read"), drugReferenceH.ListTherapeuticCategories)
	protected.GET("/therapeutic-categories/:id", middleware.RequireAnyPermission("drug.read", "guideline.read"), drugReferenceH.GetTherapeuticCategory)
	protected.POST("/therapeutic-categories", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugReferenceH.CreateTherapeuticCategory)
	protected.PATCH("/therapeutic-categories/:id", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugReferenceH.UpdateTherapeuticCategory)
	protected.DELETE("/therapeutic-categories/:id", middleware.RequireAnyPermission("drug.write", "guideline.write"), drugReferenceH.DeleteTherapeuticCategory)
}
