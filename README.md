# extraBadges

HTTP-сервис, хранящий связку `user_id → emoji_id` (бейджи) и отдающий её по запросу.

## Запуск

```bash
go run ./cmd/extrabadges
```

Адрес задаётся переменной `EXTRABADGES_ADDR` (по умолчанию `:8080`).

## API

| Метод  | Путь                 | Описание                              |
|--------|----------------------|---------------------------------------|
| GET    | `/healthz`           | health-check                          |
| GET    | `/badges`            | список всех бейджей                   |
| GET    | `/badges/{userID}`   | бейдж пользователя                    |
| POST   | `/badges`            | создать (`409`, если уже есть)        |
| PUT    | `/badges/{userID}`   | создать или заменить                  |
| DELETE | `/badges/{userID}`   | удалить                               |

Тело: `{"user_id": 1, "emoji_id": 2, "title": "..."}` (`title` опционален).

## Структура

```
cmd/extrabadges        точка входа, graceful shutdown
internal/badge         доменная модель, валидация, сервис, интерфейс Repository
internal/storage/memory потокобезопасное in-memory хранилище
internal/httpapi       HTTP-хендлеры и роутинг
internal/config        конфигурация из env
```

## Тесты

```bash
go test -race -cover ./...
```
