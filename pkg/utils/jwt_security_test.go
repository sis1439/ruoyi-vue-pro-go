package utils

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/wxlbd/ruoyi-mall-go/pkg/config"
	"strings"
	"testing"
	"time"
)

func TestJWTBoundaries(t *testing.T) {
	old := config.C.Security.JWTSecret
	t.Cleanup(func() { config.C.Security.JWTSecret = old })
	config.C.Security.JWTSecret = ""
	if _, err := GenerateTokenWithInfo(1, 1, 1, "", time.Hour); err == nil {
		t.Fatal("missing secret accepted")
	}
	config.C.Security.JWTSecret = "fixture-secret-012345678901234567890123456789"
	a, err := GenerateTokenWithInfo(1, 1, 1, "member", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := GenerateTokenWithInfo(1, 1, 1, "member", time.Hour)
	if a == b {
		t.Fatal("independent sessions reuse token")
	}
	refresh, _ := GenerateTypedToken(1, 1, 1, "", TokenRefresh, time.Hour)
	if _, err := ParseToken(refresh); err == nil {
		t.Fatal("refresh accepted as access")
	}
	if _, err := ParseTypedToken(a, TokenRefresh); err == nil {
		t.Fatal("access accepted as refresh")
	}
	for _, identity := range [][3]int64{{0, 1, 1}, {1, 0, 1}, {1, 1, 0}, {1, 3, 1}, {1, 1, -1}} {
		if _, err := GenerateTokenWithInfo(identity[0], int(identity[1]), identity[2], "", time.Hour); err == nil {
			t.Fatalf("invalid identity accepted: %v", identity)
		}
	}
	claims, _ := ParseToken(a)
	cases := []struct {
		name   string
		edit   func(*Claims)
		method jwt.SigningMethod
	}{
		{"expired", func(c *Claims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute)) }, jwt.SigningMethodHS256},
		{"expiry missing", func(c *Claims) { c.ExpiresAt = nil }, jwt.SigningMethodHS256},
		{"wrong issuer", func(c *Claims) { c.Issuer = "other" }, jwt.SigningMethodHS256},
		{"missing id", func(c *Claims) { c.ID = "" }, jwt.SigningMethodHS256},
		{"wrong algorithm", func(c *Claims) {}, jwt.SigningMethodHS384},
		{"future issued", func(c *Claims) { c.IssuedAt = jwt.NewNumericDate(time.Now().Add(time.Hour)) }, jwt.SigningMethodHS256},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			copy := *claims
			tc.edit(&copy)
			raw, err := jwt.NewWithClaims(tc.method, copy).SignedString([]byte(config.C.Security.JWTSecret))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = ParseToken(raw); err == nil {
				t.Fatal("invalid JWT accepted")
			}
		})
	}
	config.C.Security.JWTSecret = strings.Repeat("other", 10)
	if _, err := ParseToken(a); err == nil {
		t.Fatal("wrong key accepted")
	}
}
