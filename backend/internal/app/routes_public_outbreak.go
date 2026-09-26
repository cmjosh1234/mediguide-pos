package app

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/handlers"
	"mediguide/internal/middleware"
	"time"
)

func registerPublicOutbreakRoutes(public *gin.RouterGroup, rateLimiter *middleware.RateLimiter, supportH handlers.SupportHandler, outbreakH handlers.OutbreakHandler, contentHubH handlers.ContentHubHandler) {
	public.POST("/support/tickets", rateLimiter.Limit(middleware.Policy("public-support-ticket-create", 5, time.Hour, 1), middleware.IPIdentity), supportH.CreateGuestTicket)
	outbreakReadLimit := rateLimiter.Limit(middleware.Policy("public-outbreaks", 90, time.Minute, 15), middleware.IPIdentity)
	public.GET("/outbreaks", outbreakReadLimit, outbreakH.List)
	public.GET("/outbreaks/:id", outbreakReadLimit, outbreakH.Get)
	public.GET("/outbreaks/:id/hub", outbreakReadLimit, contentHubH.PublicOutbreakHub)
	public.GET("/outbreaks/:id/updates", outbreakReadLimit, outbreakH.Updates)
	public.GET("/outbreaks/:id/resources", outbreakReadLimit, outbreakH.Resources)
	public.GET("/outbreak-resources", outbreakReadLimit, outbreakH.ListResources)
	public.GET("/outbreaks/:id/documents", outbreakReadLimit, outbreakH.Documents)
	public.GET("/outbreaks/:id/documents/:documentId", outbreakReadLimit, outbreakH.GetDocument)
	public.GET("/outbreaks/:id/documents/:documentId/download", outbreakReadLimit, outbreakH.DocumentDownload)
	public.GET("/outbreak-documents", outbreakReadLimit, outbreakH.SearchDocuments)
	public.GET("/outbreak-documents/:documentId", outbreakReadLimit, outbreakH.GetDocumentGlobal)
	public.GET("/outbreak-documents/:documentId/content", outbreakReadLimit, outbreakH.DocumentContent)
	public.GET("/situation-reports", outbreakReadLimit, outbreakH.ListReports)
	public.GET("/situation-reports/:id", outbreakReadLimit, outbreakH.GetReport)
	public.GET("/situation-reports/:id/asset", outbreakReadLimit, outbreakH.ReportAsset)
}
