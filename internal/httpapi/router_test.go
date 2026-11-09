package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ZhazitDastan/kezek/internal/httpapi"
)

func TestHealthz(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"healthz ok", http.MethodGet, "/healthz",
			http.StatusOK},
		{"wrong method", http.MethodPost, "/healthz",
			http.StatusMethodNotAllowed},
		{"unknown path", http.MethodGet, "/nope",
			http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			rec := httptest.NewRecorder()
			httpapi.NewRouter().ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Errorf("got %d, want %d", rec.Code, tt.want)
			}

		})
	}
}
