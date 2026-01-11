package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/herodragmon/scalable-ecommerce/services/api-gateway/internal/config"
	"github.com/herodragmon/scalable-ecommerce/services/api-gateway/internal/proxy"
)

func main() {
	godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	cfg := &config.Config{
		UserServiceURL:    getEnvOrDefault("USER_SERVICE_URL", "http://localhost:8081"),
		ProductServiceURL: getEnvOrDefault("PRODUCT_SERVICE_URL", "http://localhost:8082"),
		CartServiceURL: os.Getenv("CART_SERVICE_URL"),
		JWTSecret:         os.Getenv("SECRET_KEY"),
	}

	mux := http.NewServeMux()
	proxy.RegisterRoutes(mux, cfg)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("API Gateway starting on port %s", port)
	log.Printf("User service: %s", cfg.UserServiceURL)
	log.Printf("Product service: %s", cfg.ProductServiceURL)
	log.Fatal(server.ListenAndServe())
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
