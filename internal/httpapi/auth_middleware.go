package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ZhazitDastan/kezek/internal/auth"
)

type userIDKey struct{}

func UserIDFrom(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey{}).(int64)
	return id, ok
}

func RequireAuth(svc AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
			return
		}
		userID, err := svc.Authenticate(r.Context(), token)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidCredentials) {
				writeError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
				return
			}
			slog.Error("failed to authenticate user", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
