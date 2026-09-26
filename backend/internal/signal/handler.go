package signal

import (
	"context"
	"database/sql"
	"net/http"
	"salesanalizer/backend/internal/shared"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/offers", h.ListSignals)
	rg.GET("/offers/:id", h.GetSignalByID)
	rg.DELETE("/offers/:id", h.DiscardSignal)
	rg.POST("/offers/analyze", h.AnalyzeURL)
	rg.POST("/scrapers/run", h.RunScrapers)
	rg.GET("/system/status", h.GetSystemStatus)
}

func (h *Handler) ListSignals(c *gin.Context) {
	status := c.DefaultQuery("status", "active")
	minScore, _ := strconv.Atoi(c.Query("min_score"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	res, err := h.service.ListSignals(c.Request.Context(), status, minScore, limit, offset)
	if err != nil {
		shared.RespondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetSignalByID(c *gin.Context) {
	id := c.Param("id")
	sig, err := h.service.GetSignalByID(c.Request.Context(), id)
	if err != nil {
		shared.RespondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}
	if sig == nil {
		shared.RespondError(c, http.StatusNotFound, "not_found", "Senyal no trobat")
		return
	}

	c.JSON(http.StatusOK, sig)
}

func (h *Handler) DiscardSignal(c *gin.Context) {
	id := c.Param("id")
	err := h.service.DiscardSignal(c.Request.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			shared.RespondError(c, http.StatusNotFound, "not_found", "Senyal no trobat")
			return
		}
		shared.RespondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Senyal descartat correctament",
	})
}

type AnalyzeURLRequest struct {
	URL string `json:"url" binding:"required"`
}

func (h *Handler) AnalyzeURL(c *gin.Context) {
	var req AnalyzeURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondError(c, http.StatusBadRequest, "invalid_payload", "El camp 'url' és obligatori")
		return
	}

	sig, err := h.service.IngestAndAnalyzeURL(c.Request.Context(), req.URL)
	if err != nil {
		if err == ErrDailyLimitReached {
			shared.RespondError(c, http.StatusTooManyRequests, "rate_limit_exceeded", err.Error())
			return
		}
		if err == ErrInvalidURL {
			shared.RespondError(c, http.StatusBadRequest, "invalid_url", err.Error())
			return
		}
		shared.RespondError(c, http.StatusInternalServerError, "processing_error", err.Error())
		return
	}

	c.JSON(http.StatusCreated, sig)
}

func (h *Handler) RunScrapers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := h.service.ProcessScrapers(ctx)
	if err != nil {
		if err == ErrDailyLimitReached {
			shared.RespondError(c, http.StatusTooManyRequests, "rate_limit_exceeded", err.Error())
			return
		}
		shared.RespondError(c, http.StatusInternalServerError, "scraper_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetSystemStatus(c *gin.Context) {
	status, err := h.service.GetSystemStatus(c.Request.Context())
	if err != nil {
		shared.RespondError(c, http.StatusInternalServerError, "system_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, status)
}
