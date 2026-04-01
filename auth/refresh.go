package auth

import (
	"crypto/rand"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

func GenerateRefreshToken() (plain string, hash string, err error) {
	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		return "", "", err
	}

	plain = hex.EncodeToString(b)

	hashed, err := bcrypt.GenerateFromPassword(
		[]byte(plain),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", "", err
	}

	return plain, string(hashed), nil
}

func CompareRefreshToken(hash string, plain string) error {
	return bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(plain),
	)
}
