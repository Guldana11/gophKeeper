// Package main — точка входа серверного приложения GophKeeper.
package main

import (
	"flag"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"

	"github.com/guldana/gophKeeperr/internal/server/auth"
	"github.com/guldana/gophKeeperr/internal/server/handler"
	"github.com/guldana/gophKeeperr/internal/server/service"
	"github.com/guldana/gophKeeperr/internal/server/storage"
	pb "github.com/guldana/gophKeeperr/proto"
)

func main() {
	addr := flag.String("addr", ":3200", "server address")
	dsn := flag.String("dsn", "postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable", "database DSN")
	jwtSecret := flag.String("jwt-secret", "supersecretkey", "JWT secret key")
	flag.Parse()

	store, err := storage.New(*dsn)
	if err != nil {
		log.Fatalf("Ошибка инициализации хранилища: %v", err)
	}
	defer store.Close()

	authManager := auth.NewManager(*jwtSecret, 24*time.Hour)
	svc := service.New(store, authManager)
	h := handler.NewGophKeeperHandler(svc)

	srv := grpc.NewServer(
		grpc.UnaryInterceptor(handler.AuthInterceptor(authManager)),
	)
	pb.RegisterGophKeeperServer(srv, h)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("Ошибка прослушивания порта: %v", err)
	}

	log.Printf("Сервер GophKeeper запущен на %s", *addr)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
