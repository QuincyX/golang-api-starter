package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims deliberately contains no permissions or profile data. Permissions are
// loaded per request, so changing a role takes effect immediately.
type Claims struct {
	SessionID    string `json:"sid"`
	TokenVersion int    `json:"ver"`
	jwt.RegisteredClaims
}

type Manager struct {
	secret   []byte
	issuer   string
	audience string
	ttl      time.Duration
}

func New(secret, issuer, audience string, ttl time.Duration) (*Manager, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("jwt secret must be at least 32 bytes")
	}
	if issuer == "" || audience == "" {
		return nil, fmt.Errorf("jwt issuer and audience are required")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("jwt ttl must be positive")
	}
	return &Manager{secret: []byte(secret), issuer: issuer, audience: audience, ttl: ttl}, nil
}

func (m *Manager) TTL() time.Duration {
	return m.ttl
}

func (m *Manager) Sign(userID, sessionID string, tokenVersion int) (string, time.Time, error) {
	if userID == "" || sessionID == "" {
		return "", time.Time{}, fmt.Errorf("userID and sessionID are required")
	}
	now := time.Now()
	expiresAt := now.Add(m.ttl)
	claims := Claims{
		SessionID:    sessionID,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

func (m *Manager) Parse(raw string) (*Claims, error) {
	var claims Claims
	token, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
		jwt.WithAudience(m.audience),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(60*time.Second),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("token is not valid")
	}
	if claims.Subject == "" || claims.SessionID == "" {
		return nil, errors.New("token is missing sub or sid")
	}
	return &claims, nil
}
