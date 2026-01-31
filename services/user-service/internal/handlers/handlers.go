package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/herodragmon/scalable-ecommerce/services/user-service/internal/auth"
	"github.com/herodragmon/scalable-ecommerce/services/user-service/internal/config"
	"github.com/herodragmon/scalable-ecommerce/services/user-service/internal/database"
	"github.com/herodragmon/scalable-ecommerce/services/user-service/internal/response"
)

func RegisterRoutes(mux *http.ServeMux, cfg *config.Config) {
	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		response.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "user-service"})
	})

	// User routes
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		handlerUsersCreate(cfg, w, r)
	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		handlerLogin(cfg, w, r)
	})

	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		handlerRefresh(cfg, w, r)
	})

	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		handlerRevoke(cfg, w, r)
	})

	// Internal routes (called by API Gateway)
	mux.HandleFunc("GET /internal/users/{userID}", func(w http.ResponseWriter, r *http.Request) {
		handlerGetUserByID(cfg, w, r)
	})

	mux.HandleFunc("POST /internal/validate-token", func(w http.ResponseWriter, r *http.Request) {
		handlerValidateToken(cfg, w, r)
	})

	// Admin routes (dev only)
	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		handlerReset(cfg, w, r)
	})
}

func handlerLogin(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	type userResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
		Role      string    `json:"role"`
	}

	type resp struct {
		User  userResponse `json:"user"`
		Token string       `json:"token"`
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	user, err := cfg.DB.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		response.RespondWithError(w, http.StatusUnauthorized, "Incorrect email or password", nil)
		return
	}

	match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil || !match {
		response.RespondWithError(w, http.StatusUnauthorized, "Incorrect email or password", nil)
		return
	}

	accessToken, err := auth.MakeJWT(
		user.ID,
		user.Role,
		cfg.JWTSecret,
		time.Hour,
	)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "Couldn't create access JWT", err)
		return
	}

	refreshToken := auth.MakeRefreshToken()

	_, err = cfg.DB.CreateRefreshToken(
		r.Context(),
		database.CreateRefreshTokenParams{
			UserID:    user.ID,
			Token:     refreshToken,
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour * 60),
		},
	)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "Couldn't save refresh token", err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   cfg.Platform != "dev",
		Path:     "/api",
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour * 60),
	})

	response.RespondWithJSON(w, http.StatusOK, resp{
		User: userResponse{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
			Role:      user.Role,
		},
		Token: accessToken,
	})
}

func handlerUsersCreate(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	if params.Password == "" || params.Email == "" {
		response.RespondWithError(w, http.StatusBadRequest, "Email and password are required", nil)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "Couldn't hash password", err)
		return
	}

	user, err := cfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "Couldn't create user", err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, user)
}

func handlerRefresh(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	type resp struct {
		Token string `json:"token"`
	}

	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		response.RespondWithError(w, http.StatusUnauthorized, "Missing refresh token", err)
		return
	}

	refreshToken := cookie.Value

	user, err := cfg.DB.GetUserByRefreshToken(r.Context(), refreshToken)
	if err != nil {
		response.RespondWithError(w, http.StatusUnauthorized, "Couldn't get user for refresh token", err)
		return
	}

	accessToken, err := auth.MakeJWT(
		user.ID,
		user.Role,
		cfg.JWTSecret,
		time.Hour,
	)
	if err != nil {
		response.RespondWithError(w, http.StatusUnauthorized, "Couldn't validate token", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, resp{
		Token: accessToken,
	})
}

func handlerRevoke(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		response.RespondWithError(w, http.StatusUnauthorized, "Missing refresh token", err)
		return
	}

	refreshToken := cookie.Value

	if err := cfg.DB.RevokeRefreshToken(r.Context(), refreshToken); err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "Couldn't revoke session", err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api",
		HttpOnly: true,
		Secure:   cfg.Platform != "dev",
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
	})

	w.WriteHeader(http.StatusNoContent)
}

func handlerGetUserByID(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	user, err := cfg.DB.GetUserByID(r.Context(), userID)
	if err != nil {
		response.RespondWithError(w, http.StatusNotFound, "User not found", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, user)
}

func handlerValidateToken(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	type validateRequest struct {
		Token string `json:"token"`
	}

	type validateResponse struct {
		Valid  bool      `json:"valid"`
		UserID uuid.UUID `json:"user_id,omitempty"`
		Role   string    `json:"role,omitempty"`
	}

	var req validateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	userID, role, err := auth.ValidateJWT(req.Token, cfg.JWTSecret)
	if err != nil {
		response.RespondWithJSON(w, http.StatusOK, validateResponse{Valid: false})
		return
	}

	response.RespondWithJSON(w, http.StatusOK, validateResponse{
		Valid:  true,
		UserID: userID,
		Role:   role,
	})
}

func handlerReset(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	if cfg.Platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Reset is only allowed in dev environment."))
		return
	}

	err := cfg.DB.Reset(r.Context())
	if err != nil {
		response.RespondWithError(
			w,
			http.StatusInternalServerError,
			"Couldn't reset database",
			err,
		)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Database reset to initial state"))
}
