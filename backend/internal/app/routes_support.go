package app

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/handlers"
	"mediguide/internal/middleware"
	"time"
)

func registerSupportRoutes(protected *gin.RouterGroup, rateLimiter *middleware.RateLimiter, supportH handlers.SupportHandler, helpContentH handlers.HelpContentHandler) {
	protected.GET("/support/tickets", supportH.ListTickets)
	protected.GET("/support/tickets/:id", supportH.GetTicket)
	protected.POST("/support/tickets", rateLimiter.Limit(middleware.Policy("support-ticket-create", 5, time.Hour, 1), middleware.UserIdentity), supportH.CreateTicket)
	protected.PATCH("/support/tickets/:id", supportH.UpdateTicket)
	protected.DELETE("/support/tickets/:id", supportH.DeleteTicket)
	protected.GET("/support/tickets/:id/replies", supportH.ListReplies)
	protected.POST("/support/tickets/:id/replies", rateLimiter.Limit(middleware.Policy("support-reply-create", 30, time.Minute, 5), middleware.UserIdentity), supportH.CreateReply)

	protected.GET("/faqs", helpContentH.ListFAQs)
	protected.GET("/faqs/:id", helpContentH.GetFAQ)
	protected.POST("/faqs", middleware.RequireAnyPermission("admin.all", "content.write", "guideline.write"), helpContentH.CreateFAQ)
	protected.PATCH("/faqs/:id", middleware.RequireAnyPermission("admin.all", "content.write", "guideline.write"), helpContentH.UpdateFAQ)
	protected.DELETE("/faqs/:id", middleware.RequireAnyPermission("admin.all", "content.write", "guideline.write"), helpContentH.DeleteFAQ)
	protected.GET("/faq-tags", helpContentH.ListTags)
	protected.GET("/faq-tags/:id", helpContentH.GetTag)
	protected.POST("/faq-tags", middleware.RequireAnyPermission("admin.all", "content.write", "guideline.write"), helpContentH.CreateTag)
	protected.PATCH("/faq-tags/:id", middleware.RequireAnyPermission("admin.all", "content.write", "guideline.write"), helpContentH.UpdateTag)
	protected.DELETE("/faq-tags/:id", middleware.RequireAnyPermission("admin.all", "content.write", "guideline.write"), helpContentH.DeleteTag)
	protected.POST("/faq-tags/recalculate-usage", middleware.RequireAnyPermission("admin.all", "content.write", "guideline.write"), rateLimiter.Limit(middleware.Policy("faq-tag-recalculation", 5, 15*time.Minute, 0), middleware.UserIdentity), helpContentH.RecalculateTagUsage)
	protected.GET("/documentation", helpContentH.ListDocumentation)
	protected.GET("/documentation/:id", helpContentH.GetDocumentation)
	protected.POST("/documentation", middleware.RequireAnyPermission("admin.all", "content.write", "guideline.write"), helpContentH.CreateDocumentation)
	protected.PATCH("/documentation/:id", middleware.RequireAnyPermission("admin.all", "content.write", "guideline.write"), helpContentH.UpdateDocumentation)
	protected.DELETE("/documentation/:id", middleware.RequireAnyPermission("admin.all", "content.write", "guideline.write"), helpContentH.DeleteDocumentation)
}
