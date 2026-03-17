package handler

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/guldana/gophKeeperr/internal/models"
	"github.com/guldana/gophKeeperr/internal/server/service"
	"github.com/guldana/gophKeeperr/internal/server/storage"
	pb "github.com/guldana/gophKeeperr/proto"
)

// ServiceProvider определяет интерфейс бизнес-логики для хендлера.
type ServiceProvider interface {
	Register(ctx context.Context, login, password string) (string, error)
	Login(ctx context.Context, login, password string) (string, error)
	CreateItem(ctx context.Context, item *models.Item) (*models.Item, error)
	GetItem(ctx context.Context, id, userID string) (*models.Item, error)
	ListItems(ctx context.Context, userID string) ([]*models.Item, error)
	UpdateItem(ctx context.Context, item *models.Item) error
	DeleteItem(ctx context.Context, id, userID string) error
	Sync(ctx context.Context, userID string, clientItems []*models.Item, lastSync time.Time) ([]*models.Item, []string, error)
}

// GophKeeperHandler реализует gRPC сервис GophKeeper.
type GophKeeperHandler struct {
	pb.UnimplementedGophKeeperServer
	service ServiceProvider
}

// NewGophKeeperHandler создаёт новый обработчик gRPC запросов.
func NewGophKeeperHandler(svc ServiceProvider) *GophKeeperHandler {
	return &GophKeeperHandler{service: svc}
}

// Register обрабатывает запрос на регистрацию пользователя.
func (h *GophKeeperHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Login == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "логин и пароль обязательны")
	}

	token, err := h.service.Register(ctx, req.Login, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, "пользователь уже существует")
		}
		return nil, status.Error(codes.Internal, "ошибка регистрации")
	}

	return &pb.RegisterResponse{Token: token}, nil
}

// Login обрабатывает запрос на аутентификацию пользователя.
func (h *GophKeeperHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Login == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "логин и пароль обязательны")
	}

	token, err := h.service.Login(ctx, req.Login, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "неверные учётные данные")
		}
		return nil, status.Error(codes.Internal, "ошибка входа")
	}

	return &pb.LoginResponse{Token: token}, nil
}

// CreateItem обрабатывает запрос на создание элемента данных.
func (h *GophKeeperHandler) CreateItem(ctx context.Context, req *pb.CreateItemRequest) (*pb.CreateItemResponse, error) {
	userID := UserIDFromContext(ctx)

	item := protoToItem(req.Item)
	item.UserID = userID

	created, err := h.service.CreateItem(ctx, item)
	if err != nil {
		return nil, status.Error(codes.Internal, "ошибка создания элемента")
	}

	return &pb.CreateItemResponse{Id: created.ID}, nil
}

// GetItem обрабатывает запрос на получение элемента данных.
func (h *GophKeeperHandler) GetItem(ctx context.Context, req *pb.GetItemRequest) (*pb.GetItemResponse, error) {
	userID := UserIDFromContext(ctx)

	item, err := h.service.GetItem(ctx, req.Id, userID)
	if err != nil {
		if errors.Is(err, storage.ErrItemNotFound) {
			return nil, status.Error(codes.NotFound, "элемент не найден")
		}
		return nil, status.Error(codes.Internal, "ошибка получения элемента")
	}

	return &pb.GetItemResponse{Item: itemToProto(item)}, nil
}

// ListItems обрабатывает запрос на получение списка всех элементов пользователя.
func (h *GophKeeperHandler) ListItems(ctx context.Context, _ *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	userID := UserIDFromContext(ctx)

	items, err := h.service.ListItems(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "ошибка получения списка элементов")
	}

	pbItems := make([]*pb.Item, len(items))
	for i, item := range items {
		pbItems[i] = itemToProto(item)
	}

	return &pb.ListItemsResponse{Items: pbItems}, nil
}

// UpdateItem обрабатывает запрос на обновление элемента данных.
func (h *GophKeeperHandler) UpdateItem(ctx context.Context, req *pb.UpdateItemRequest) (*pb.UpdateItemResponse, error) {
	userID := UserIDFromContext(ctx)

	item := protoToItem(req.Item)
	item.UserID = userID

	if err := h.service.UpdateItem(ctx, item); err != nil {
		if errors.Is(err, storage.ErrItemNotFound) {
			return nil, status.Error(codes.NotFound, "элемент не найден")
		}
		return nil, status.Error(codes.Internal, "ошибка обновления элемента")
	}

	return &pb.UpdateItemResponse{}, nil
}

// DeleteItem обрабатывает запрос на удаление элемента данных.
func (h *GophKeeperHandler) DeleteItem(ctx context.Context, req *pb.DeleteItemRequest) (*pb.DeleteItemResponse, error) {
	userID := UserIDFromContext(ctx)

	if err := h.service.DeleteItem(ctx, req.Id, userID); err != nil {
		if errors.Is(err, storage.ErrItemNotFound) {
			return nil, status.Error(codes.NotFound, "элемент не найден")
		}
		return nil, status.Error(codes.Internal, "ошибка удаления элемента")
	}

	return &pb.DeleteItemResponse{}, nil
}

// SyncItems обрабатывает запрос на синхронизацию данных.
func (h *GophKeeperHandler) SyncItems(ctx context.Context, req *pb.SyncRequest) (*pb.SyncResponse, error) {
	userID := UserIDFromContext(ctx)

	clientItems := make([]*models.Item, len(req.Items))
	for i, pbItem := range req.Items {
		clientItems[i] = protoToItem(pbItem)
	}

	lastSync := req.LastSyncTime.AsTime()

	updated, deletedIDs, err := h.service.Sync(ctx, userID, clientItems, lastSync)
	if err != nil {
		return nil, status.Error(codes.Internal, "ошибка синхронизации")
	}

	pbItems := make([]*pb.Item, len(updated))
	for i, item := range updated {
		pbItems[i] = itemToProto(item)
	}

	return &pb.SyncResponse{UpdatedItems: pbItems, DeletedIds: deletedIDs}, nil
}

func itemToProto(item *models.Item) *pb.Item {
	return &pb.Item{
		Id:            item.ID,
		DataType:      pb.DataType(item.DataType),
		EncryptedData: item.EncryptedData,
		Metadata:      item.Metadata,
		CreatedAt:     timestamppb.New(item.CreatedAt),
		UpdatedAt:     timestamppb.New(item.UpdatedAt),
	}
}

func protoToItem(pbItem *pb.Item) *models.Item {
	item := &models.Item{
		ID:            pbItem.Id,
		DataType:      models.DataType(pbItem.DataType),
		EncryptedData: pbItem.EncryptedData,
		Metadata:      pbItem.Metadata,
	}
	if pbItem.CreatedAt != nil {
		item.CreatedAt = pbItem.CreatedAt.AsTime()
	}
	if pbItem.UpdatedAt != nil {
		item.UpdatedAt = pbItem.UpdatedAt.AsTime()
	}
	return item
}
