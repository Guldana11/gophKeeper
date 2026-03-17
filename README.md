# GophKeeper

Клиент-серверная система для безопасного хранения паролей, текстовых и бинарных данных, а также данных банковских карт.

## Архитектура

```
┌──────────────┐         gRPC (TLS)         ┌──────────────┐
│  CLI Client  │ ◄─────────────────────────► │    Server    │
│              │                             │              │
│  AES-256-GCM │                             │  JWT Auth    │
│  шифрование  │                             │  PostgreSQL  │
└──────────────┘                             └──────────────┘
```

- **Транспорт** — gRPC с опциональным TLS (минимум TLS 1.2)
- **Аутентификация** — JWT токены (HS256, 24 часа)
- **Шифрование данных** — AES-256-GCM на стороне клиента, ключ выводится из пароля через PBKDF2 (100 000 итераций, SHA-256)
- **БД** — PostgreSQL, подключение через pgxpool, запросы через squirrel
- **Синхронизация** — last-write-wins с отслеживанием удалённых элементов

## Типы хранимых данных

| Тип | Флаг CLI | Описание |
|-----|----------|----------|
| `credential` | `-type credential` | Пары логин/пароль |
| `text` | `-type text` | Произвольный текст |
| `binary` | `-type binary` | Бинарные файлы (через `-file`) |
| `card` | `-type card` | Данные банковских карт |

Для всех типов поддерживается произвольная текстовая метаинформация (`-label`).

## Быстрый старт

### Требования

- Go 1.25+
- PostgreSQL 16+
- Docker (для интеграционных тестов)
- protoc + protoc-gen-go + protoc-gen-go-grpc (для генерации proto)

### Сборка

```bash
# Сборка под текущую платформу
make build

# Кроссплатформенная сборка (Linux, macOS, Windows)
make build-all

# Бинарники в каталоге bin/
ls bin/
```

### Запуск сервера

```bash
# Без TLS (разработка)
./bin/server -addr :3200 \
  -dsn "postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable" \
  -jwt-secret "your-secret-key"

# С TLS (продакшен)
./bin/server -addr :3200 \
  -dsn "postgres://..." \
  -jwt-secret "your-secret-key" \
  -tls-cert certs/server-cert.pem \
  -tls-key certs/server-key.pem
```

Миграции применяются автоматически при запуске.

### Генерация TLS-сертификатов

```bash
./certs/generate.sh
```

### Использование клиента

```bash
# Версия клиента
./bin/client version

# Регистрация
./bin/client register -login alice -password mypassword

# Вход
./bin/client login -login alice -password mypassword

# Добавление данных
./bin/client add -type credential -label "GitHub" -data '{"login":"alice","password":"secret"}'
./bin/client add -type text -label "Заметка" -data "Текст заметки"
./bin/client add -type binary -label "Документ" -file /path/to/file.pdf
./bin/client add -type card -label "Сбербанк" -data '{"number":"4111...","cvv":"123"}'

# Список элементов
./bin/client list

# Просмотр элемента (с расшифровкой)
./bin/client get -id <uuid>

# Обновление
./bin/client update -id <uuid> -data "новые данные"
./bin/client update -id <uuid> -label "Новая метка"

# Удаление
./bin/client delete -id <uuid>
```

## Флаги сервера

| Флаг | По умолчанию | Описание |
|------|-------------|----------|
| `-addr` | `:3200` | Адрес сервера |
| `-dsn` | `postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable` | Строка подключения к БД |
| `-jwt-secret` | `supersecretkey` | Секрет для подписи JWT |
| `-tls-cert` | — | Путь к TLS-сертификату |
| `-tls-key` | — | Путь к приватному ключу TLS |

## Конфигурация клиента

Клиент хранит конфигурацию в `~/.gophkeeper.json` (права 0600):

```json
{
  "token": "jwt-token",
  "server_addr": "localhost:3200",
  "encryption_key": "пароль пользователя",
  "ca_cert_path": "certs/ca-cert.pem"
}
```

Для TLS-подключения укажите `ca_cert_path` вручную в конфиге.

## Структура проекта

```
├── cmd/
│   ├── client/main.go          # CLI-приложение
│   └── server/main.go          # gRPC-сервер
├── internal/
│   ├── client/
│   │   ├── config/             # Конфигурация клиента
│   │   ├── crypto/             # AES-256-GCM шифрование
│   │   └── grpcclient/         # gRPC-клиент
│   ├── server/
│   │   ├── auth/               # JWT + bcrypt
│   │   ├── handler/            # gRPC-обработчики + interceptor
│   │   ├── service/            # Бизнес-логика
│   │   └── storage/            # PostgreSQL (pgxpool + squirrel)
│   └── models/                 # Доменные модели
├── proto/                      # Protobuf-контракт
├── migrations/                 # SQL-миграции
├── certs/                      # TLS-сертификаты (generate.sh)
└── Makefile
```

## Схема БД

**users** — пользователи (UUID PK, уникальный логин, bcrypt-хеш пароля)

**items** — зашифрованные данные (UUID PK, FK на users, тип, BYTEA данные, JSONB метаданные, временные метки)

**deleted_items** — журнал удалений для синхронизации (UUID PK, FK на users, время удаления)

## API (gRPC)

| Метод | Описание | Авторизация |
|-------|----------|-------------|
| `Register` | Регистрация пользователя | Нет |
| `Login` | Аутентификация | Нет |
| `CreateItem` | Создание элемента | JWT |
| `GetItem` | Получение элемента | JWT |
| `ListItems` | Список элементов | JWT |
| `UpdateItem` | Обновление элемента | JWT |
| `DeleteItem` | Удаление элемента | JWT |
| `SyncItems` | Синхронизация | JWT |

## Безопасность

- Данные шифруются на клиенте **до** отправки на сервер (AES-256-GCM)
- Ключ шифрования выводится из пароля пользователя (PBKDF2, 100k итераций)
- Каждое шифрование использует уникальные salt и nonce
- Сервер хранит только зашифрованные данные и не имеет ключа расшифровки
- Пароли хешируются через bcrypt
- Поддержка TLS для защиты транспорта
- Конфиг клиента сохраняется с правами 0600

## Тестирование

```bash
# Все тесты
make test

# Покрытие
make cover

# Линтер
make lint
```

Покрытие тестами: **79.1%**

Интеграционные тесты storage используют testcontainers (требуется Docker).

## Сборка с указанием версии

```bash
make build-client BUILD_VERSION=v1.0.0
./bin/client version
# GophKeeper Client v1.0.0 (built: 2026-03-17T12:00:00Z)
```
