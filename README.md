# extraBadges

HTTP-сервис для клиента exteraless. Хранит и отдаёт две вещи:

1. **Метку клиента** — «пользуется exteraless». Ставится, когда клиент
   подтверждает аккаунт подписанным Telegram `initData`.
2. **Бейдж справа от ника** — custom emoji с описанием, выдаётся вручную через админку.

## Запуск

```bash
EXTRABADGES_ADMIN_TOKEN=... EXTRABADGES_BOT_TOKEN=... go run ./cmd/extrabadges
```

| Переменная                     | По умолчанию      | Описание                                             |
|--------------------------------|-------------------|------------------------------------------------------|
| `EXTRABADGES_ADMIN_TOKEN`      | — (обязательно)   | токен админских ручек                                |
| `EXTRABADGES_BOT_TOKEN`        | — (обязательно)   | токен бота для проверки `initData`                   |
| `EXTRABADGES_ADDR`             | `:8080`           | адрес HTTP-сервера                                   |
| `EXTRABADGES_DB`               | `extrabadges.db`  | путь к файлу SQLite (создаётся сам)                  |
| `EXTRABADGES_CACHE_TTL`        | `5m`              | TTL кэша бейджей и меток; `0` — выкл                 |
| `EXTRABADGES_CLIENT_TTL`       | `720h`            | срок жизни метки клиента без продления               |
| `EXTRABADGES_INITDATA_MAX_AGE` | `24h`             | максимальный возраст `auth_date` в `initData`        |
| `EXTRABADGES_MARK_EMOJI_ID`    | —                 | значок метки клиента                                 |
| `EXTRABADGES_MARK_TEXT`        | `exteraless user` | текст метки клиента                                  |

Схема SQLite мигрирует сама при старте (`PRAGMA user_version`).

## Публичный API

Все ответы — JSON, ошибки — `{"error": "…"}`.

| Метод  | Путь                    | Тело                       | Ответ |
|--------|-------------------------|----------------------------|-------|
| GET    | `/healthz`              | —                          | `{"status":"ok"}` |
| GET    | `/v1/config`            | —                          | `{"client_mark":{"emoji_id":123,"text":"exteraless user"}}` |
| POST   | `/v1/lookup`            | `{"ids":[1,2,3]}`, ≤ 100   | `{"users":{"1":{"client":true,"badge":{"emoji_id":5,"description":"…"}}}}` |
| POST   | `/v1/clients/register`  | `{"init_data":"…"}`        | `204`; `401` — плохая подпись или устарела |
| DELETE | `/v1/clients/me`        | `{"init_data":"…"}`        | `204`; `401` — плохая подпись или устарела |

`lookup`: пользователи без метки и без бейджа в ответ не попадают; `client`
есть, только если метка не скрыта и моложе `EXTRABADGES_CLIENT_TTL`; больше
100 id → `400`. Тела `lookup` не логируются.

## Админский API

Все ручки требуют `Authorization: Bearer $EXTRABADGES_ADMIN_TOKEN`, иначе `401`.

| Метод  | Путь                        | Описание                                      |
|--------|-----------------------------|-----------------------------------------------|
| GET    | `/v1/admin/badges`          | список всех бейджей                           |
| GET    | `/v1/admin/badges/{userID}` | бейдж пользователя                            |
| POST   | `/v1/admin/badges`          | создать (`409`, если уже есть)                |
| PUT    | `/v1/admin/badges/{userID}` | создать или заменить                          |
| PATCH  | `/v1/admin/badges/{userID}` | частично изменить (`emoji_id`, `description`) |
| DELETE | `/v1/admin/badges/{userID}` | удалить                                       |

Тело: `{"user_id": 1, "emoji_id": 2, "description": "..."}`. Пустое
`description` — клиент не показывает подсказку.

## Структура

```
cmd/extrabadges              точка входа, graceful shutdown
internal/badge               доменная модель бейджа, валидация, сервис, интерфейс Repository
internal/client              метка клиента, интерфейс Repository
internal/telegram            проверка Telegram initData
internal/storage/sqlite      SQLite-хранилище и миграции (modernc.org/sqlite, без cgo)
internal/storage/cached      TTL-кэш поверх любого Repository (github.com/x-chunk/anvil)
internal/storage/memory      in-memory хранилища (для тестов)
internal/storage/storagetest общие контрактные тесты для всех Repository
internal/httpapi             HTTP-хендлеры и роутинг
internal/config              конфигурация из env
```

## Тесты

```bash
go test -race -cover ./...
```
