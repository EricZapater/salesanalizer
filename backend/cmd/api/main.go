package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"salesanalizer/backend/internal/auth"
	"salesanalizer/backend/internal/db"
	"salesanalizer/backend/internal/shared"
	"salesanalizer/backend/internal/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Carregar fitxer .env si existeix
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ginMode := os.Getenv("GIN_MODE")
	if ginMode != "" {
		gin.SetMode(ginMode)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbHost := os.Getenv("DB_HOST")
		if dbHost != "" {
			dbPort := os.Getenv("DB_PORT")
			if dbPort == "" {
				dbPort = "5432"
			}
			dbUser := os.Getenv("DB_USER")
			dbPass := os.Getenv("DB_PASSWORD")
			dbName := os.Getenv("DB_NAME")
			if dbName == "" {
				dbName = "salesanalizer"
			}
			dbSSL := os.Getenv("DB_SSLMODE")
			if dbSSL == "" {
				dbSSL = "disable"
			}
			// Utilitzem el format DSN clau-valor de Postgres per evitar problemes amb caràcters especials com '#' o '@'
			dbURL = fmt.Sprintf("host=%s port=%s user=%s password='%s' dbname=%s sslmode=%s", dbHost, dbPort, dbUser, dbPass, dbName, dbSSL)
		} else {
			dbURL = "postgres://salesanalizer:salesanalizer_password@localhost:5432/salesanalizer?sslmode=disable"
		}
	}

	log.Println("Iniciant SalesAnalizer API...")

	// Connexió a Base de dades
	database, err := db.Connect(dbURL)
	if err != nil {
		log.Printf("Avis de connexió BD: %v. Reintentant en background...", err)
	} else {
		autoMigrate := os.Getenv("AUTO_MIGRATE")
		if autoMigrate == "" || autoMigrate == "true" {
			if err := database.RunAutoMigrations(); err != nil {
				log.Fatalf("Error executant migracions: %v", err)
			}
		}
	}

	// Inicialització de dominis
	authService := auth.NewService()
	authHandler := auth.NewHandler(authService)

	var signalService *signal.Service
	var signalHandler *signal.Handler
	if database != nil {
		signalRepo := signal.NewRepository(database)
		signalService = signal.NewService(signalRepo)
		signalHandler = signal.NewHandler(signalService)
	}

	// Iniciar Scheduler de fons (03:00h cron nocturn amb extractors multicanal)
	if signalService != nil {
		startNightlyCron(signalService)
	}

	// Configuració de Gin
	router := gin.Default()

	// CORS per al frontend
	router.Use(shared.CORSMiddleware())

	// Rutes públiques
	apiGroup := router.Group("/api")
	{
		apiGroup.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":    "healthy",
				"timestamp": time.Now().Format(time.RFC3339),
			})
		})

		authHandler.RegisterRoutes(apiGroup)
	}

	// Rutes protegides per AuthMiddleware
	protectedGroup := apiGroup.Group("")
	protectedGroup.Use(shared.AuthMiddleware())
	{
		if signalHandler != nil {
			signalHandler.RegisterRoutes(protectedGroup)
		}
	}

	log.Printf("Servidor escoltant al port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Error executant el servidor HTTP: %v", err)
	}
}

// startNightlyCron executa una rutina de fons que comprova l'hora per executar la cerca a les 03:00h
func startNightlyCron(service *signal.Service) {
	go func() {
		for {
			now := time.Now()
			// Calcular el proper 03:00h
			next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			duration := time.Until(next)
			log.Printf("Scheduler nocturn multicanal programat per a d'aquí: %v (%s)", duration, next.Format(time.RFC3339))
			time.Sleep(duration)

			log.Println("Executant rastreig nocturn multicanal (03:00h)...")
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			result, err := service.ProcessScrapers(ctx)
			if err != nil {
				log.Printf("Error al cron nocturn: %v", err)
			} else {
				log.Printf("Cron nocturn finalitzat amb èxit: %+v", result)
			}
			cancel()
			time.Sleep(1 * time.Minute)
		}
	}()
}
