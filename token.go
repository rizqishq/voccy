package main

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenClaims struct {
	OrgID uuid.UUID `json:"org_id"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(orgID uuid.UUID) (string, error) {
	expiry, err := parseDuration(os.Getenv("JWT_EXPIRY"), "15m")
	if err != nil {
		return "", err
	}

	claims := TokenClaims{
		OrgID: orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   orgID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(getJWTSecret()))
}

func GenerateRefreshToken(orgID uuid.UUID) (string, error) {
	expiry, err := parseDuration(os.Getenv("JWT_REFRESH_EXPIRY"), "168h")
	if err != nil {
		return "", err
	}

	claims := TokenClaims{
		OrgID: orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   orgID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(getJWTSecret()))
}

func ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(getJWTSecret()), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}

func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default-secret-change-me"
	}
	return secret
}

func parseDuration(val, fallback string) (time.Duration, error) {
	if val == "" {
		val = fallback
	}
	return time.ParseDuration(val)
}
