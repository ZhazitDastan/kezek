package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	uniqueErr := &pgconn.PgError{Code: uniqueViolation}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "unique violation",
			err:  uniqueErr,
			want: true,
		},
		{
			name: "wrapped unique violation",
			err:  fmt.Errorf("insert: %w", uniqueErr),
			want: true,
		},
		{
			name: "other pg error",
			err:  &pgconn.PgError{Code: "23503"},
			want: false,
		},
		{
			name: "plain error",
			err:  errors.New("x"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUniqueViolation(tt.err); got != tt.want {
				t.Errorf("IsUniqueViolation() = %v, want %v", got, tt.want)
			}
		})
	}
}
