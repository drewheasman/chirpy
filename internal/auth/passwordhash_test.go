package auth

import "testing"

func TestPasswordHashUnhash(t *testing.T) {
	password := "mypassword"

	hashedPassword, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword(%v) resulted in error: %v", password, err)
	}

	if err := CheckPasswordHash(password, hashedPassword); err != nil {
		t.Fatalf("CheckPasswordHash(%v) resulted in error: %v", password, err)
	}
}

func TestPasswordHashUnhashEmpty(t *testing.T) {
	password := ""

	hashedPassword, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword(%v) resulted in error: %v", password, err)
	}

	if err := CheckPasswordHash(password, hashedPassword); err != nil {
		t.Fatalf("CheckPasswordHash(%v) resulted in error: %v", password, err)
	}
}
