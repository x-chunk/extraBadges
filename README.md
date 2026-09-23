# extraBadges

HTTP-сервис, хранящий связку `user_id → emoji_id` (бейджи) и отдающий её по запросу.

## Запуск

```bash
go run ./cmd/extrabadges
```

| Переменная              | По умолчанию     | Описание                                   |
|-------------------------|------------------|--------------------------------------------|
| `EXTRABADGES_ADDR`      | `:8080`          | адрес HTTP-сервера                         |
| `EXTRABADGES_DB`        | `extrabadges.db` | путь к файлу SQLite (создаётся сам)        |
| `EXTRABADGES_CACHE_TTL` | `5m`             | TTL кэша для `GET /badges/{id}`; `0` — выкл |

## API

| Метод  | Путь                 | Описание                              |
|--------|----------------------|---------------------------------------|
| GET    | `/healthz`           | health-check                          |
| GET    | `/badges`            | список всех бейджей                   |
| GET    | `/badges/{userID}`   | бейдж пользователя                    |
| POST   | `/badges`            | создать (`409`, если уже есть)        |
| PUT    | `/badges/{userID}`   | создать или заменить                  |
| PATCH  | `/badges/{userID}`   | частично изменить (`emoji_id`, `title`) |
| DELETE | `/badges/{userID}`   | удалить                               |

Тело: `{"user_id": 1, "emoji_id": 2, "title": "..."}` (`title` опционален).

## Структура

```
cmd/extrabadges        точка входа, graceful shutdown
internal/badge         доменная модель, валидация, сервис, интерфейс Repository
internal/storage/sqlite  SQLite-хранилище (modernc.org/sqlite, без cgo)
internal/storage/cached  TTL-кэш поверх любого Repository (github.com/x-chunk/anvil)
internal/storage/memory  in-memory хранилище (для тестов)
internal/storage/storagetest общий контрактный тест для всех Repository
internal/httpapi       HTTP-хендлеры и роутинг
internal/config        конфигурация из env
```

## Тесты

```bash
go test -race -cover ./...
```
