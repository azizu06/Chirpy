package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAndMakeValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"
	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	gotID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatal(err)
	}
	if gotID != userID {
		t.Fatalf("Wrong user")
	}
}

func TestExpiredJWT(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"
	token, err := MakeJWT(userID, secret, -time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Fatal("expected expirted token to return an error")
	}
}

func TestJWTWrongSecret(t *testing.T) {
	userID := uuid.New()
	token, err := MakeJWT(userID, "correct-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ValidateJWT(token, "wrong-secret")
	if err == nil {
		t.Fatal("expected wrong secret to return an error")
	}
}
