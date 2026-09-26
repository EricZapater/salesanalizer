package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidSecret = errors.New("contrasenya o clau d'administrador invàlida")

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) ValidateSecret(inputPassword string) (string, error) {
	passwordHash := os.Getenv("ADMIN_PASSWORD_HASH")
	adminSecret := os.Getenv("ADMIN_SECRET")

	isValid := false

	if passwordHash != "" {
		err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(inputPassword))
		if err == nil {
			isValid = true
		}
	} else if adminSecret != "" {
		if inputPassword == adminSecret {
			isValid = true
		}
	} else {
		// Default dev fallback
		if inputPassword == "sales_analizer_secret_key" {
			isValid = true
		}
	}

	if !isValid {
		return "", ErrInvalidSecret
	}

	// Generate JWT signed token
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		if adminSecret != "" {
			jwtSecret = adminSecret
		} else {
			jwtSecret = "salesanalizer-jwt-secret-key-32-chars-long"
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "admin",
		"role": "admin",
		"exp":  time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 dies
		"iat":  time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
