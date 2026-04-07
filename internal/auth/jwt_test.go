package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func generateTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

func TestGenerateJWT(t *testing.T) {
	key := generateTestKey(t)
	now := time.Now()

	token, err := GenerateJWT("12345", key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return &key.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("unexpected claims type")
	}

	iss, err := claims.GetIssuer()
	if err != nil {
		t.Fatalf("get issuer: %v", err)
	}
	if iss != "12345" {
		t.Errorf("iss = %q, want %q", iss, "12345")
	}

	iat, err := claims.GetIssuedAt()
	if err != nil {
		t.Fatalf("get iat: %v", err)
	}
	drift := now.Sub(iat.UTC())
	if drift < 50*time.Second || drift > 70*time.Second {
		t.Errorf("iat drift = %v, want ~60s", drift)
	}

	exp, err := claims.GetExpirationTime()
	if err != nil {
		t.Fatalf("get exp: %v", err)
	}
	ttl := exp.UTC().Sub(now)
	if ttl < 9*time.Minute || ttl > 11*time.Minute {
		t.Errorf("ttl = %v, want ~10m", ttl)
	}
}
