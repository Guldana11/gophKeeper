// Package main — точка входа клиентского CLI-приложения GophKeeper.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/guldana/gophKeeperr/internal/client/config"
	"github.com/guldana/gophKeeperr/internal/client/crypto"
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
	case "get":
		cmdGet(serverAddr, cfg)
	case "add":
		cmdAdd(serverAddr, cfg)
	case "update":
		cmdUpdate(serverAddr, cfg)
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
	fmt.Println("  get        Получить элемент по ID (с расшифровкой)")
	fmt.Println("  add        Добавить новый элемент")
	fmt.Println("  update     Обновить существующий элемент")
	fmt.Println("  delete     Удалить элемент")
}

// newClient создаёт подключённый gRPC клиент с токеном авторизации.
func newClient(addr string, cfg *config.Config) *grpcclient.Client {
	client, err := grpcclient.New(addr, cfg.CACertPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	client.SetToken(cfg.Token)
	return client
}

// requireAuth проверяет наличие токена и ключа шифрования.
func requireAuth(cfg *config.Config) {
	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "Необходимо сначала выполнить login или register")
		os.Exit(1)
	}
}

// requireEncryptionKey проверяет наличие ключа шифрования.
func requireEncryptionKey(cfg *config.Config) {
	if cfg.EncryptionKey == "" {
		fmt.Fprintln(os.Stderr, "Ключ шифрования не найден. Выполните login или register.")
		os.Exit(1)
	}
}

// parseDataType преобразует строковый тип в proto enum.
func parseDataType(s string) pb.DataType {
	switch s {
	case "credential":
		return pb.DataType_DATA_TYPE_CREDENTIAL
	case "text":
		return pb.DataType_DATA_TYPE_TEXT
	case "binary":
		return pb.DataType_DATA_TYPE_BINARY
	case "card":
		return pb.DataType_DATA_TYPE_BANK_CARD
	default:
		fmt.Fprintf(os.Stderr, "Неизвестный тип: %s (допустимые: credential, text, binary, card)\n", s)
		os.Exit(1)
		return 0
	}
}

var dataTypeNames = map[pb.DataType]string{
	pb.DataType_DATA_TYPE_CREDENTIAL: "логин/пароль",
	pb.DataType_DATA_TYPE_TEXT:       "текст",
	pb.DataType_DATA_TYPE_BINARY:     "бинарные данные",
	pb.DataType_DATA_TYPE_BANK_CARD:  "банковская карта",
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

	client, err := grpcclient.New(addr, cfg.CACertPath)
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
	cfg.EncryptionKey = *password
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

	client, err := grpcclient.New(addr, cfg.CACertPath)
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
	cfg.EncryptionKey = *password
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка сохранения токена: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Вход выполнен. Токен сохранён.")
}

func cmdList(addr string, cfg *config.Config) {
	requireAuth(cfg)

	client := newClient(addr, cfg)
	defer client.Close()

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

func cmdGet(addr string, cfg *config.Config) {
	requireAuth(cfg)
	requireEncryptionKey(cfg)

	fs := flag.NewFlagSet("get", flag.ExitOnError)
	id := fs.String("id", "", "ID элемента")
	fs.Parse(os.Args[2:])

	if *id == "" {
		fmt.Fprintln(os.Stderr, "Необходимо указать -id")
		os.Exit(1)
	}

	client := newClient(addr, cfg)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	item, err := client.GetItem(ctx, *id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}

	decrypted, err := crypto.Decrypt(item.EncryptedData, cfg.EncryptionKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка дешифрования: %v\n", err)
		os.Exit(1)
	}

	typeName := dataTypeNames[item.DataType]
	if typeName == "" {
		typeName = "неизвестный"
	}

	fmt.Printf("ID:        %s\n", item.Id)
	fmt.Printf("Тип:       %s\n", typeName)
	fmt.Printf("Метка:     %s\n", metaLabel(item.Metadata))
	fmt.Printf("Создан:    %s\n", item.CreatedAt.AsTime().Format("2006-01-02 15:04"))
	fmt.Printf("Обновлён:  %s\n", item.UpdatedAt.AsTime().Format("2006-01-02 15:04"))

	if item.DataType == pb.DataType_DATA_TYPE_BINARY {
		fmt.Printf("Данные:    [бинарные, %d байт]\n", len(decrypted))

		outFile := *id + ".bin"
		if err := os.WriteFile(outFile, decrypted, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка сохранения файла: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Файл сохранён: %s\n", outFile)
	} else {
		fmt.Printf("Данные:    %s\n", string(decrypted))
	}

	if len(item.Metadata) > 0 {
		fmt.Println("Метаданные:")
		for k, v := range item.Metadata {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}
}

func cmdAdd(addr string, cfg *config.Config) {
	requireAuth(cfg)
	requireEncryptionKey(cfg)

	fs := flag.NewFlagSet("add", flag.ExitOnError)
	dataType := fs.String("type", "", "тип данных: credential, text, binary, card")
	label := fs.String("label", "", "название/метка элемента")
	data := fs.String("data", "", "данные (в текстовом виде)")
	file := fs.String("file", "", "путь к файлу (для типа binary)")
	fs.Parse(os.Args[2:])

	if *dataType == "" {
		fmt.Fprintln(os.Stderr, "Необходимо указать -type")
		os.Exit(1)
	}

	pbType := parseDataType(*dataType)

	var plaintext []byte
	if *file != "" {
		fileData, err := os.ReadFile(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка чтения файла: %v\n", err)
			os.Exit(1)
		}
		plaintext = fileData
	} else if *data != "" {
		plaintext = []byte(*data)
	} else {
		fmt.Fprintln(os.Stderr, "Необходимо указать -data или -file")
		os.Exit(1)
	}

	encryptedData, err := crypto.Encrypt(plaintext, cfg.EncryptionKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка шифрования: %v\n", err)
		os.Exit(1)
	}

	meta := map[string]string{}
	if *label != "" {
		meta["label"] = *label
	}
	if *file != "" {
		meta["filename"] = *file
	}

	client := newClient(addr, cfg)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	id, err := client.CreateItem(ctx, &pb.Item{
		DataType:      pbType,
		EncryptedData: encryptedData,
		Metadata:      meta,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Элемент создан: %s\n", id)
}

func cmdUpdate(addr string, cfg *config.Config) {
	requireAuth(cfg)
	requireEncryptionKey(cfg)

	fs := flag.NewFlagSet("update", flag.ExitOnError)
	id := fs.String("id", "", "ID элемента для обновления")
	dataType := fs.String("type", "", "новый тип данных: credential, text, binary, card")
	label := fs.String("label", "", "новая метка элемента")
	data := fs.String("data", "", "новые данные (в текстовом виде)")
	file := fs.String("file", "", "путь к файлу (для типа binary)")
	fs.Parse(os.Args[2:])

	if *id == "" {
		fmt.Fprintln(os.Stderr, "Необходимо указать -id")
		os.Exit(1)
	}

	client := newClient(addr, cfg)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Получаем текущий элемент.
	item, err := client.GetItem(ctx, *id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка получения элемента: %v\n", err)
		os.Exit(1)
	}

	// Обновляем тип, если указан.
	if *dataType != "" {
		item.DataType = parseDataType(*dataType)
	}

	// Обновляем данные, если указаны.
	if *file != "" {
		fileData, err := os.ReadFile(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка чтения файла: %v\n", err)
			os.Exit(1)
		}
		encrypted, err := crypto.Encrypt(fileData, cfg.EncryptionKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка шифрования: %v\n", err)
			os.Exit(1)
		}
		item.EncryptedData = encrypted
		if item.Metadata == nil {
			item.Metadata = map[string]string{}
		}
		item.Metadata["filename"] = *file
	} else if *data != "" {
		encrypted, err := crypto.Encrypt([]byte(*data), cfg.EncryptionKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка шифрования: %v\n", err)
			os.Exit(1)
		}
		item.EncryptedData = encrypted
	}

	// Обновляем метку, если указана.
	if *label != "" {
		if item.Metadata == nil {
			item.Metadata = map[string]string{}
		}
		item.Metadata["label"] = *label
	}

	if err := client.UpdateItem(ctx, item); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка обновления: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Элемент обновлён.")
}

func cmdDelete(addr string, cfg *config.Config) {
	requireAuth(cfg)

	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	id := fs.String("id", "", "ID элемента для удаления")
	fs.Parse(os.Args[2:])

	if *id == "" {
		fmt.Fprintln(os.Stderr, "Необходимо указать -id")
		os.Exit(1)
	}

	client := newClient(addr, cfg)
	defer client.Close()

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
