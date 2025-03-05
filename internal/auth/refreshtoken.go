package auth

import (
	"crypto/rand"
	"encoding/hex"
)

func MakeRefreshToken() (string, error) {
	randData := make([]byte, 32)
	rand.Read(randData)

	return hex.EncodeToString(randData), nil
}
