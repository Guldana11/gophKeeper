// Package storage реализует работу с базой данных PostgreSQL.
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guldana/gophKeeperr/internal/models"
)

// psql — билдер запросов с плейсхолдерами PostgreSQL ($1, $2, ...).
var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// Ошибки хранилища.
var (
	ErrUserExists   = errors.New("пользователь уже существует")
	ErrUserNotFound = errors.New("пользователь не найден")
	ErrItemNotFound = errors.New("элемент не найден")
)

// Storage предоставляет методы для работы с БД.
type Storage struct {
	pool *pgxpool.Pool
}

// New создаёт новое подключение к БД и применяет миграции.
func New(dsn string) (*Storage, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("ошибка проверки соединения с БД: %w", err)
	}

	s := &Storage{pool: pool}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("ошибка миграции: %w", err)
	}

	return s, nil
}

// Close закрывает соединение с БД.
func (s *Storage) Close() {
	s.pool.Close()
}

func (s *Storage) migrate() error {
	data, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("ошибка чтения миграции: %w", err)
	}
	_, err = s.pool.Exec(context.Background(), string(data))
	return err
}

// CreateUser создаёт нового пользователя. Возвращает ErrUserExists, если логин занят.
func (s *Storage) CreateUser(ctx context.Context, login, passwordHash string) (*models.User, error) {
	query, args, err := psql.
		Insert("users").
		Columns("login", "password_hash").
		Values(login, passwordHash).
		Suffix("RETURNING id, login, password_hash, created_at").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("ошибка построения запроса: %w", err)
	}

	user := &models.User{}
	err = s.pool.QueryRow(ctx, query, args...).
		Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("ошибка создания пользователя: %w", err)
	}
	return user, nil
}

// GetUserByLogin возвращает пользователя по логину.
func (s *Storage) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	query, args, err := psql.
		Select("id", "login", "password_hash", "created_at").
		From("users").
		Where(sq.Eq{"login": login}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("ошибка построения запроса: %w", err)
	}

	user := &models.User{}
	err = s.pool.QueryRow(ctx, query, args...).
		Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
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

	query, args, err := psql.
		Insert("items").
		Columns("user_id", "data_type", "encrypted_data", "metadata").
		Values(item.UserID, item.DataType, item.EncryptedData, meta).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("ошибка построения запроса: %w", err)
	}

	err = s.pool.QueryRow(ctx, query, args...).
		Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания элемента: %w", err)
	}
	return item, nil
}

// itemColumns — список колонок таблицы items для SELECT-запросов.
var itemColumns = []string{"id", "user_id", "data_type", "encrypted_data", "metadata", "created_at", "updated_at"}

// GetItem возвращает элемент данных по ID и userID.
func (s *Storage) GetItem(ctx context.Context, id, userID string) (*models.Item, error) {
	query, args, err := psql.
		Select(itemColumns...).
		From("items").
		Where(sq.Eq{"id": id, "user_id": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("ошибка построения запроса: %w", err)
	}

	item := &models.Item{}
	var meta []byte
	err = s.pool.QueryRow(ctx, query, args...).
		Scan(&item.ID, &item.UserID, &item.DataType, &item.EncryptedData, &meta, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
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
	query, args, err := psql.
		Select(itemColumns...).
		From("items").
		Where(sq.Eq{"user_id": userID}).
		OrderBy("updated_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("ошибка построения запроса: %w", err)
	}

	rows, err := s.pool.Query(ctx, query, args...)
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

	query, args, err := psql.
		Update("items").
		Set("data_type", item.DataType).
		Set("encrypted_data", item.EncryptedData).
		Set("metadata", meta).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": item.ID, "user_id": item.UserID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("ошибка построения запроса: %w", err)
	}

	ct, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ошибка обновления элемента: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return ErrItemNotFound
	}
	return nil
}

// DeleteItem удаляет элемент данных.
func (s *Storage) DeleteItem(ctx context.Context, id, userID string) error {
	query, args, err := psql.
		Delete("items").
		Where(sq.Eq{"id": id, "user_id": userID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("ошибка построения запроса: %w", err)
	}

	ct, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ошибка удаления элемента: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return ErrItemNotFound
	}
	return nil
}

// GetItemsUpdatedAfter возвращает элементы, обновлённые после указанного времени.
func (s *Storage) GetItemsUpdatedAfter(ctx context.Context, userID string, after time.Time) ([]*models.Item, error) {
	query, args, err := psql.
		Select(itemColumns...).
		From("items").
		Where(sq.Eq{"user_id": userID}).
		Where(sq.Gt{"updated_at": after}).
		OrderBy("updated_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("ошибка построения запроса: %w", err)
	}

	rows, err := s.pool.Query(ctx, query, args...)
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
