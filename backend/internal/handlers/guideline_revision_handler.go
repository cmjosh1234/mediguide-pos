package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"mediguide/internal/httpx"
	"mediguide/internal/middleware"
	"mediguide/internal/security"
	"mediguide/internal/services"
	"net/http"
)

type GuidelineRevisionEnvelope struct {
	Success bool                             `json:"success"`
	Data    services.GuidelineRevisionResult `json:"data"`
}

// ReviseWithdrawnDocument godoc
// @Summary Create or continue an unapproved draft from a withdrawn publication
// @Tags guidelines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Document ID" format(uuid)
// @Param payload body services.ReviseGuidelineInput true "Withdrawn source and new version"
// @Success 200 {object} handlers.GuidelineRevisionEnvelope
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 403 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 409 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v2/guidelines/{id}/revise [post]
func (h GuidelineHandler) ReviseWithdrawnDocument(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, 400, "invalid id")
		return
	}
	var input services.ReviseGuidelineInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "source_version_id and version are required")
		return
	}
	actor := c.MustGet(middleware.ClaimsKey).(*security.Claims).UserID
	result, err := h.Service.ReviseWithdrawnDocument(c.Request.Context(), id, actor, input, c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGuidelineRevisionInput):
			httpx.Error(c, 400, err.Error())
		case errors.Is(err, services.ErrGuidelinePublicationConflict), errors.Is(err, services.ErrGuidelineVersionExists):
			httpx.Error(c, 409, err.Error())
		case errors.Is(err, gorm.ErrRecordNotFound):
			httpx.Error(c, 404, "not found")
		default:
			httpx.Error(c, http.StatusInternalServerError, "Could not create the revision draft. Check that its retained source is available and try again.")
		}
		return
	}
	httpx.OK(c, result)
}
