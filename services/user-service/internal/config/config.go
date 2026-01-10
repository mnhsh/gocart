package config

import "github.com/herodragmon/scalable-ecommerce/services/user-service/internal/database"

type Config struct {
	DB        *database.Queries
	Platform  string
	JWTSecret string
}
