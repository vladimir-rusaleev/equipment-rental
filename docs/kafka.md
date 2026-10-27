# Kafka

Kafka передаёт события изменения заказов из Rental в Notifier:

```text
Rental → outbox в PostgreSQL → Kafka → Notifier → очередь в PostgreSQL → лог
```

Rental удаляет событие из outbox после подтверждения брокера. Notifier подтверждает
чтение после сохранения в своей БД. При сбоях возможны повторы; Notifier убирает
дубликаты по ID события. Если Notifier остановлен, события остаются в Kafka.

## Запуск

Весь проект, включая Kafka и мониторинг:

```bash
docker compose up --build -d
```

`kafka-init` создаёт topic и завершается с кодом 0. Это нормально.
Для запуска Go-сервисов локально, а PostgreSQL и Kafka в Docker:

```bash
make infra-up
NOTIFICATION_TRANSPORT=kafka make run
```

Обычный `make run` использует gRPC, если в `.env` не выбран другой транспорт.

| Настройка | Значение по умолчанию |
|---|---|
| `NOTIFICATION_TRANSPORT` | Локально `grpc`, в Compose `kafka` |
| `KAFKA_BROKERS` | Локально `127.0.0.1:19092`, в Compose `kafka:9092` |
| `KAFKA_TOPIC` | `rental.orders` |
| `KAFKA_GROUP` | `rent-notifier` |

## Проверка

Оформите заказ и посмотрите уведомление:

```bash
docker compose logs -f notifier
```

Для проверки доставки после простоя можно остановить Notifier, оформить заказ
и снова запустить его: `docker compose stop notifier` / `docker compose start notifier`.
Автоматическая проверка этого сценария — `make smoke-kafka`.

Используется один брокер и одна partition, без отказоустойчивых реплик.
Хранение событий ограничено 24 часами и 256 МБ на partition. Некорректное событие
блокирует дальнейшее чтение: отдельная очередь ошибок пока не реализована.
