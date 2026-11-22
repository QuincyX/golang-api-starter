package auth

import (
	"strings"
	"testing"
	"testing/synctest"
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
	for i := range 3 {
		if !limiter.allow("ip:127.0.0.1", 3, time.Hour) {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
	if limiter.allow("ip:127.0.0.1", 3, time.Hour) {
		t.Fatal("fourth attempt should be rejected")
	}
}

// TestLoginLimiterWindowResets uses synctest's fake clock, so the expiry is
// verified instantly and does not depend on wall-clock sleep.
func TestLoginLimiterWindowResets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		limiter := newLoginLimiter()
		for i := range 3 {
			if !limiter.allow("ip:127.0.0.1", 3, time.Hour) {
				t.Fatalf("attempt %d should be allowed", i+1)
			}
		}
		if limiter.allow("ip:127.0.0.1", 3, time.Hour) {
			t.Fatal("fourth attempt should be rejected")
		}

		synctest.Sleep(time.Hour + time.Second)
		if !limiter.allow("ip:127.0.0.1", 3, time.Hour) {
			t.Fatal("attempt after the window reset should be allowed")
		}
	})
}
