package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/herodragmon/scalable-ecommerce/services/product-service/internal/config"
	"github.com/herodragmon/scalable-ecommerce/services/product-service/internal/database"
	"github.com/herodragmon/scalable-ecommerce/services/product-service/internal/response"
	"github.com/herodragmon/scalable-ecommerce/services/product-service/internal/validation"
)

func RegisterRoutes(mux *http.ServeMux, cfg *config.Config) {
	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		response.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "product-service"})
	})

	// Public routes
	mux.HandleFunc("GET /api/products", func(w http.ResponseWriter, r *http.Request) {
		handlerProductsGet(cfg, w, r)
	})

	mux.HandleFunc("GET /api/products/{productID}", func(w http.ResponseWriter, r *http.Request) {
		handlerProductsGetByID(cfg, w, r)
	})

	// Admin routes (authorization handled by API Gateway)
	mux.HandleFunc("POST /api/products", func(w http.ResponseWriter, r *http.Request) {
		handlerProductsCreate(cfg, w, r)
	})

	mux.HandleFunc("PATCH /api/products/{productID}", func(w http.ResponseWriter, r *http.Request) {
		handlerProductsUpdate(cfg, w, r)
	})

	mux.HandleFunc("DELETE /api/products/{productID}", func(w http.ResponseWriter, r *http.Request) {
		handlerProductsDelete(cfg, w, r)
	})
}

func handlerProductsGet(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	products, err := cfg.DB.GetProducts(r.Context())
	if err != nil {
		response.RespondWithError(
			w,
			http.StatusInternalServerError,
			"Couldn't get products",
			err,
		)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, products)
}

func handlerProductsGetByID(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	productIDStr := r.PathValue("productID")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	product, err := cfg.DB.GetProductByID(r.Context(), productID)
	if err != nil {
		response.RespondWithError(w, http.StatusNotFound, "Couldn't get product", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, product)
}

func handlerProductsCreate(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	type createProductReq struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		PriceCents  int    `json:"price_cents"`
		Stock       int    `json:"stock"`
		IsActive    *bool  `json:"is_active"`
	}

	var body createProductReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid body", err)
		return
	}

	if !validation.Required(body.Name) {
		response.RespondWithError(w, http.StatusBadRequest, "name is required", nil)
		return
	}

	if !validation.GreaterThan(body.PriceCents, 0) {
		response.RespondWithError(w, http.StatusBadRequest, "price_cents must be > 0", nil)
		return
	}

	if !validation.MinInt(body.Stock, 0) {
		response.RespondWithError(w, http.StatusBadRequest, "stock cannot be negative", nil)
		return
	}

	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}

	product, err := cfg.DB.CreateProduct(
		r.Context(),
		database.CreateProductParams{
			Name: body.Name,
			Description: sql.NullString{
				String: body.Description,
				Valid:  body.Description != "",
			},
			PriceCents: int32(body.PriceCents),
			Stock:      int32(body.Stock),
			IsActive:   isActive,
		},
	)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "couldn't create product", err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, product)
}

func handlerProductsUpdate(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	productIDStr := r.PathValue("productID")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid product ID", err)
		return
	}

	type updateReq struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		PriceCents  int    `json:"price_cents"`
		Stock       int    `json:"stock"`
		IsActive    bool   `json:"is_active"`
	}

	var body updateReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid body", err)
		return
	}

	product, err := cfg.DB.UpdateProduct(
		r.Context(),
		database.UpdateProductParams{
			ID:   productID,
			Name: body.Name,
			Description: sql.NullString{
				String: body.Description,
				Valid:  body.Description != "",
			},
			PriceCents: int32(body.PriceCents),
			Stock:      int32(body.Stock),
			IsActive:   body.IsActive,
		},
	)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "couldn't update product", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, product)
}

func handlerProductsDelete(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	productIDStr := r.PathValue("productID")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid product ID", err)
		return
	}

	err = cfg.DB.DeleteProduct(r.Context(), productID)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "couldn't delete product", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
