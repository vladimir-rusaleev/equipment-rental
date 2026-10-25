# HTTP API

## HTTP API

JSON использует `snake_case`. По правилам ProtoJSON поля `int64` выводятся строками;
при вводе допустимы строковые и числовые значения.
Защищённые маршруты требуют `Authorization: Bearer <token>`.

| Метод | Маршрут `/api/v1` | Назначение |
|---|---|---|
| POST | `/auth/register` | Регистрация: `email`, `password` (10–72 байта) |
| POST | `/auth/login` | Вход |
| GET | `/me` | Текущий пользователь |
| GET | `/models` | Каталог: `query`, `category`, необязательная пара `start`, `end` |
| GET | `/models/{id}` | Карточка модели |
| GET | `/models/{id}/calendar?start=…&end=…` | Свободное количество по дням |
| POST | `/models` | Создать модель — admin |
| PUT | `/models/{id}` | Полностью обновить модель/тариф — admin |
| GET | `/models/{id}/units` | Экземпляры — staff/admin |
| POST | `/models/{id}/units` | Добавить экземпляр — staff/admin |
| PATCH | `/units/{id}` | Состояние: `ready`, `maintenance`, `retired` — staff/admin |
| PATCH | `/users/{id}/role` | Изменить роль — admin |
| GET | `/cart` | Своя корзина |
| PUT | `/cart` | Заменить корзину, `items: []` очищает её |
| POST | `/orders` | Оформить с `request_id` |
| GET | `/orders` | Свои заказы; для staff/admin — все |
| GET | `/orders/{id}` | Заказ с историей и экземплярами |
| POST | `/orders/{id}/cancel` | Отмена, тело `{"reason":"..."}` |
| POST | `/orders/{id}/issue` | Выдача — staff/admin, тело `{}` |
| POST | `/orders/{id}/return` | Возврат — staff/admin, тело `{"unit_ids":["..."]}` |

`GET /healthz` — проверка живости Gateway, не проверка готовности всех зависимостей.
Успешные вызовы возвращают HTTP 200. Некорректные данные — 400, отсутствие входа — 401,
нет прав — 403, нет объекта — 404, конфликт — 409, недоступный сервис — 503.
При отложенном выполнении checkout возвращает заказ в промежуточном состоянии:
результат следует читать через `GET /orders/{id}`.

### Пример клиентского сценария

```bash
curl -s http://localhost:8080/api/v1/models

curl -s http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"client@example.com","password":"my-long-password"}'

# Подставьте token и model_id из ответов.
curl -s -X PUT http://localhost:8080/api/v1/cart \
  -H 'Authorization: Bearer TOKEN' -H 'Content-Type: application/json' \
  -d '{"items":[{"model_id":"MODEL_ID","quantity":1,"start":"2026-11-10","end":"2026-11-12"}]}'

curl -s http://localhost:8080/api/v1/orders \
  -H 'Authorization: Bearer TOKEN' -H 'Content-Type: application/json' \
  -d '{"request_id":"checkout-001"}'
```


## Роли и авторизация

- `customer`: своя корзина, оформление, просмотр и отмена своих броней.
- `staff`: дополнительно все заказы, выдача/возврат, отмена чужих броней с причиной,
  добавление экземпляров и изменение их состояния.
- `admin`: дополнительно каталог/тарифы и назначение ролей.

Обычная регистрация всегда создаёт `customer`. Пароли хешируются bcrypt.
Сессии — случайные bearer-токены на 24 часа; в хранилище находятся их SHA-256-хеши.
Роль читается заново при аутентификации, поэтому изменение роли действует на текущую сессию.
Последнего администратора нельзя понизить.

Gateway игнорирует присланные HTTP-заголовки ролей и пользователя и получает
личность через Rental.Authenticate. Внутренние вызовы защищены `SERVICE_KEY`.
Проверки бизнес-доступа есть в usecase Rental и Inventory.
Схема рассчитана на доверенную внутреннюю сеть; перед публичным развёртыванием
нужно добавить TLS, ограничение частоты запросов и отдельные удостоверения сервисов.


## Protobuf

Контракты: `api/rent/v1/rent.proto`. Сгенерированные файлы включены в проект.
Для обычной сборки protoc не нужен. Для изменения контрактов установите protoc
и плагины, затем выполните:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0
# Добавьте $(go env GOPATH)/bin в PATH.
make generate
```

