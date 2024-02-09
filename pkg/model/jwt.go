package model

import "github.com/golang-jwt/jwt/v5"

type Identity struct {
	UserID string `json:"user_id"`
}

type JWTAccessCustomClaims struct {
	Identity
	jwt.RegisteredClaims
}

type JWTRefreshCustomClaims struct {
	Identity
	jwt.RegisteredClaims
}
