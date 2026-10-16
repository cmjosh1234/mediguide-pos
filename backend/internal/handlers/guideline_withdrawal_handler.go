package handlers

import (
	"errors"
	"net/http"

	"mediguide/internal/httpx"
	"mediguide/internal/middleware"
	"mediguide/internal/security"
	"mediguide/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Withdraw godoc
// @Summary Withdraw a published guideline, retaining files and review history
// @Tags guidelines
// @Accept json
// @Security BearerAuth
// @Param id path string true "Document ID" format(uuid)
// @Param payload body services.WithdrawGuidelineInput true "Withdrawal reason and expected current version"
// @Success 204
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 403 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 409 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v2/guidelines/{id}/withdraw [post]
func (h GuidelineHandler) Withdraw(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in services.WithdrawGuidelineInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "reason and current_version_id are required")
		return
	}
	claims := c.MustGet(middleware.ClaimsKey).(*security.Claims)
	if err := h.Service.WithdrawDocument(id, claims.UserID, in, c.ClientIP()); err != nil {
		switch {
		case errors.Is(err, services.ErrGuidelineWithdrawalReason):
			httpx.Error(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, services.ErrGuidelinePublicationConflict):
			httpx.Error(c, http.StatusConflict, err.Error())
		case errors.Is(err, gorm.ErrRecordNotFound):
			httpx.Error(c, http.StatusNotFound, "not found")
		default:
			httpx.Error(c, http.StatusInternalServerError, "failed to withdraw guideline")
		}
		return
	}
	c.Status(http.StatusNoContent)
}
