package app

import (
	cachepkg "mediguide/internal/cache"
	"mediguide/internal/config"
	"mediguide/internal/handlers"
	"mediguide/internal/services"
	"mediguide/internal/storage"

	"gorm.io/gorm"
)

func wireGuidelines(cfg config.Config, database *gorm.DB, store *storage.MinioStore, cacheStore *cachepkg.Store) (services.GuidelineService, handlers.GuidelineHandler, handlers.PublicGuidelineHandler, handlers.GuidelineContentHandler, handlers.GuidelineLibraryHandler) {
	guidelineSvc := services.GuidelineService{DB: database, Store: store, Cache: cacheStore}
	publicSvc := services.PublicGuidelineService{DB: database, Store: store, Cache: cacheStore}
	return guidelineSvc,
		handlers.GuidelineHandler{Service: guidelineSvc, MaxUploadMB: cfg.MaxUploadMB, DirectUploads: cfg.GuidelineDirectUploads && cfg.S3PublicEndpoint != ""},
		handlers.PublicGuidelineHandler{Service: publicSvc, Content: publicSvc},
		handlers.GuidelineContentHandler{Service: services.GuidelineContentService{DB: database, Cache: cacheStore}},
		handlers.GuidelineLibraryHandler{Service: services.GuidelineLibraryService{DB: database}}
}
