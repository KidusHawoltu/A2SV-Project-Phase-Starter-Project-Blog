package infrastructure

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// JWTService defines the operations for JWT token management.
type JWTService interface {
	GenerateAccessToken(userID string, role domain.Role) (string, *JWTClaims, error)
	GenerateRefreshToken(userID string) (string, *JWTClaims, error)
	ValidateToken(tokenString string) (*JWTClaims, error)
	ParseExpiredToken(tokenString string) (*JWTClaims, error)
	GetRefreshTokenExpiry() time.Duration
}

// JWTClaims contains the claims for the JWT.
type JWTClaims struct {
	UserID string      `json:"user_id"`
	Role   domain.Role `json:"role"`
	jwt.RegisteredClaims
}

type jwtService struct {
	secretKey       string
	issuer          string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewJWTService creates a new JWT service instance.
func NewJWTService(secret, issuer string, accessTokenTTL, refreshTokenTTL time.Duration) JWTService {
	fmt.Println(refreshTokenTTL)
	return &jwtService{
		secretKey:       secret,
		issuer:          issuer,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (s *jwtService) GenerateAccessToken(userID string, role domain.Role) (string, *JWTClaims, error) {
	claims := &JWTClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        primitive.NewObjectID().Hex(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenTTL)),
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.secretKey))
	return tokenString, claims, err
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

// GenerateRefreshToken creates a long-lived refresh token.
func (s *jwtService) GenerateRefreshToken(userID string) (string, *JWTClaims, error) {
	claims := &JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate((time.Now().Add((s.refreshTokenTTL)))),
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        primitive.NewObjectID().Hex(),
		},
	}
	fmt.Println(claims.RegisteredClaims.ExpiresAt.Date())
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.secretKey))
	return tokenString, claims, err
}

func (s *jwtService) ParseExpiredToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.secretKey), nil
	})

	if claims, ok := token.Claims.(*JWTClaims); ok {
		if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
			return nil, err
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

func (s *jwtService) GetRefreshTokenExpiry() time.Duration {
	return s.refreshTokenTTL
}
