package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"mediguide/internal/httpx"
	"mediguide/internal/security"
	"mediguide/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OutbreakAdminHandler struct {
	Service services.OutbreakAdminService
}

func outbreakActor(c *gin.Context) services.OutbreakActor {
	return services.OutbreakActor{ID: supportClaims(c).UserID, IP: c.ClientIP()}
}
func outbreakAdminID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid "+name)
		return uuid.Nil, false
	}
	return id, true
}
func outbreakAdminPage(c *gin.Context) (services.PageInput, bool) {
	p, err := parsePageQuery(c, 20, 100)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid pagination")
		return services.PageInput{}, false
	}
	return p, true
}
func outbreakLock(c *gin.Context) (int, bool) {
	v, err := strconv.Atoi(c.Query("lock_version"))
	if err != nil || v < 1 {
		httpx.Error(c, http.StatusBadRequest, "valid lock_version is required")
		return 0, false
	}
	return v, true
}
func bindTransition(c *gin.Context) (services.TransitionInput, bool) {
	var in services.TransitionInput
	if c.ShouldBindJSON(&in) != nil || in.LockVersion < 1 {
		httpx.Error(c, http.StatusBadRequest, "valid transition payload is required")
		return in, false
	}
	return in, true
}
func (h OutbreakAdminHandler) result(c *gin.Context, status int, value any, err error) {
	if err == nil {
		c.JSON(status, value)
		return
	}
	var validation *services.OutbreakValidationError
	switch {
	case errors.As(err, &validation) && len(validation.Fields) > 0:
		// The fields let the dashboard highlight what to fix.
		httpx.ErrorWithMeta(c, http.StatusBadRequest, validation.Message, gin.H{"fields": validation.Fields})
	case errors.As(err, &validation):
		// A specific, administrator-facing reason (for example which field to fix).
		httpx.Error(c, http.StatusBadRequest, validation.Message)
	case errors.Is(err, services.ErrOutbreakInvalid):
		httpx.Error(c, http.StatusBadRequest, "invalid outbreak operation")
	case errors.Is(err, services.ErrOutbreakConflict):
		httpx.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, services.ErrOutbreakImmutable):
		httpx.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, services.ErrOutbreakForbidden):
		httpx.Error(c, http.StatusForbidden, err.Error())
	case errors.Is(err, gorm.ErrRecordNotFound):
		httpx.Error(c, http.StatusNotFound, "outbreak content not found")
	default:
		httpx.Error(c, http.StatusInternalServerError, "outbreak administration operation failed")
	}
}

// ListOutbreaks godoc
// @Summary List outbreak administration records
// @Tags outbreak-administration
// @Security BearerAuth
// @Param search query string false "Title, summary, or disease search"
// @Param status query string false "Lifecycle status"
// @Param disease query string false "Disease ID"
// @Param area query string false "Geographic text"
// @Param region_id query string false "Region UUID"
// @Param visual_tone query string false "Visual tone"
// @Param effective_from query string false "Effective from (RFC3339)"
// @Param effective_to query string false "Effective to (RFC3339)"
// @Param updated_from query string false "Updated from (RFC3339)"
// @Param updated_to query string false "Updated to (RFC3339)"
// @Param sort query string false "Allowlisted sort field"
// @Param order query string false "asc or desc"
// @Success 200 {object} services.PageResult[services.OutbreakAdminDTO]
// @Router /api/v2/outbreaks [get]
func (h OutbreakAdminHandler) ListOutbreaks(c *gin.Context) {
	p, ok := outbreakAdminPage(c)
	if !ok {
		return
	}
	v, e := h.Service.ListOutbreaks(services.OutbreakAdminQuery{Page: p, Search: c.Query("search"), Status: c.Query("status"), Disease: c.Query("disease"), Area: c.Query("area"), RegionID: c.Query("region_id"), VisualTone: c.Query("visual_tone"), EffectiveFrom: c.Query("effective_from"), EffectiveTo: c.Query("effective_to"), UpdatedFrom: c.Query("updated_from"), UpdatedTo: c.Query("updated_to"), Sort: c.Query("sort"), Order: c.Query("order")})
	h.result(c, 200, v, e)
}

// CreateOutbreak godoc
// @Summary Create an outbreak draft
// @Tags outbreak-administration
// @Security BearerAuth
// @Param payload body services.OutbreakInput true "Outbreak draft"
// @Success 201 {object} services.OutbreakAdminDTO
// @Router /api/v2/outbreaks [post]
func (h OutbreakAdminHandler) CreateOutbreak(c *gin.Context) {
	var in services.OutbreakInput
	if c.ShouldBindJSON(&in) != nil {
		httpx.Error(c, 400, "invalid request body")
		return
	}
	v, e := h.Service.CreateOutbreak(outbreakActor(c), in)
	h.result(c, 201, v, e)
}

// GetOutbreak godoc
// @Summary Get an outbreak administration record
// @Tags outbreak-administration
// @Security BearerAuth
// @Success 200 {object} services.OutbreakAdminDTO
// @Router /api/v2/outbreaks/{id} [get]
func (h OutbreakAdminHandler) GetOutbreak(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	v, e := h.Service.GetOutbreak(id)
	h.result(c, 200, v, e)
}

// UpdateOutbreak godoc
// @Summary Update an unpublished outbreak draft using optimistic locking
// @Tags outbreak-administration
// @Security BearerAuth
// @Param payload body services.OutbreakInput true "Outbreak changes"
// @Success 200 {object} services.OutbreakAdminDTO
// @Router /api/v2/outbreaks/{id} [patch]
func (h OutbreakAdminHandler) UpdateOutbreak(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	var in services.OutbreakInput
	if c.ShouldBindJSON(&in) != nil {
		httpx.Error(c, 400, "invalid request body")
		return
	}
	v, e := h.Service.UpdateOutbreak(outbreakActor(c), id, in)
	h.result(c, 200, v, e)
}

// UpdateMetrics godoc
// @Summary Update an outbreak's metrics regardless of its lifecycle status
// @Tags outbreak-administration
// @Security BearerAuth
// @Param payload body services.OutbreakMetricsInput true "Outbreak metrics"
// @Success 200 {object} services.OutbreakAdminDTO
// @Router /api/v2/outbreaks/{id}/metrics [patch]
func (h OutbreakAdminHandler) UpdateMetrics(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	var in services.OutbreakMetricsInput
	if c.ShouldBindJSON(&in) != nil {
		httpx.Error(c, 400, "invalid request body")
		return
	}
	v, e := h.Service.UpdateMetrics(outbreakActor(c), id, in)
	h.result(c, 200, v, e)
}

// DeleteOutbreak godoc
// @Summary Soft-delete an outbreak
// @Description Deletes the outbreak with its updates, resources, open corrections and unsent alerts. Situation reports and content hubs linked to it are kept and unlinked; each report stays as visible as it was. Deleting anything beyond a draft or a submission in review also needs outbreak.withdraw.
// @Tags outbreak-administration
// @Security BearerAuth
// @Param lock_version query int true "Optimistic lock version"
// @Param payload body services.OutbreakDeleteInput true "Reason for deleting"
// @Success 200 {object} services.OutbreakDeleteResult
// @Router /api/v2/outbreaks/{id} [delete]
func (h OutbreakAdminHandler) DeleteOutbreak(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	lock, ok := outbreakLock(c)
	if !ok {
		return
	}
	var in services.OutbreakDeleteInput
	if c.ShouldBindJSON(&in) != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	canWithdraw := security.HasPerm(supportClaims(c), "outbreak.withdraw")
	v, e := h.Service.DeleteOutbreak(outbreakActor(c), id, lock, in, canWithdraw)
	h.result(c, 200, v, e)
}

// TransitionOutbreak godoc
// @Summary Submit, approve, publish, or withdraw an outbreak
// @Description Approving a correction applies its core metadata and source to the live outbreak, removes the correction and returns the live outbreak. Corrections are never published.
// @Tags outbreak-administration
// @Security BearerAuth
// @Param payload body services.TransitionInput true "Transition"
// @Success 200 {object} services.OutbreakAdminDTO
// @Router /api/v2/outbreaks/{id}/submit [post]
// @Router /api/v2/outbreaks/{id}/approve [post]
// @Router /api/v2/outbreaks/{id}/publish [post]
// @Router /api/v2/outbreaks/{id}/withdraw [post]
func (h OutbreakAdminHandler) TransitionOutbreak(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := outbreakAdminID(c, "id")
		if !ok {
			return
		}
		in, ok := bindTransition(c)
		if !ok {
			return
		}
		v, e := h.Service.TransitionOutbreak(outbreakActor(c), id, action, in)
		h.result(c, 200, v, e)
	}
}

// CorrectOutbreak godoc
// @Summary Create a correction draft of a published outbreak's core metadata and source
// @Description Only one correction of an outbreak can be open at a time.
// @Tags outbreak-administration
// @Security BearerAuth
// @Param payload body services.TransitionInput true "Correction reason and lock version"
// @Success 201 {object} services.OutbreakAdminDTO
// @Router /api/v2/outbreaks/{id}/correct [post]
func (h OutbreakAdminHandler) CorrectOutbreak(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	in, ok := bindTransition(c)
	if !ok {
		return
	}
	v, e := h.Service.CorrectOutbreak(outbreakActor(c), id, in)
	h.result(c, 201, v, e)
}

// ListUpdates godoc
// @Summary List outbreak updates for administration
// @Tags outbreak-administration
// @Security BearerAuth
// @Success 200 {object} services.PageResult[services.OutbreakUpdateAdminDTO]
// @Router /api/v2/outbreaks/{id}/updates [get]
func (h OutbreakAdminHandler) ListUpdates(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	p, ok := outbreakAdminPage(c)
	if !ok {
		return
	}
	v, e := h.Service.ListUpdates(id, p)
	h.result(c, 200, v, e)
}

// CreateUpdate godoc
// @Summary Create an outbreak update draft
// @Tags outbreak-administration
// @Security BearerAuth
// @Param payload body services.ChildContentInput true "Update draft"
// @Success 201 {object} services.OutbreakUpdateAdminDTO
// @Router /api/v2/outbreaks/{id}/updates [post]
func (h OutbreakAdminHandler) CreateUpdate(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	var in services.ChildContentInput
	if c.ShouldBindJSON(&in) != nil {
		httpx.Error(c, 400, "invalid request body")
		return
	}
	v, e := h.Service.CreateUpdate(outbreakActor(c), id, in)
	h.result(c, 201, v, e)
}

// GetUpdate godoc
// @Summary Get an outbreak update
// @Tags outbreak-administration
// @Security BearerAuth
// @Success 200 {object} services.OutbreakUpdateAdminDTO
// @Router /api/v2/outbreaks/{id}/updates/{updateId} [get]
func (h OutbreakAdminHandler) GetUpdate(c *gin.Context) {
	id, child, ok := twoOutbreakIDs(c, "updateId")
	if !ok {
		return
	}
	v, e := h.Service.GetUpdate(id, child)
	h.result(c, 200, v, e)
}

// UpdateUpdate godoc
// @Summary Update an outbreak update draft
// @Tags outbreak-administration
// @Security BearerAuth
// @Param payload body services.ChildContentInput true "Update changes"
// @Success 200 {object} services.OutbreakUpdateAdminDTO
// @Router /api/v2/outbreaks/{id}/updates/{updateId} [patch]
func (h OutbreakAdminHandler) UpdateUpdate(c *gin.Context) {
	id, child, ok := twoOutbreakIDs(c, "updateId")
	if !ok {
		return
	}
	var in services.ChildContentInput
	if c.ShouldBindJSON(&in) != nil {
		httpx.Error(c, 400, "invalid request body")
		return
	}
	v, e := h.Service.UpdateUpdate(outbreakActor(c), id, child, in)
	h.result(c, 200, v, e)
}

// DeleteUpdate godoc
// @Summary Delete an unpublished outbreak update
// @Tags outbreak-administration
// @Security BearerAuth
// @Success 204
// @Router /api/v2/outbreaks/{id}/updates/{updateId} [delete]
func (h OutbreakAdminHandler) DeleteUpdate(c *gin.Context) {
	id, child, ok := twoOutbreakIDs(c, "updateId")
	if !ok {
		return
	}
	lock, ok := outbreakLock(c)
	if !ok {
		return
	}
	e := h.Service.DeleteUpdate(outbreakActor(c), id, child, lock)
	if e == nil {
		c.Status(204)
		return
	}
	h.result(c, 0, nil, e)
}

// TransitionUpdate godoc
// @Summary Submit, approve, publish, or withdraw an outbreak update
// @Tags outbreak-administration
// @Security BearerAuth
// @Param payload body services.TransitionInput true "Transition"
// @Success 200 {object} services.OutbreakUpdateAdminDTO
// @Router /api/v2/outbreaks/{id}/updates/{updateId}/submit [post]
// @Router /api/v2/outbreaks/{id}/updates/{updateId}/approve [post]
// @Router /api/v2/outbreaks/{id}/updates/{updateId}/publish [post]
// @Router /api/v2/outbreaks/{id}/updates/{updateId}/withdraw [post]
func (h OutbreakAdminHandler) TransitionUpdate(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, child, ok := twoOutbreakIDs(c, "updateId")
		if !ok {
			return
		}
		in, ok := bindTransition(c)
		if !ok {
			return
		}
		v, e := h.Service.TransitionUpdate(outbreakActor(c), id, child, action, in)
		h.result(c, 200, v, e)
	}
}

// CorrectUpdate godoc
// @Summary Create a correction for a published outbreak update
// @Description With changes, the edited version is submitted for review and replaces the original once published. Without changes, an unedited correction draft is created.
// @Tags outbreak-administration
// @Security BearerAuth
// @Param payload body services.ResourceCorrectionInput true "Correction"
// @Success 201 {object} services.OutbreakUpdateAdminDTO
// @Router /api/v2/outbreaks/{id}/updates/{updateId}/correct [post]
func (h OutbreakAdminHandler) CorrectUpdate(c *gin.Context) {
	id, child, ok := twoOutbreakIDs(c, "updateId")
	if !ok {
		return
	}
	var in services.ResourceCorrectionInput
	if c.ShouldBindJSON(&in) != nil || in.LockVersion < 1 {
		httpx.Error(c, http.StatusBadRequest, "valid correction payload is required")
		return
	}
	if in.Changes == nil {
		v, e := h.Service.CorrectUpdate(outbreakActor(c), id, child, services.TransitionInput{LockVersion: in.LockVersion, Reason: in.Reason})
		h.result(c, 201, v, e)
		return
	}
	v, e := h.Service.EditPublishedUpdate(outbreakActor(c), id, child, in)
	h.result(c, 201, v, e)
}

// ListResources godoc
// @Summary List outbreak resources for administration
// @Tags outbreak-administration
// @Security BearerAuth
// @Success 200 {object} services.PageResult[services.OutbreakResourceAdminDTO]
// @Router /api/v2/outbreaks/{id}/resources [get]
func (h OutbreakAdminHandler) ListResources(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	p, ok := outbreakAdminPage(c)
	if !ok {
		return
	}
	v, e := h.Service.ListResources(id, p)
	h.result(c, 200, v, e)
}

// CreateResource godoc
// @Summary Create an outbreak resource draft
// @Tags outbreak-administration
// @Security BearerAuth
// @Param payload body services.ChildContentInput true "Resource draft"
// @Success 201 {object} services.OutbreakResourceAdminDTO
// @Router /api/v2/outbreaks/{id}/resources [post]
func (h OutbreakAdminHandler) CreateResource(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	var in services.ChildContentInput
	if c.ShouldBindJSON(&in) != nil {
		httpx.Error(c, 400, "invalid request body")
		return
	}
	v, e := h.Service.CreateResource(outbreakActor(c), id, in)
	h.result(c, 201, v, e)
}

// GetResource godoc
// @Summary Get an outbreak resource administration record
// @Tags outbreak-administration
// @Security BearerAuth
// @Success 200 {object} services.OutbreakResourceAdminDTO
// @Router /api/v2/outbreaks/{id}/resources/{resourceId} [get]
func (h OutbreakAdminHandler) GetResource(c *gin.Context) {
	id, child, ok := twoOutbreakIDs(c, "resourceId")
	if !ok {
		return
	}
	v, e := h.Service.GetResource(id, child)
	h.result(c, 200, v, e)
}

// UpdateResource godoc
// @Summary Update an unpublished outbreak resource
// @Tags outbreak-administration
// @Security BearerAuth
// @Param payload body services.ChildContentInput true "Resource changes"
// @Success 200 {object} services.OutbreakResourceAdminDTO
// @Router /api/v2/outbreaks/{id}/resources/{resourceId} [patch]
func (h OutbreakAdminHandler) UpdateResource(c *gin.Context) {
	id, child, ok := twoOutbreakIDs(c, "resourceId")
	if !ok {
		return
	}
	var in services.ChildContentInput
	if c.ShouldBindJSON(&in) != nil {
		httpx.Error(c, 400, "invalid request body")
		return
	}
	v, e := h.Service.UpdateResource(outbreakActor(c), id, child, in)
	h.result(c, 200, v, e)
}

// DeleteResource godoc
// @Summary Delete an unpublished outbreak resource
// @Tags outbreak-administration
// @Security BearerAuth
// @Success 204
// @Router /api/v2/outbreaks/{id}/resources/{resourceId} [delete]
func (h OutbreakAdminHandler) DeleteResource(c *gin.Context) {
	id, child, ok := twoOutbreakIDs(c, "resourceId")
	if !ok {
		return
	}
	lock, ok := outbreakLock(c)
	if !ok {
		return
	}
	e := h.Service.DeleteResource(outbreakActor(c), id, child, lock)
	if e == nil {
		c.Status(204)
		return
	}
	h.result(c, 0, nil, e)
}

// TransitionResource godoc
// @Summary Submit, approve, publish, or withdraw an outbreak resource
// @Tags outbreak-administration
// @Security BearerAuth
// @Param payload body services.TransitionInput true "Transition"
// @Success 200 {object} services.OutbreakResourceAdminDTO
// @Router /api/v2/outbreaks/{id}/resources/{resourceId}/submit [post]
// @Router /api/v2/outbreaks/{id}/resources/{resourceId}/approve [post]
// @Router /api/v2/outbreaks/{id}/resources/{resourceId}/publish [post]
// @Router /api/v2/outbreaks/{id}/resources/{resourceId}/withdraw [post]
func (h OutbreakAdminHandler) TransitionResource(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, child, ok := twoOutbreakIDs(c, "resourceId")
		if !ok {
			return
		}
		in, ok := bindTransition(c)
		if !ok {
			return
		}
		v, e := h.Service.TransitionResource(outbreakActor(c), id, child, action, in)
		h.result(c, 200, v, e)
	}
}

// CorrectResource godoc
// @Summary Create a correction for a published outbreak resource
// @Description With changes, the edited version is submitted for review and replaces the original once published. Without changes, an unedited correction draft is created.
// @Tags outbreak-administration
// @Security BearerAuth
// @Param payload body services.ResourceCorrectionInput true "Correction"
// @Success 201 {object} services.OutbreakResourceAdminDTO
// @Router /api/v2/outbreaks/{id}/resources/{resourceId}/correct [post]
func (h OutbreakAdminHandler) CorrectResource(c *gin.Context) {
	id, child, ok := twoOutbreakIDs(c, "resourceId")
	if !ok {
		return
	}
	var in services.ResourceCorrectionInput
	if c.ShouldBindJSON(&in) != nil || in.LockVersion < 1 {
		httpx.Error(c, http.StatusBadRequest, "valid correction payload is required")
		return
	}
	if in.Changes == nil {
		v, e := h.Service.CorrectResource(outbreakActor(c), id, child, services.TransitionInput{LockVersion: in.LockVersion, Reason: in.Reason})
		h.result(c, 201, v, e)
		return
	}
	v, e := h.Service.EditPublishedResource(outbreakActor(c), id, child, in)
	h.result(c, 201, v, e)
}

// ListReports godoc
// @Summary List situation reports for administration
// @Tags situation-report-administration
// @Security BearerAuth
// @Success 200 {object} services.PageResult[services.SituationReportAdminDTO]
// @Router /api/v2/situation-reports [get]
func (h OutbreakAdminHandler) ListReports(c *gin.Context) {
	p, ok := outbreakAdminPage(c)
	if !ok {
		return
	}
	v, e := h.Service.ListReports(services.OutbreakAdminQuery{Page: p, Search: c.Query("search"), Status: c.Query("status"), Sort: c.Query("sort"), Order: c.Query("order")}, c.Query("outbreak_id"))
	h.result(c, 200, v, e)
}

// CreateReport godoc
// @Summary Create a situation report draft
// @Tags situation-report-administration
// @Security BearerAuth
// @Param payload body services.SituationReportInput true "Report draft"
// @Success 201 {object} services.SituationReportAdminDTO
// @Router /api/v2/situation-reports [post]
func (h OutbreakAdminHandler) CreateReport(c *gin.Context) {
	var in services.SituationReportInput
	if c.ShouldBindJSON(&in) != nil {
		httpx.Error(c, 400, "invalid request body")
		return
	}
	v, e := h.Service.CreateReport(outbreakActor(c), in)
	h.result(c, 201, v, e)
}

// GetReport godoc
// @Summary Get a situation report administration record
// @Tags situation-report-administration
// @Security BearerAuth
// @Success 200 {object} services.SituationReportAdminDTO
// @Router /api/v2/situation-reports/{id} [get]
func (h OutbreakAdminHandler) GetReport(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	v, e := h.Service.GetReport(id)
	h.result(c, 200, v, e)
}

// UpdateReport godoc
// @Summary Update an unpublished situation report
// @Tags situation-report-administration
// @Security BearerAuth
// @Param payload body services.SituationReportInput true "Report changes"
// @Success 200 {object} services.SituationReportAdminDTO
// @Router /api/v2/situation-reports/{id} [patch]
func (h OutbreakAdminHandler) UpdateReport(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	var in services.SituationReportInput
	if c.ShouldBindJSON(&in) != nil {
		httpx.Error(c, 400, "invalid request body")
		return
	}
	v, e := h.Service.UpdateReport(outbreakActor(c), id, in)
	h.result(c, 200, v, e)
}

// DeleteReport godoc
// @Summary Delete an unpublished situation report
// @Tags situation-report-administration
// @Security BearerAuth
// @Success 204
// @Router /api/v2/situation-reports/{id} [delete]
func (h OutbreakAdminHandler) DeleteReport(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	lock, ok := outbreakLock(c)
	if !ok {
		return
	}
	e := h.Service.DeleteReport(outbreakActor(c), id, lock)
	if e == nil {
		c.Status(204)
		return
	}
	h.result(c, 0, nil, e)
}

// TransitionReport godoc
// @Summary Submit, approve, publish, or withdraw a situation report
// @Tags situation-report-administration
// @Security BearerAuth
// @Param payload body services.TransitionInput true "Transition"
// @Success 200 {object} services.SituationReportAdminDTO
// @Router /api/v2/situation-reports/{id}/submit [post]
// @Router /api/v2/situation-reports/{id}/approve [post]
// @Router /api/v2/situation-reports/{id}/publish [post]
// @Router /api/v2/situation-reports/{id}/withdraw [post]
func (h OutbreakAdminHandler) TransitionReport(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := outbreakAdminID(c, "id")
		if !ok {
			return
		}
		in, ok := bindTransition(c)
		if !ok {
			return
		}
		v, e := h.Service.TransitionReport(outbreakActor(c), id, action, in)
		h.result(c, 200, v, e)
	}
}

// CorrectReport godoc
// @Summary Create a correction draft for a published situation report
// @Tags situation-report-administration
// @Security BearerAuth
// @Param payload body services.TransitionInput true "Correction"
// @Success 201 {object} services.SituationReportAdminDTO
// @Router /api/v2/situation-reports/{id}/correct [post]
func (h OutbreakAdminHandler) CorrectReport(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	in, ok := bindTransition(c)
	if !ok {
		return
	}
	v, e := h.Service.CorrectReport(outbreakActor(c), id, in)
	h.result(c, 201, v, e)
}

// UpdateReportMetrics godoc
// @Summary Save an unpublished situation report's metrics
// @Tags situation-report-administration
// @Security BearerAuth
// @Param payload body services.OutbreakMetricsInput true "Report metrics"
// @Success 200 {object} services.SituationReportAdminDTO
// @Router /api/v2/situation-reports/{id}/metrics [patch]
func (h OutbreakAdminHandler) UpdateReportMetrics(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	var in services.OutbreakMetricsInput
	if c.ShouldBindJSON(&in) != nil {
		httpx.Error(c, 400, "invalid request body")
		return
	}
	v, e := h.Service.UpdateReportMetrics(outbreakActor(c), id, in)
	h.result(c, 200, v, e)
}

// ListReportAttachments godoc
// @Summary List the guideline-library documents attached to a situation report
// @Tags situation-report-administration
// @Security BearerAuth
// @Success 200 {array} services.SituationReportAttachmentDTO
// @Router /api/v2/situation-reports/{id}/attachments [get]
func (h OutbreakAdminHandler) ListReportAttachments(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	v, e := h.Service.ListReportAttachments(id)
	h.result(c, 200, v, e)
}

// CreateReportAttachment godoc
// @Summary Attach a published guideline-library document to an unpublished situation report
// @Tags situation-report-administration
// @Security BearerAuth
// @Param payload body services.SituationReportAttachmentInput true "Attachment"
// @Success 201 {object} services.SituationReportAttachmentDTO
// @Router /api/v2/situation-reports/{id}/attachments [post]
func (h OutbreakAdminHandler) CreateReportAttachment(c *gin.Context) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return
	}
	var in services.SituationReportAttachmentInput
	if c.ShouldBindJSON(&in) != nil {
		httpx.Error(c, 400, "invalid request body")
		return
	}
	v, e := h.Service.CreateReportAttachment(outbreakActor(c), id, in)
	h.result(c, 201, v, e)
}

// UpdateReportAttachment godoc
// @Summary Update an attachment of an unpublished situation report
// @Tags situation-report-administration
// @Security BearerAuth
// @Param payload body services.SituationReportAttachmentInput true "Attachment changes"
// @Success 200 {object} services.SituationReportAttachmentDTO
// @Router /api/v2/situation-reports/{id}/attachments/{attachmentId} [patch]
func (h OutbreakAdminHandler) UpdateReportAttachment(c *gin.Context) {
	id, attachment, ok := twoOutbreakIDs(c, "attachmentId")
	if !ok {
		return
	}
	var in services.SituationReportAttachmentInput
	if c.ShouldBindJSON(&in) != nil {
		httpx.Error(c, 400, "invalid request body")
		return
	}
	v, e := h.Service.UpdateReportAttachment(outbreakActor(c), id, attachment, in)
	h.result(c, 200, v, e)
}

// DeleteReportAttachment godoc
// @Summary Remove an attachment from an unpublished situation report
// @Tags situation-report-administration
// @Security BearerAuth
// @Success 204
// @Router /api/v2/situation-reports/{id}/attachments/{attachmentId} [delete]
func (h OutbreakAdminHandler) DeleteReportAttachment(c *gin.Context) {
	id, attachment, ok := twoOutbreakIDs(c, "attachmentId")
	if !ok {
		return
	}
	lock, ok := outbreakLock(c)
	if !ok {
		return
	}
	e := h.Service.DeleteReportAttachment(outbreakActor(c), id, attachment, lock)
	if e == nil {
		c.Status(204)
		return
	}
	h.result(c, 0, nil, e)
}

// ListAudit godoc
// @Summary List immutable audit history for outbreak content
// @Tags outbreak-administration
// @Security BearerAuth
// @Success 200 {object} services.PageResult[services.OutbreakAuditDTO]
// @Router /api/v2/outbreaks/{id}/audit [get]
// @Router /api/v2/situation-reports/{id}/audit [get]
func (h OutbreakAdminHandler) ListAudit(entityType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := outbreakAdminID(c, "id")
		if !ok {
			return
		}
		page, ok := outbreakAdminPage(c)
		if !ok {
			return
		}
		value, err := h.Service.ListAudit(entityType, id, page)
		h.result(c, http.StatusOK, value, err)
	}
}

// AddReviewComment godoc
// @Summary Add an auditable review comment to outbreak content
// @Tags outbreak-administration
// @Security BearerAuth
// @Param payload body services.OutbreakReviewCommentInput true "Review comment"
// @Success 204
// @Router /api/v2/outbreaks/{id}/review-comments [post]
// @Router /api/v2/situation-reports/{id}/review-comments [post]
func (h OutbreakAdminHandler) AddReviewComment(entityType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := outbreakAdminID(c, "id")
		if !ok {
			return
		}
		var in services.OutbreakReviewCommentInput
		if c.ShouldBindJSON(&in) != nil {
			httpx.Error(c, http.StatusBadRequest, "valid review comment is required")
			return
		}
		if err := h.Service.AddReviewComment(outbreakActor(c), entityType, id, in.Comment); err != nil {
			h.result(c, 0, nil, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func twoOutbreakIDs(c *gin.Context, childName string) (uuid.UUID, uuid.UUID, bool) {
	id, ok := outbreakAdminID(c, "id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	child, ok := outbreakAdminID(c, childName)
	return id, child, ok
}
