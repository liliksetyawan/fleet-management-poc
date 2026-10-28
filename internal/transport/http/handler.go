package httptransport

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

	"fleet-management-poc/internal/domain"
	"github.com/gin-gonic/gin"
)

type Reader interface {
	Latest(context.Context, string) (domain.Location, error)
	History(context.Context, string, int64, int64) ([]domain.Location, error)
}

type Handler struct{ reader Reader }

func New(reader Reader) *Handler {
	return &Handler{
		reader: reader,
	}
}

func (h *Handler) Router() *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/vehicles/:vehicle_id/location", h.latest)
	r.GET("/vehicles/:vehicle_id/history", h.history)
	return r
}

func (h *Handler) latest(c *gin.Context) {
	l, err := h.reader.Latest(c.Request.Context(), c.Param("vehicle_id"))
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "vehicle location not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve latest location"})
		return
	}
	c.JSON(http.StatusOK, l)
}

func (h *Handler) history(c *gin.Context) {
	start, e1 := strconv.ParseInt(c.Query("start"), 10, 64)
	end, e2 := strconv.ParseInt(c.Query("end"), 10, 64)
	if e1 != nil || e2 != nil || start <= 0 || end < start {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start and end must be valid Unix timestamps and end >= start"})
		return
	}
	locations, err := h.reader.History(c.Request.Context(), c.Param("vehicle_id"), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve location history"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"vehicle_id": c.Param("vehicle_id"), "start": start, "end": end, "count": len(locations), "data": locations})
}
