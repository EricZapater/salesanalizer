package db_test

import (
	"os"
	"salesanalizer/backend/internal/db"
	"testing"
)

func TestAutoMigrations(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL / DATABASE_URL no configurada, ometent test d'integració de migracions")
	}

	database, err := db.Connect(dbURL)
	if err != nil {
		t.Fatalf("error connectant a la BD de test: %v", err)
	}

	if err := database.RunAutoMigrations(); err != nil {
		t.Fatalf("error executant RunAutoMigrations: %v", err)
	}
}
