package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Public directory reads never depend on a session or expose inactive entries.
func (h ContentReferenceHandler) PublicListDirectory(c *gin.Context) {
	q, ok := contentReferenceQuery(c)
	if !ok {
		return
	}
	v, err := h.Service.ListDirectory(false, q)
	h.result(c, v, err, http.StatusOK)
}

func (h ContentReferenceHandler) PublicGetDirectory(c *gin.Context) {
	id, ok := contentReferenceID(c)
	if !ok {
		return
	}
	v, err := h.Service.GetDirectory(id, false)
	h.result(c, v, err, http.StatusOK)
}
