# trip-service

Сервис поездок TripGo: создание, получение и завершение поездки. HTTP API на
`chi`, хранение в PostgreSQL. Лабораторная работа 1.

## Требования

- Go 1.26+
- `tripgoctl` для локального окружения с PostgreSQL
- `make`

## Запуск

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect        # генерирует .env с DATABASE_URL

make migrate             # накатить миграции
make run                 # запустить сервис
```

Остальные цели `Makefile`:

| Цель | Что делает |
|---|---|
| `make generate` | генерирует типы и chi-сервер из `contracts/openapi/trip-service.openapi.yaml` |
| `make migrate-down` | откатить последнюю миграцию |
| `make build` | собрать бинарь в `bin/` |
| `make test` | тесты с `-race` |

## Запуск в Docker

```bash
make migrate        # миграции накатываются с хоста
make docker-build   # образ trip-service:local из deploy/Dockerfile
make docker-run     # берёт .env, адрес бд подменяется на host.docker.internal
curl -i localhost:8080/ready
```

Образ собирается в два этапа: бинарь компилируется в `golang:1.26-alpine`, в
итоговый `scratch` кладутся только статический бинарь и корневые сертификаты.
Внутри нет шелла, пакетного менеджера, исходников и тулчейна: образ меньше, а
поверхность атаки минимальна. Процесс запускается от непривилегированного
пользователя `65532`. `.dockerignore` пропускает в контекст сборки только
`go.mod`, `go.sum`, `cmd/` и `internal/`.

Размер итогового образа: **TODO МБ** (`docker images trip-service:local`).

## Переменные окружения

Пример лежит в [`.env.example`](.env.example).

| Переменная | Обязательная | По умолчанию | Что задаёт |
|---|---|---|---|
| `HTTP_ADDR` | да | | адрес HTTP-сервера, например `:8080` |
| `LOG_LEVEL` | да | | `debug`, `info`, `warn`, `error` |
| `SHUTDOWN_TIMEOUT` | да | | бюджет на graceful shutdown |
| `HTTP_READ_TIMEOUT` | нет | `10s` | `http.Server.ReadTimeout` |
| `HTTP_READ_HEADER_TIMEOUT` | нет | `5s` | `http.Server.ReadHeaderTimeout` |
| `HTTP_WRITE_TIMEOUT` | нет | `10s` | `http.Server.WriteTimeout` |
| `HTTP_IDLE_TIMEOUT` | нет | `60s` | `http.Server.IdleTimeout` |
| `DATABASE_URL` | да | | строка подключения к PostgreSQL |
| `DATABASE_MAX_CONNS` | да | | максимальный размер пула |
| `DATABASE_MIN_CONNS` | да | | минимальный размер пула |
| `DATABASE_MAX_CONN_LIFETIME` | да | | время жизни соединения |
| `DATABASE_CONNECT_TIMEOUT` | да | | таймаут установки соединения |
| `DATABASE_QUERY_TIMEOUT` | да | | `statement_timeout` для запросов |
| `IDEMPOTENCY_KEY_TTL` | нет | `24h` | сколько живёт ключ идемпотентности |

## Решения

### Уровень изоляции

`READ COMMITTED`, задаётся явно при старте транзакции в `TxManager.Do`. Более
строгий уровень не нужен: от гонок защищают ограничения базы и условный `UPDATE`,
а не видимость данных внутри транзакции.

### Менеджер транзакций

`repository.TxManager.Do` открывает транзакцию, кладёт её в `context` и вызывает
`fn`. Если `fn` вернула ошибку или паникнула, делается `ROLLBACK`, иначе
`COMMIT`. Репозитории берут исполнителя через `extractTx`: транзакцию из
контекста, если она есть, иначе пул. Вложенный `Do` видит транзакцию в контексте
и новую не открывает. Юзкейс знает только интерфейс `Do(ctx, fn)`.

Создание поездки пишет в `trips` и `trip_status_history` внутри одного `Do`.

### Одна активная поездка на водителя

Частичный уникальный индекс
`trips_driver_active_uidx ON trips (driver_id) WHERE status = 'active'`. При
параллельных вставках вторая получает `23505`, репозиторий переводит его в
`ErrDriverBusy`, HTTP отдаёт `409 driver_busy`.

### Завершение поездки

`UPDATE ... SET status = 'completed' WHERE id = $1 AND status = 'active'`.
Параллельный `UPDATE` ждёт блокировку строки, после коммита первого перепроверяет
условие и не находит строк. Ноль строк различаем дочитыванием: поездки нет, это
`404`, иначе `409 trip_completed`.

### Валидация запросов

Запросы проверяются по тому же OpenAPI-контракту, из которого сгенерирован код:
middleware `oapi-codegen/nethttp-middleware` проверяет обязательные поля, типы,
диапазоны и лишние поля и отдаёт `400 invalid_request`. Вручную проверяется
только то, чего нет в схеме: `user_id` и `driver_id` не нулевые UUID.

### Идемпотентность создания

Ключ из заголовка `Idempotency-Key` хранится в таблице `idempotency_keys` вместе
с sha256 полей запроса и id созданной поездки. В той же транзакции, что и
создание поездки, выполняется
`INSERT ... ON CONFLICT (key) DO NOTHING`:

- ключ вставился — это первый запрос, создаём поездку, привязываем её к ключу, `201`;
- ключ уже есть, хеш совпал — повтор, отдаём ту же поездку, `200`;
- ключ уже есть, хеш другой — `409 idempotency_conflict`.

Два одновременных запроса с одним ключом: второй `INSERT` ждёт на первичном ключе,
пока первая транзакция не завершится, и затем видит её результат. Если первая
откатилась (например, `driver_busy`), ключ не сохраняется, и второй запрос
выполняется как новый.

Ключ живёт `IDEMPOTENCY_KEY_TTL` (по умолчанию 24 часа). Просроченный ключ
удаляется при следующем обращении к нему и считается новым.
