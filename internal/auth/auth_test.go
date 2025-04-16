package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "secret"

	t.Run("Valid Token", func(t *testing.T) {
		token, err := MakeJWT(userID, tokenSecret, time.Hour)
		if err != nil {
			t.Fatalf("failed to create token: %v", err)
		}

		parsedID, err := ValidateJWT(token, tokenSecret)
		if err != nil {
			t.Fatalf("failed to validate token: %v", err)
		}

		if parsedID != userID {
			t.Errorf("id mismatch, got: %v, not: %v", parsedID, userID)
		}
	})
	t.Run("Expired Token", func(t *testing.T) {
		// Create a token that expires immediately (negative duration)
		token, err := MakeJWT(userID, tokenSecret, -time.Hour)
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		// Try to validate the expired token
		_, err = ValidateJWT(token, tokenSecret)
		if err == nil {
			t.Error("Expected error for expired token, got nil")
		}
	})
}
