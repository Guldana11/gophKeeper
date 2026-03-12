package auth

import (
	"testing"
	"time"
)

func TestHashPassword(t *testing.T) {
	m := NewManager("secret", time.Hour)

	hash, err := m.HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword() returned empty hash")
	}
	if hash == "mypassword" {
		t.Fatal("HashPassword() returned plaintext password")
	}
}

func TestCheckPassword(t *testing.T) {
	m := NewManager("secret", time.Hour)

	hash, _ := m.HashPassword("mypassword")

	if !m.CheckPassword("mypassword", hash) {
		t.Error("CheckPassword() should return true for correct password")
	}
	if m.CheckPassword("wrongpassword", hash) {
		t.Error("CheckPassword() should return false for wrong password")
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	m := NewManager("secret", time.Hour)

	token, err := m.GenerateToken("user-123")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken() returned empty token")
	}

	userID, err := m.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if userID != "user-123" {
		t.Errorf("ValidateToken() userID = %v, want user-123", userID)
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	m := NewManager("secret", time.Hour)

	_, err := m.ValidateToken("invalid-token")
	if err == nil {
		t.Error("ValidateToken() should return error for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	m1 := NewManager("secret1", time.Hour)
	m2 := NewManager("secret2", time.Hour)

	token, _ := m1.GenerateToken("user-123")

	_, err := m2.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() should return error for token signed with different secret")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	m := NewManager("secret", -time.Hour) // отрицательный срок = токен сразу просрочен

	token, _ := m.GenerateToken("user-123")

	_, err := m.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() should return error for expired token")
	}
}
