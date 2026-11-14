package auth

import (
	"golang.org/x/crypto/bcrypt"
)

const maxPasswordLen = 72
const minPasswordLen = 8

var bcryptCost = bcrypt.DefaultCost

var dummyHash = func() string {
	h, err := bcrypt.GenerateFromPassword([]byte("dummy-password"),
		bcryptCost)
	if err != nil {
		panic(err)
	}
	return string(h)
}()

func hashPassword(pw string) (string, error) {
	if len(pw) < minPasswordLen || len(pw) > maxPasswordLen {
		return "", ErrPasswordLength
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcryptCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func checkPassword(hash, pw string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
}
