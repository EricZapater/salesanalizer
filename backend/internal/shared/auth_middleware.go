package shared

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func RespondError(c *gin.Context, status int, errCode, message string) {
	c.JSON(status, ErrorResponse{
		Error:   errCode,
		Message: message,
	})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			RespondError(c, http.StatusUnauthorized, "unauthorized", "Capçalera d'autorització absent")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		tokenString := ""
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			tokenString = parts[1]
		} else {
			tokenString = authHeader
		}

		jwtSecret := os.Getenv("JWT_SECRET")
		adminSecret := os.Getenv("ADMIN_SECRET")
		if jwtSecret == "" {
			if adminSecret != "" {
				jwtSecret = adminSecret
			} else {
				jwtSecret = "salesanalizer-jwt-secret-key-32-chars-long"
			}
		}

		// 1. Try parsing as signed JWT
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err == nil && token.Valid {
			c.Next()
			return
		}

		// 2. Direct secret fallback (for curl / raw tokens)
		if adminSecret != "" && tokenString == adminSecret {
			c.Next()
			return
		}
		if adminSecret == "" && tokenString == "sales_analizer_secret_key" {
			c.Next()
			return
		}

		RespondError(c, http.StatusUnauthorized, "unauthorized", "Token d'autorització invàlid o expirat")
		c.Abort()
	}
}
