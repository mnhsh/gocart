package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/herodragmon/scalable-ecommerce/services/cart-service/internal/config"
	"github.com/herodragmon/scalable-ecommerce/services/cart-service/internal/database"
	"github.com/herodragmon/scalable-ecommerce/services/cart-service/internal/handlers"
	"github.com/herodragmon/scalable-ecommerce/services/cart-service/internal/client"
)

func main() {
	godotenv.Load()

	platform := os.Getenv("PLATFORM")
	dbURL := os.Getenv("DB_URL")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	productServiceURL := os.Getenv("PRODUCT_SERVICE_URL")
  if productServiceURL == "" {
		log.Fatal("PRODUCT_SERVICE_URL is not set")
  }
	productClient := client.NewProductClient(productServiceURL, 10*time.Second)

	dbQueries := database.New(db)

	cfg := &config.Config{
		DB:       dbQueries,
		Platform: platform,
		ProductClient: productClient,
	}

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux, cfg)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Cart service starting on port %s", port)
	log.Fatal(server.ListenAndServe())
}
