package storage

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/guldana/gophKeeperr/internal/models"
)

func setupTestDB(t *testing.T) *Storage {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("gophkeeper_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("не удалось запустить контейнер: %v", err)
	}
	t.Cleanup(func() { pgContainer.Terminate(ctx) })

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("не удалось получить строку подключения: %v", err)
	}

	// Миграции читаются относительно рабочей директории — переключаемся в корень проекта.
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	t.Chdir(projectRoot)

	store, err := New(connStr)
	if err != nil {
		t.Fatalf("не удалось инициализировать хранилище: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	return store
}

func TestStorage_CreateAndGetUser(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	user, err := store.CreateUser(ctx, "alice", "hashed_password")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if user.ID == "" {
		t.Fatal("CreateUser() returned empty ID")
	}
	if user.Login != "alice" {
		t.Errorf("CreateUser() login = %q, want alice", user.Login)
	}

	got, err := store.GetUserByLogin(ctx, "alice")
	if err != nil {
		t.Fatalf("GetUserByLogin() error = %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("GetUserByLogin() ID = %q, want %q", got.ID, user.ID)
	}
}

func TestStorage_CreateUser_Duplicate(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	_, err := store.CreateUser(ctx, "alice", "hash")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	_, err = store.CreateUser(ctx, "alice", "hash2")
	if err != ErrUserExists {
		t.Errorf("CreateUser() duplicate error = %v, want ErrUserExists", err)
	}
}

func TestStorage_GetUserByLogin_NotFound(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	_, err := store.GetUserByLogin(ctx, "nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("GetUserByLogin() error = %v, want ErrUserNotFound", err)
	}
}

func TestStorage_CRUD_Items(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	user, _ := store.CreateUser(ctx, "bob", "hash")

	// Create
	item := &models.Item{
		UserID:        user.ID,
		DataType:      models.DataTypeCredential,
		EncryptedData: []byte("encrypted-login-pass"),
		Metadata:      map[string]string{"site": "example.com"},
	}
	created, err := store.CreateItem(ctx, item)
	if err != nil {
		t.Fatalf("CreateItem() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("CreateItem() returned empty ID")
	}

	// Get
	got, err := store.GetItem(ctx, created.ID, user.ID)
	if err != nil {
		t.Fatalf("GetItem() error = %v", err)
	}
	if string(got.EncryptedData) != "encrypted-login-pass" {
		t.Errorf("GetItem() data = %q, want encrypted-login-pass", got.EncryptedData)
	}
	if got.Metadata["site"] != "example.com" {
		t.Errorf("GetItem() metadata = %v", got.Metadata)
	}

	// List
	items, err := store.ListItems(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("ListItems() count = %d, want 1", len(items))
	}

	// Update
	created.EncryptedData = []byte("updated-data")
	created.Metadata = map[string]string{"site": "new.com"}
	if err := store.UpdateItem(ctx, created); err != nil {
		t.Fatalf("UpdateItem() error = %v", err)
	}

	updated, _ := store.GetItem(ctx, created.ID, user.ID)
	if string(updated.EncryptedData) != "updated-data" {
		t.Errorf("after UpdateItem() data = %q, want updated-data", updated.EncryptedData)
	}

	// Delete
	if err := store.DeleteItem(ctx, created.ID, user.ID); err != nil {
		t.Fatalf("DeleteItem() error = %v", err)
	}

	_, err = store.GetItem(ctx, created.ID, user.ID)
	if err != ErrItemNotFound {
		t.Errorf("after DeleteItem() GetItem error = %v, want ErrItemNotFound", err)
	}
}

func TestStorage_GetItem_NotFound(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	_, err := store.GetItem(ctx, "00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000001")
	if err != ErrItemNotFound {
		t.Errorf("GetItem() error = %v, want ErrItemNotFound", err)
	}
}

func TestStorage_UpdateItem_NotFound(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	err := store.UpdateItem(ctx, &models.Item{
		ID:            "00000000-0000-0000-0000-000000000000",
		UserID:        "00000000-0000-0000-0000-000000000001",
		EncryptedData: []byte("data"),
		Metadata:      map[string]string{},
	})
	if err != ErrItemNotFound {
		t.Errorf("UpdateItem() error = %v, want ErrItemNotFound", err)
	}
}

func TestStorage_DeleteItem_NotFound(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	err := store.DeleteItem(ctx, "00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000001")
	if err != ErrItemNotFound {
		t.Errorf("DeleteItem() error = %v, want ErrItemNotFound", err)
	}
}

func TestStorage_GetItemsUpdatedAfter(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	user, _ := store.CreateUser(ctx, "carol", "hash")

	before := time.Now().Add(-time.Second)

	item := &models.Item{
		UserID:        user.ID,
		DataType:      models.DataTypeText,
		EncryptedData: []byte("text"),
		Metadata:      map[string]string{},
	}
	store.CreateItem(ctx, item)

	items, err := store.GetItemsUpdatedAfter(ctx, user.ID, before)
	if err != nil {
		t.Fatalf("GetItemsUpdatedAfter() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("GetItemsUpdatedAfter() count = %d, want 1", len(items))
	}

	future := time.Now().Add(time.Hour)
	items, err = store.GetItemsUpdatedAfter(ctx, user.ID, future)
	if err != nil {
		t.Fatalf("GetItemsUpdatedAfter() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("GetItemsUpdatedAfter() future count = %d, want 0", len(items))
	}
}

func TestStorage_GetDeletedIDsAfter(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	user, _ := store.CreateUser(ctx, "dave", "hash")

	item := &models.Item{
		UserID:        user.ID,
		DataType:      models.DataTypeBinary,
		EncryptedData: []byte("binary"),
		Metadata:      map[string]string{},
	}
	created, _ := store.CreateItem(ctx, item)

	before := time.Now().Add(-time.Second)
	store.DeleteItem(ctx, created.ID, user.ID)

	ids, err := store.GetDeletedIDsAfter(ctx, user.ID, before)
	if err != nil {
		t.Fatalf("GetDeletedIDsAfter() error = %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("GetDeletedIDsAfter() count = %d, want 1", len(ids))
	}
	if ids[0] != created.ID {
		t.Errorf("GetDeletedIDsAfter() id = %q, want %q", ids[0], created.ID)
	}
}
