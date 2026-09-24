package app

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/handlers"
)

func registerPublicHubsRoutes(public *gin.RouterGroup, contentHubH handlers.ContentHubHandler, diseaseH handlers.DiseaseHandler) {
	public.GET("/hubs", contentHubH.PublicList)
	public.GET("/hubs/:slug", contentHubH.PublicGet)
	public.GET("/hubs/:slug/pillars/:pillarSlug", contentHubH.PublicPillar)
	public.GET("/diseases", diseaseH.PublicList)
	public.GET("/diseases/hierarchy", diseaseH.PublicHierarchy)
	public.GET("/diseases/:slug", diseaseH.PublicGet)
}
