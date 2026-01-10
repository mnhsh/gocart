package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/herodragmon/scalable-ecommerce/services/user-service/internal/config"
	"github.com/herodragmon/scalable-ecommerce/services/user-service/internal/database"
	"github.com/herodragmon/scalable-ecommerce/services/user-service/internal/handlers"
)

func main() {
	godotenv.Load()

	platform := os.Getenv("PLATFORM")
	secretKey := os.Getenv("SECRET_KEY")
	dbURL := os.Getenv("DB_URL")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	dbQueries := database.New(db)

	cfg := &config.Config{
		DB:        dbQueries,
		Platform:  platform,
		JWTSecret: secretKey,
	}

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux, cfg)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("User service starting on port %s", port)
	log.Fatal(server.ListenAndServe())
}
