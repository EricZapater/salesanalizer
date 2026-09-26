package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"salesanalizer/backend/internal/auth"
	"salesanalizer/backend/internal/db"
	"salesanalizer/backend/internal/prospector"
	"salesanalizer/backend/internal/shared"
	"time"

	"github.com/gin-contrib/cors"
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
			dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPass, dbHost, dbPort, dbName, dbSSL)
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

	var prospectorService *prospector.Service
	var prospectorHandler *prospector.Handler
	if database != nil {
		prospectorRepo := prospector.NewRepository(database)
		prospectorService = prospector.NewService(prospectorRepo)
		prospectorHandler = prospector.NewHandler(prospectorService)
	}

	// Iniciar Scheduler de fons (03:00h cron nocturn)
	if prospectorService != nil {
		startNightlyCron(prospectorService)
	}

	// Configuració de Gin
	router := gin.Default()

	// CORS per al frontend
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000", "http://127.0.0.1:5173", "*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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
		if prospectorHandler != nil {
			prospectorHandler.RegisterRoutes(protectedGroup)
		}
	}

	log.Printf("Servidor escoltant al port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Error executant el servidor HTTP: %v", err)
	}
}

// startNightlyCron executa una rutina de fons que comprova l'hora per executar la cerca a les 03:00h
func startNightlyCron(service *prospector.Service) {
	go func() {
		for {
			now := time.Now()
			// Calcular el proper 03:00h
			next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			duration := time.Until(next)
			log.Printf("Scheduler nocturn programat per a d'aquí: %v (%s)", duration, next.Format(time.RFC3339))
			time.Sleep(duration)

			log.Println("Executant rastreig nocturn programat (03:00h)...")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			result, err := service.RunScrapers(ctx)
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
