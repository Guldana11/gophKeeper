// Package main — точка входа серверного приложения GophKeeper.
package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

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
	tlsCert := flag.String("tls-cert", "", "path to TLS certificate file")
	tlsKey := flag.String("tls-key", "", "path to TLS private key file")
	flag.Parse()

	store, err := storage.New(*dsn)
	if err != nil {
		log.Fatalf("Ошибка инициализации хранилища: %v", err)
	}
	defer store.Close()

	authManager := auth.NewManager(*jwtSecret, 24*time.Hour)
	svc := service.New(store, authManager)
	h := handler.NewGophKeeperHandler(svc)

	var opts []grpc.ServerOption
	opts = append(opts, grpc.UnaryInterceptor(handler.AuthInterceptor(authManager)))

	if *tlsCert != "" && *tlsKey != "" {
		cert, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
		if err != nil {
			log.Fatalf("Ошибка загрузки TLS сертификата: %v", err)
		}
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsCfg)))
		log.Println("TLS включён")
	} else {
		log.Println("ВНИМАНИЕ: TLS отключён, соединение не защищено")
	}

	srv := grpc.NewServer(opts...)
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
