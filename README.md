# Go REST API - Profile Service

Пет-проект REST API на Go для управления профилями пользователей. Часть микросервисной архитектуры, работающий совместно с Auth Service.

## Технологии

- **Go 1.23.1** - основной язык
- **PostgreSQL** - база данных
- **Gorilla Mux** - HTTP роутинг
- **JWT** - токены для аутентификации (интеграция с Auth Service)
- **Logrus** - структурированное логирование
- **Docker & Docker Compose** - контейнеризация
- **Flyway** - миграции БД

## Структура проекта

```
gorestapi_profile/
├── cmd/apiserver/          # Точка входа приложения
├── internal/app/
│   ├── apiserver/          # HTTP сервер, роутинг, handlers
│   ├── model/              # Модели данных (Profile, ShortProfile)
│   └── store/              # Слой работы с БД (репозитории)
├── configs/                # Конфигурационные файлы (TOML)
├── migrations/             # SQL миграции для Flyway
├── certs/                  # TLS сертификаты
├── docker-compose.yaml     # Оркестрация сервисов
└── Dockerfile              # Образ приложения
```

## Установка и запуск

### Требования

- Go 1.23+
- Docker & Docker Compose
- PostgreSQL (или через Docker)

### Настройка

1. Клонируйте репозиторий:

```bash
git clone <repository-url>
cd gorestapi_profile
```

2. Создайте файл `.env` в корне проекта:

```env
BIND_ADDR=:8081
LOG_LEVEL=debug
DB_URL=postgres
DB_PORT=5432
DB_USER=your_user
DB_PASSWORD=your_password
DB_NAME=profiles
JWTSKEY=your_jwt_secret_key
```

**Важно:** `JWTSKEY` должен совпадать с ключом в Auth Service для корректной валидации JWT токенов.

3. Соберите проект:

```bash
make build
```

Или вручную:

```bash
go build -v ./cmd/apiserver
```

### Запуск через Docker Compose

```bash
docker-compose up -d
```

Это запустит:
- PostgreSQL контейнер
- Flyway для применения миграций
- Profile Service

```bash
docker network create my-app-network
```

### Запуск локально

1. Убедитесь, что PostgreSQL запущен и доступен
2. Примените миграции
3. Запустите сервер:

```bash
./apiserver -config-path=configs/apiserver.toml
```

## API Endpoints

Все endpoints требуют JWT токен в заголовке `Authorization: Bearer <token>`, полученный от Auth Service.

### `GET /api/profiles/me`

Получить свой профиль.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
- `200 OK` - успешный запрос
```json
{
  "userid": 1,
  "username": "johndoe",
  "description": "Software developer",
  "avatarurl": "https://example.com/avatar.jpg",
  "birthday": "1990-01-01",
  "followerscount": 42,
  "isownprofile": true,
  "isfollowed": true
}
```
- `401 Unauthorized` - невалидный или отсутствующий токен
- `404 Not Found` - профиль не найден

### `GET /api/profiles/{username}`

Получить профиль пользователя по username.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
- `200 OK` - успешный запрос
```json
{
  "userid": 2,
  "username": "janedoe",
  "description": "Designer",
  "avatarurl": "https://example.com/avatar2.jpg",
  "birthday": "1992-05-15",
  "followerscount": 100,
  "isownprofile": false,
  "isfollowed": true
}
```
- `401 Unauthorized` - невалидный или отсутствующий токен
- `404 Not Found` - профиль не найден

### `GET /api/profiles?search={pattern}`

Поиск профилей по паттерну (username).

**Query Parameters:**
- `search` (required) - паттерн для поиска

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
- `200 OK` - успешный запрос
```json
[
  {
    "username": "johndoe",
    "avatarurl": "https://example.com/avatar.jpg"
  },
  {
    "username": "janedoe",
    "avatarurl": "https://example.com/avatar2.jpg"
  }
]
```
- `400 Bad Request` - отсутствует параметр search
- `401 Unauthorized` - невалидный или отсутствующий токен
- `404 Not Found` - профили не найдены

### `POST /api/profiles/crprofile`

Создать новый профиль.

**Headers:**
```
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "userid": 1,
  "username": "johndoe",
  "description": "Software developer",
  "avatarurl": "https://example.com/avatar.jpg",
  "birthday": "1990-01-01"
}
```

**Response:**
- `200 OK` - профиль успешно создан
- `400 Bad Request` - невалидные данные
- `409 Conflict` - профиль с таким username уже существует

### `POST /api/profiles/subscribe/{username}`

Подписаться на пользователя.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
- `200 OK` - успешная подписка
- `401 Unauthorized` - невалидный или отсутствующий токен
- `404 Not Found` - пользователь не найден

### `POST /api/profiles/unsubscribe/{username}`

Отписаться от пользователя.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
- `200 OK` - успешная отписка
- `401 Unauthorized` - невалидный или отсутствующий токен
- `404 Not Found` - пользователь не найден

### `GET /api/profiles/followers/{username}`

Получить список подписчиков пользователя.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
- `200 OK` - успешный запрос
```json
[
  {
    "username": "user1",
    "avatarurl": "https://example.com/avatar1.jpg"
  },
  {
    "username": "user2",
    "avatarurl": "https://example.com/avatar2.jpg"
  }
]
```
- `401 Unauthorized` - невалидный или отсутствующий токен
- `404 Not Found` - пользователь не найден

### `GET /api/profiles/followed/{username}`

Получить список подписок пользователя (на кого подписан).

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
- `200 OK` - успешный запрос
```json
[
  {
    "username": "user3",
    "avatarurl": "https://example.com/avatar3.jpg"
  },
  {
    "username": "user4",
    "avatarurl": "https://example.com/avatar4.jpg"
  }
]
```
- `401 Unauthorized` - невалидный или отсутствующий токен
- `404 Not Found` - пользователь не найден

## Конфигурация

Конфигурация хранится в `configs/apiserver.toml` и поддерживает переменные окружения:

```toml
bind_addr = "${BIND_ADDR}"
log_level = "${LOG_LEVEL}"

[store]
database_url = "host=${DB_URL} port=${DB_PORT} user=${DB_USER} password=${DB_PASSWORD} dbname=${DB_NAME} sslmode=disable"
```

## База данных

### Структура таблиц

**user_profiles:**
- `id` - первичный ключ
- `user_id` - ID пользователя из Auth Service (уникальный)
- `username` - имя пользователя (уникальное)
- `birthday_date` - дата рождения
- `description` - описание профиля
- `avatar_url` - URL аватара
- `followers_count` - количество подписчиков

**subscribtions:**
- `follower_id` - ID подписчика (FK на user_profiles.user_id)
- `followee_id` - ID того, на кого подписались (FK на user_profiles.user_id)
- Составной первичный ключ (follower_id, followee_id)

## Безопасность

- JWT токены валидируются с использованием секретного ключа (должен совпадать с Auth Service)
- CORS настроен (в текущей версии разрешены все источники)
- Подготовлена инфраструктура для TLS (закомментировано)
- Все операции с подписками выполняются в транзакциях

## Интеграция с Auth Service

Profile Service работает совместно с Auth Service:

1. Пользователь регистрируется/авторизуется через Auth Service
2. Auth Service возвращает JWT токен
3. Profile Service использует этот токен для:
   - Идентификации пользователя (из поля `sub` в JWT)
   - Проверки прав доступа
   - Связывания профиля с пользователем через `user_id`

**Важно:** При создании профиля `user_id` должен соответствовать ID пользователя из Auth Service.

## Тестирование

Запуск тестов:

```bash
make test
```

Или:

```bash
go test -v -race -timeout 30s ./...
```

Пет-проект для обучения и портфолио.

