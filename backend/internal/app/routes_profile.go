package app

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/handlers"
	"mediguide/internal/middleware"
	"time"
)

func registerProfileRoutes(protected *gin.RouterGroup, rateLimiter *middleware.RateLimiter, referenceH handlers.ReferenceHandler, contentReferenceH handlers.ContentReferenceHandler, progressUsageH handlers.ProgressUsageHandler, guidelineLibraryH handlers.GuidelineLibraryHandler, conversationH handlers.ConversationHandler) {
	protected.GET("/settings", middleware.RequirePermission("admin.all"), referenceH.ListSettings)
	protected.POST("/settings", middleware.RequirePermission("admin.all"), referenceH.CreateSetting)
	protected.GET("/languages", contentReferenceH.ListLanguages)
	protected.GET("/languages/:id", contentReferenceH.GetLanguage)
	protected.POST("/languages", middleware.RequirePermission("admin.all"), contentReferenceH.CreateLanguage)
	protected.PATCH("/languages/:id", middleware.RequirePermission("admin.all"), contentReferenceH.UpdateLanguage)
	protected.DELETE("/languages/:id", middleware.RequirePermission("admin.all"), contentReferenceH.DeleteLanguage)
	protected.GET("/reading-progress", progressUsageH.ListProgress)
	protected.GET("/reading-progress/:guidelineId", progressUsageH.GetProgress)
	protected.PUT("/reading-progress/:guidelineId", rateLimiter.Limit(middleware.Policy("reading-progress-write", 120, time.Minute, 20), middleware.UserIdentity), progressUsageH.UpsertProgress)
	protected.DELETE("/reading-progress/:guidelineId", progressUsageH.DeleteProgress)
	protected.GET("/library/collections", guidelineLibraryH.ListCollections)
	protected.POST("/library/collections", guidelineLibraryH.CreateCollection)
	protected.GET("/library/collections/:id", guidelineLibraryH.GetCollection)
	protected.PATCH("/library/collections/:id", guidelineLibraryH.UpdateCollection)
	protected.DELETE("/library/collections/:id", guidelineLibraryH.DeleteCollection)
	protected.POST("/library/collections/:id/items", guidelineLibraryH.AddCollectionItem)
	protected.GET("/library/collections/:id/items", guidelineLibraryH.ListCollectionItems)
	protected.DELETE("/library/collections/:id/items/:guidelineId", guidelineLibraryH.RemoveCollectionItem)
	protected.GET("/library/downloads", guidelineLibraryH.ListDownloads)
	protected.POST("/library/downloads", rateLimiter.Limit(middleware.Policy("guideline-download-record", 120, time.Minute, 20), middleware.UserIdentity), guidelineLibraryH.RecordDownload)
	protected.POST("/usage/guidelines", rateLimiter.Limit(middleware.Policy("usage-event-write", 120, time.Minute, 20), middleware.UserIdentity), progressUsageH.RecordGuidelineUsage)
	protected.POST("/usage/abbreviations", rateLimiter.Limit(middleware.Policy("usage-event-write", 120, time.Minute, 20), middleware.UserIdentity), progressUsageH.RecordAbbreviationUsage)
	protected.POST("/usage/ai", rateLimiter.Limit(middleware.Policy("usage-event-write", 120, time.Minute, 20), middleware.UserIdentity), progressUsageH.RecordAIUsage)
	protected.GET("/analytics/usage", middleware.RequireAnyPermission("admin.all", "analytics.read", "sync.read"), rateLimiter.Limit(middleware.Policy("analytics-read", 30, time.Minute, 5), middleware.UserIdentity), progressUsageH.UsageAggregates)
	protected.GET("/conversations", conversationH.List)
	protected.POST("/conversations", rateLimiter.Limit(middleware.Policy("conversation-create", 10, time.Hour, 2), middleware.UserIdentity), conversationH.Create)
	protected.GET("/conversations/:id", conversationH.Get)
	protected.DELETE("/conversations/:id", conversationH.Delete)
	protected.GET("/conversations/:id/messages", conversationH.ListMessages)
	protected.POST("/conversations/:id/messages", rateLimiter.Limit(middleware.Policy("conversation-message", 30, time.Minute, 5), middleware.UserIdentity), conversationH.CreateMessage)
	protected.POST("/conversations/:id/messages/:messageId/read", conversationH.MarkRead)
	protected.POST("/conversations/:id/messages/:messageId/reaction", conversationH.React)
}
