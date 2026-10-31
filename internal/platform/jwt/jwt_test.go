package jwt

import (
	"strings"
	"testing"
	"time"
)

func TestSignAndParse(t *testing.T) {
	manager, err := New(strings.Repeat("k", 32), "homebrew-api", "admin-web", 15*time.Minute)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	token, expiresAt, err := manager.Sign("user-1", "session-1", 2)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if !expiresAt.After(time.Now()) {
		t.Fatal("expiry should be in the future")
	}

	claims, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if claims.Subject != "user-1" || claims.SessionID != "session-1" || claims.TokenVersion != 2 {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestRejectTamperedToken(t *testing.T) {
	manager, _ := New(strings.Repeat("k", 32), "homebrew-api", "admin-web", 15*time.Minute)
	token, _, _ := manager.Sign("user-1", "session-1", 0)

	if _, err := manager.Parse(token + "x"); err == nil {
		t.Fatal("expected a tampered token to fail")
	}
}

func TestRejectWrongAudience(t *testing.T) {
	manager, _ := New(strings.Repeat("k", 32), "homebrew-api", "admin-web", 15*time.Minute)
	other, _ := New(strings.Repeat("k", 32), "homebrew-api", "other-app", 15*time.Minute)

	token, _, _ := manager.Sign("user-1", "session-1", 0)
	if _, err := other.Parse(token); err == nil {
		t.Fatal("expected a token for another audience to fail")
	}
}
