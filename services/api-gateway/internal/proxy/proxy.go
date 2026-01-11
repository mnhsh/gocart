package proxy

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/herodragmon/scalable-ecommerce/services/api-gateway/internal/auth"
	"github.com/herodragmon/scalable-ecommerce/services/api-gateway/internal/config"
	"github.com/herodragmon/scalable-ecommerce/services/api-gateway/internal/response"
)

func RegisterRoutes(mux *http.ServeMux, cfg *config.Config) {
	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		response.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "api-gateway"})
	})

	// ===== USER SERVICE ROUTES =====
	// Public user routes (no auth required)
	mux.HandleFunc("POST /api/users", proxyHandler(cfg.UserServiceURL, "/api/users"))
	mux.HandleFunc("POST /api/login", proxyHandler(cfg.UserServiceURL, "/api/login"))
	mux.HandleFunc("POST /api/refresh", proxyHandler(cfg.UserServiceURL, "/api/refresh"))
	mux.HandleFunc("POST /api/revoke", proxyHandler(cfg.UserServiceURL, "/api/revoke"))

	// Protected user routes
	mux.HandleFunc("GET /api/me", authMiddleware(cfg, proxyToUserService(cfg, "/internal/users/{userID}")))

	// ===== PRODUCT SERVICE ROUTES =====
	// Public product routes (no auth required)
	mux.HandleFunc("GET /api/products", proxyHandler(cfg.ProductServiceURL, "/api/products"))
	mux.HandleFunc("GET /api/products/{productID}", proxyWithPathHandler(cfg.ProductServiceURL, "/api/products/"))

	// Admin product routes (auth + admin role required)
	mux.HandleFunc("POST /admin/products", adminMiddleware(cfg, proxyHandler(cfg.ProductServiceURL, "/api/products")))
	mux.HandleFunc("PATCH /admin/products/{productID}", adminMiddleware(cfg, proxyWithPathHandler(cfg.ProductServiceURL, "/api/products/")))

	// Cart routes (all require auth, all need X-User-ID header)
	mux.HandleFunc("GET /api/cart", authMiddleware(cfg, proxyWithUserIDHandler(cfg.CartServiceURL, "/api/cart")))
	mux.HandleFunc("POST /api/cart/items", authMiddleware(cfg, proxyWithUserIDHandler(cfg.CartServiceURL, "/api/cart/items")))
	mux.HandleFunc("PATCH /api/cart/items/{itemID}", authMiddleware(cfg, proxyWithUserIDAndPathHandler(cfg.CartServiceURL, "/api/cart/items/", "itemID")))
	mux.HandleFunc("DELETE /api/cart/items/{itemID}", authMiddleware(cfg, proxyWithUserIDAndPathHandler(cfg.CartServiceURL, "/api/cart/items/", "itemID")))
	mux.HandleFunc("DELETE /api/cart", authMiddleware(cfg, proxyWithUserIDHandler(cfg.CartServiceURL, "/api/cart")))
}

// proxyHandler creates a simple proxy handler for a target service
func proxyHandler(targetURL, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		proxyRequest(w, r, targetURL+path)
	}
}

// proxyWithPathHandler proxies requests and preserves the path parameter
func proxyWithPathHandler(targetURL, basePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract path value from the request
		pathValue := r.PathValue("productID")
		proxyRequest(w, r, targetURL+basePath+pathValue)
	}
}

// proxyToUserService handles the /api/me endpoint by getting user details
func proxyToUserService(cfg *config.Config, pathPattern string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.GetUserIDFromContext(r.Context())
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		targetURL := cfg.UserServiceURL + strings.Replace(pathPattern, "{userID}", userID.String(), 1)
		proxyRequest(w, r, targetURL)
	}
}

// authMiddleware validates JWT token and adds user info to context
func authMiddleware(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			response.RespondWithError(w, http.StatusUnauthorized, "missing or invalid token", err)
			return
		}

		userID, role, err := auth.ValidateJWT(token, cfg.JWTSecret)
		if err != nil {
			response.RespondWithError(w, http.StatusUnauthorized, "invalid or expired token", err)
			return
		}

		ctx := context.WithValue(r.Context(), auth.UserIDContextKey, userID)
		ctx = context.WithValue(ctx, auth.UserRoleContextKey, role)

		next(w, r.WithContext(ctx))
	}
}

// adminMiddleware validates JWT and checks for admin role
func adminMiddleware(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			response.RespondWithError(w, http.StatusUnauthorized, "missing or invalid token", err)
			return
		}

		userID, role, err := auth.ValidateJWT(token, cfg.JWTSecret)
		if err != nil {
			response.RespondWithError(w, http.StatusUnauthorized, "invalid or expired token", err)
			return
		}

		if role != "admin" {
			response.RespondWithError(w, http.StatusForbidden, "admin access required", nil)
			return
		}

		ctx := context.WithValue(r.Context(), auth.UserIDContextKey, userID)
		ctx = context.WithValue(ctx, auth.UserRoleContextKey, role)

		next(w, r.WithContext(ctx))
	}
}

// proxyRequest forwards the request to the target URL
func proxyRequest(w http.ResponseWriter, r *http.Request, targetURL string) {
	client := &http.Client{}

	// Create new request
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		log.Printf("Error creating proxy request: %v", err)
		response.RespondWithError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Copy cookies
	for _, cookie := range r.Cookies() {
		proxyReq.AddCookie(cookie)
	}

	// Make the request
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("Error proxying request to %s: %v", targetURL, err)
		response.RespondWithError(w, http.StatusBadGateway, "Service unavailable", err)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(w, resp.Body)
}

func proxyWithUserIDHandler(targetURL, path string) http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.GetUserIDFromContext(r.Context())
    if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
      return
    }
    // Add X-User-ID header
    r.Header.Set("X-User-ID", userID.String())
		proxyRequest(w, r, targetURL+path)
	}
}

func proxyWithUserIDAndPathHandler(targetURL, basePath, paramName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
    userID, ok := auth.GetUserIDFromContext(r.Context())
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
      return
    }
    r.Header.Set("X-User-ID", userID.String())
    pathValue := r.PathValue(paramName)
    proxyRequest(w, r, targetURL+basePath+pathValue)
  }
}
