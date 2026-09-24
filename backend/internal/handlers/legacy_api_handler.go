package handlers

import (
	"net/http"
	"strings"
	"time"

	"mediguide/internal/config"
	"mediguide/internal/security"
	"mediguide/internal/services"

	"github.com/gin-gonic/gin"
)

type LegacyAPIHandler struct {
	Service services.LegacyAPIService
	Cfg     config.Config
}

// HealthFacilitiesTree godoc
// @Summary Get health facilities tree
// @Description Legacy v1 endpoint that groups facilities by region, district, then facility level.
// @Tags legacy-v1
// @Produce json
// @Param level query int false "Tree level" minimum(0) maximum(2)
// @Param filters query string false "JSON encoded filters"
// @Param context query string false "Optional JSON context"
// @Success 200 {object} handlers.LegacyTreeResult
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/health-facilities/tree [get]
func (h LegacyAPIHandler) HealthFacilitiesTree(c *gin.Context) {
	level, filters := services.ParseTreeRequest(c.Query("level"), c.Query("filters"), 2, []string{"region", "district", "facility_level"})
	result, err := h.Service.HealthFacilitiesTree(level, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to load health facilities tree"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// MinistryDirectoryTree godoc
// @Summary Get ministry directory tree
// @Description Legacy v1 endpoint that groups ministry directory contacts by region, district, then ministry.
// @Tags legacy-v1
// @Produce json
// @Param level query int false "Tree level" minimum(0) maximum(2)
// @Param filters query string false "JSON encoded filters"
// @Param context query string false "Optional JSON context"
// @Success 200 {object} handlers.LegacyTreeResult
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/ministry-directory/tree [get]
func (h LegacyAPIHandler) MinistryDirectoryTree(c *gin.Context) {
	level, filters := services.ParseTreeRequest(c.Query("level"), c.Query("filters"), 2, []string{"region", "district", "ministry", "department", "status"})
	result, err := h.Service.MinistryDirectoryTree(level, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to load ministry directory tree"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// Overview godoc
// @Summary Get legacy overview metrics
// @Description Dashboard-oriented legacy v1 overview endpoint with metrics, pipeline, engagement, support and coverage totals.
// @Tags legacy-v1
// @Produce json
// @Security BearerAuth
// @Success 200 {object} handlers.LegacyOverviewResult
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/overview [get]
func (h LegacyAPIHandler) Overview(c *gin.Context) {
	result, err := h.Service.Overview()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to fetch overview data",
			"cached_at": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}
	c.JSON(http.StatusOK, result)
}

// Stats godoc
// @Summary Get legacy mobile stats
// @Description Legacy v1 stats endpoint used by the mobile home screen. Authentication is optional, but user-specific message counts are only returned when a valid bearer token is provided.
// @Tags legacy-v1
// @Produce json
// @Success 200 {object} handlers.LegacyStatsResult
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/stats [get]
func (h LegacyAPIHandler) Stats(c *gin.Context) {
	userID := ""
	if claims := h.optionalClaims(c.GetHeader("Authorization")); claims != nil {
		userID = claims.UserID.String()
	}
	result, err := h.Service.Stats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to fetch statistics",
			"cached_at": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h LegacyAPIHandler) optionalClaims(header string) *security.Claims {
	if header == "" || !strings.HasPrefix(header, "Bearer ") {
		return nil
	}
	claims, err := security.ParseJWT(h.Cfg.JWTSecret, strings.TrimPrefix(header, "Bearer "))
	if err != nil {
		return nil
	}
	return claims
}
