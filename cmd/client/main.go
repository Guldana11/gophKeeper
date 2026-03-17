// Package main — точка входа клиентского CLI-приложения GophKeeper.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/guldana/gophKeeperr/internal/client/config"
	"github.com/guldana/gophKeeperr/internal/client/grpcclient"
	pb "github.com/guldana/gophKeeperr/proto"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	if command == "version" {
		fmt.Printf("GophKeeper Client %s (built: %s)\n", buildVersion, buildDate)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка загрузки конфигурации: %v\n", err)
		os.Exit(1)
	}

	serverAddr := cfg.ServerAddr
	if serverAddr == "" {
		serverAddr = "localhost:3200"
	}

	switch command {
	case "register":
		cmdRegister(serverAddr, cfg)
	case "login":
		cmdLogin(serverAddr, cfg)
	case "list":
		cmdList(serverAddr, cfg)
	case "add":
		cmdAdd(serverAddr, cfg)
	case "delete":
		cmdDelete(serverAddr, cfg)
	default:
		fmt.Fprintf(os.Stderr, "Неизвестная команда: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("GophKeeper Client — менеджер паролей")
	fmt.Println()
	fmt.Println("Использование: gophkeeper <команда> [флаги]")
	fmt.Println()
	fmt.Println("Команды:")
	fmt.Println("  version    Показать версию клиента")
	fmt.Println("  register   Регистрация нового пользователя")
	fmt.Println("  login      Аутентификация пользователя")
	fmt.Println("  list       Список сохранённых элементов")
	fmt.Println("  add        Добавить новый элемент")
	fmt.Println("  delete     Удалить элемент")
}

func cmdRegister(addr string, cfg *config.Config) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	login := fs.String("login", "", "логин пользователя")
	password := fs.String("password", "", "пароль пользователя")
	fs.Parse(os.Args[2:])

	if *login == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "Необходимо указать -login и -password")
		os.Exit(1)
	}

	client, err := grpcclient.New(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := client.Register(ctx, *login, *password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка регистрации: %v\n", err)
		os.Exit(1)
	}

	cfg.Token = token
	cfg.ServerAddr = addr
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка сохранения токена: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Регистрация успешна. Токен сохранён.")
}

func cmdLogin(addr string, cfg *config.Config) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	login := fs.String("login", "", "логин пользователя")
	password := fs.String("password", "", "пароль пользователя")
	fs.Parse(os.Args[2:])

	if *login == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "Необходимо указать -login и -password")
		os.Exit(1)
	}

	client, err := grpcclient.New(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := client.Login(ctx, *login, *password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка входа: %v\n", err)
		os.Exit(1)
	}

	cfg.Token = token
	cfg.ServerAddr = addr
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка сохранения токена: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Вход выполнен. Токен сохранён.")
}

func cmdList(addr string, cfg *config.Config) {
	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "Необходимо сначала выполнить login или register")
		os.Exit(1)
	}

	client, err := grpcclient.New(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()
	client.SetToken(cfg.Token)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	items, err := client.ListItems(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}

	if len(items) == 0 {
		fmt.Println("Нет сохранённых элементов.")
		return
	}

	dataTypeNames := map[pb.DataType]string{
		pb.DataType_DATA_TYPE_CREDENTIAL: "логин/пароль",
		pb.DataType_DATA_TYPE_TEXT:       "текст",
		pb.DataType_DATA_TYPE_BINARY:     "бинарные данные",
		pb.DataType_DATA_TYPE_BANK_CARD:  "банковская карта",
	}

	for _, item := range items {
		typeName := dataTypeNames[item.DataType]
		if typeName == "" {
			typeName = "неизвестный"
		}
		fmt.Printf("[%s] %s | тип: %s | обновлён: %s\n",
			item.Id,
			metaLabel(item.Metadata),
			typeName,
			item.UpdatedAt.AsTime().Format("2006-01-02 15:04"),
		)
	}
}

func cmdAdd(addr string, cfg *config.Config) {
	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "Необходимо сначала выполнить login или register")
		os.Exit(1)
	}

	fs := flag.NewFlagSet("add", flag.ExitOnError)
	dataType := fs.String("type", "", "тип данных: credential, text, card")
	label := fs.String("label", "", "название/метка элемента")
	data := fs.String("data", "", "данные (в текстовом виде)")
	fs.Parse(os.Args[2:])

	if *dataType == "" || *data == "" {
		fmt.Fprintln(os.Stderr, "Необходимо указать -type и -data")
		os.Exit(1)
	}

	var pbType pb.DataType
	switch *dataType {
	case "credential":
		pbType = pb.DataType_DATA_TYPE_CREDENTIAL
	case "text":
		pbType = pb.DataType_DATA_TYPE_TEXT
	case "card":
		pbType = pb.DataType_DATA_TYPE_BANK_CARD
	default:
		fmt.Fprintf(os.Stderr, "Неизвестный тип: %s (допустимые: credential, text, card)\n", *dataType)
		os.Exit(1)
	}

	meta := map[string]string{}
	if *label != "" {
		meta["label"] = *label
	}

	client, err := grpcclient.New(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()
	client.SetToken(cfg.Token)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id, err := client.CreateItem(ctx, &pb.Item{
		DataType:      pbType,
		EncryptedData: []byte(*data),
		Metadata:      meta,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Элемент создан: %s\n", id)
}

func cmdDelete(addr string, cfg *config.Config) {
	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "Необходимо сначала выполнить login или register")
		os.Exit(1)
	}

	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	id := fs.String("id", "", "ID элемента для удаления")
	fs.Parse(os.Args[2:])

	if *id == "" {
		fmt.Fprintln(os.Stderr, "Необходимо указать -id")
		os.Exit(1)
	}

	client, err := grpcclient.New(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()
	client.SetToken(cfg.Token)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.DeleteItem(ctx, *id); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Элемент удалён.")
}

func metaLabel(meta map[string]string) string {
	if label, ok := meta["label"]; ok && label != "" {
		return label
	}
	return "(без названия)"
}
