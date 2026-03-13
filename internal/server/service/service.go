// Package service реализует бизнес-логику сервера GophKeeper.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/guldana/gophKeeper/internal/models"
	"github.com/guldana/gophKeeper/internal/server/storage"
)

// Ошибки сервиса.
var (
	ErrInvalidCredentials = errors.New("неверный логин или пароль")
	ErrUserExists         = errors.New("пользователь уже существует")
)

// StorageProvider определяет интерфейс для работы с хранилищем.
type StorageProvider interface {
	CreateUser(ctx context.Context, login, passwordHash string) (*models.User, error)
	GetUserByLogin(ctx context.Context, login string) (*models.User, error)
	CreateItem(ctx context.Context, item *models.Item) (*models.Item, error)
	GetItem(ctx context.Context, id, userID string) (*models.Item, error)
	ListItems(ctx context.Context, userID string) ([]*models.Item, error)
	UpdateItem(ctx context.Context, item *models.Item) error
	DeleteItem(ctx context.Context, id, userID string) error
	GetItemsUpdatedAfter(ctx context.Context, userID string, after time.Time) ([]*models.Item, error)
}

// AuthProvider определяет интерфейс для аутентификации.
type AuthProvider interface {
	HashPassword(password string) (string, error)
	CheckPassword(password, hash string) bool
	GenerateToken(userID string) (string, error)
}

// Service предоставляет бизнес-логику приложения.
type Service struct {
	storage StorageProvider
	auth    AuthProvider
}

// New создаёт новый экземпляр сервиса.
func New(storage StorageProvider, auth AuthProvider) *Service {
	return &Service{
		storage: storage,
		auth:    auth,
	}
}

// Register регистрирует нового пользователя и возвращает JWT токен.
func (s *Service) Register(ctx context.Context, login, password string) (string, error) {
	hash, err := s.auth.HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("ошибка хеширования пароля: %w", err)
	}

	user, err := s.storage.CreateUser(ctx, login, hash)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			return "", ErrUserExists
		}
		return "", fmt.Errorf("ошибка создания пользователя: %w", err)
	}

	token, err := s.auth.GenerateToken(user.ID)
	if err != nil {
		return "", fmt.Errorf("ошибка генерации токена: %w", err)
	}

	return token, nil
}

// Login аутентифицирует пользователя и возвращает JWT токен.
func (s *Service) Login(ctx context.Context, login, password string) (string, error) {
	user, err := s.storage.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("ошибка получения пользователя: %w", err)
	}

	if !s.auth.CheckPassword(password, user.PasswordHash) {
		return "", ErrInvalidCredentials
	}

	token, err := s.auth.GenerateToken(user.ID)
	if err != nil {
		return "", fmt.Errorf("ошибка генерации токена: %w", err)
	}

	return token, nil
}

// CreateItem создаёт новый элемент данных.
func (s *Service) CreateItem(ctx context.Context, item *models.Item) (*models.Item, error) {
	return s.storage.CreateItem(ctx, item)
}

// GetItem возвращает элемент данных по ID.
func (s *Service) GetItem(ctx context.Context, id, userID string) (*models.Item, error) {
	return s.storage.GetItem(ctx, id, userID)
}

// ListItems возвращает все элементы пользователя.
func (s *Service) ListItems(ctx context.Context, userID string) ([]*models.Item, error) {
	return s.storage.ListItems(ctx, userID)
}

// UpdateItem обновляет элемент данных.
func (s *Service) UpdateItem(ctx context.Context, item *models.Item) error {
	return s.storage.UpdateItem(ctx, item)
}

// DeleteItem удаляет элемент данных.
func (s *Service) DeleteItem(ctx context.Context, id, userID string) error {
	return s.storage.DeleteItem(ctx, id, userID)
}

// Sync синхронизирует данные: принимает изменения от клиента, возвращает серверные обновления.
func (s *Service) Sync(ctx context.Context, userID string, clientItems []*models.Item, lastSync time.Time) ([]*models.Item, error) {
	for _, item := range clientItems {
		item.UserID = userID
		existing, err := s.storage.GetItem(ctx, item.ID, userID)
		if err != nil {
			if errors.Is(err, storage.ErrItemNotFound) {
				if _, err := s.storage.CreateItem(ctx, item); err != nil {
					return nil, fmt.Errorf("ошибка создания элемента при синхронизации: %w", err)
				}
				continue
			}
			return nil, fmt.Errorf("ошибка получения элемента при синхронизации: %w", err)
		}

		// Last-write-wins: обновляем если клиентская версия новее
		if item.UpdatedAt.After(existing.UpdatedAt) {
			if err := s.storage.UpdateItem(ctx, item); err != nil {
				return nil, fmt.Errorf("ошибка обновления элемента при синхронизации: %w", err)
			}
		}
	}

	updated, err := s.storage.GetItemsUpdatedAfter(ctx, userID, lastSync)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения обновлённых элементов: %w", err)
	}

	return updated, nil
}
