// Package storage реализует работу с базой данных PostgreSQL.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/guldana/gophKeeper/internal/models"
)

// Ошибки хранилища.
var (
	ErrUserExists   = errors.New("пользователь уже существует")
	ErrUserNotFound = errors.New("пользователь не найден")
	ErrItemNotFound = errors.New("элемент не найден")
)

// Storage предоставляет методы для работы с БД.
type Storage struct {
	db *sql.DB
}

// New создаёт новое подключение к БД и применяет миграции.
func New(dsn string) (*Storage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка проверки соединения с БД: %w", err)
	}

	s := &Storage{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("ошибка миграции: %w", err)
	}

	return s, nil
}

// Close закрывает соединение с БД.
func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) migrate() error {
	data, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("ошибка чтения миграции: %w", err)
	}
	_, err = s.db.Exec(string(data))
	return err
}

// CreateUser создаёт нового пользователя. Возвращает ErrUserExists, если логин занят.
func (s *Storage) CreateUser(ctx context.Context, login, passwordHash string) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (login, password_hash) VALUES ($1, $2)
		 RETURNING id, login, password_hash, created_at`,
		login, passwordHash,
	).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("ошибка создания пользователя: %w", err)
	}
	return user, nil
}

// GetUserByLogin возвращает пользователя по логину.
func (s *Storage) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, login, password_hash, created_at FROM users WHERE login = $1`,
		login,
	).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователя: %w", err)
	}
	return user, nil
}

// CreateItem сохраняет новый элемент данных.
func (s *Storage) CreateItem(ctx context.Context, item *models.Item) (*models.Item, error) {
	meta, err := json.Marshal(item.Metadata)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации метаданных: %w", err)
	}

	err = s.db.QueryRowContext(ctx,
		`INSERT INTO items (user_id, data_type, encrypted_data, metadata)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at, updated_at`,
		item.UserID, item.DataType, item.EncryptedData, meta,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("ошибка создания элемента: %w", err)
	}
	return item, nil
}

// GetItem возвращает элемент данных по ID и userID.
func (s *Storage) GetItem(ctx context.Context, id, userID string) (*models.Item, error) {
	item := &models.Item{}
	var meta []byte

	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, data_type, encrypted_data, metadata, created_at, updated_at
		 FROM items WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&item.ID, &item.UserID, &item.DataType, &item.EncryptedData, &meta, &item.CreatedAt, &item.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения элемента: %w", err)
	}

	if err := json.Unmarshal(meta, &item.Metadata); err != nil {
		return nil, fmt.Errorf("ошибка десериализации метаданных: %w", err)
	}
	return item, nil
}

// ListItems возвращает все элементы данных пользователя.
func (s *Storage) ListItems(ctx context.Context, userID string) ([]*models.Item, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, data_type, encrypted_data, metadata, created_at, updated_at
		 FROM items WHERE user_id = $1 ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка элементов: %w", err)
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		item := &models.Item{}
		var meta []byte
		if err := rows.Scan(&item.ID, &item.UserID, &item.DataType, &item.EncryptedData, &meta, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ошибка чтения элемента: %w", err)
		}
		if err := json.Unmarshal(meta, &item.Metadata); err != nil {
			return nil, fmt.Errorf("ошибка десериализации метаданных: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// UpdateItem обновляет элемент данных.
func (s *Storage) UpdateItem(ctx context.Context, item *models.Item) error {
	meta, err := json.Marshal(item.Metadata)
	if err != nil {
		return fmt.Errorf("ошибка сериализации метаданных: %w", err)
	}

	result, err := s.db.ExecContext(ctx,
		`UPDATE items SET data_type = $1, encrypted_data = $2, metadata = $3, updated_at = NOW()
		 WHERE id = $4 AND user_id = $5`,
		item.DataType, item.EncryptedData, meta, item.ID, item.UserID,
	)
	if err != nil {
		return fmt.Errorf("ошибка обновления элемента: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества строк: %w", err)
	}
	if rows == 0 {
		return ErrItemNotFound
	}
	return nil
}

// DeleteItem удаляет элемент данных.
func (s *Storage) DeleteItem(ctx context.Context, id, userID string) error {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM items WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("ошибка удаления элемента: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества строк: %w", err)
	}
	if rows == 0 {
		return ErrItemNotFound
	}
	return nil
}

// GetItemsUpdatedAfter возвращает элементы, обновлённые после указанного времени.
func (s *Storage) GetItemsUpdatedAfter(ctx context.Context, userID string, after time.Time) ([]*models.Item, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, data_type, encrypted_data, metadata, created_at, updated_at
		 FROM items WHERE user_id = $1 AND updated_at > $2 ORDER BY updated_at DESC`,
		userID, after,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения обновлённых элементов: %w", err)
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		item := &models.Item{}
		var meta []byte
		if err := rows.Scan(&item.ID, &item.UserID, &item.DataType, &item.EncryptedData, &meta, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ошибка чтения элемента: %w", err)
		}
		if err := json.Unmarshal(meta, &item.Metadata); err != nil {
			return nil, fmt.Errorf("ошибка десериализации метаданных: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func isUniqueViolation(err error) bool {
	return err != nil && (err.Error() == `pq: duplicate key value violates unique constraint "users_login_key"` ||
		contains(err.Error(), "unique constraint"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
