package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	store      Store
	sessionTTL time.Duration
	now        func() time.Time
}

func NewService(store Store, sessionTTL time.Duration) *Service {
	return &Service{store: store, sessionTTL: sessionTTL, now: time.Now}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Service) Register(ctx context.Context, email, password string) (User, error) {
	email = normalizeEmail(email)
	if email == "" || !strings.Contains(email, "@") {
		return User{}, ErrInvalidEmail
	}
	hash, err := hashPassword(password)
	if err != nil {
		return User{}, err
	}

	user, err := s.store.CreateUser(ctx, email, hash)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	email = normalizeEmail(email)

	user, err := s.store.UserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}
	if err := checkPassword(user.PasswordHash, password); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("check password: %w", err)
	}
	token, hash, err := newToken()
	if err != nil {
		return "", err
	}
	expiresAt := s.now().Add(s.sessionTTL)
	err = s.store.CreateSession(ctx, hash, user.ID, expiresAt)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (int64, error) {
	hash := hashToken(token)
	sess, err := s.store.SessionByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return 0, ErrInvalidCredentials
		}
		return 0, err
	}
	if !s.now().Before(sess.ExpiresAt) {
		return 0, ErrInvalidCredentials
	}

	return sess.UserID, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, hashToken(token))
}
