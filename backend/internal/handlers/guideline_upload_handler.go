package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"mediguide/internal/httpx"
	"mediguide/internal/middleware"
	"mediguide/internal/models"
	"mediguide/internal/security"
	"mediguide/internal/services"
	"net/http"
	"strconv"
)

type SourceUploadResponse struct {
	Data services.GuidelineUploadState `json:"data"`
}
type SourceUploadListResponse struct {
	Data []models.GuidelineUpload `json:"data"`
}
type SourceUploadJobResponse struct {
	Data models.IngestionJob `json:"data"`
}
type SourceUploadCapabilities struct {
	DirectUploads bool  `json:"direct_uploads"`
	MaxSizeBytes  int64 `json:"max_size_bytes"`
	PartSize      int64 `json:"part_size"`
}
type SourceUploadCapabilitiesResponse struct {
	Data SourceUploadCapabilities `json:"data"`
}
type SourceUploadPartURL struct {
	URL string `json:"url"`
}
type SourceUploadPartResponse struct {
	Data SourceUploadPartURL `json:"data"`
}

// UploadCapabilities godoc
// @Summary Discover guideline source upload capabilities
// @Tags guideline-uploads
// @Security BearerAuth
// @Param id path string true "Version ID" format(uuid)
// @Success 200 {object} SourceUploadCapabilitiesResponse
// @Router /api/v2/guideline-versions/{id}/upload-capabilities [get]
func (h GuidelineHandler) UploadCapabilities(c *gin.Context) {
	direct := h.DirectUploads
	if direct {
		// Kinds published as uploaded (forms) accept Word files and skip extraction,
		// which only the standard upload supports.
		version, err := uuid.Parse(c.Param("id"))
		if err != nil {
			httpx.Error(c, 400, "invalid version")
			return
		}
		asUploaded, err := h.Service.VersionPublishesAsUploaded(version)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.Error(c, 404, "guideline version not found")
			return
		}
		if err != nil {
			httpx.Error(c, 500, "failed to load guideline version")
			return
		}
		direct = !asUploaded
	}
	httpx.OK(c, gin.H{"direct_uploads": direct, "max_size_bytes": h.MaxUploadMB << 20, "part_size": 8 << 20})
}

// SourceUploadJob godoc
// @Summary Read source ingestion status and timing metrics
// @Tags guideline-uploads
// @Security BearerAuth
// @Param id path string true "Version ID" format(uuid)
// @Param jobId path string true "Job ID" format(uuid)
// @Success 200 {object} SourceUploadJobResponse
// @Router /api/v2/guideline-versions/{id}/upload-jobs/{jobId} [get]
func (h GuidelineHandler) SourceUploadJob(c *gin.Context) {
	version, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, 400, "invalid version")
		return
	}
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		httpx.Error(c, 400, "invalid job")
		return
	}
	var job models.IngestionJob
	if err = h.Service.DB.WithContext(c.Request.Context()).First(&job, "id=? AND version_id=?", jobID, version).Error; err != nil {
		httpx.Error(c, 404, "Job not found")
		return
	}
	job.PayloadJSON = ""
	if job.Error != "" {
		job.Error = "Document processing failed. Open the version review workspace or ask an administrator to inspect the job logs."
	}
	c.Header("Cache-Control", "private, no-store")
	httpx.OK(c, job)
}

func (h GuidelineHandler) SourceUpload(c *gin.Context) {
	if !h.DirectUploads {
		httpx.Error(c, 503, "Direct uploads are disabled; use the standard upload.")
		return
	}
	version, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, 400, "invalid version")
		return
	}
	actor := c.MustGet(middleware.ClaimsKey).(*security.Claims).UserID
	var result any
	session := uuid.Nil
	if c.Param("session") != "" {
		session, err = uuid.Parse(c.Param("session"))
		if err != nil {
			httpx.Error(c, 400, "invalid upload session")
			return
		}
	}
	switch {
	case c.Request.Method == "GET" && session == uuid.Nil:
		rows := []models.GuidelineUpload{}
		err = h.Service.DB.WithContext(c.Request.Context()).Where("version_id=? AND user_id=?", version, actor).Order("created_at DESC").Limit(20).Find(&rows).Error
		result = rows
	case c.Request.Method == "POST" && session == uuid.Nil:
		var in services.BeginGuidelineUpload
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
		if c.ShouldBindJSON(&in) != nil {
			httpx.Error(c, 400, "invalid upload metadata")
			return
		}
		result, err = h.Service.BeginSourceUpload(c.Request.Context(), version, actor, in, h.MaxUploadMB<<20)
	case c.Request.Method == "GET":
		result, err = h.Service.UploadSession(c.Request.Context(), version, actor, session)
	case c.Request.Method == "DELETE":
		err = h.Service.AbortSourceUpload(c.Request.Context(), version, actor, session)
		result = gin.H{"aborted": err == nil}
	case c.Param("part") != "":
		number, e := strconv.Atoi(c.Param("part"))
		if e != nil {
			httpx.Error(c, 400, "invalid part number")
			return
		}
		var signed string
		signed, err = h.Service.SignSourcePart(c.Request.Context(), version, actor, session, number)
		result = gin.H{"url": signed}
	default:
		result, err = h.Service.CompleteSourceUpload(c.Request.Context(), version, actor, session)
	}
	if err != nil {
		status := 500
		message := "Unable to process upload. Retry shortly."
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			status = 404
			message = "Upload or version not found."
		case errors.Is(err, services.ErrUploadInvalid):
			status = 400
			message = err.Error()
		case errors.Is(err, services.ErrUploadConflict), errors.Is(err, services.ErrPublishedVersionImmutable):
			status = 409
			message = err.Error()
		}
		httpx.Error(c, status, message)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	httpx.OK(c, result)
}

// BeginSourceUpload godoc
// @Summary Begin a resumable source upload owned by the current editor
// @Tags guideline-uploads
// @Security BearerAuth
// @Param id path string true "Version ID" format(uuid)
// @Param body body services.BeginGuidelineUpload true "Source identity"
// @Success 200 {object} SourceUploadResponse
// @Router /api/v2/guideline-versions/{id}/uploads [post]
func (h GuidelineHandler) BeginSourceUpload(c *gin.Context) { h.SourceUpload(c) }

// ListSourceUploads godoc
// @Summary List the current editor's recent source uploads
// @Tags guideline-uploads
// @Security BearerAuth
// @Param id path string true "Version ID" format(uuid)
// @Success 200 {object} SourceUploadListResponse
// @Router /api/v2/guideline-versions/{id}/uploads [get]
func (h GuidelineHandler) ListSourceUploads(c *gin.Context) { h.SourceUpload(c) }

// GetSourceUpload godoc
// @Summary Recover source upload parts and processing job
// @Tags guideline-uploads
// @Security BearerAuth
// @Param id path string true "Version ID" format(uuid)
// @Param session path string true "Upload session ID" format(uuid)
// @Success 200 {object} SourceUploadResponse
// @Router /api/v2/guideline-versions/{id}/uploads/{session} [get]
func (h GuidelineHandler) GetSourceUpload(c *gin.Context) { h.SourceUpload(c) }

// SignSourceUploadPart godoc
// @Summary Sign one bounded source upload part
// @Tags guideline-uploads
// @Security BearerAuth
// @Param id path string true "Version ID" format(uuid)
// @Param session path string true "Upload session ID" format(uuid)
// @Param part path int true "One-based part number"
// @Success 200 {object} SourceUploadPartResponse
// @Router /api/v2/guideline-versions/{id}/uploads/{session}/parts/{part} [post]
func (h GuidelineHandler) SignSourceUploadPart(c *gin.Context) { h.SourceUpload(c) }

// CompleteSourceUpload godoc
// @Summary Verify source bytes and idempotently queue ingestion
// @Tags guideline-uploads
// @Security BearerAuth
// @Param id path string true "Version ID" format(uuid)
// @Param session path string true "Upload session ID" format(uuid)
// @Success 200 {object} SourceUploadJobResponse
// @Router /api/v2/guideline-versions/{id}/uploads/{session}/complete [post]
func (h GuidelineHandler) CompleteSourceUpload(c *gin.Context) { h.SourceUpload(c) }

// AbortSourceUpload godoc
// @Summary Cancel an unattached source upload
// @Tags guideline-uploads
// @Security BearerAuth
// @Param id path string true "Version ID" format(uuid)
// @Param session path string true "Upload session ID" format(uuid)
// @Success 200 {object} map[string]interface{}
// @Router /api/v2/guideline-versions/{id}/uploads/{session} [delete]
func (h GuidelineHandler) AbortSourceUpload(c *gin.Context) { h.SourceUpload(c) }
