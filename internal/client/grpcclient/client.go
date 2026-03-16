// Package grpcclient реализует gRPC клиент для взаимодействия с сервером GophKeeper.
package grpcclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/guldana/gophKeeper/proto"
)

// Client предоставляет методы для взаимодействия с сервером GophKeeper.
type Client struct {
	conn    *grpc.ClientConn
	service pb.GophKeeperClient
	token   string
}

// New создаёт новый gRPC клиент и подключается к серверу.
func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к серверу: %w", err)
	}

	return &Client{
		conn:    conn,
		service: pb.NewGophKeeperClient(conn),
	}, nil
}

// Close закрывает соединение с сервером.
func (c *Client) Close() error {
	return c.conn.Close()
}

// SetToken устанавливает JWT токен для авторизованных запросов.
func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) authCtx(ctx context.Context) context.Context {
	if c.token == "" {
		return ctx
	}
	md := metadata.Pairs("authorization", "Bearer "+c.token)
	return metadata.NewOutgoingContext(ctx, md)
}

// Register регистрирует нового пользователя и возвращает JWT токен.
func (c *Client) Register(ctx context.Context, login, password string) (string, error) {
	resp, err := c.service.Register(ctx, &pb.RegisterRequest{
		Login:    login,
		Password: password,
	})
	if err != nil {
		return "", fmt.Errorf("ошибка регистрации: %w", err)
	}
	c.token = resp.Token
	return resp.Token, nil
}

// Login аутентифицирует пользователя и возвращает JWT токен.
func (c *Client) Login(ctx context.Context, login, password string) (string, error) {
	resp, err := c.service.Login(ctx, &pb.LoginRequest{
		Login:    login,
		Password: password,
	})
	if err != nil {
		return "", fmt.Errorf("ошибка входа: %w", err)
	}
	c.token = resp.Token
	return resp.Token, nil
}

// ListItems возвращает все элементы данных пользователя.
func (c *Client) ListItems(ctx context.Context) ([]*pb.Item, error) {
	resp, err := c.service.ListItems(c.authCtx(ctx), &pb.ListItemsRequest{})
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка: %w", err)
	}
	return resp.Items, nil
}

// CreateItem создаёт новый элемент данных на сервере.
func (c *Client) CreateItem(ctx context.Context, item *pb.Item) (string, error) {
	resp, err := c.service.CreateItem(c.authCtx(ctx), &pb.CreateItemRequest{Item: item})
	if err != nil {
		return "", fmt.Errorf("ошибка создания элемента: %w", err)
	}
	return resp.Id, nil
}

// DeleteItem удаляет элемент данных по ID.
func (c *Client) DeleteItem(ctx context.Context, id string) error {
	_, err := c.service.DeleteItem(c.authCtx(ctx), &pb.DeleteItemRequest{Id: id})
	if err != nil {
		return fmt.Errorf("ошибка удаления элемента: %w", err)
	}
	return nil
}
