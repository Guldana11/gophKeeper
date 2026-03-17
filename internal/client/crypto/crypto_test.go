package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	password := "my-secret-password"
	plaintext := []byte("hello, GophKeeper!")

	encrypted, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Зашифрованные данные должны быть длиннее исходных (salt + nonce + tag).
	if len(encrypted) <= len(plaintext) {
		t.Fatal("encrypted data should be longer than plaintext")
	}

	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted data mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptWrongPassword(t *testing.T) {
	plaintext := []byte("secret data")

	encrypted, err := Encrypt(plaintext, "correct-password")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(encrypted, "wrong-password")
	if err == nil {
		t.Fatal("Decrypt with wrong password should fail")
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	password := "password"
	plaintext := []byte("same data")

	enc1, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	enc2, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if bytes.Equal(enc1, enc2) {
		t.Fatal("two encryptions of the same data should produce different ciphertexts")
	}
}

func TestDecryptTooShort(t *testing.T) {
	_, err := Decrypt([]byte("short"), "password")
	if err == nil {
		t.Fatal("Decrypt with too short data should fail")
	}
}

func TestEncryptEmptyData(t *testing.T) {
	password := "password"
	plaintext := []byte{}

	encrypted, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt empty data failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt empty data failed: %v", err)
	}

	if len(decrypted) != 0 {
		t.Fatalf("expected empty decrypted data, got %d bytes", len(decrypted))
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	salt := []byte("1234567890123456")
	key1 := DeriveKey("password", salt)
	key2 := DeriveKey("password", salt)

	if !bytes.Equal(key1, key2) {
		t.Fatal("DeriveKey should be deterministic for same password and salt")
	}

	if len(key1) != KeySize {
		t.Fatalf("key size should be %d, got %d", KeySize, len(key1))
	}
}

func TestDeriveKeyDifferentPasswords(t *testing.T) {
	salt := []byte("1234567890123456")
	key1 := DeriveKey("password1", salt)
	key2 := DeriveKey("password2", salt)

	if bytes.Equal(key1, key2) {
		t.Fatal("different passwords should produce different keys")
	}
}

func TestEncryptDecryptLargeData(t *testing.T) {
	password := "password"
	plaintext := make([]byte, 1024*1024) // 1MB
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	encrypted, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt large data failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt large data failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatal("large data mismatch after decrypt")
	}
}
