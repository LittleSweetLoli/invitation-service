# Invitation Service

Сервис для регистрации пользователей по приглашениям с ограниченным числом использований.

---

## Быстрый старт (Docker Compose)

```bash
# 1. Клонировать / распаковать проект
cd invitation-service

# 2. Собрать и запустить сервис + PostgreSQL
docker compose up --build

# 3. Сервис слушает на http://localhost:3000
#    Миграции и seed данных применяются автоматически при старте.
```

### seed для приглашений:
```SQL
INSERT INTO invitations (code, max_uses)
VALUES
    ('twitter-reg1',    100),
    ('telegram-test',    50),
    ('instagram-hello', 200)
ON CONFLICT (code) DO NOTHING;
```

Сервис готов к работе, когда в логах появится строка `server listening addr=:3000`.

### Пример POST запроса на регистрацию
```bash
Invoke-WebRequest -Uri "http://localhost:3000/api/v1/invitations/instagram-hello" -Method POST -ContentType "application/json" -Body '{"Email":"test@example.com"}'
# Или 
curl -X POST http://localhost:3000/api/v1/invitations/instagram-hello -H "Content-Type: application/json" -d "{\"Email\":\"test@example.com\"}"
```

---

## Запуск тестов

```bash
# Unit-тесты (не требуют БД)
go test ./...

# Unit + интеграционные тесты
# В ENV записать
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/invitations?sslmode=disable"
go test ./... -v -count=1
```

## Текущая покрытость тестами
```
go test -cover ./...
        github.com/example/invitation-service/cmd/server                        coverage: 0.0% of statements
        github.com/example/invitation-service/internal/db                       coverage: 0.0% of statements
ok      github.com/example/invitation-service/internal/handler          0.783s  coverage: 100.0% of statements
?       github.com/example/invitation-service/internal/model                    [no test files]
ok      github.com/example/invitation-service/internal/repository       1.239s  coverage: 82.6% of statements
ok      github.com/example/invitation-service/internal/service          0.691s  coverage: 100.0% of statements
```

---

## Архитектура

```
cmd/server/          - точка входа
internal/
  handler/           - HTTP слой
  service/           - бизнес-логика
  repository/        - работа с БД, транзакции
  model/             - доменные модели
  db/                - миграции, запуск при старте
```
---

## Сопроводительное письмо

### Почему выбраны pq и chi
- Драйвер pq - проверенный драйвер, очень прост в использовании и полностью совместим с database/sql.
- Роутер chi - легкий, быстрый и понятный.

### Что бы я добавил в Production | TODO

#### Безопасность
- Rate limiting для резких всплесков трафика, ограничение по IP и отдельный лимит по коду приглашения.  
- Защита от очевидного абуза, для слишком частых попыток с одного и того же email.

#### Observability
- Структурированные логи в JSON-формате.
- Метрики prometheus, количество успешных/неуспешных регистраций, latency, ошибки по кодам.
- Трассировка в OpenTelemetry, чтобы видеть где конкретно тормозит.
- Health'чеки `/healthz` и `/readyz`.

#### Надёжность
- Retry с backoff если БД еще не поднялась.
- Circuit breaker на случай сильного торможения БД.
- Заменить самописный раннер на golang-migrate.

#### Тесты
- Интеграционные тесты запускать через "testcontainers-go"
- Добавить нагрузочные тесты дабы проверить корректность работы при высокой конкурентной нагрузке.

#### В CI/CD
- GitHub Actions с линтерами, тестами, и сборкой Docker образа, пуш в registry.

#### Масштабирование
- Горизонтальное масштабирование через Kubernetes HPA + rolling updates для обновлений без даунтайма.
- Managed PostgreSQL с read replica, point-in-time recovery, автоматическими бэкапами через AWS RDS или Google Cloud SQL.
- При очень высокой нагрузке, для дополнительной защиты от гонки, можно добавить Redis как distributed lock.

