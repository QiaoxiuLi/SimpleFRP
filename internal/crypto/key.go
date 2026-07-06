package crypto

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
)

func RandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func DeriveSessionKey(password, salt string) []byte {
	return argon2.IDKey([]byte(password), []byte(salt), 3, 64*1024, 4, 32)
}
