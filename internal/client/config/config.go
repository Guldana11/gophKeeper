// Package config управляет конфигурацией клиента GophKeeper,
// включая сохранение и загрузку JWT токена.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = ".gophkeeper.json"

// Config хранит конфигурацию клиента.
type Config struct {
	Token         string `json:"token"`
	ServerAddr    string `json:"server_addr"`
	EncryptionKey string `json:"encryption_key"`
	CACertPath    string `json:"ca_cert_path,omitempty"`
}

// configPath возвращает путь к файлу конфигурации в домашней директории.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("ошибка получения домашней директории: %w", err)
	}
	return filepath.Join(home, configFileName), nil
}

// Load загружает конфигурацию из файла. Если файл не существует, возвращает пустую конфигурацию.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("ошибка чтения конфигурации: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка разбора конфигурации: %w", err)
	}
	return &cfg, nil
}

// Save сохраняет конфигурацию в файл.
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации конфигурации: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("ошибка записи конфигурации: %w", err)
	}
	return nil
}
