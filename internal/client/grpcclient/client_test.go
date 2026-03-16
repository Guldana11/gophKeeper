package grpcclient

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/guldana/gophKeeper/proto"
)

// fakeServer реализует минимальный gRPC сервер для тестирования клиента.
type fakeServer struct {
	pb.UnimplementedGophKeeperServer
}

func (f *fakeServer) Register(_ context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Login == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "логин и пароль обязательны")
	}
	return &pb.RegisterResponse{Token: "fake-token-register"}, nil
}

func (f *fakeServer) Login(_ context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Login == "unknown" {
		return nil, status.Error(codes.Unauthenticated, "неверные учётные данные")
	}
	return &pb.LoginResponse{Token: "fake-token-login"}, nil
}

func (f *fakeServer) ListItems(_ context.Context, _ *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	return &pb.ListItemsResponse{
		Items: []*pb.Item{
			{Id: "item-1", DataType: pb.DataType_DATA_TYPE_TEXT},
			{Id: "item-2", DataType: pb.DataType_DATA_TYPE_CREDENTIAL},
		},
	}, nil
}

func (f *fakeServer) CreateItem(_ context.Context, req *pb.CreateItemRequest) (*pb.CreateItemResponse, error) {
	return &pb.CreateItemResponse{Id: "new-item-id"}, nil
}

func (f *fakeServer) DeleteItem(_ context.Context, req *pb.DeleteItemRequest) (*pb.DeleteItemResponse, error) {
	if req.Id == "not-found" {
		return nil, status.Error(codes.NotFound, "элемент не найден")
	}
	return &pb.DeleteItemResponse{}, nil
}

func startFakeServer(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("не удалось запустить слушатель: %v", err)
	}

	srv := grpc.NewServer()
	pb.RegisterGophKeeperServer(srv, &fakeServer{})

	go func() {
		if err := srv.Serve(lis); err != nil {
			// сервер остановлен
		}
	}()
	t.Cleanup(srv.GracefulStop)

	return lis.Addr().String()
}

func TestRegister(t *testing.T) {
	addr := startFakeServer(t)
	client, err := New(addr)
	if err != nil {
		t.Fatalf("New() вернул ошибку: %v", err)
	}
	defer client.Close()

	token, err := client.Register(context.Background(), "user", "pass")
	if err != nil {
		t.Fatalf("Register() вернул ошибку: %v", err)
	}
	if token != "fake-token-register" {
		t.Errorf("ожидался токен %q, получен %q", "fake-token-register", token)
	}
}

func TestLogin(t *testing.T) {
	addr := startFakeServer(t)
	client, err := New(addr)
	if err != nil {
		t.Fatalf("New() вернул ошибку: %v", err)
	}
	defer client.Close()

	token, err := client.Login(context.Background(), "user", "pass")
	if err != nil {
		t.Fatalf("Login() вернул ошибку: %v", err)
	}
	if token != "fake-token-login" {
		t.Errorf("ожидался токен %q, получен %q", "fake-token-login", token)
	}
}

func TestLoginFail(t *testing.T) {
	addr := startFakeServer(t)
	client, err := New(addr)
	if err != nil {
		t.Fatalf("New() вернул ошибку: %v", err)
	}
	defer client.Close()

	_, err = client.Login(context.Background(), "unknown", "pass")
	if err == nil {
		t.Fatal("ожидалась ошибка при неверном логине")
	}
}

func TestListItems(t *testing.T) {
	addr := startFakeServer(t)
	client, err := New(addr)
	if err != nil {
		t.Fatalf("New() вернул ошибку: %v", err)
	}
	defer client.Close()
	client.SetToken("fake-token")

	items, err := client.ListItems(context.Background())
	if err != nil {
		t.Fatalf("ListItems() вернул ошибку: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("ожидалось 2 элемента, получено %d", len(items))
	}
}

func TestCreateItem(t *testing.T) {
	addr := startFakeServer(t)
	client, err := New(addr)
	if err != nil {
		t.Fatalf("New() вернул ошибку: %v", err)
	}
	defer client.Close()
	client.SetToken("fake-token")

	id, err := client.CreateItem(context.Background(), &pb.Item{
		DataType:      pb.DataType_DATA_TYPE_TEXT,
		EncryptedData: []byte("test"),
	})
	if err != nil {
		t.Fatalf("CreateItem() вернул ошибку: %v", err)
	}
	if id != "new-item-id" {
		t.Errorf("ожидался ID %q, получен %q", "new-item-id", id)
	}
}

func TestDeleteItem(t *testing.T) {
	addr := startFakeServer(t)
	client, err := New(addr)
	if err != nil {
		t.Fatalf("New() вернул ошибку: %v", err)
	}
	defer client.Close()
	client.SetToken("fake-token")

	if err := client.DeleteItem(context.Background(), "item-1"); err != nil {
		t.Fatalf("DeleteItem() вернул ошибку: %v", err)
	}

	if err := client.DeleteItem(context.Background(), "not-found"); err == nil {
		t.Fatal("ожидалась ошибка при удалении несуществующего элемента")
	}
}
