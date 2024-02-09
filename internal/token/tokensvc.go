package tokensvc

import (
	"bikefest/pkg/model"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/golang-jwt/jwt/v5"
)

func CreateRefreshToken(user *model.User, secret string, expiry int64) (refreshToken string, err error) {
	claimRefresh := &model.JWTRefreshCustomClaims{
		Identity: model.Identity{
			UserID: user.ID,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiry) * time.Second)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claimRefresh)
	tkn, err := t.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return tkn, nil
}

func CreateAccessToken(user *model.User, secret string, expiry int64) (accessToken string, err error) {
	claimAccess := &model.JWTAccessCustomClaims{
		Identity: model.Identity{
			UserID: user.ID,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiry) * time.Second)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claimAccess)
	tkn, err := t.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return tkn, nil
}

func ExtractIdentityFromToken(tokenString string, secret string) (identity *model.Identity, err error) {
	claims := &model.JWTAccessCustomClaims{}
	tkn, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok { // check signing method
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !tkn.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	return &claims.Identity, nil
}

func ExtractCustomClaimsFromToken(tokenString *string, secret string) (claims *model.JWTAccessCustomClaims, err error) {
	claims = &model.JWTAccessCustomClaims{}
	tkn, err := jwt.ParseWithClaims(*tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok { // check signing method
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !tkn.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	return claims, nil
}

func VerifyRefreshToken(tokenString string, secret string) (user *model.User, err error) {
	claims := &model.JWTRefreshCustomClaims{}
	tkn, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok { // check signing method
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if !tkn.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	return &model.User{
		ID: claims.UserID,
	}, nil
}
