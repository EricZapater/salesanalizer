package signal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"salesanalizer/backend/internal/shared"
	"strconv"
	"strings"
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
	rg.PATCH("/offers/:id/status", h.UpdateStatus)
	rg.PUT("/offers/:id/status", h.UpdateStatus)
	rg.DELETE("/offers/:id", h.DiscardSignal)
	rg.POST("/offers/analyze", h.AnalyzeURL)
	rg.POST("/scrapers/run", h.RunScrapers)
	rg.GET("/scrapers/stream", h.StreamScrapers)
	rg.GET("/system/status", h.GetSystemStatus)
	rg.GET("/signals/settings", h.GetScraperSettings)
	rg.PUT("/signals/settings", h.UpdateScraperSettings)
	rg.GET("/settings", h.GetScraperSettings)
	rg.PUT("/settings", h.UpdateScraperSettings)
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

func (h *Handler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondError(c, http.StatusBadRequest, "invalid_payload", "El camp 'status' és obligatori")
		return
	}

	validStatuses := map[string]bool{
		"pendent":    true,
		"pending":    true,
		"analyzed":   true,
		"enviada":    true,
		"acceptada":  true,
		"rebutjada":  true,
		"descartada": true,
		"discarded":  true,
	}
	if !validStatuses[strings.ToLower(req.Status)] {
		shared.RespondError(c, http.StatusBadRequest, "invalid_status", "Estat no vàlid. Valors admesos: pendent, enviada, acceptada, rebutjada, descartada")
		return
	}

	if err := h.service.UpdateSignalStatus(c.Request.Context(), id, req.Status); err != nil {
		if err == sql.ErrNoRows {
			shared.RespondError(c, http.StatusNotFound, "not_found", "Senyal no trobat")
			return
		}
		shared.RespondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Estat actualitzat correctament",
		"status":  req.Status,
	})
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

func (h *Handler) StreamScrapers(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		shared.RespondError(c, http.StatusInternalServerError, "streaming_unsupported", "Streaming no suportat")
		return
	}

	sendEvent := func(event ProgressEvent) {
		bytes, err := json.Marshal(event)
		if err == nil {
			_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", string(bytes))
			flusher.Flush()
		}
	}

	_, err := h.service.ProcessScrapersWithProgress(ctx, sendEvent)
	if err != nil {
		sendEvent(ProgressEvent{
			Type:     "error",
			Message:  err.Error(),
			Progress: 100,
		})
	}
}

func (h *Handler) GetSystemStatus(c *gin.Context) {
	status, err := h.service.GetSystemStatus(c.Request.Context())
	if err != nil {
		shared.RespondError(c, http.StatusInternalServerError, "system_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, status)
}

func (h *Handler) GetScraperSettings(c *gin.Context) {
	deepFetch, err := h.service.GetScraperSettings(c.Request.Context())
	if err != nil {
		shared.RespondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, ScraperSettingsResponse{
		DeepFetchEnabled: deepFetch,
	})
}

func (h *Handler) UpdateScraperSettings(c *gin.Context) {
	var req UpdateScraperSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondError(c, http.StatusBadRequest, "invalid_payload", "Format invàlid per a la configuració")
		return
	}

	if err := h.service.UpdateScraperSettings(c.Request.Context(), req.DeepFetchEnabled); err != nil {
		shared.RespondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"message":            "Configuració d'extracció actualitzada correctament",
		"deep_fetch_enabled": req.DeepFetchEnabled,
	})
}

