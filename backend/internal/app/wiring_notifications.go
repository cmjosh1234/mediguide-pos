package app

import (
	"time"

	"mediguide/internal/config"
	"mediguide/internal/handlers"
	"mediguide/internal/services"

	"gorm.io/gorm"
)

func wireNotifications(cfg config.Config, database *gorm.DB) (handlers.NotificationHandler, handlers.FirebaseHandler, error) {
	firebaseSvc, err := services.NewFirebaseService(database, cfg)
	if err != nil {
		return handlers.NotificationHandler{}, handlers.FirebaseHandler{}, err
	}
	notificationSvc := services.NotificationService{DB: database, AllowedActionHosts: cfg.NotificationActionExternalHosts, DeviceStaleAfter: time.Duration(cfg.FirebaseDeviceStaleDays) * 24 * time.Hour}
	return handlers.NotificationHandler{Service: notificationSvc, Outbox: services.NotificationOutboxService{DB: database}}, handlers.FirebaseHandler{Service: firebaseSvc}, nil
}
