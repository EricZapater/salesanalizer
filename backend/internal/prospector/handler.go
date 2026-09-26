package prospector

import (
	"database/sql"
	"net/http"
	"salesanalizer/backend/internal/shared"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/offers", h.ListOffers)
	rg.GET("/offers/:id", h.GetOfferByID)
	rg.DELETE("/offers/:id", h.DiscardOffer)
	rg.POST("/offers/analyze", h.AnalyzeURL)
	rg.POST("/scrapers/run", h.RunScrapers)
	rg.GET("/system/status", h.GetSystemStatus)
}

func (h *Handler) ListOffers(c *gin.Context) {
	status := c.DefaultQuery("status", "active")
	minScore, _ := strconv.Atoi(c.Query("min_score"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	res, err := h.service.ListOffers(c.Request.Context(), status, minScore, limit, offset)
	if err != nil {
		shared.RespondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetOfferByID(c *gin.Context) {
	id := c.Param("id")
	offer, err := h.service.GetOfferByID(c.Request.Context(), id)
	if err != nil {
		shared.RespondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}
	if offer == nil {
		shared.RespondError(c, http.StatusNotFound, "not_found", "Oferta no trobada")
		return
	}

	c.JSON(http.StatusOK, offer)
}

func (h *Handler) DiscardOffer(c *gin.Context) {
	id := c.Param("id")
	err := h.service.DiscardOffer(c.Request.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			shared.RespondError(c, http.StatusNotFound, "not_found", "Oferta no trobada")
			return
		}
		shared.RespondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Oferta descartada correctament",
	})
}

func (h *Handler) AnalyzeURL(c *gin.Context) {
	var req AnalyzeURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondError(c, http.StatusBadRequest, "invalid_payload", "El camp 'url' és obligatori")
		return
	}

	offer, err := h.service.IngestAndAnalyzeURL(c.Request.Context(), req.URL)
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

	c.JSON(http.StatusCreated, offer)
}

func (h *Handler) RunScrapers(c *gin.Context) {
	result, err := h.service.RunScrapers(c.Request.Context())
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
