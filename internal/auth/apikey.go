package auth

import (
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", AuthHeaderMissing
	}

	parts := strings.Split(authHeader, " ")

	if len(parts) != 2 || parts[0] != "ApiKey" {
		return "", AuthHeaderIncorrectlyFormatted
	}

	return parts[1], nil
}
