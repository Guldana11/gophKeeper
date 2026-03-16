package handler

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/guldana/gophKeeperr/internal/server/auth"
)

func TestAuthInterceptor_PublicMethod(t *testing.T) {
	m := auth.NewManager("secret", time.Hour)
	interceptor := AuthInterceptor(m)

	called := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.GophKeeper/Register"}
	_, err := interceptor(context.Background(), nil, info, handler)
	if err != nil {
		t.Fatalf("interceptor error = %v", err)
	}
	if !called {
		t.Error("handler should be called for public method")
	}
}

func TestAuthInterceptor_NoMetadata(t *testing.T) {
	m := auth.NewManager("secret", time.Hour)
	interceptor := AuthInterceptor(m)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.GophKeeper/CreateItem"}
	_, err := interceptor(context.Background(), nil, info, handler)
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("code = %v, want Unauthenticated", status.Code(err))
	}
}

func TestAuthInterceptor_NoToken(t *testing.T) {
	m := auth.NewManager("secret", time.Hour)
	interceptor := AuthInterceptor(m)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.GophKeeper/CreateItem"}
	_, err := interceptor(ctx, nil, info, handler)
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("code = %v, want Unauthenticated", status.Code(err))
	}
}

func TestAuthInterceptor_InvalidToken(t *testing.T) {
	m := auth.NewManager("secret", time.Hour)
	interceptor := AuthInterceptor(m)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	md := metadata.Pairs("authorization", "Bearer invalid-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.GophKeeper/CreateItem"}
	_, err := interceptor(ctx, nil, info, handler)
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("code = %v, want Unauthenticated", status.Code(err))
	}
}

func TestAuthInterceptor_ValidToken(t *testing.T) {
	m := auth.NewManager("secret", time.Hour)
	interceptor := AuthInterceptor(m)

	token, _ := m.GenerateToken("user-123")

	var capturedUserID string
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		capturedUserID = UserIDFromContext(ctx)
		return "ok", nil
	}

	md := metadata.Pairs("authorization", "Bearer "+token)
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.GophKeeper/CreateItem"}
	_, err := interceptor(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("interceptor error = %v", err)
	}
	if capturedUserID != "user-123" {
		t.Errorf("userID = %v, want user-123", capturedUserID)
	}
}

func TestUserIDFromContext_Empty(t *testing.T) {
	id := UserIDFromContext(context.Background())
	if id != "" {
		t.Errorf("UserIDFromContext() = %v, want empty", id)
	}
}
