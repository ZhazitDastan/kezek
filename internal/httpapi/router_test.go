package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ZhazitDastan/kezek/internal/httpapi"
)

type fakePinger struct {
	err error
}

type hangPinger struct{}

func (hangPinger) Ping(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func (f fakePinger) Ping(ctx context.Context) error { return f.err }

func TestHealthz(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		path          string
		pingErr       error
		wantStatus    int
		wantStatusVal string
		wantErrCode   string
	}{
		{
			name:          "healthz ok",
			method:        http.MethodGet,
			path:          "/healthz",
			pingErr:       nil,
			wantStatus:    http.StatusOK,
			wantStatusVal: "ok",
		},
		{
			name:        "db down",
			method:      http.MethodGet,
			path:        "/healthz",
			pingErr:     errors.New("secret: password auth failed"),
			wantStatus:  http.StatusServiceUnavailable,
			wantErrCode: "db_unavailable",
		},
		{
			name:       "method not allowed",
			method:     http.MethodPost,
			path:       "/healthz",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "unknown path",
			method:     http.MethodGet,
			path:       "/nope",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			httpapi.NewRouter(fakePinger{err: tt.pingErr}).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("got %d, want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			wantJSON := tt.wantStatusVal != "" || tt.wantErrCode != ""
			if wantJSON {
				if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("content-type = %q, want application/json", ct)
				}
			}

			if tt.pingErr != nil && strings.Contains(rec.Body.String(), tt.pingErr.Error()) {
				t.Errorf("body leaks error: %s", rec.Body.String())
			}

			if wantJSON {
				var body struct {
					Status string `json:"status"`
					Error  struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("unmarshal: %v, body: %s", err, rec.Body.String())
				}
				if body.Status != tt.wantStatusVal {
					t.Errorf("status = %q, want %q", body.Status, tt.wantStatusVal)
				}
				if body.Error.Code != tt.wantErrCode {
					t.Errorf("error.code = %q, want %q", body.Error.Code, tt.wantErrCode)
				}
			}

		})
	}
}

func TestHealthz_PingTimeout(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	start := time.Now()
	httpapi.NewRouter(hangPinger{}).ServeHTTP(rec, req)
	elapsed := time.Since(start)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if elapsed > 3*time.Second {
		t.Errorf("handler hung for %v", elapsed)
	}
}
