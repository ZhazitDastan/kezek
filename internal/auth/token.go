package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

const tokenBytes = 32

func newToken() (token string, hash []byte, err error) {
	b := make([]byte, tokenBytes)
	_, err = rand.Read(b)
	if err != nil {
		return "", nil, err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	hash = hashToken(token)
	return token, hash, nil
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
