# Subscription Aggregation Service

REST-сервис для CRUDL-операций над онлайн-подписками и подсчета суммарной стоимости подписок за период.

## Запуск

```bash
cp .env.example .env
docker compose up --build
```

Сервис будет доступен на `http://localhost:8080`.

## Документация

OpenAPI-спецификация: `docs/openapi.yaml`.

## Основные ручки

- `POST /subscriptions` - создать подписку
- `GET /subscriptions` - список подписок с пагинацией
- `GET /subscriptions/{id}` - получить подписку
- `PUT /subscriptions/{id}` - обновить подписку
- `DELETE /subscriptions/{id}` - удалить подписку
- `GET /subscriptions/total?from=07-2025&to=12-2025&user_id=...&service_name=Yandex%20Plus` - сумма за период

Даты передаются в формате `MM-YYYY`. Цена хранится целым числом рублей.
