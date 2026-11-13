package auth

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func TestNewToken(t *testing.T) {
	token, hash, err := newToken()
	if err != nil {
		t.Fatalf("newToken err: %v", err)
	}

	if len(token) != 43 {
		t.Errorf("token length should be 43, but got %d", len(token))
	}

	if len(hash) != sha256.Size {
		t.Errorf("hash length should be %d, but got %d", sha256.Size, len(hash))
	}

	computedHash := hashToken(token)
	if !bytes.Equal(hash, computedHash) {
		t.Errorf("hashToken(%q) = %x, want %x", token, computedHash, hash)
	}
}

func TestNewToken_Unique(t *testing.T) {
	token, _, err := newToken()
	if err != nil {
		t.Fatalf("newToken err: %v", err)
	}
	token2, _, err := newToken()
	if err != nil {
		t.Fatalf("newToken err: %v", err)
	}

	if token == token2 {
		t.Errorf("newToken returned same token")
	}
}
