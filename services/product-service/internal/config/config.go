package config

import "github.com/herodragmon/scalable-ecommerce/services/product-service/internal/database"

type Config struct {
	DB       *database.Queries
	Platform string
}
