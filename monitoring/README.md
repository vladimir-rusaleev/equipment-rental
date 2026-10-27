# Мониторинг

**Prometheus** собирает и хранит метрики сервисов каждые 15 секунд.
**Grafana** получает их из Prometheus и показывает на готовом дашборде.

Оба инструмента запускаются вместе с проектом:

```bash
docker compose up --build -d
```

- [Grafana](http://localhost:3000/d/rent-overview) — дашборд, первый вход `admin` / `admin`.
- [Prometheus](http://localhost:9090/) — запросы к метрикам.
- [Prometheus Targets](http://localhost:9090/targets) — проверка сбора данных.

Порты задаются через `GRAFANA_PORT` и `PROMETHEUS_PORT` в `.env`.
Например, при `PROMETHEUS_PORT=19090` открывайте `http://localhost:19090`.
Начальный пароль Grafana задаётся через `GRAFANA_ADMIN_PASSWORD`.
Дашборд и источник данных подключаются автоматически; история Prometheus хранится 7 дней.

## Что отслеживается

| Метрика | Назначение |
|---|---|
| `up` | Удалось ли собрать метрики сервиса |
| `rent_http_requests_total` | Количество HTTP-запросов и их коды ответа |
| `rent_http_request_duration_seconds` | Время ответа Gateway |
| `rent_grpc_requests_total` | Количество gRPC-вызовов и их коды ответа |
| `rent_grpc_request_duration_seconds` | Время выполнения gRPC-вызовов |
| `rent_queue_pending` | Ожидающие события в outbox Rental и очереди Notifier в БД |
| `rent_notification_attempts_total` | Успешные и неудачные попытки передачи и доставки событий |

Дашборд показывает скорость запросов, ошибки, p95 времени ответа и очереди.
Откройте каталог или оформите заказ, затем подождите около 30 секунд.
Без запросов часть графиков может показывать «No data».
`up` проверяет сбор метрик; размер очереди Kafka в `rent_queue_pending` не входит.

Конфигурация находится в `monitoring/prometheus/prometheus.yml`,
источник данных и дашборд — в `monitoring/grafana/`.
Настройки Prometheus рассчитаны на запуск всего стека через Compose.
