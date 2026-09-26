package app

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/handlers"
	"mediguide/internal/middleware"
	"time"
)

func registerDiscoveryRoutes(protected *gin.RouterGroup, rateLimiter *middleware.RateLimiter, searchH handlers.SearchHandler, ragH handlers.RAGHandler, protocolH handlers.ProtocolHandler) {
	protected.GET("/search", middleware.RequirePermission("guideline.read"), rateLimiter.Limit(middleware.Policy("guideline-search", 60, time.Minute, 10), middleware.UserIdentity), searchH.Search)
	protected.POST("/chat/ask", middleware.RequirePermission("chat.ask"),
		rateLimiter.Limit(middleware.Policy("ai-chat-minute", 10, time.Minute, 2), middleware.UserIdentity),
		rateLimiter.Limit(middleware.Policy("ai-chat-daily", 100, 24*time.Hour, 0), middleware.UserIdentity),
		rateLimiter.Concurrency("ai-chat-user", 2, 3*time.Minute, middleware.UserIdentity),
		rateLimiter.Concurrency("ai-chat-global", 20, 3*time.Minute, middleware.StaticIdentity("global")), ragH.Ask)

	protected.POST("/protocols", middleware.RequirePermission("protocol.write"), protocolH.Create)
	protected.GET("/protocols", middleware.RequirePermission("protocol.read"), protocolH.List)
	protected.GET("/protocols/:id", middleware.RequirePermission("protocol.read"), protocolH.Get)
	protected.POST("/protocols/:id/run", middleware.RequirePermission("protocol.read"), rateLimiter.Limit(middleware.Policy("protocol-run", 60, time.Minute, 10), middleware.UserIdentity), protocolH.Run)
}
