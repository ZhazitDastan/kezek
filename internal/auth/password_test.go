package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	pw := "secret123"
	hash, err := hashPassword(pw)
	if err != nil {
		t.Fatalf("error hashing password: %v", err)
	}

	if hash == "" || hash == pw {
		t.Fatalf("hash is empty or equals plain password")
	}

	err = checkPassword(hash, pw)
	if err != nil {
		t.Fatalf("correct password rejected: %v", err)
	}

	err = checkPassword(hash, "secureasd")
	if err == nil {
		t.Errorf("wrong password accepted, want error")
	}

	hash2, err := hashPassword(pw)
	if err != nil {
		t.Fatalf("error hashing password: %v", err)
	}

	if hash2 == hash {
		t.Errorf("hashes of same password are equal, want different (salt)")
	}

	if err := checkPassword(hash2, pw); err != nil {
		t.Errorf("hash2 does not match password: %v", err)
	}

}

func TestHashPassword_Length(t *testing.T) {
	tests := []struct {
		name string
		pw   string
	}{
		{name: "too short", pw: "short"},
		{name: "too long", pw: strings.Repeat("a", 73)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := hashPassword(tt.pw)
			if !errors.Is(err, ErrPasswordLength) {
				t.Errorf("got error %v, want %v", err, ErrPasswordLength)
			}
		})
	}
}
