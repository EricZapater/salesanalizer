package auth

import (
	"net/http"
	"salesanalizer/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/auth/login", h.Login)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondError(c, http.StatusBadRequest, "bad_request", "Format de petició invàlid")
		return
	}

	token, err := h.service.ValidateSecret(req.Secret)
	if err != nil {
		shared.RespondError(c, http.StatusUnauthorized, "unauthorized", "Clau d'administrador incorrecta")
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:   token,
		Message: "Login correcte",
	})
}
