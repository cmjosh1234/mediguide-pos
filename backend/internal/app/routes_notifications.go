package app

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/handlers"
	"mediguide/internal/middleware"
	"time"
	"gorm.io/gorm"
)

func registerNotificationsRoutes(protected *gin.RouterGroup, rateLimiter *middleware.RateLimiter, database *gorm.DB, notificationH handlers.NotificationHandler, firebaseH handlers.FirebaseHandler) {
	protected.GET("/notifications", middleware.RequirePermission("notification.read"), notificationH.List)
	protected.GET("/notifications/:id", middleware.RequirePermission("notification.read"), notificationH.Get)
	protected.POST("/notifications", middleware.RequirePermission("notification.publish"), rateLimiter.Limit(middleware.Policy("notification-publication", 20, time.Hour, 0), middleware.UserIdentity), rateLimiter.LimitWhen(middleware.Policy("notification-urgent-publication", 3, time.Hour, 0), middleware.UserIdentity, middleware.JSONFieldEquals("priority", "urgent")), notificationH.Create)
	protected.POST("/notifications/read-all", middleware.RequirePermission("notification.read"), notificationH.MarkAllRead)
	protected.POST("/notifications/:id/read", middleware.RequirePermission("notification.read"), notificationH.MarkRead)
	protected.POST("/notifications/:id/unread", middleware.RequirePermission("notification.read"), notificationH.MarkUnread)
	protected.POST("/notification-deliveries/:id/open", middleware.RequirePermission("notification.read"), rateLimiter.Limit(middleware.Policy("notification-delivery-event", 120, time.Hour, 10), middleware.UserIdentity), notificationH.RecordDeliveryOpen)
	protected.POST("/notification-deliveries/:id/click", middleware.RequirePermission("notification.read"), rateLimiter.Limit(middleware.Policy("notification-delivery-event", 120, time.Hour, 10), middleware.UserIdentity), notificationH.RecordDeliveryClick)
	protected.GET("/firebase/status", middleware.RequirePermission("firebase.status.read"), firebaseH.Status)
	protected.GET("/firebase/devices", firebaseH.ListDevices)
	protected.POST("/firebase/devices", firebaseH.RegisterDevice)
	protected.PATCH("/firebase/devices/:id", firebaseH.UpdateDevice)
	protected.DELETE("/firebase/devices/:id", firebaseH.DeleteDevice)
	protected.POST("/firebase/push/test", middleware.RequirePermission("firebase.push.test"), rateLimiter.Limit(middleware.Policy("firebase-test-push", 10, time.Hour, 0), middleware.UserIdentity), firebaseH.SendTestPush)
	protected.GET("/firebase/test-recipients", middleware.RequirePermission("firebase.push.test"), firebaseH.SearchTestRecipients)
	protected.GET("/firebase/remote-config", middleware.RequirePermission("firebase.config.manage"), firebaseH.GetRemoteConfig)
	protected.PUT("/firebase/remote-config", middleware.RequirePermission("firebase.config.manage"), rateLimiter.Limit(middleware.Policy("firebase-remote-config-write", 10, time.Hour, 0), middleware.UserIdentity), firebaseH.PutRemoteConfig)
	protected.GET("/notification-preferences", notificationH.GetPreferences)
	protected.PATCH("/notification-preferences", rateLimiter.Limit(middleware.Policy("notification-preference-write", 30, time.Hour, 0), middleware.UserIdentity), notificationH.UpdatePreferences)
	protected.GET("/notification-preferences/aggregates", middleware.RequirePermission("notification.analytics.read"), notificationH.PreferenceAggregates)

	protected.GET("/notification-templates", middleware.RequirePermission("notification.template.read"), notificationH.ListTemplates)
	protected.GET("/notification-templates/:id", middleware.RequirePermission("notification.template.read"), notificationH.GetTemplate)
	protected.POST("/notification-templates", middleware.RequirePermission("notification.template.manage"), notificationH.CreateTemplate)
	protected.PATCH("/notification-templates/:id", middleware.RequirePermission("notification.template.manage"), notificationH.UpdateTemplate)
	protected.PATCH("/notification-templates/:id/status", middleware.RequirePermission("notification.template.manage"), notificationH.UpdateTemplateStatus)
	protected.GET("/notification-templates/:id/versions", middleware.RequirePermission("notification.template.read"), notificationH.ListTemplateVersions)
	protected.POST("/notification-template-versions/:id/preview", middleware.RequirePermission("notification.template.read"), notificationH.PreviewTemplateVersion)
	protected.POST("/notification-templates/:id/clone", middleware.RequirePermission("notification.template.manage"), notificationH.CloneTemplate)
	protected.DELETE("/notification-templates/:id", middleware.RequirePermission("notification.template.manage"), notificationH.DeleteTemplate)
	protected.GET("/notification-campaigns", middleware.RequirePermission("notification.campaign.read"), notificationH.ListCampaigns)
	protected.POST("/notification-campaigns/audience-estimate", middleware.RequirePermission("notification.campaign.manage"), rateLimiter.Limit(middleware.Policy("notification-audience-estimate", 60, time.Hour, 10), middleware.UserIdentity), notificationH.EstimateAudience)
	protected.GET("/notification-campaigns/:id", middleware.RequirePermission("notification.campaign.read"), notificationH.GetCampaign)
	campaignCreateLimit := rateLimiter.Limit(middleware.Policy("notification-campaign-creation", 10, time.Hour, 0), middleware.UserIdentity)
	urgentCampaignCreateLimit := rateLimiter.LimitWhen(middleware.Policy("notification-urgent-campaign-creation", 3, time.Hour, 0), middleware.UserIdentity, middleware.JSONFieldEquals("priority", "urgent"))
	protected.POST("/notification-campaigns", middleware.RequirePermission("notification.campaign.manage"), campaignCreateLimit, urgentCampaignCreateLimit, notificationH.CreateCampaign)
	protected.POST("/guidelines/:id/notification-campaign", middleware.RequirePermission("notification.campaign.manage"), campaignCreateLimit, urgentCampaignCreateLimit, notificationH.CreateGuidelineCampaign)
	protected.POST("/outbreaks/:id/notification-campaign", middleware.RequirePermission("notification.campaign.manage"), middleware.RequirePermission("outbreak.publish"), campaignCreateLimit, urgentCampaignCreateLimit, notificationH.CreateOutbreakCampaign)
	protected.POST("/situation-reports/:id/notification-campaign", middleware.RequirePermission("notification.campaign.manage"), middleware.RequirePermission("situation_report.publish"), campaignCreateLimit, urgentCampaignCreateLimit, notificationH.CreateSituationReportCampaign)
	protected.PATCH("/notification-campaigns/:id", middleware.RequirePermission("notification.campaign.manage"), notificationH.UpdateCampaign)
	protected.POST("/notification-campaigns/:id/submit", middleware.RequirePermission("notification.campaign.manage"), rateLimiter.Limit(middleware.Policy("notification-campaign-write", 10, time.Hour, 0), middleware.UserIdentity), notificationH.TransitionCampaign)
	protected.POST("/notification-campaigns/:id/approve", middleware.RequirePermission("notification.campaign.approve"), rateLimiter.Limit(middleware.Policy("notification-campaign-approval", 20, time.Hour, 0), middleware.UserIdentity), rateLimiter.LimitWhen(middleware.Policy("notification-urgent-campaign-approval", 3, time.Hour, 0), middleware.UserIdentity, middleware.CampaignIsUrgent(database)), notificationH.TransitionCampaign)
	protected.POST("/notification-campaigns/:id/reject", middleware.RequirePermission("notification.campaign.approve"), notificationH.TransitionCampaign)
	protected.POST("/notification-campaigns/:id/schedule", middleware.RequirePermission("notification.campaign.manage"), rateLimiter.Limit(middleware.Policy("notification-campaign-scheduling", 10, time.Hour, 0), middleware.UserIdentity), rateLimiter.LimitWhen(middleware.Policy("notification-urgent-campaign-scheduling", 3, time.Hour, 0), middleware.UserIdentity, middleware.CampaignIsUrgent(database)), notificationH.TransitionCampaign)
	protected.POST("/notification-campaigns/:id/cancel", middleware.RequirePermission("notification.campaign.manage"), notificationH.TransitionCampaign)
	protected.POST("/notification-campaigns/:id/pause", middleware.RequirePermission("notification.campaign.manage"), notificationH.TransitionCampaign)
	protected.POST("/notification-campaigns/:id/resume", middleware.RequirePermission("notification.campaign.manage"), notificationH.TransitionCampaign)
	protected.DELETE("/notification-campaigns/:id", middleware.RequirePermission("notification.campaign.manage"), notificationH.DeleteCampaign)
	protected.GET("/notification-delivery-jobs", middleware.RequirePermission("notification.analytics.read"), notificationH.ListDeliveryJobs)
	protected.GET("/notification-deliveries", middleware.RequirePermission("notification.analytics.read"), notificationH.ListDeliveries)
	protected.GET("/notification-delivery-analytics/daily", middleware.RequirePermission("notification.analytics.read"), notificationH.DeliveryAnalytics)
	protected.POST("/notification-delivery-jobs/:id/requeue", middleware.RequirePermission("notification.campaign.manage"), rateLimiter.Limit(middleware.Policy("notification-delivery-requeue", 20, time.Hour, 0), middleware.UserIdentity), notificationH.RequeueDeliveryJob)
}
