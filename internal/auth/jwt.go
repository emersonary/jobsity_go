package auth

import (
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/you/go-jobsity-flights/internal/config"
	"log"
	"net/http"
	"strings"
	"time"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func IssueToken(cfg *config.Config, username string) (string, error) {
	claims := jwt.MapClaims{
		"sub": username,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(cfg.JWTSecret))
}

func JWTMiddleware(public, protected *http.ServeMux, cfg *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/auth/") {
			public.ServeHTTP(w, r)
			return
		}
		authH := r.Header.Get("Authorization")
		if authH == "" {
			if t := r.URL.Query().Get("token"); t != "" {
				// inject a Bearer header so the rest of the middleware works unchanged
				r.Header.Set("Authorization", "Bearer "+t)
				authH = r.Header.Get("Authorization")
			}
		}
		if !strings.HasPrefix(authH, "Bearer ") {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		tok := strings.TrimPrefix(authH, "Bearer ")
		_, err := jwt.Parse(tok, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrTokenUnverifiable
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil {
			log.Printf("JWT error: %v", err)
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		protected.ServeHTTP(w, r)
	})
}

func LoginHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if req.Username != cfg.JWTUser || req.Password != cfg.JWTPassword {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		tok, err := IssueToken(cfg, req.Username)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(loginResponse{Token: tok})
	}
}
