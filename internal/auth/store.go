package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionNotFound    = errors.New("session not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrPasswordLength     = errors.New("password length invalid")
	ErrInvalidEmail       = errors.New("invalid email")
)

type User struct {
	ID           int64
	Email        string
	PasswordHash string
}

type Session struct {
	UserID    int64
	ExpiresAt time.Time
}

type Store interface {
	// CreateUser → ErrEmailTaken when email is duplicated
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	// UserByEmail → ErrUserNotFound if not found
	UserByEmail(ctx context.Context, email string) (User, error)
	CreateSession(ctx context.Context, tokenHash []byte, userID int64, expiresAt time.Time) error
	// SessionByHash → ErrSessionNotFound  if not found
	SessionByHash(ctx context.Context, tokenHash []byte) (Session, error)
	// DeleteSession returns nil even if the session does not exist
	DeleteSession(ctx context.Context, tokenHash []byte) error
}
