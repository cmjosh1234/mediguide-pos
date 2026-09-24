package app

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/handlers"
	"mediguide/internal/middleware"
)

func registerUsersRoutes(protected *gin.RouterGroup, userH handlers.UserHandler) {
	protected.GET("/users", middleware.RequirePermission("admin.all"), userH.List)
	protected.GET("/users/:id", userH.Get)
	protected.POST("/users", middleware.RequirePermission("admin.all"), userH.Create)
	protected.PATCH("/users/:id", userH.Update)
	protected.DELETE("/users/:id", middleware.RequirePermission("admin.all"), userH.Delete)
	protected.POST("/users/:id/verification", middleware.RequirePermission("admin.all"), userH.Verify)
	protected.GET("/roles", middleware.RequirePermission("admin.all"), userH.ListRoles)
	protected.GET("/roles/:id", middleware.RequirePermission("admin.all"), userH.GetRole)
	protected.POST("/roles", middleware.RequirePermission("admin.all"), userH.CreateRole)
	protected.PATCH("/roles/:id", middleware.RequirePermission("admin.all"), userH.UpdateRole)
	protected.DELETE("/roles/:id", middleware.RequirePermission("admin.all"), userH.DeleteRole)
	protected.GET("/permissions", middleware.RequirePermission("admin.all"), userH.ListPermissions)
	protected.GET("/roles/:id/permissions", middleware.RequirePermission("admin.all"), userH.GetRolePermissions)
	protected.PUT("/roles/:id/permissions", middleware.RequirePermission("admin.all"), userH.SetRolePermissions)
}
