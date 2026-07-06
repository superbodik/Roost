package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid or expired token")

type TokenType string

const (
	TokenAccess  TokenType = "access"
	TokenRefresh TokenType = "refresh"
)

type Claims struct {
	UserID  int64     `json:"uid"`
	Email   string    `json:"email"`
	IsAdmin bool      `json:"is_admin"`
	Type    TokenType `json:"type"`
	jwt.RegisteredClaims

	KeyPermissions *[]string `json:"-"`
}

func (c *Claims) HasKeyPermission(code string) bool {
	if c.KeyPermissions == nil {
		return true
	}
	for _, p := range *c.KeyPermissions {
		if p == code {
			return true
		}
	}
	return false
}

type TokenManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenManager(secret string, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func (m *TokenManager) issue(userID int64, email string, isAdmin bool, typ TokenType, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:  userID,
		Email:   email,
		IsAdmin: isAdmin,
		Type:    typ,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *TokenManager) Issue(userID int64, email string, isAdmin bool) (string, error) {
	return m.issue(userID, email, isAdmin, TokenAccess, m.accessTTL)
}

func (m *TokenManager) IssueRefresh(userID int64, email string, isAdmin bool) (string, error) {
	return m.issue(userID, email, isAdmin, TokenRefresh, m.refreshTTL)
}

func (m *TokenManager) Parse(raw string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (m *TokenManager) ParseRefresh(raw string) (*Claims, error) {
	claims, err := m.Parse(raw)
	if err != nil {
		return nil, err
	}
	if claims.Type != TokenRefresh {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
