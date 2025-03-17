package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeJWTValidateJWT(t *testing.T) {
	secretString := "1234567890"
	id := uuid.New()

	jwt, err := MakeJWT(id, secretString, time.Duration(time.Second))
	if err != nil {
		t.Fatalf("MakeJWT() resulted in error: %v", err)
	}

	actualId, err := ValidateJWT(jwt, secretString)
	if err != nil {
		t.Fatalf("ValidateJWT() resulted in error: %v", err)
	}
	if id != actualId {
		t.Fatalf("ValidateJWT() resulted in %v, was %v", actualId, id)
	}
}

func TestGetBearerToken(t *testing.T) {
	testTable := []struct {
		name          string
		key           string
		value         string
		expectedToken string
		gotError      error
	}{
		{
			name:          "Valid bearer format",
			key:           "Authorization",
			value:         "Bearer 12345",
			expectedToken: "12345",
			gotError:      nil,
		},
		{
			name:          "Missing bearer keyword",
			key:           "Authorization",
			value:         "12345",
			expectedToken: "",
			gotError:      AuthHeaderIncorrectlyFormatted,
		},
		{
			name:          "Empty header",
			key:           "Authorization",
			value:         "",
			expectedToken: "",
			gotError:      AuthHeaderMissing,
		},
		{
			name:          "Too many parts",
			key:           "Authorization",
			value:         "Bearer 12345 67890",
			expectedToken: "",
			gotError:      AuthHeaderIncorrectlyFormatted,
		},
	}

	headers := http.Header{}
	for _, v := range testTable {
		t.Run(v.name, func(t *testing.T) {
			headers.Set(v.key, v.value)
			token, err := GetBearerToken(headers)
			if err != v.gotError {
				t.Fatalf("GetBearerToken() resulted in error: %v", err)
			}

			if token != v.expectedToken {
				t.Fatalf("Expected %v got %v", v.expectedToken, token)
			}
		})
	}
}
