package infrastructure

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService defines the operations for JWT token management.
type JWTService interface {
	GenerateAccessToken(userID string, role domain.Role) (string, error)
	ValidateToken(tokenString string) (*JWTClaims, error)
}

// JWTClaims contains the claims for the JWT.
type JWTClaims struct {
	UserID string      `json:"user_id"`
	Role   domain.Role `json:"role"`
	jwt.RegisteredClaims
}

type jwtService struct {
	secretKey      string
	issuer         string
	accessTokenTTL time.Duration
}

// NewJWTService creates a new JWT service instance.
func NewJWTService(secret, issuer string, accessTokenTTL time.Duration) JWTService {
	return &jwtService{
		secretKey:      secret,
		issuer:         issuer,
		accessTokenTTL: accessTokenTTL,
	}
}

func (s *jwtService) GenerateAccessToken(userID string, role domain.Role) (string, error) {
	claims := &JWTClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenTTL)),
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

func (s *jwtService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}