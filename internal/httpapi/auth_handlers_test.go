package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZhazitDastan/kezek/internal/auth"
	"github.com/ZhazitDastan/kezek/internal/httpapi"
)

type fakeAuthService struct {
	registerUser auth.User
	registerErr  error
	loginToken   string
	loginErr     error
	logoutErr    error
	authUserID   int64
	authErr      error
}

var _ httpapi.AuthService = (*fakeAuthService)(nil)

func (f *fakeAuthService) Register(ctx context.Context, email, password string) (auth.User, error) {
	return f.registerUser, f.registerErr
}

func (f *fakeAuthService) Login(ctx context.Context, email, password string) (string, error) {
	return f.loginToken, f.loginErr
}

func (f *fakeAuthService) Logout(ctx context.Context, token string) error {
	return f.logoutErr
}

func (f *fakeAuthService) Authenticate(ctx context.Context, token string) (int64, error) {
	return f.authUserID, f.authErr
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		svc        *fakeAuthService
		wantStatus int
		wantCode   string
	}{
		{name: "ok",
			body:       "{\"email\":\"a@x.kz\",\"password\":\"secret123\"}",
			svc:        &fakeAuthService{registerUser: auth.User{ID: 1, Email: "a@x.kz"}},
			wantStatus: http.StatusCreated,
			wantCode:   "",
		},
		{name: "bad json",
			body:       "{bad",
			svc:        &fakeAuthService{},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_json",
		},
		{
			name:       "email taken",
			body:       "{\"email\":\"a@x.kz\",\"password\":\"secret123\"}",
			svc:        &fakeAuthService{registerErr: auth.ErrEmailTaken},
			wantStatus: http.StatusConflict,
			wantCode:   "email_taken",
		},
		{
			name:       "invalid email",
			body:       `{"email":"not-an-email","password":"secret123"}`,
			svc:        &fakeAuthService{registerErr: auth.ErrInvalidEmail},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "invalid_email",
		},
		{
			name:       "invalid password",
			body:       `{"email":"a@x.kz","password":"123"}`,
			svc:        &fakeAuthService{registerErr: auth.ErrPasswordLength},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "invalid_password",
		},
		{
			name:       "unknown error",
			body:       `{"email":"a@x.kz","password":"secret123"}`,
			svc:        &fakeAuthService{registerErr: errors.New("db secret")},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
		{
			name:       "body too large",
			body:       `{"email":"` + strings.Repeat("a", 2<<20),
			svc:        &fakeAuthService{},
			wantStatus: http.StatusRequestEntityTooLarge,
			wantCode:   "body_too_large",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			httpapi.NewRouter(fakePinger{}, tt.svc).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if strings.Contains(rec.Body.String(), "db secret") {
				t.Errorf("body leaks internal error: %s", rec.Body.String())
			}

			if tt.wantCode != "" {
				var body struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("unmarshal: %v, body: %s", err, rec.Body.String())
				}
				if body.Error.Code != tt.wantCode {
					t.Errorf("error code = %q, want %q", body.Error.Code, tt.wantCode)
				}
			}

			if tt.wantStatus == http.StatusCreated {
				var got struct {
					ID    int64  `json:"id"`
					Email string `json:"email"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("unmarshal: %v, body: %s", err, rec.Body.String())
				}
				if got.ID != 1 || got.Email != "a@x.kz" {
					t.Errorf("got %+v, want id=1 email=a@x.kz", got)
				}
				if strings.Contains(rec.Body.String(), "password") {
					t.Errorf("body contains password field: %s", rec.Body.String())
				}
			}
		})
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		svc        *fakeAuthService
		wantStatus int
		wantToken  string
		wantCode   string
	}{
		{
			name:       "ok",
			body:       `{"email":"a@x.kz","password":"secret123"}`,
			svc:        &fakeAuthService{loginToken: "tok"},
			wantStatus: http.StatusOK,
			wantToken:  "tok",
		},
		{
			name:       "invalid credentials",
			body:       `{"email":"a@x.kz","password":"wrong"}`,
			svc:        &fakeAuthService{loginErr: auth.ErrInvalidCredentials},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "invalid_credentials",
		},
		{
			name:       "bad json",
			body:       `{bad`,
			svc:        &fakeAuthService{},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_json",
		},
		{
			name:       "unknown error",
			body:       `{"email":"a@x.kz","password":"secret123"}`,
			svc:        &fakeAuthService{loginErr: errors.New("db secret")},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			httpapi.NewRouter(fakePinger{}, tt.svc).ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "db secret") {
				t.Errorf("body leaks internal error: %s", rec.Body.String())
			}

			if tt.wantCode != "" {
				var body struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("unmarshal: %v, body: %s", err, rec.Body.String())
				}
				if body.Error.Code != tt.wantCode {
					t.Errorf("error code = %q, want %q", body.Error.Code, tt.wantCode)
				}
			}

			if tt.wantToken != "" {
				var got struct {
					Token string `json:"token"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("unmarshal: %v, body: %s", err, rec.Body.String())
				}
				if got.Token != tt.wantToken {
					t.Errorf("token = %q, want %q", got.Token, tt.wantToken)
				}
			}
		})
	}
}

func TestRequireAuth(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		svc        *fakeAuthService
		wantStatus int
		wantCode   string
		wantID     int64
	}{
		{
			name:       "no header",
			header:     "",
			svc:        &fakeAuthService{},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
		{
			name:       "wrong scheme",
			header:     "Token abc",
			svc:        &fakeAuthService{},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
		{
			name:       "bad token",
			header:     "Bearer abc",
			svc:        &fakeAuthService{authErr: auth.ErrInvalidCredentials},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
		{
			name:       "service error",
			header:     "Bearer abc",
			svc:        &fakeAuthService{authErr: errors.New("db secret")},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
		{
			name:       "ok",
			header:     "Bearer abc",
			svc:        &fakeAuthService{authUserID: 7},
			wantStatus: http.StatusOK,
			wantID:     7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()

			httpapi.NewRouter(fakePinger{}, tt.svc).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if strings.Contains(rec.Body.String(), "db secret") {
				t.Errorf("response body leaked internal error: %s", rec.Body.String())
			}

			if tt.wantCode != "" {
				var errResp struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
					t.Fatalf("failed to unmarshal error body: %v", err)
				}
				if errResp.Error.Code != tt.wantCode {
					t.Errorf("got error code %q, want %q", errResp.Error.Code, tt.wantCode)
				}
			}

			if tt.wantID != 0 {
				var okResp struct {
					ID int64 `json:"id"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &okResp); err != nil {
					t.Fatalf("failed to unmarshal ok body: %v", err)
				}
				if okResp.ID != tt.wantID {
					t.Errorf("got user ID %d, want %d", okResp.ID, tt.wantID)
				}
			}
		})
	}
}

func TestLogout(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		svc        *fakeAuthService
		wantStatus int
		wantCode   string
	}{
		{
			name:       "no header",
			header:     "",
			svc:        &fakeAuthService{},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
		{
			name:       "wrong scheme",
			header:     "Token abc",
			svc:        &fakeAuthService{},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
		{
			name:       "service error",
			header:     "Bearer abc",
			svc:        &fakeAuthService{logoutErr: errors.New("db secret")},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
		{
			name:       "ok",
			header:     "Bearer abc",
			svc:        &fakeAuthService{},
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()

			httpapi.NewRouter(fakePinger{}, tt.svc).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if strings.Contains(rec.Body.String(), "db secret") {
				t.Errorf("body leaks internal error: %s", rec.Body.String())
			}

			if tt.wantCode != "" {
				var body struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("unmarshal: %v, body: %s", err, rec.Body.String())
				}
				if body.Error.Code != tt.wantCode {
					t.Errorf("error code = %q, want %q", body.Error.Code, tt.wantCode)
				}
			}

			if tt.wantStatus == http.StatusNoContent && rec.Body.Len() != 0 {
				t.Errorf("204 response has body: %q", rec.Body.String())
			}
		})
	}
}
