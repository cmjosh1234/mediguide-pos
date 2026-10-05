package handlers

import (
	"mediguide/internal/httpx"
	"mediguide/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PublicList always restricts the catalogue to active tools, including when a
// caller requests draft or archived records through the status filter.
func (h CalculatorHandler) PublicList(c *gin.Context) {
	h.list(c, "active")
}

func (h CalculatorHandler) PublicGet(c *gin.Context) {
	if item := h.publicCalculator(c); item != nil {
		httpx.OK(c, item)
	}
}

func (h CalculatorHandler) PublicDefinition(c *gin.Context) {
	if h.publicCalculator(c) != nil {
		// Definition only resolves the current published version.
		h.Definition(c)
	}
}

func (h CalculatorHandler) PublicContent(c *gin.Context) {
	if h.publicCalculator(c) != nil {
		h.Content(c)
	}
}

func (h CalculatorHandler) publicCalculator(c *gin.Context) *models.Calculator {
	id, ok := calculatorID(c)
	if !ok {
		return nil
	}
	item, err := h.Service.Get(id)
	if err != nil {
		h.writeError(c, err)
		return nil
	}
	if item.Status != "active" {
		h.writeError(c, gorm.ErrRecordNotFound)
		return nil
	}
	return item
}
