package models

import (
	"testing"
)

func TestDataType_Constants(t *testing.T) {
	if DataTypeCredential != 1 {
		t.Errorf("DataTypeCredential = %d, want 1", DataTypeCredential)
	}
	if DataTypeText != 2 {
		t.Errorf("DataTypeText = %d, want 2", DataTypeText)
	}
	if DataTypeBinary != 3 {
		t.Errorf("DataTypeBinary = %d, want 3", DataTypeBinary)
	}
	if DataTypeBankCard != 4 {
		t.Errorf("DataTypeBankCard = %d, want 4", DataTypeBankCard)
	}
}

func TestCredential_Fields(t *testing.T) {
	c := Credential{Login: "user", Password: "pass"}
	if c.Login != "user" || c.Password != "pass" {
		t.Error("Credential fields not set correctly")
	}
}

func TestBankCard_Fields(t *testing.T) {
	card := BankCard{
		Number:     "4111111111111111",
		ExpMonth:   12,
		ExpYear:    2025,
		CVV:        "123",
		HolderName: "John Doe",
	}
	if card.Number != "4111111111111111" {
		t.Error("BankCard.Number not set correctly")
	}
	if card.ExpMonth != 12 || card.ExpYear != 2025 {
		t.Error("BankCard expiry not set correctly")
	}
}

func TestTextData_Fields(t *testing.T) {
	td := TextData{Text: "secret note"}
	if td.Text != "secret note" {
		t.Error("TextData.Text not set correctly")
	}
}

func TestBinaryData_Fields(t *testing.T) {
	bd := BinaryData{FileName: "file.pdf", Data: []byte("content")}
	if bd.FileName != "file.pdf" {
		t.Error("BinaryData.FileName not set correctly")
	}
	if string(bd.Data) != "content" {
		t.Error("BinaryData.Data not set correctly")
	}
}
