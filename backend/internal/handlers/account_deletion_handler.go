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
)

type AccountDeletionRequest struct {
    CurrentPassword string `json:"current_password" binding:"required,max=1024"`
    Confirm bool `json:"confirm" binding:"required"`
}

type PublicAccountDeletionRequest struct {
    Email string `json:"email" binding:"required,email,max=254"`
    CurrentPassword string `json:"current_password" binding:"required,max=1024"`
    Confirm bool `json:"confirm" binding:"required"`
}

// RequestAccountDeletion godoc
// @Summary Request deletion of the current account and associated data
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body handlers.AccountDeletionRequest true "Explicit confirmation and current password"
// @Success 202 {object} httpx.Response
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 401 {object} handlers.ErrorResponse
// @Router /api/v2/me/deletion-request [post]
func (h AuthHandler) RequestAccountDeletion(c *gin.Context) {
    var req AccountDeletionRequest
    if c.ShouldBindJSON(&req) != nil || !req.Confirm {
        httpx.Error(c, http.StatusBadRequest, "Confirm the deletion request and enter your current password.")
        return
    }
    claims := c.MustGet(middleware.ClaimsKey).(*security.Claims)
    h.requestDeletion(c, &claims.UserID, "", req.CurrentPassword)
}

// RequestPublicAccountDeletion godoc
// @Summary Request account deletion without installing or signing into the app
// @Tags auth
// @Accept json
// @Produce json
// @Param payload body handlers.PublicAccountDeletionRequest true "Account credentials and confirmation"
// @Success 202 {object} httpx.Response
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 401 {object} handlers.ErrorResponse
// @Router /api/v2/auth/deletion-request [post]
func (h AuthHandler) RequestPublicAccountDeletion(c *gin.Context) {
    var req PublicAccountDeletionRequest
    if c.ShouldBindJSON(&req) != nil || !req.Confirm {
        httpx.Error(c, http.StatusBadRequest, "Enter your email and password and confirm the deletion request.")
        return
    }
    h.requestDeletion(c, nil, req.Email, req.CurrentPassword)
}

func (h AuthHandler) requestDeletion(c *gin.Context, actor *uuid.UUID, email, password string) {
    service := h.Service
    service.DB = service.DB.WithContext(c.Request.Context())
    result, err := service.RequestAccountDeletion(actor, email, password)
    if errors.Is(err, services.ErrDeletionCredentials) {
        status := http.StatusUnauthorized
        if actor != nil { status = http.StatusBadRequest }
        httpx.Error(c, status, "Unable to verify account credentials.")
        return
    }
    if err != nil {
        httpx.Error(c, http.StatusInternalServerError, "Unable to record your request. Please try again.")
        return
    }
    c.JSON(http.StatusAccepted, httpx.Response{Success: true, Data: result})
}
