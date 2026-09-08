package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/wxlbd/ruoyi-mall-go/pkg/config"
	"time"
)

const TokenAccess = "access"
const TokenRefresh = "refresh"

type Claims struct {
	UserID    int64  `json:"userId"`
	UserType  int    `json:"userType"`
	TenantID  int64  `json:"tenantId"`
	Nickname  string `json:"nickname"`
	TokenType string `json:"tokenType"`
	jwt.RegisteredClaims
}

func GenerateToken(userID int64, duration time.Duration) (string, error) {
	return "", errors.New("explicit user type and tenant required")
}
func GenerateTokenWithInfo(userID int64, userType int, tenantID int64, nickname string, duration time.Duration) (string, error) {
	return GenerateTypedToken(userID, userType, tenantID, nickname, TokenAccess, duration)
}
func GenerateTypedToken(userID int64, userType int, tenantID int64, nickname, kind string, duration time.Duration) (string, error) {
	if len(config.C.Security.JWTSecret) < 32 {
		return "", errors.New("JWT secret must contain at least 32 bytes")
	}
	if userID <= 0 || tenantID <= 0 || (userType != 1 && userType != 2) || (kind != TokenAccess && kind != TokenRefresh) || duration <= 0 {
		return "", errors.New("invalid token identity or type")
	}
	id := make([]byte, 32)
	if _, err := rand.Read(id); err != nil {
		return "", err
	}
	claims := Claims{UserID: userID, UserType: userType, TenantID: tenantID, Nickname: nickname, TokenType: kind, RegisteredClaims: jwt.RegisteredClaims{ID: hex.EncodeToString(id), Issuer: "yudao-go", IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration))}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(config.C.Security.JWTSecret))
}
func ParseToken(raw string) (*Claims, error) { return ParseTypedToken(raw, TokenAccess) }
func ParseTypedToken(raw, kind string) (*Claims, error) {
	if len(config.C.Security.JWTSecret) < 32 {
		return nil, errors.New("JWT secret is not configured")
	}
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (interface{}, error) { return []byte(config.C.Security.JWTSecret), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("yudao-go"), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil {
		return nil, err
	}
	c, ok := token.Claims.(*Claims)
	if !ok || !token.Valid || c.TokenType != kind || c.ID == "" || c.IssuedAt == nil || c.UserID <= 0 || c.TenantID <= 0 || (c.UserType != 1 && c.UserType != 2) {
		return nil, errors.New("invalid token claims")
	}
	return c, nil
}
