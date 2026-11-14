package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type healthHandler struct {
	p Pinger
}

func NewRouter(p Pinger, authSvc AuthService) http.Handler {
	h := &healthHandler{p: p}
	ah := &authHandler{svc: authSvc}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("POST /auth/register", ah.register)
	mux.HandleFunc("POST /auth/login", ah.login)
	mux.HandleFunc("POST /auth/logout", ah.logout)
	mux.Handle("GET /me", RequireAuth(authSvc, http.HandlerFunc(ah.me)))
	return mux
}

func (h *healthHandler) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	err := h.p.Ping(ctx)
	if err != nil {
		slog.Error("ping failed", "err", err)
		writeError(w, http.StatusServiceUnavailable, "db_unavailable", "service unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
