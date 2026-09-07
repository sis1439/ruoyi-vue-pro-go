package config

import (
	"strings"
	"testing"
)

func TestSecretValidation(t *testing.T) {
	for _, secret := range []string{"", strings.Repeat("x", 32), "change-me-change-me-change-me-change-me", "your-secret-placeholder-that-is-long-enough"} {
		if (SecurityConfig{JWTSecret: secret}).Validate() == nil {
			t.Fatalf("accepted weak secret")
		}
	}
	if err := (SecurityConfig{JWTSecret: "Synthetic-Independent-Secret-8293!abcd"}).Validate(); err != nil {
		t.Fatal(err)
	}
}
