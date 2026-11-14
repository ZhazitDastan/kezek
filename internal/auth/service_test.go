package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	users    map[string]User
	sessions map[string]Session
	nextID   int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: make(map[string]User), sessions: make(map[string]Session), nextID: 1}
}

var _ Store = (*fakeStore)(nil)

func setup(t *testing.T) (*Service, *fakeStore, User, string) {
	t.Helper()
	const pw = "validpassword123"
	ctx := context.Background()
	store := newFakeStore()
	svc := NewService(store, time.Hour)
	user, err := svc.Register(ctx, "existing@example.com", pw)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return svc, store, user, pw
}

func (f *fakeStore) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	_, ok := f.users[email]
	if ok {
		return User{}, ErrEmailTaken
	}
	f.users[email] = User{ID: f.nextID, Email: email, PasswordHash: passwordHash}
	f.nextID++
	return f.users[email], nil
}

func (f *fakeStore) UserByEmail(ctx context.Context, email string) (User, error) {
	user, ok := f.users[email]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}
func (f *fakeStore) CreateSession(ctx context.Context, tokenHash []byte, userID int64, expiresAt time.Time) error {
	f.sessions[string(tokenHash)] = Session{UserID: userID, ExpiresAt: expiresAt}
	return nil
}

func (f *fakeStore) SessionByHash(ctx context.Context, tokenHash []byte) (Session, error) {
	session, ok := f.sessions[string(tokenHash)]
	if !ok {
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (f *fakeStore) DeleteSession(ctx context.Context, tokenHash []byte) error {
	delete(f.sessions, string(tokenHash))
	return nil
}

func TestRegister_Errors(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{
			name:     "invalid email",
			email:    "not-an-email",
			password: "validpassword123",
			wantErr:  ErrInvalidEmail,
		},
		{
			name:     "short password",
			email:    "new@example.com",
			password: "123",
			wantErr:  ErrPasswordLength,
		},
		{
			name:     "duplicate email",
			email:    "existing@example.com",
			password: "validpassword123",
			wantErr:  ErrEmailTaken,
		},
		{
			name:     "invalid email",
			email:    "@",
			password: "validpassword123",
			wantErr:  ErrInvalidEmail,
		},
		{
			name:     "invalid email 2",
			email:    "a b@c.com",
			password: "validpassword123",
			wantErr:  ErrInvalidEmail,
		},
		{
			name:     "invalid email 2",
			email:    "",
			password: "validpassword123",
			wantErr:  ErrInvalidEmail,
		},
		{
			name:     "valid email",
			email:    "a@b.com",
			password: "validpassword123",
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, _ := setup(t)
			ctx := context.Background()
			_, err := svc.Register(ctx, tt.email, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}

		})
	}

}

func TestRegister_OK(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, time.Hour)
	user, err := svc.Register(context.Background(), "   A@X.kz", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != "a@x.kz" {
		t.Errorf("email = %q, want %q", user.Email, "a@x.kz")
	}
	if user.PasswordHash == "secret123" || user.PasswordHash == "" {
		t.Errorf("password hash is empty or equals plain password")
	}
	if err := checkPassword(user.PasswordHash, "secret123"); err != nil {
		t.Errorf("stored hash does not match password: %v", err)
	}
	if user.ID == 0 {
		t.Errorf("user ID is zero, want assigned ID")
	}
}

func TestLogin_Errors(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
	}{
		{
			name:     "wrong password",
			email:    "existing@example.com",
			password: "wrongpassword",
		},
		{
			name:     "unknown email",
			email:    "unknown@example.com",
			password: "validpassword123",
		},
		{name: "malformed email",
			email:    "not-an-email",
			password: "validpassword123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			svc, _, _, _ := setup(t)
			token, err := svc.Login(ctx, tt.email, tt.password)
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf("got %v, want ErrInvalidCredentials", err)
			}
			if token != "" {
				t.Errorf("token = %q, want empty", token)
			}
		})
	}
}

func TestLogin_OK(t *testing.T) {
	svc, store, _, pw := setup(t)
	ctx := context.Background()

	token, err := svc.Login(ctx, "EXISTING@example.com", pw)
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if len(token) != 43 {
		t.Errorf("token length = %d, want 43", len(token))
	}

	if len(store.sessions) != 1 {
		t.Errorf("sessions = %d, want 1", len(store.sessions))
	}
}

func TestAuthenticate_OK(t *testing.T) {
	svc, _, user, pw := setup(t)
	ctx := context.Background()
	token, err := svc.Login(ctx, user.Email, pw)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	id, err := svc.Authenticate(ctx, token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if id != user.ID {
		t.Errorf("id = %d, want %d", id, user.ID)
	}
}

func TestAuthenticate_BadToken(t *testing.T) {
	svc, _, _, _ := setup(t)
	_, err := svc.Authenticate(context.Background(), "garbage")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthenticate_Expired(t *testing.T) {
	svc, _, user, pw := setup(t)
	ctx := context.Background()
	token, err := svc.Login(ctx, user.Email, pw)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	svc.now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	_, err = svc.Authenticate(ctx, token)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLogout(t *testing.T) {
	svc, store, user, pw := setup(t)
	ctx := context.Background()
	token, err := svc.Login(ctx, user.Email, pw)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := svc.Logout(ctx, token); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if len(store.sessions) != 0 {
		t.Errorf("sessions = %d, want 0", len(store.sessions))
	}
	if _, err := svc.Authenticate(ctx, token); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("after logout: got %v, want ErrInvalidCredentials", err)
	}
	if err := svc.Logout(ctx, token); err != nil {
		t.Errorf("repeated logout: want nil, got %v", err)
	}
}
