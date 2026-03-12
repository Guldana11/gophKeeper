// Package auth реализует аутентификацию: хеширование паролей и работу с JWT токенами.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Ошибки аутентификации.
var (
	ErrInvalidToken = errors.New("невалидный токен")
	ErrExpiredToken = errors.New("токен истёк")
)

// Manager управляет аутентификацией пользователей.
type Manager struct {
	secretKey   []byte
	tokenExpiry time.Duration
}

// NewManager создаёт новый менеджер аутентификации.
func NewManager(secretKey string, tokenExpiry time.Duration) *Manager {
	return &Manager{
		secretKey:   []byte(secretKey),
		tokenExpiry: tokenExpiry,
	}
}

// HashPassword хеширует пароль с помощью bcrypt.
func (m *Manager) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("ошибка хеширования пароля: %w", err)
	}
	return string(hash), nil
}

// CheckPassword проверяет соответствие пароля хешу.
func (m *Manager) CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateToken создаёт JWT токен для пользователя.
func (m *Manager) GenerateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(m.tokenExpiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", fmt.Errorf("ошибка подписи токена: %w", err)
	}
	return signed, nil
}

// ValidateToken проверяет JWT токен и возвращает ID пользователя.
func (m *Manager) ValidateToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи: %v", t.Header["alg"])
		}
		return m.secretKey, nil
	})
	if err != nil {
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", ErrInvalidToken
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", ErrInvalidToken
	}

	return userID, nil
}
