package auth_test

import (
	"os"
	"salesanalizer/backend/internal/auth"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestValidateSecret_WithBcrypt(t *testing.T) {
	rawPassword := "ElMeuPasswordSecret123!"
	hash, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("error generant hash: %v", err)
	}

	os.Setenv("ADMIN_PASSWORD_HASH", string(hash))
	os.Setenv("JWT_SECRET", "super-jwt-secret-key-32-bytes-long!")
	defer os.Unsetenv("ADMIN_PASSWORD_HASH")
	defer os.Unsetenv("JWT_SECRET")

	service := auth.NewService()

	// Correct password
	token, err := service.ValidateSecret(rawPassword)
	if err != nil {
		t.Fatalf("esperava èxit, obtingut error: %v", err)
	}
	if len(token) < 20 {
		t.Errorf("esperava un token JWT vàlid, obtingut: %s", token)
	}

	// Incorrect password
	_, err = service.ValidateSecret("wrong_password")
	if err == nil {
		t.Fatalf("esperava error per contrasenya invàlida, obtingut nil")
	}
}

func TestValidateSecret_WithPlainSecret(t *testing.T) {
	os.Setenv("ADMIN_SECRET", "test_secret_123")
	os.Setenv("JWT_SECRET", "super-jwt-secret-key-32-bytes-long!")
	defer os.Unsetenv("ADMIN_SECRET")
	defer os.Unsetenv("JWT_SECRET")

	service := auth.NewService()

	token, err := service.ValidateSecret("test_secret_123")
	if err != nil {
		t.Fatalf("esperava èxit, obtingut error: %v", err)
	}
	if len(token) < 20 {
		t.Errorf("esperava un token JWT vàlid, obtingut: %s", token)
	}
}
