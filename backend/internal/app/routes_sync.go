package app

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/handlers"
	"mediguide/internal/middleware"
	"time"
)

func registerSyncRoutes(protected *gin.RouterGroup, rateLimiter *middleware.RateLimiter, syncH handlers.SyncHandler) {
	protected.GET("/sync/manifest", middleware.RequirePermission("sync.read"), syncH.Manifest)
	protected.POST("/sync/packages", middleware.RequirePermission("admin.all"), rateLimiter.Limit(middleware.Policy("sync-package-create", 5, time.Hour, 0), middleware.UserIdentity), rateLimiter.Concurrency("sync-package-create", 1, 30*time.Minute, middleware.UserIdentity), syncH.CreatePackage)
	protected.GET("/sync/packages/:id/download", middleware.RequirePermission("sync.read"), rateLimiter.Limit(middleware.Policy("sync-download-url", 30, time.Minute, 5), middleware.UserIdentity), syncH.Download)
}
