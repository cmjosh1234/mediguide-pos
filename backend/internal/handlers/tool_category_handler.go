package handlers

import (
	"errors"
	"net/http"

	"mediguide/internal/httpx"
	"mediguide/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListToolCategories godoc
// @Summary List tool categories
// @Tags tool-categories
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number"
// @Param per_page query int false "Items per page"
// @Param search query string false "Name, slug, or description"
// @Param status query string false "active or inactive"
// @Param sort query string false "name, sort_order, created_at or updated_at"
// @Param order query string false "asc or desc"
// @Success 200 {object} handlers.PaginatedToolCategoriesEnvelope
// @Failure 400 {object} handlers.ErrorResponse
// @Router /api/v2/tool-categories [get]
func (h CalculatorHandler) ListToolCategories(c *gin.Context) {
	page, err := parsePageQuery(c, 50, 100)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid pagination")
		return
	}
	result, err := h.Categories.List(services.ToolCategoryQuery{
		Page:   page,
		Search: c.Query("search"),
		Status: c.Query("status"),
		Sort:   c.Query("sort"),
		Order:  c.Query("order"),
	})
	if err != nil {
		h.writeToolCategoryError(c, err)
		return
	}
	httpx.OK(c, result)
}

// GetToolCategory godoc
// @Summary Get a tool category
// @Tags tool-categories
// @Produce json
// @Security BearerAuth
// @Param id path string true "Category ID" format(uuid)
// @Success 200 {object} handlers.ToolCategoryEnvelope
// @Failure 404 {object} handlers.ErrorResponse
// @Router /api/v2/tool-categories/{id} [get]
func (h CalculatorHandler) GetToolCategory(c *gin.Context) {
	id, ok := toolCategoryID(c)
	if !ok {
		return
	}
	item, err := h.Categories.Get(id)
	if err != nil {
		h.writeToolCategoryError(c, err)
		return
	}
	httpx.OK(c, item)
}

// CreateToolCategory godoc
// @Summary Create a tool category
// @Tags tool-categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body services.ToolCategoryInput true "Category"
// @Success 201 {object} handlers.ToolCategoryEnvelope
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 409 {object} handlers.ErrorResponse
// @Router /api/v2/tool-categories [post]
func (h CalculatorHandler) CreateToolCategory(c *gin.Context) {
	var in services.ToolCategoryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	item, err := h.Categories.Save(nil, in)
	if err != nil {
		h.writeToolCategoryError(c, err)
		return
	}
	httpx.Created(c, item)
}

// UpdateToolCategory godoc
// @Summary Update a tool category
// @Tags tool-categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Category ID" format(uuid)
// @Param payload body services.ToolCategoryInput true "Category changes"
// @Success 200 {object} handlers.ToolCategoryEnvelope
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 409 {object} handlers.ErrorResponse
// @Router /api/v2/tool-categories/{id} [patch]
func (h CalculatorHandler) UpdateToolCategory(c *gin.Context) {
	id, ok := toolCategoryID(c)
	if !ok {
		return
	}
	var in services.ToolCategoryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	item, err := h.Categories.Save(&id, in)
	if err != nil {
		h.writeToolCategoryError(c, err)
		return
	}
	httpx.OK(c, item)
}

// DeleteToolCategory godoc
// @Summary Delete a tool category with no tools assigned
// @Tags tool-categories
// @Security BearerAuth
// @Param id path string true "Category ID" format(uuid)
// @Success 204
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 409 {object} handlers.ErrorResponse
// @Router /api/v2/tool-categories/{id} [delete]
func (h CalculatorHandler) DeleteToolCategory(c *gin.Context) {
	id, ok := toolCategoryID(c)
	if !ok {
		return
	}
	if err := h.Categories.Delete(id); err != nil {
		h.writeToolCategoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func toolCategoryID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid tool category id")
		return uuid.Nil, false
	}
	return id, true
}

func (h CalculatorHandler) writeToolCategoryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrToolCategoryInvalid):
		httpx.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, services.ErrToolCategoryConflict), errors.Is(err, services.ErrToolCategoryInUse):
		httpx.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, gorm.ErrRecordNotFound):
		httpx.Error(c, http.StatusNotFound, "tool category not found")
	default:
		httpx.Error(c, http.StatusInternalServerError, "tool category operation failed")
	}
}
