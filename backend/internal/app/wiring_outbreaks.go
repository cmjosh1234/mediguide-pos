package app

import (
	"mediguide/internal/config"
	"mediguide/internal/handlers"
	"mediguide/internal/services"
	"mediguide/internal/storage"

	"gorm.io/gorm"
)

func wireOutbreaks(cfg config.Config, database *gorm.DB, store *storage.MinioStore) (handlers.OutbreakHandler, handlers.OutbreakAdminHandler) {
	publicSvc := services.OutbreakService{DB: database, Store: store, AllowedExternalHosts: cfg.NotificationActionExternalHosts}
	adminSvc := services.OutbreakAdminService{DB: database, Store: store, AllowedExternalHosts: cfg.NotificationActionExternalHosts}
	adminSvc.DocumentNotifications = &services.OutbreakDocumentNotificationService{DB: database, AllowedActionHosts: cfg.NotificationActionExternalHosts}
	return handlers.OutbreakHandler{Service: publicSvc}, handlers.OutbreakAdminHandler{Service: adminSvc, MaxUploadMB: cfg.MaxUploadMB}
}
