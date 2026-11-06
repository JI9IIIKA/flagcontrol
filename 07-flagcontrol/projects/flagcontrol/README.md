# FlagControl — Feature flags

Проект №07. Самостоятельное Go-приложение. Сборка: `./cmd/flagcontrol`.
Общие пакеты находятся в `internal/platform`, поэтому сохраняйте структуру репозитория.

## Реализовано в v0.1

Версии конфигурации; optimistic locking; процентный rollout; allow list; стабильный хеш; откат через новую версию; Go SDK с last-known config; HTTP и gRPC получение конфигурации.

Данные: `flags, flag_heads`. Схема: `../../migrations/flagcontrol_001.sql`.
Все записи изолированы по tenant; роль определяется серверным API-токеном.

## Запуск

Из корня репозитория:

```bash
python3 scripts/setup.py  # один раз для всего набора
PROJECT=flagcontrol docker compose up --build -d app
python3 scripts/demo.py flagcontrol
```

Для JobRunner запускайте ещё `--profile jobs`; точные команды профилей — в корневом README.
Демо создаёт тестовые записи. Остановить: `docker compose down`.

## API

Все пути требуют `Authorization: Bearer <token>`. Ошибки возвращаются как JSON.
`Idempotency-Key` необязателен, но рекомендуется для команд создания и отправки.
Повтор того же запроса с тем же ключом возвращает сохранённый ответ; изменение запроса даёт 409.

| Метод | Путь |
|---|---|
| PUT | `/flags/{key}/{environment}` |
| GET | `/flags/{key}/{environment}` |
| GET | `/flags/{key}/{environment}/history` |
| POST | `/flags/{key}/{environment}/rollback` |
| GET | `/evaluate/{key}/{environment}` |

## Примеры тел запросов

`PUT /flags/{key}/{environment}`:

```json
{
  "expected_version": 0,
  "enabled": true,
  "percentage": 50,
  "allow_users": []
}
```

`POST /flags/{key}/{environment}/rollback`:

```json
{
  "expected_version": 2,
  "version": 1
}
```

## Проверка

```bash
TEST_DATABASE_URL='postgres://portfolio:portfolio-local-only@localhost:5432/portfolio?sslmode=disable' \
  go test -count=1 -v ./tests -run '^TestFlagControl$'
```

Интеграционные тесты пишут данные в указанную БД. Используйте отдельную тестовую базу.
Полный сценарий проверки находится в `../../tests/integration_test.go`.

## Границы версии и дальнейшая работа

Правила по произвольным атрибутам пользователя, фоновая синхронизация SDK и streaming updates.

Это реализованное ядро проекта, а не заявление о завершении всех расширений исходной идеи.
Подробнее об ограничениях окружения проверки — `../../VALIDATION.md`.
