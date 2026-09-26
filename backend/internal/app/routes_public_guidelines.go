package app

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/handlers"
	"mediguide/internal/middleware"
	"time"
)

func registerPublicGuidelinesRoutes(public *gin.RouterGroup, rateLimiter *middleware.RateLimiter, publicGuidelineH handlers.PublicGuidelineHandler, searchH handlers.SearchHandler) {
	public.GET("/guidelines", publicGuidelineH.List)
	public.GET("/search", rateLimiter.Limit(middleware.Policy("public-search", 60, time.Minute, 10), middleware.IPIdentity), searchH.PublicSearch)
	public.GET("/guidelines/:id", publicGuidelineH.Get)
	public.GET("/guidelines/:id/manifest", publicGuidelineH.Manifest)
	public.GET("/guidelines/:id/content", publicGuidelineH.ContentBundle)
	public.GET("/guidelines/:id/sections", publicGuidelineH.Sections)
	public.GET("/guidelines/:id/sections/:sectionId", publicGuidelineH.Section)
	public.GET("/guidelines/:id/tables", publicGuidelineH.Tables)
	public.GET("/guidelines/:id/figures", publicGuidelineH.Figures)
	public.GET("/guidelines/:id/algorithms", publicGuidelineH.Algorithms)
	public.GET("/guidelines/:id/original", rateLimiter.Limit(middleware.Policy("public-guideline-original", 30, time.Minute, 5), middleware.IPIdentity), publicGuidelineH.Original)
	public.GET("/guidelines/:id/original/download", rateLimiter.Limit(middleware.Policy("public-guideline-original-download", 20, time.Minute, 3), middleware.IPIdentity), publicGuidelineH.OriginalDownload)
	public.GET("/guidelines/:id/offline-package", rateLimiter.Limit(middleware.Policy("public-guideline-offline", 20, time.Minute, 3), middleware.IPIdentity), publicGuidelineH.OfflinePackage)
	public.GET("/guidelines/:id/offline-package/download", rateLimiter.Limit(middleware.Policy("public-guideline-offline-download", 10, time.Minute, 2), middleware.IPIdentity), publicGuidelineH.OfflinePackageDownload)
	public.GET("/guidelines/:id/assets/:assetId/download", rateLimiter.Limit(middleware.Policy("public-guideline-asset-download", 60, time.Minute, 10), middleware.IPIdentity), publicGuidelineH.AssetDownload)
	public.GET("/guidelines/:id/markdown", rateLimiter.Limit(middleware.Policy("public-markdown", 60, time.Minute, 10), middleware.IPIdentity), publicGuidelineH.Markdown)
}
