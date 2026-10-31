package account

import (
	"crypto/sha256"
	"strings"
	"testing"
)

func TestRandomTokenAndHash(t *testing.T) {
	first, err := randomToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := randomToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) < 32 {
		t.Fatalf("token too short: %d", len(first))
	}
	if first == second {
		t.Fatal("tokens should be unique")
	}
	sum := sha256.Sum256([]byte(first))
	if got := hashToken(first); string(got) != string(sum[:]) {
		t.Fatal("hashToken does not match sha256")
	}
}

func TestActivationURL(t *testing.T) {
	svc := NewService(nil, nil, "https://admin.example.com/")
	if got := svc.activationURL("a b+c"); !strings.HasPrefix(got, "https://admin.example.com/activate?token=") {
		t.Fatalf("activation URL = %q", got)
	}
	if strings.Contains(svc.activationURL("a b"), " ") {
		t.Fatal("activation token must be URL escaped")
	}
}

func TestNormalizePage(t *testing.T) {
	cases := []struct {
		page, size, wantPage, wantSize int
	}{
		{0, 0, 1, 20},
		{-1, 500, 1, 100},
		{3, 10, 3, 10},
	}
	for _, tc := range cases {
		page, size := normalizePage(tc.page, tc.size)
		if page != tc.wantPage || size != tc.wantSize {
			t.Errorf("normalizePage(%d,%d) = (%d,%d), want (%d,%d)",
				tc.page, tc.size, page, size, tc.wantPage, tc.wantSize)
		}
	}
}
