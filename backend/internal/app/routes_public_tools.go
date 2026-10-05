package app

import (
	"mediguide/internal/handlers"

	"github.com/gin-gonic/gin"
)

// Guest reads share the public group's IP rate limit. Editorial and usage
// endpoints remain on the authenticated v2 group.
func registerPublicToolsRoutes(public *gin.RouterGroup, calculatorH handlers.CalculatorHandler, contentReferenceH handlers.ContentReferenceHandler) {
	public.GET("/calculators", calculatorH.PublicList)
	public.GET("/calculators/:id", calculatorH.PublicGet)
	public.GET("/calculators/:id/definition", calculatorH.PublicDefinition)
	public.GET("/calculators/:id/content", calculatorH.PublicContent)
	public.GET("/ministry-directory", contentReferenceH.PublicListDirectory)
	public.GET("/ministry-directory/:id", contentReferenceH.PublicGetDirectory)
}
