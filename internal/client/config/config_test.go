package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSave(t *testing.T) {
	// Подменяем путь конфигурации на временную директорию.
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Загрузка несуществующего файла — пустая конфигурация.
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() вернул ошибку: %v", err)
	}
	if cfg.Token != "" {
		t.Errorf("ожидался пустой токен, получен %q", cfg.Token)
	}

	// Сохранение и повторная загрузка.
	cfg.Token = "test-token-123"
	cfg.ServerAddr = "localhost:3200"
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() вернул ошибку: %v", err)
	}

	// Проверяем права файла.
	path := filepath.Join(tmpDir, configFileName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("файл конфигурации не найден: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("ожидались права 0600, получены %o", perm)
	}

	// Повторная загрузка.
	cfg2, err := Load()
	if err != nil {
		t.Fatalf("Load() вернул ошибку: %v", err)
	}
	if cfg2.Token != "test-token-123" {
		t.Errorf("ожидался токен %q, получен %q", "test-token-123", cfg2.Token)
	}
	if cfg2.ServerAddr != "localhost:3200" {
		t.Errorf("ожидался адрес %q, получен %q", "localhost:3200", cfg2.ServerAddr)
	}
}
