package auth

import (
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
