# trip-service

Сервис поездок TripGo: создание, получение и завершение поездки. HTTP API на
`chi`, хранение в PostgreSQL. Лабораторная работа 2.

## Требования

- Go 1.26+
- `tripgoctl` для локального окружения с PostgreSQL
- `make`

## Запуск

```bash
tripgoctl cluster start
tripgoctl environment start  # поднимает PostgreSQL и генерирует .env
tripgoctl connect            # адреса компонентов

make migrate             # накатить миграции
make run                 # запустить сервис
```

Остальные цели `Makefile`:

| Цель | Что делает |
|---|---|
| `make generate` | генерирует типы и chi-сервер из `contracts/openapi/trip-service.openapi.yaml` |
| `make migrate-down` | откатить последнюю миграцию |
| `make build` | собрать бинарь в `bin/` |
| `make test` | unit-тесты с `-race` и итоговым покрытием |
| `make test-integration` | миграции и интеграционные тесты PostgreSQL; нужен `TEST_DATABASE_URL` |

Для интеграционных тестов задайте `TEST_DATABASE_URL` — адрес тестовой базы PostgreSQL
из окружения `tripgoctl`. Команда применяет миграции к этой базе. Тесты создают записи
со случайными UUID и удаляют свои данные после проверки. Для запуска используется
билд-тег `integration`; обычный `make test` подключения к базе не требует.

## Запуск в Docker

```bash
make migrate        # миграции накатываются с хоста
make docker-build   # образ service:local из deploy/Dockerfile
make docker-run     # берёт .env, адрес бд подменяется на host.docker.internal
curl -i localhost:8080/ready
```

Образ собирается в два этапа: бинарь компилируется в `golang:1.26-alpine`, в
итоговый `scratch` кладётся только статический бинарь. Внутри нет шелла,
пакетного менеджера, исходников и тулчейна. Процесс запускается от пользователя
`65532`, не от root. `.dockerignore` пропускает в контекст сборки только
`go.mod`, `go.sum`, `cmd/` и `internal/`.

Размер итогового образа: **23 МБ** (`docker images trip-service:local`).

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

## Архитектура

### Раскладка пакетов

HTTP и usecase разделены по действиям: `trip/create`, `trip/get`, `trip/finish`.
Интерфейсы зависимостей лежат в `contract.go` у потребителя, конкретные
реализации связывает `internal/container`. Общие преобразования ответа поездки
лежат в `internal/http/trip/convert.go`, запись JSON — в `internal/http/response`,
маппинг ошибок — в `internal/http/domainerror`.

Репозитории разделены по данным: `trip`, `triphistory`, `idempotency`.
Операции с поездкой находятся в отдельных файлах одного пакета `trip`:
они используют общие колонки и преобразование строки БД. Отдельные пакеты
истории и идемпотентности сохраняют существующие границы зависимостей usecase.

Менеджер транзакций лежит в `internal/infra/postgres/transaction`.
Usecase зависит только от интерфейса `Do`, а репозитории вызывают
`transaction.ExtractTx`, чтобы использовать транзакцию из контекста или пул.
Прямой импорт допустим здесь: репозиторий реализует PostgreSQL-хранилище
и уже зависит от pgx. Выделение дополнительного интерфейса доступа к соединению
потребует отдельного изменения конструкторов; для переноса оно не требуется.

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
диапазоны и лишние поля и отдаёт `400 invalid_request`. В хендлере дополнительно
проверяется, что `user_id` и `driver_id` не нулевые UUID: по схеме нулевой UUID
валиден.

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

## Проверка лабораторной 2

`make lint` запускает полный набор линтеров курса. `make generate` обновляет OpenAPI и моки.
`make test` запускает unit-тесты с `-race` и печатает покрытие; повторная проверка: `go test -race -count=2 ./...`.
Для интеграционных тестов задайте `TEST_DATABASE_URL` и выполните `make test-integration` дважды.

Зависимости создаются в `internal/container`. `main` читает конфигурацию и запускает приложение.
Интерфейсы зависимостей объявлены в `contract.go` потребителя: usecase и HTTP-ручки не знают конкретного хранилища.
Домен проверяет данные поездки и переход active -> completed. HTTP и PostgreSQL зависят от домена; обратные импорты запрещает depguard.

Телеметрия находится в `internal/infra/telemetry`: OTel SDK, OTLP-экспорт, инструментация пула через otelpgx и JSON slog.
Для `OTEL_EXPORTER_OTLP_PROTOCOL` поддерживаются `http/protobuf` (по умолчанию) и `grpc`.
Адрес коллектора задаёт `OTEL_EXPORTER_OTLP_ENDPOINT`; sampler — `OTEL_TRACES_SAMPLER` и `OTEL_TRACES_SAMPLER_ARG`.
`OTEL_SERVICE_NAME` и `OTEL_RESOURCE_ATTRIBUTES` задают атрибуты ресурса; безопасный пример находится в `.env.example`.
Экспорт работает асинхронно: недоступный коллектор не останавливает HTTP-запросы.

Порядок middleware: Recover -> Tracing -> Logging -> Metrics -> router.
Логи запросов содержат шаблон маршрута, статус, длительность и trace_id/span_id, тела запросов не записываются.
Бизнес-счётчики увеличиваются после успешного commit; повтор идемпотентного создания их не меняет.
Gauge активных поездок читает актуальное число из PostgreSQL, поэтому корректен после рестарта.
