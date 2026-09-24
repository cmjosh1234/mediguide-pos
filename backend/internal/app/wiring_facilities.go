package app

import (
	cachepkg "mediguide/internal/cache"
	"mediguide/internal/handlers"
	"mediguide/internal/services"

	"gorm.io/gorm"
)

func wireFacilities(database *gorm.DB, cacheStore *cachepkg.Store) handlers.FacilityHandler {
	return handlers.NewFacilityHandler(services.FacilityService{DB: database, Cache: cacheStore})
}
