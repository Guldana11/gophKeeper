package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/guldana/gophKeeper/internal/models"
	"github.com/guldana/gophKeeper/internal/server/storage"
)

// --- Моки ---

type mockStorage struct {
	users map[string]*models.User
	items map[string]*models.Item
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		users: make(map[string]*models.User),
		items: make(map[string]*models.Item),
	}
}

func (m *mockStorage) CreateUser(_ context.Context, login, passwordHash string) (*models.User, error) {
	if _, exists := m.users[login]; exists {
		return nil, storage.ErrUserExists
	}
	user := &models.User{
		ID:           "user-" + login,
		Login:        login,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
	m.users[login] = user
	return user, nil
}

func (m *mockStorage) GetUserByLogin(_ context.Context, login string) (*models.User, error) {
	user, ok := m.users[login]
	if !ok {
		return nil, storage.ErrUserNotFound
	}
	return user, nil
}

func (m *mockStorage) CreateItem(_ context.Context, item *models.Item) (*models.Item, error) {
	item.ID = "item-" + item.UserID
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()
	m.items[item.ID] = item
	return item, nil
}

func (m *mockStorage) GetItem(_ context.Context, id, userID string) (*models.Item, error) {
	item, ok := m.items[id]
	if !ok || item.UserID != userID {
		return nil, storage.ErrItemNotFound
	}
	return item, nil
}

func (m *mockStorage) ListItems(_ context.Context, userID string) ([]*models.Item, error) {
	var result []*models.Item
	for _, item := range m.items {
		if item.UserID == userID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (m *mockStorage) UpdateItem(_ context.Context, item *models.Item) error {
	existing, ok := m.items[item.ID]
	if !ok || existing.UserID != item.UserID {
		return storage.ErrItemNotFound
	}
	item.UpdatedAt = time.Now()
	m.items[item.ID] = item
	return nil
}

func (m *mockStorage) DeleteItem(_ context.Context, id, userID string) error {
	item, ok := m.items[id]
	if !ok || item.UserID != userID {
		return storage.ErrItemNotFound
	}
	delete(m.items, id)
	return nil
}

func (m *mockStorage) GetItemsUpdatedAfter(_ context.Context, userID string, after time.Time) ([]*models.Item, error) {
	var result []*models.Item
	for _, item := range m.items {
		if item.UserID == userID && item.UpdatedAt.After(after) {
			result = append(result, item)
		}
	}
	return result, nil
}

type mockAuth struct{}

func (m *mockAuth) HashPassword(password string) (string, error) {
	return "hashed_" + password, nil
}

func (m *mockAuth) CheckPassword(password, hash string) bool {
	return "hashed_"+password == hash
}

func (m *mockAuth) GenerateToken(userID string) (string, error) {
	return "token_" + userID, nil
}

// --- Тесты ---

func TestRegister_Success(t *testing.T) {
	svc := New(newMockStorage(), &mockAuth{})

	token, err := svc.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if token != "token_user-alice" {
		t.Errorf("Register() token = %v, want token_user-alice", token)
	}
}

func TestRegister_UserExists(t *testing.T) {
	store := newMockStorage()
	svc := New(store, &mockAuth{})

	_, _ = svc.Register(context.Background(), "alice", "password123")

	_, err := svc.Register(context.Background(), "alice", "password123")
	if !errors.Is(err, ErrUserExists) {
		t.Errorf("Register() error = %v, want ErrUserExists", err)
	}
}

func TestLogin_Success(t *testing.T) {
	store := newMockStorage()
	svc := New(store, &mockAuth{})

	_, _ = svc.Register(context.Background(), "alice", "password123")

	token, err := svc.Login(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if token != "token_user-alice" {
		t.Errorf("Login() token = %v, want token_user-alice", token)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	store := newMockStorage()
	svc := New(store, &mockAuth{})

	_, _ = svc.Register(context.Background(), "alice", "password123")

	_, err := svc.Login(context.Background(), "alice", "wrongpassword")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	svc := New(newMockStorage(), &mockAuth{})

	_, err := svc.Login(context.Background(), "bob", "password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestCreateItem(t *testing.T) {
	svc := New(newMockStorage(), &mockAuth{})

	item := &models.Item{
		UserID:        "user-1",
		DataType:      models.DataTypeCredential,
		EncryptedData: []byte("encrypted"),
		Metadata:      map[string]string{"site": "example.com"},
	}

	created, err := svc.CreateItem(context.Background(), item)
	if err != nil {
		t.Fatalf("CreateItem() error = %v", err)
	}
	if created.ID == "" {
		t.Error("CreateItem() returned item with empty ID")
	}
}

func TestGetItem(t *testing.T) {
	store := newMockStorage()
	svc := New(store, &mockAuth{})

	item := &models.Item{
		UserID:        "user-1",
		DataType:      models.DataTypeText,
		EncryptedData: []byte("data"),
	}
	created, _ := svc.CreateItem(context.Background(), item)

	got, err := svc.GetItem(context.Background(), created.ID, "user-1")
	if err != nil {
		t.Fatalf("GetItem() error = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("GetItem() ID = %v, want %v", got.ID, created.ID)
	}
}

func TestGetItem_NotFound(t *testing.T) {
	svc := New(newMockStorage(), &mockAuth{})

	_, err := svc.GetItem(context.Background(), "nonexistent", "user-1")
	if !errors.Is(err, storage.ErrItemNotFound) {
		t.Errorf("GetItem() error = %v, want ErrItemNotFound", err)
	}
}

func TestListItems(t *testing.T) {
	store := newMockStorage()
	svc := New(store, &mockAuth{})

	item := &models.Item{UserID: "user-1", DataType: models.DataTypeText, EncryptedData: []byte("data")}
	_, _ = svc.CreateItem(context.Background(), item)

	items, err := svc.ListItems(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("ListItems() count = %d, want 1", len(items))
	}
}

func TestUpdateItem(t *testing.T) {
	store := newMockStorage()
	svc := New(store, &mockAuth{})

	item := &models.Item{UserID: "user-1", DataType: models.DataTypeText, EncryptedData: []byte("old")}
	created, _ := svc.CreateItem(context.Background(), item)

	created.EncryptedData = []byte("new")
	err := svc.UpdateItem(context.Background(), created)
	if err != nil {
		t.Fatalf("UpdateItem() error = %v", err)
	}
}

func TestUpdateItem_NotFound(t *testing.T) {
	svc := New(newMockStorage(), &mockAuth{})

	err := svc.UpdateItem(context.Background(), &models.Item{ID: "nonexistent", UserID: "user-1"})
	if !errors.Is(err, storage.ErrItemNotFound) {
		t.Errorf("UpdateItem() error = %v, want ErrItemNotFound", err)
	}
}

func TestDeleteItem(t *testing.T) {
	store := newMockStorage()
	svc := New(store, &mockAuth{})

	item := &models.Item{UserID: "user-1", DataType: models.DataTypeText, EncryptedData: []byte("data")}
	created, _ := svc.CreateItem(context.Background(), item)

	err := svc.DeleteItem(context.Background(), created.ID, "user-1")
	if err != nil {
		t.Fatalf("DeleteItem() error = %v", err)
	}

	_, err = svc.GetItem(context.Background(), created.ID, "user-1")
	if !errors.Is(err, storage.ErrItemNotFound) {
		t.Error("DeleteItem() item should be deleted")
	}
}

func TestDeleteItem_NotFound(t *testing.T) {
	svc := New(newMockStorage(), &mockAuth{})

	err := svc.DeleteItem(context.Background(), "nonexistent", "user-1")
	if !errors.Is(err, storage.ErrItemNotFound) {
		t.Errorf("DeleteItem() error = %v, want ErrItemNotFound", err)
	}
}

func TestSync_NewItem(t *testing.T) {
	store := newMockStorage()
	svc := New(store, &mockAuth{})

	clientItems := []*models.Item{
		{
			ID:            "client-item-1",
			DataType:      models.DataTypeCredential,
			EncryptedData: []byte("data"),
			UpdatedAt:     time.Now(),
		},
	}

	updated, err := svc.Sync(context.Background(), "user-1", clientItems, time.Time{})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if len(updated) == 0 {
		t.Error("Sync() should return updated items")
	}
}

func TestSync_LastWriteWins(t *testing.T) {
	store := newMockStorage()
	svc := New(store, &mockAuth{})

	// Создаём элемент на сервере
	serverItem := &models.Item{
		ID:            "item-1",
		UserID:        "user-1",
		DataType:      models.DataTypeText,
		EncryptedData: []byte("server-data"),
		UpdatedAt:     time.Now(),
	}
	store.items["item-1"] = serverItem

	// Клиент отправляет более новую версию
	clientItems := []*models.Item{
		{
			ID:            "item-1",
			DataType:      models.DataTypeText,
			EncryptedData: []byte("client-data"),
			UpdatedAt:     time.Now().Add(time.Hour),
		},
	}

	_, err := svc.Sync(context.Background(), "user-1", clientItems, time.Time{})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	// Проверяем что данные обновились
	item := store.items["item-1"]
	if string(item.EncryptedData) != "client-data" {
		t.Errorf("Sync() item data = %s, want client-data", item.EncryptedData)
	}
}
