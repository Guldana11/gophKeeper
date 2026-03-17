// Package crypto реализует шифрование и дешифрование данных
// с использованием AES-256-GCM и ключа, производного от пароля через PBKDF2.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// SaltSize — размер соли для PBKDF2 в байтах.
	SaltSize = 16
	// KeySize — размер ключа AES-256 в байтах.
	KeySize = 32
	// Iter — количество итераций PBKDF2.
	Iter = 100_000
)

// DeriveKey генерирует ключ AES-256 из пароля и соли с помощью PBKDF2.
func DeriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, Iter, KeySize, sha256.New)
}

// Encrypt шифрует данные с использованием AES-256-GCM.
// Возвращает: salt (16) + nonce (12) + ciphertext.
func Encrypt(plaintext []byte, password string) ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("ошибка генерации соли: %w", err)
	}

	key := DeriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания шифра: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("ошибка генерации nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// salt + nonce + ciphertext
	result := make([]byte, 0, SaltSize+len(nonce)+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// Decrypt дешифрует данные, зашифрованные функцией Encrypt.
// Ожидает формат: salt (16) + nonce (12) + ciphertext.
func Decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < SaltSize+12 {
		return nil, errors.New("данные слишком короткие для дешифрования")
	}

	salt := data[:SaltSize]
	key := DeriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания шифра: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < SaltSize+nonceSize {
		return nil, errors.New("данные слишком короткие для дешифрования")
	}

	nonce := data[SaltSize : SaltSize+nonceSize]
	ciphertext := data[SaltSize+nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка дешифрования: %w", err)
	}

	return plaintext, nil
}
