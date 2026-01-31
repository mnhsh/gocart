package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"context"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/herodragmon/scalable-ecommerce/services/product-service/internal/config"
	"github.com/herodragmon/scalable-ecommerce/services/product-service/internal/database"
	"github.com/herodragmon/scalable-ecommerce/services/product-service/internal/handlers"
	"github.com/herodragmon/scalable-ecommerce/services/product-service/internal/rabbitmq"
)

func main() {
	godotenv.Load()

	platform := os.Getenv("PLATFORM")
	dbURL := os.Getenv("DB_URL")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
    log.Fatal("RABBITMQ_URL is not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	dbQueries := database.New(db)

	consumer, err := rabbitmq.NewConsumer(rabbitmqURL, db, dbQueries)
	if err != nil {
		log.Fatalf("failed to consume: %v",err)
	}
	defer consumer.Close()

	go consumer.Start(context.Background())


	cfg := &config.Config{
		DB:       dbQueries,
		Platform: platform,
	}

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux, cfg)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Product service starting on port %s", port)
	log.Fatal(server.ListenAndServe())
}
