// Package handler реализует gRPC хендлеры сервера GophKeeper.
package handler

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/guldana/gophKeeperr/internal/server/auth"
)

type contextKey string

const userIDKey contextKey = "user_id"

// Методы, не требующие аутентификации.
var publicMethods = map[string]bool{
	"/gophkeeper.GophKeeper/Register": true,
	"/gophkeeper.GophKeeper/Login":    true,
}

// AuthInterceptor создаёт gRPC перехватчик для проверки JWT токена.
func AuthInterceptor(authManager *auth.Manager) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "отсутствуют метаданные")
		}

		values := md.Get("authorization")
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "отсутствует токен")
		}

		token := strings.TrimPrefix(values[0], "Bearer ")
		userID, err := authManager.ValidateToken(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "невалидный токен")
		}

		ctx = context.WithValue(ctx, userIDKey, userID)
		return handler(ctx, req)
	}
}

// UserIDFromContext извлекает ID пользователя из контекста.
func UserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}
