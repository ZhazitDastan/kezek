package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ZhazitDastan/kezek/internal/platform/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgStore struct {
	pool *pgxpool.Pool
}

func NewPgStore(pool *pgxpool.Pool) *PgStore {
	return &PgStore{pool: pool}
}

var _ Store = (*PgStore)(nil)

func (p *PgStore) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var id int64
	err := p.pool.QueryRow(ctx, "INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id",
		email, passwordHash).Scan(&id)
	if err != nil {
		if postgres.IsUniqueViolation(err) {
			return User{}, ErrEmailTaken
		}
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return User{ID: id, Email: email, PasswordHash: passwordHash}, nil
}

func (p *PgStore) UserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := p.pool.QueryRow(ctx,
		"SELECT id, email, password_hash FROM users WHERE email = $1", email).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("get user: %w", err)
	}

	return user, nil
}

func (p *PgStore) CreateSession(ctx context.Context, tokenHash []byte, userID int64, expiresAt time.Time) error {
	_, err := p.pool.Exec(ctx, "INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)",
		tokenHash, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (p *PgStore) SessionByHash(ctx context.Context, tokenHash []byte) (Session, error) {
	var session Session
	err := p.pool.QueryRow(ctx, "SELECT user_id, expires_at FROM sessions WHERE token_hash = $1",
		tokenHash).Scan(&session.UserID, &session.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, fmt.Errorf("get session: %w", err)
	}
	return session, nil
}

func (p *PgStore) DeleteSession(ctx context.Context, tokenHash []byte) error {
	_, err := p.pool.Exec(ctx, "DELETE FROM sessions WHERE token_hash = $1", tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
