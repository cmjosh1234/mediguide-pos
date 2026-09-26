package app

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/handlers"
	"mediguide/internal/middleware"
	"time"
)

func registerPublicAiRoutes(public *gin.RouterGroup, rateLimiter *middleware.RateLimiter, ragH handlers.RAGHandler) {
	public.POST("/assistant/ask",
		middleware.PrivateNoStore(),
		rateLimiter.Limit(middleware.Policy("public-general-ai-chat-minute", 6, time.Minute, 1), middleware.IPIdentity),
		rateLimiter.Limit(middleware.Policy("public-general-ai-chat-daily", 30, 24*time.Hour, 0), middleware.IPIdentity),
		rateLimiter.Concurrency("public-general-ai-chat-ip", 1, time.Minute, middleware.IPIdentity),
		rateLimiter.Concurrency("public-general-ai-chat-global", 10, 2*time.Minute, middleware.StaticIdentity("global")),
		ragH.AskPublic)
	public.POST("/guidelines/:id/ask",
		middleware.PrivateNoStore(),
		rateLimiter.Limit(middleware.Policy("public-ai-chat-minute", 6, time.Minute, 1), middleware.IPIdentity),
		rateLimiter.Limit(middleware.Policy("public-ai-chat-daily", 30, 24*time.Hour, 0), middleware.IPIdentity),
		rateLimiter.Concurrency("public-ai-chat-ip", 1, time.Minute, middleware.IPIdentity),
		rateLimiter.Concurrency("public-ai-chat-global", 10, 2*time.Minute, middleware.StaticIdentity("global")),
		ragH.AskPublishedGuideline)
}
