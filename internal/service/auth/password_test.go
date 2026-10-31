package auth

import (
	"strings"
	"testing"
	"time"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("CorrectHorseBatteryStaple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("unexpected hash format: %s", hash)
	}
	ok, err := VerifyPassword(hash, "CorrectHorseBatteryStaple")
	if err != nil || !ok {
		t.Fatalf("VerifyPassword = %v, %v", ok, err)
	}
	ok, _ = VerifyPassword(hash, "wrong")
	if ok {
		t.Fatal("wrong password accepted")
	}
}

func TestNeedsRehash(t *testing.T) {
	hash, err := HashPassword("password-123")
	if err != nil {
		t.Fatal(err)
	}
	if NeedsRehash(hash) {
		t.Fatal("fresh hash should not need rehash")
	}
	if !NeedsRehash("$argon2id$v=19$m=1,t=1,p=1$c2FsdA$aGFzaA") {
		t.Fatal("weak parameters should need rehash")
	}
}

func TestLoginLimiter(t *testing.T) {
	limiter := newLoginLimiter()
	for i := 0; i < 3; i++ {
		if !limiter.allow("ip:127.0.0.1", 3, time.Hour) {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
	if limiter.allow("ip:127.0.0.1", 3, time.Hour) {
		t.Fatal("fourth attempt should be rejected")
	}
}
