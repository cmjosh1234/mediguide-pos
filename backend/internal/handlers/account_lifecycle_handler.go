package handlers

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/httpx"
	"mediguide/internal/models"
	"net/http"
	"strconv"
	"time"
)

// AccountLifecycle returns private aggregate counts, never emails or tokens.
// @Summary Account lifecycle analytics
// @Tags analytics
// @Security BearerAuth
// @Param days query int false "Lookback in days (1–90, default 30)"
// @Success 200 {object} httpx.Response{data=AccountLifecycleResponse}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Router /analytics/accounts [get]
func (h AuthHandler) AccountLifecycle(c *gin.Context) {
	days := 30
	if raw := c.Query("days"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 90 {
			httpx.Error(c, http.StatusBadRequest, "days must be between 1 and 90")
			return
		}
		days = n
	}
	since := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	type count struct {
		Action string `json:"event"`
		Count  int64  `json:"count"`
	}
	events := []count{}
	actions := []string{"user.account_created", "user.password_reset_requested", "user.password_reset_completed", "user.email_verification_requested", "user.email_verified", "user.verified", "user.account_email_sent", "user.account_email_send_failed", "user.account_email_failed", "user.account_email_cancelled"}
	if err := h.Service.DB.WithContext(c.Request.Context()).Model(&models.AuditLog{}).Select("action, count(*) as count").Where("created_at >= ? AND action IN ?", since, actions).Group("action").Order("action").Scan(&events).Error; err != nil {
		httpx.Error(c, 500, "account lifecycle analytics unavailable")
		return
	}
	type deliveryCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	deliveries := []deliveryCount{}
	if err := h.Service.DB.WithContext(c.Request.Context()).Model(&models.AccountEmailDelivery{}).Select("status,count(*) as count").Group("status").Order("status").Scan(&deliveries).Error; err != nil {
		httpx.Error(c, 500, "account email analytics unavailable")
		return
	}
	httpx.OK(c, gin.H{"since": since, "events": events, "email_queue": deliveries, "email_delivery_semantics": "sent means SMTP accepted, not confirmed inbox delivery"})
}

// AccountLifecycleResponse contains only aggregate operational measurements.
type AccountLifecycleResponse struct {
	Since                  time.Time                    `json:"since"`
	Events                 []AccountLifecycleEventCount `json:"events"`
	EmailQueue             []AccountEmailStatusCount    `json:"email_queue"`
	EmailDeliverySemantics string                       `json:"email_delivery_semantics"`
}
type AccountLifecycleEventCount struct {
	Event string `json:"event"`
	Count int64  `json:"count"`
}
type AccountEmailStatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}
