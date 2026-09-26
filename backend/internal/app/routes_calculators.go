package app

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/handlers"
	"mediguide/internal/middleware"
	"time"
)

func registerCalculatorsRoutes(protected *gin.RouterGroup, rateLimiter *middleware.RateLimiter, calculatorH handlers.CalculatorHandler) {
	protected.GET("/calculators", middleware.RequireAnyPermission("calculator.read", "guideline.read"), calculatorH.List)
	protected.GET("/calculators/:id", middleware.RequireAnyPermission("calculator.read", "guideline.read"), calculatorH.Get)
	protected.POST("/calculators", middleware.RequireAnyPermission("calculator.write", "guideline.write"), calculatorH.Create)
	protected.PATCH("/calculators/:id", middleware.RequireAnyPermission("calculator.write", "guideline.write"), calculatorH.Update)
	protected.DELETE("/calculators/:id", middleware.RequireAnyPermission("calculator.write", "guideline.write"), calculatorH.Delete)
	protected.GET("/calculators/:id/content", middleware.RequireAnyPermission("calculator.read", "guideline.read"), calculatorH.Content)
	protected.GET("/calculators/:id/definition", middleware.RequireAnyPermission("calculator.read", "guideline.read"), calculatorH.Definition)
	protected.GET("/calculators/:id/versions", middleware.RequirePermission("calculator.write"), calculatorH.ListVersions)
	protected.POST("/calculators/:id/versions", middleware.RequirePermission("calculator.write"), calculatorH.CreateVersion)
	protected.GET("/calculator-versions/review-queue", middleware.RequirePermission("calculator.review"), calculatorH.ReviewQueue)
	protected.GET("/calculator-versions/:id", middleware.RequirePermission("calculator.write"), calculatorH.GetVersion)
	protected.GET("/calculator-versions/:id/preview", middleware.RequirePermission("calculator.review"), calculatorH.PreviewVersion)
	protected.PATCH("/calculator-versions/:id", middleware.RequirePermission("calculator.write"), calculatorH.UpdateVersion)
	protected.DELETE("/calculator-versions/:id", middleware.RequirePermission("calculator.write"), calculatorH.DeleteVersion)
	protected.POST("/calculator-versions/:id/duplicate", middleware.RequirePermission("calculator.write"), calculatorH.DuplicateVersion)
	protected.POST("/calculator-versions/:id/validate", middleware.RequirePermission("calculator.write"), rateLimiter.Limit(middleware.Policy("calculator-version-validate", 30, time.Minute, 5), middleware.UserIdentity), calculatorH.ValidateVersion)
	protected.POST("/calculator-versions/:id/test", middleware.RequirePermission("calculator.write"), rateLimiter.Limit(middleware.Policy("calculator-version-test", 20, time.Minute, 3), middleware.UserIdentity), calculatorH.TestVersion)
	protected.POST("/calculator-versions/:id/submit", middleware.RequirePermission("calculator.write"), calculatorH.SubmitVersion)
	protected.POST("/calculator-versions/:id/approve", middleware.RequirePermission("calculator.review"), calculatorH.ApproveVersion)
	protected.POST("/calculator-versions/:id/publish", middleware.RequirePermission("calculator.publish"), rateLimiter.Limit(middleware.Policy("calculator-version-publish", 10, time.Hour, 1), middleware.UserIdentity), calculatorH.PublishVersion)
	protected.POST("/calculators/:id/runtime/legacy", middleware.RequirePermission("calculator.publish"), rateLimiter.Limit(middleware.Policy("calculator-runtime-rollback", 10, time.Hour, 1), middleware.UserIdentity), calculatorH.SelectLegacyRuntime)
	protected.POST("/calculator-versions/:id/withdraw", middleware.RequirePermission("calculator.withdraw"), calculatorH.WithdrawVersion)
	protected.GET("/calculator-versions/:id/audit", middleware.RequirePermission("calculator.review"), calculatorH.VersionAudit)
	protected.POST("/calculator-versions/:id/review-comments", middleware.RequirePermission("calculator.review"), calculatorH.AddVersionReviewComment)
	protected.POST("/calculators/:id/usage", calculatorH.StartUsage)
	protected.PATCH("/calculator-usage/:usageId", calculatorH.FinishUsage)
}
