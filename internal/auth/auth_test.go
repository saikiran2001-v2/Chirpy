package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "supersecret"

	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	gotID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}
	if gotID != userID {
		t.Errorf("gotID %v does not match userID %v", gotID, userID)
	}
}

func TestExpiredJWT(t *testing.T) {
	userID := uuid.New()
	secret := "supersecret"

	token, _ := MakeJWT(userID, secret, -time.Second) // already expired
	gotID, err := ValidateJWT(token, secret)
	if err == nil {
		t.Errorf("ValidateJWT should have failed, got %v", gotID)
	}
}

func TestWrongSecretJWT(t *testing.T) {
	userID := uuid.New()

	token, _ := MakeJWT(userID, "correctsecret", time.Hour)
	gotID, err := ValidateJWT(token, "wrongsecret")
	if err == nil {
		t.Errorf("ValidateJWT should have failed, got %v", gotID)
	}
}
