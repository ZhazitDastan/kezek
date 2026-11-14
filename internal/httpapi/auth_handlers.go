package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ZhazitDastan/kezek/internal/auth"
)

const maxBodyBytes = 1 << 20

type AuthService interface {
	Register(ctx context.Context, email, password string) (auth.User, error)
	Login(ctx context.Context, email, password string) (string, error)
	Logout(ctx context.Context, token string) error
	Authenticate(ctx context.Context, token string) (int64, error)
}

type authHandler struct {
	svc AuthService
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	err := json.NewDecoder(r.Body).Decode(dst)
	if err == nil {
		return true
	}
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		writeError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body too large")
		return false
	}
	writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
	return false
}

func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	user, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidEmail):
			writeError(w, http.StatusUnprocessableEntity, "invalid_email", "invalid email")
		case errors.Is(err, auth.ErrPasswordLength):
			writeError(w, http.StatusUnprocessableEntity, "invalid_password", "invalid password")
		case errors.Is(err, auth.ErrEmailTaken):
			writeError(w, http.StatusConflict, "email_taken", "email is taken")
		default:
			slog.Error("failed to register user", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, userResponse{ID: user.ID, Email: user.Email})
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	token, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
			return
		}
		slog.Error("failed to login", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{Token: token})
}

func bearerToken(r *http.Request) (string, bool) {
	head := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(head, "Bearer ")
	if !ok || token == "" {
		return "", false
	}
	return token, true
}

func (h *authHandler) logout(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	err := h.svc.Logout(r.Context(), token)
	if err != nil {
		slog.Error("failed to logout", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *authHandler) me(w http.ResponseWriter, r *http.Request) {
	id, ok := UserIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}
