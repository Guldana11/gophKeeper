// Package models содержит доменные модели данных приложения GophKeeper.
package models

import "time"

// DataType определяет тип хранимых данных.
type DataType int

const (
	DataTypeCredential DataType = iota + 1
	DataTypeText
	DataTypeBinary
	DataTypeBankCard
)

// User представляет зарегистрированного пользователя системы.
type User struct {
	ID           string
	Login        string
	PasswordHash string
	CreatedAt    time.Time
}

// Item представляет единицу хранения данных пользователя.
// Поле EncryptedData содержит зашифрованные данные (шифрование на стороне клиента).
type Item struct {
	ID            string
	UserID        string
	DataType      DataType
	EncryptedData []byte
	Metadata      map[string]string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Credential хранит пару логин/пароль.
type Credential struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// TextData хранит произвольные текстовые данные.
type TextData struct {
	Text string `json:"text"`
}

// BinaryData хранит произвольные бинарные данные.
type BinaryData struct {
	FileName string `json:"file_name"`
	Data     []byte `json:"data"`
}

// BankCard хранит данные банковской карты.
type BankCard struct {
	Number     string `json:"number"`
	ExpMonth   int    `json:"exp_month"`
	ExpYear    int    `json:"exp_year"`
	CVV        string `json:"cvv"`
	HolderName string `json:"holder_name"`
}
