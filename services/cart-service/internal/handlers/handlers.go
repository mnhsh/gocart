package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/herodragmon/scalable-ecommerce/services/cart-service/internal/config"
	"github.com/herodragmon/scalable-ecommerce/services/cart-service/internal/database"
	"github.com/herodragmon/scalable-ecommerce/services/cart-service/internal/response"
)

func RegisterRoutes(mux *http.ServeMux, cfg *config.Config) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		response.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "cart-service"})
	})

	mux.HandleFunc("GET /api/cart", func(w http.ResponseWriter, r *http.Request) {
		handlerCartGet(cfg, w, r)
	})

	mux.HandleFunc("POST /api/cart/items", func(w http.ResponseWriter, r *http.Request) {
		handlerCartAddItem(cfg, w, r)
	})

	mux.HandleFunc("PATCH /api/cart/items/{itemID}", func(w http.ResponseWriter, r *http.Request) {
		handlerCartUpdateItem(cfg, w, r)
	})

	mux.HandleFunc("DELETE /api/cart/items/{itemID}", func(w http.ResponseWriter, r *http.Request) {
		handlerCartDeleteItem(cfg, w, r)
	})

	mux.HandleFunc("DELETE /api/cart", func(w http.ResponseWriter, r *http.Request) {
		handlerCartClear(cfg, w, r)
	})

	mux.HandleFunc("GET /internal/cart/{userID}", func(w http.ResponseWriter, r *http.Request) {
		handlerInternalCartGet(cfg, w, r)
	})

	mux.HandleFunc("DELETE /internal/cart/{userID}", func(w http.ResponseWriter, r *http.Request) {
		handlerInternalCartClear(cfg, w, r)
	})
}

type CartItemResponse struct {
	ID         uuid.UUID `json:"id"`
	ProductID  uuid.UUID `json:"product_id"`
	Quantity   int32     `json:"quantity"`
	PriceCents int32     `json:"price_cents"`
}

type CartResponse struct {
	ID         uuid.UUID          `json:"id,omitempty"`
	Items      []CartItemResponse `json:"items"`
	TotalCents int64              `json:"total_cents"`
}

func handlerCartGet(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		response.RespondWithError(w, http.StatusUnauthorized, "missing user ID", nil)
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid user ID", err)
		return
	}

	cart, err := cfg.DB.GetCartByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.RespondWithJSON(w, http.StatusOK, CartResponse{
				Items:      []CartItemResponse{},
				TotalCents: 0,
			})
			return
		}
		response.RespondWithError(w, http.StatusInternalServerError, "couldn't get cart", err)
		return
	}

	items, err := cfg.DB.GetCartItems(r.Context(), cart.ID)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "couldn't get cart items", err)
		return
	}

	var totalCents int64
	itemResponses := make([]CartItemResponse, len(items))
	for i, item := range items {
		totalCents += int64(item.PriceCents) * int64(item.Quantity)
		itemResponses[i] = CartItemResponse{
			ID:         item.ID,
			ProductID:  item.ProductID,
			Quantity:   item.Quantity,
			PriceCents: item.PriceCents,
		}
	}

	response.RespondWithJSON(w, http.StatusOK, CartResponse{
		ID:         cart.ID,
		Items:      itemResponses,
		TotalCents: totalCents,
	})
}

func handlerCartAddItem(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	type addItemRequest struct {
		ProductID uuid.UUID `json:"product_id"`
		Quantity  int32     `json:"quantity"`
	}

	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		response.RespondWithError(w, http.StatusUnauthorized, "missing user ID", nil)
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid user ID", err)
		return
	}

	var item addItemRequest
	err = json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "couldn't decode json", err)
		return
	}

	if item.ProductID == uuid.Nil || item.Quantity <= 0 {
		response.RespondWithError(w, http.StatusBadRequest, "invalid product_id or quantity", err)
		return
	}

	product, exists, err := cfg.ProductClient.GetProduct(r.Context(), item.ProductID)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "error checking product", err)
		return
	}
	if !exists {
		response.RespondWithError(w, http.StatusNotFound, "product not found", nil)
		return
	}

	cart, err := cfg.DB.GetCartByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			cart, err = cfg.DB.CreateCart(r.Context(), userID)
			if err != nil {
				response.RespondWithError(w, http.StatusInternalServerError, "couldn't create cart", err)
				return
			}
		} else {
			response.RespondWithError(w, http.StatusInternalServerError, "couldn't get cart", err)
			return
		}
	}

	existingItem, err := cfg.DB.GetCartItemByProductID(r.Context(), database.GetCartItemByProductIDParams{
		CartID:    cart.ID,
		ProductID: item.ProductID,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, http.StatusInternalServerError, "couldn't check cart", err)
		return
	}
	var cartItem database.CartItem
	if err == nil {
		cartItem, err = cfg.DB.UpdateCartItemQuantity(r.Context(), database.UpdateCartItemQuantityParams{
			ID:       existingItem.ID,
			Quantity: existingItem.Quantity + item.Quantity,
		})
	} else {
		cartItem, err = cfg.DB.AddCartItem(r.Context(), database.AddCartItemParams{
			CartID:     cart.ID,
			ProductID:  item.ProductID,
			Quantity:   item.Quantity,
			PriceCents: product.PriceCents,
		})
	}
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "couldn't add item to cart", err)
		return
	}
	response.RespondWithJSON(w, http.StatusCreated, CartItemResponse{
		ID:         cartItem.ID,
		ProductID:  cartItem.ProductID,
		Quantity:   cartItem.Quantity,
		PriceCents: cartItem.PriceCents,
	})
}

func handlerCartUpdateItem(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	type updateItemRequest struct {
		Quantity int32 `json:"quantity"`
	}

	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		response.RespondWithError(w, http.StatusUnauthorized, "missing user ID", nil)
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid user ID", err)
		return
	}

	itemIDStr := r.PathValue("itemID")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid item ID", err)
		return
	}

	var req updateItemRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "could not decode json", err)
		return
	}

	if req.Quantity <= 0 {
		response.RespondWithError(w, http.StatusBadRequest, "invalid quantity", nil)
		return
	}

	cartItem, err := cfg.DB.GetCartItemByID(r.Context(), itemID)
	if err != nil {
		response.RespondWithError(w, http.StatusNotFound, "cart item not found", err)
		return
	}

	cart, err := cfg.DB.GetCartByUserID(r.Context(), userID)
	if err != nil {
		response.RespondWithError(w, http.StatusNotFound, "cart not found", err)
		return
	}
	if cartItem.CartID != cart.ID {
		response.RespondWithError(w, http.StatusForbidden, "item does not belong to your cart", nil)
		return
	}

	updatedItem, err := cfg.DB.UpdateCartItemQuantity(r.Context(), database.UpdateCartItemQuantityParams{
		ID:       itemID,
		Quantity: req.Quantity,
	})
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not update item", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, CartItemResponse{
		ID:         updatedItem.ID,
		ProductID:  updatedItem.ProductID,
		Quantity:   updatedItem.Quantity,
		PriceCents: updatedItem.PriceCents,
	})
}

func handlerCartDeleteItem(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		response.RespondWithError(w, http.StatusUnauthorized, "missing user ID", nil)
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid user ID", err)
		return
	}
	itemIDStr := r.PathValue("itemID")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid item ID", err)
		return
	}

	cartItem, err := cfg.DB.GetCartItemByID(r.Context(), itemID)
	if err != nil {
		response.RespondWithError(w, http.StatusNotFound, "cart item not found", err)
		return
	}

	cart, err := cfg.DB.GetCartByUserID(r.Context(), userID)
	if err != nil {
		response.RespondWithError(w, http.StatusNotFound, "cart not found", err)
		return
	}
	if cartItem.CartID != cart.ID {
		response.RespondWithError(w, http.StatusForbidden, "item does not belong to your cart", nil)
		return
	}
	err = cfg.DB.DeleteCartItem(r.Context(), itemID)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "couldn't delete item", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handlerCartClear(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		response.RespondWithError(w, http.StatusUnauthorized, "missing user ID", nil)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid user ID", err)
		return
	}

	cart, err := cfg.DB.GetCartByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		response.RespondWithError(w, http.StatusInternalServerError, "couldn't get cart", err)
		return
	}

	err = cfg.DB.ClearCart(r.Context(), cart.ID)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "couldn't clear cart", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handlerInternalCartGet(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid user ID", err)
		return
	}

	cart, err := cfg.DB.GetCartByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.RespondWithJSON(w, http.StatusOK, CartResponse{
				Items:      []CartItemResponse{},
				TotalCents: 0,
			})
			return
		}
		response.RespondWithError(w, http.StatusInternalServerError, "couldn't get cart", err)
		return
	}

	items, err := cfg.DB.GetCartItems(r.Context(), cart.ID)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "couldn't get cart items", err)
		return
	}

	var totalCents int64
	itemResponses := make([]CartItemResponse, len(items))
	for i, item := range items {
		totalCents += int64(item.PriceCents) * int64(item.Quantity)
		itemResponses[i] = CartItemResponse{
			ID:         item.ID,
			ProductID:  item.ProductID,
			Quantity:   item.Quantity,
			PriceCents: item.PriceCents,
		}
	}

	response.RespondWithJSON(w, http.StatusOK, CartResponse{
		ID:         cart.ID,
		Items:      itemResponses,
		TotalCents: totalCents,
	})
}

func handlerInternalCartClear(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid user ID", err)
		return
	}
	cart, err := cfg.DB.GetCartByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		response.RespondWithError(w, http.StatusInternalServerError, "couldn't get cart", err)
		return
	}
	err = cfg.DB.ClearCart(r.Context(), cart.ID)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "couldn't clear cart", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
