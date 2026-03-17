package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/guldana/gophKeeperr/internal/models"
	"github.com/guldana/gophKeeperr/internal/server/service"
	"github.com/guldana/gophKeeperr/internal/server/storage"
	pb "github.com/guldana/gophKeeperr/proto"
)

// --- Мок сервиса ---

type mockService struct {
	registerFunc   func(ctx context.Context, login, password string) (string, error)
	loginFunc      func(ctx context.Context, login, password string) (string, error)
	createItemFunc func(ctx context.Context, item *models.Item) (*models.Item, error)
	getItemFunc    func(ctx context.Context, id, userID string) (*models.Item, error)
	listItemsFunc  func(ctx context.Context, userID string) ([]*models.Item, error)
	updateItemFunc func(ctx context.Context, item *models.Item) error
	deleteItemFunc func(ctx context.Context, id, userID string) error
	syncFunc       func(ctx context.Context, userID string, items []*models.Item, lastSync time.Time) ([]*models.Item, []string, error)
}

func (m *mockService) Register(ctx context.Context, login, password string) (string, error) {
	return m.registerFunc(ctx, login, password)
}
func (m *mockService) Login(ctx context.Context, login, password string) (string, error) {
	return m.loginFunc(ctx, login, password)
}
func (m *mockService) CreateItem(ctx context.Context, item *models.Item) (*models.Item, error) {
	return m.createItemFunc(ctx, item)
}
func (m *mockService) GetItem(ctx context.Context, id, userID string) (*models.Item, error) {
	return m.getItemFunc(ctx, id, userID)
}
func (m *mockService) ListItems(ctx context.Context, userID string) ([]*models.Item, error) {
	return m.listItemsFunc(ctx, userID)
}
func (m *mockService) UpdateItem(ctx context.Context, item *models.Item) error {
	return m.updateItemFunc(ctx, item)
}
func (m *mockService) DeleteItem(ctx context.Context, id, userID string) error {
	return m.deleteItemFunc(ctx, id, userID)
}
func (m *mockService) Sync(ctx context.Context, userID string, items []*models.Item, lastSync time.Time) ([]*models.Item, []string, error) {
	return m.syncFunc(ctx, userID, items, lastSync)
}

func ctxWithUserID(userID string) context.Context {
	return context.WithValue(context.Background(), userIDKey, userID)
}

// --- Тесты ---

func TestHandler_Register_Success(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		registerFunc: func(_ context.Context, _, _ string) (string, error) {
			return "token123", nil
		},
	})

	resp, err := h.Register(context.Background(), &pb.RegisterRequest{Login: "alice", Password: "pass"})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if resp.Token != "token123" {
		t.Errorf("Register() token = %v, want token123", resp.Token)
	}
}

func TestHandler_Register_EmptyFields(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{})

	_, err := h.Register(context.Background(), &pb.RegisterRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("Register() code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestHandler_Register_UserExists(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		registerFunc: func(_ context.Context, _, _ string) (string, error) {
			return "", service.ErrUserExists
		},
	})

	_, err := h.Register(context.Background(), &pb.RegisterRequest{Login: "alice", Password: "pass"})
	if status.Code(err) != codes.AlreadyExists {
		t.Errorf("Register() code = %v, want AlreadyExists", status.Code(err))
	}
}

func TestHandler_Login_Success(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		loginFunc: func(_ context.Context, _, _ string) (string, error) {
			return "token123", nil
		},
	})

	resp, err := h.Login(context.Background(), &pb.LoginRequest{Login: "alice", Password: "pass"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if resp.Token != "token123" {
		t.Errorf("Login() token = %v, want token123", resp.Token)
	}
}

func TestHandler_Login_EmptyFields(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{})

	_, err := h.Login(context.Background(), &pb.LoginRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("Login() code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		loginFunc: func(_ context.Context, _, _ string) (string, error) {
			return "", service.ErrInvalidCredentials
		},
	})

	_, err := h.Login(context.Background(), &pb.LoginRequest{Login: "alice", Password: "wrong"})
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("Login() code = %v, want Unauthenticated", status.Code(err))
	}
}

func TestHandler_CreateItem_Success(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		createItemFunc: func(_ context.Context, item *models.Item) (*models.Item, error) {
			item.ID = "item-123"
			return item, nil
		},
	})

	ctx := ctxWithUserID("user-1")
	resp, err := h.CreateItem(ctx, &pb.CreateItemRequest{
		Item: &pb.Item{DataType: pb.DataType_DATA_TYPE_CREDENTIAL, EncryptedData: []byte("data")},
	})
	if err != nil {
		t.Fatalf("CreateItem() error = %v", err)
	}
	if resp.Id != "item-123" {
		t.Errorf("CreateItem() id = %v, want item-123", resp.Id)
	}
}

func TestHandler_GetItem_Success(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		getItemFunc: func(_ context.Context, id, _ string) (*models.Item, error) {
			return &models.Item{
				ID:            id,
				DataType:      models.DataTypeText,
				EncryptedData: []byte("data"),
				Metadata:      map[string]string{"key": "value"},
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}, nil
		},
	})

	ctx := ctxWithUserID("user-1")
	resp, err := h.GetItem(ctx, &pb.GetItemRequest{Id: "item-1"})
	if err != nil {
		t.Fatalf("GetItem() error = %v", err)
	}
	if resp.Item.Id != "item-1" {
		t.Errorf("GetItem() id = %v, want item-1", resp.Item.Id)
	}
}

func TestHandler_GetItem_NotFound(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		getItemFunc: func(_ context.Context, _, _ string) (*models.Item, error) {
			return nil, storage.ErrItemNotFound
		},
	})

	ctx := ctxWithUserID("user-1")
	_, err := h.GetItem(ctx, &pb.GetItemRequest{Id: "nonexistent"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("GetItem() code = %v, want NotFound", status.Code(err))
	}
}

func TestHandler_ListItems_Success(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		listItemsFunc: func(_ context.Context, _ string) ([]*models.Item, error) {
			return []*models.Item{
				{ID: "1", DataType: models.DataTypeText, EncryptedData: []byte("a"), Metadata: map[string]string{}},
				{ID: "2", DataType: models.DataTypeBankCard, EncryptedData: []byte("b"), Metadata: map[string]string{}},
			}, nil
		},
	})

	ctx := ctxWithUserID("user-1")
	resp, err := h.ListItems(ctx, &pb.ListItemsRequest{})
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}
	if len(resp.Items) != 2 {
		t.Errorf("ListItems() count = %d, want 2", len(resp.Items))
	}
}

func TestHandler_UpdateItem_Success(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		updateItemFunc: func(_ context.Context, _ *models.Item) error {
			return nil
		},
	})

	ctx := ctxWithUserID("user-1")
	_, err := h.UpdateItem(ctx, &pb.UpdateItemRequest{
		Item: &pb.Item{Id: "item-1", DataType: pb.DataType_DATA_TYPE_TEXT, EncryptedData: []byte("new")},
	})
	if err != nil {
		t.Fatalf("UpdateItem() error = %v", err)
	}
}

func TestHandler_UpdateItem_NotFound(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		updateItemFunc: func(_ context.Context, _ *models.Item) error {
			return storage.ErrItemNotFound
		},
	})

	ctx := ctxWithUserID("user-1")
	_, err := h.UpdateItem(ctx, &pb.UpdateItemRequest{
		Item: &pb.Item{Id: "nonexistent", EncryptedData: []byte("data")},
	})
	if status.Code(err) != codes.NotFound {
		t.Errorf("UpdateItem() code = %v, want NotFound", status.Code(err))
	}
}

func TestHandler_DeleteItem_Success(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		deleteItemFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
	})

	ctx := ctxWithUserID("user-1")
	_, err := h.DeleteItem(ctx, &pb.DeleteItemRequest{Id: "item-1"})
	if err != nil {
		t.Fatalf("DeleteItem() error = %v", err)
	}
}

func TestHandler_DeleteItem_NotFound(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		deleteItemFunc: func(_ context.Context, _, _ string) error {
			return storage.ErrItemNotFound
		},
	})

	ctx := ctxWithUserID("user-1")
	_, err := h.DeleteItem(ctx, &pb.DeleteItemRequest{Id: "nonexistent"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("DeleteItem() code = %v, want NotFound", status.Code(err))
	}
}

func TestHandler_SyncItems_Success(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		syncFunc: func(_ context.Context, _ string, _ []*models.Item, _ time.Time) ([]*models.Item, []string, error) {
			return []*models.Item{
				{ID: "item-1", DataType: models.DataTypeText, EncryptedData: []byte("data"), Metadata: map[string]string{}},
			}, []string{"deleted-1", "deleted-2"}, nil
		},
	})

	ctx := ctxWithUserID("user-1")
	resp, err := h.SyncItems(ctx, &pb.SyncRequest{
		Items:        []*pb.Item{{Id: "item-1", EncryptedData: []byte("data")}},
		LastSyncTime: timestamppb.Now(),
	})
	if err != nil {
		t.Fatalf("SyncItems() error = %v", err)
	}
	if len(resp.UpdatedItems) != 1 {
		t.Errorf("SyncItems() updated count = %d, want 1", len(resp.UpdatedItems))
	}
	if len(resp.DeletedIds) != 2 {
		t.Errorf("SyncItems() deleted count = %d, want 2", len(resp.DeletedIds))
	}
}

func TestHandler_SyncItems_Error(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		syncFunc: func(_ context.Context, _ string, _ []*models.Item, _ time.Time) ([]*models.Item, []string, error) {
			return nil, nil, errors.New("sync failed")
		},
	})

	ctx := ctxWithUserID("user-1")
	_, err := h.SyncItems(ctx, &pb.SyncRequest{LastSyncTime: timestamppb.Now()})
	if status.Code(err) != codes.Internal {
		t.Errorf("SyncItems() code = %v, want Internal", status.Code(err))
	}
}

func TestHandler_CreateItem_InternalError(t *testing.T) {
	h := NewGophKeeperHandler(&mockService{
		createItemFunc: func(_ context.Context, _ *models.Item) (*models.Item, error) {
			return nil, errors.New("db error")
		},
	})

	ctx := ctxWithUserID("user-1")
	_, err := h.CreateItem(ctx, &pb.CreateItemRequest{
		Item: &pb.Item{EncryptedData: []byte("data")},
	})
	if status.Code(err) != codes.Internal {
		t.Errorf("CreateItem() code = %v, want Internal", status.Code(err))
	}
}
