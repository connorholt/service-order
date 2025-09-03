Сервис заказов (test-only).

Важное предупреждение
- Это учебный (тестовый) сервис, собранный для демонстрации подходов и задач курса.
- Его не следует использовать как образец «красивого» или промышленного кода.
- Архитектурные и инженерные решения упрощены ради наглядности.

Запуск локально
- Требования: Docker/Docker Compose, GNU Make (по желанию), Go 1.21+ (для запуска без контейнеров).
- Быстрый старт через Docker Compose:
  - make run — поднимет Kafka, Kafka UI и сам сервис на :8080
  - make logs — посмотреть логи сервиса
  - make down — остановить и удалить контейнеры (с томами)
- Сервис слушает порт 8080. Базовый префикс API: /public/api/v1

Технологии и стек
- Язык: Go
- HTTP роутер: go-chi/chi
- DI (внедрение зависимостей): uber-go/dig
- Сообщения/события: Kafka Producer (Sarama, при наличии KAFKA_BROKERS; иначе no-op)
- Хранилище: In-Memory репозиторий (без базы), см. internal/repository/order
- Архитектурные слои: domain, usecase, repository, gateway, handlers (упрощённая clean-структура)
- Тесты: стандартный testing, testify/assert, упрощённые gomock-совместимые моки
- OpenAPI: api_openapi.yaml (может расходиться с фактической реализацией в учебных целях)

Аутентификация и заголовки
- Для учебных сценариев поддерживаются заголовки:
  - X-User-ID — идентификатор пользователя (строка)
  - X-Bypass-Auth=true — режим обхода, симулирует пользователя "default-user"
- Важная особенность:
  - GET-ручки (получение заказа и статуса, а также список) не требуют обязательного X-User-ID. Если заголовок не передан, доступ к чтению не блокируется (публичный просмотр в учебных целях).
  - Модифицирующие операции (создание/обновление/удаление) используют переданный userID и проверяют владение.

HTTP ручки
Базовый префикс: /public/api/v1

1) Создать заказ
- POST /order
- Заголовки: рекомендуется X-User-ID или X-Bypass-Auth=true
- Тело (JSON):
  {
    "restaurant_id": "rest-1",
    "items": [{"food_id":"f1","name":"Pizza","quantity":1,"price":500}],
    "total_price": 500,
    "address": {"street": "Main"}
  }
- Ответ 201: объект заказа

Пример:
  curl -X POST http://localhost:8080/public/api/v1/order \
       -H 'Content-Type: application/json' \
       -H 'X-Bypass-Auth: true' \
       -d '{"restaurant_id":"rest-1","items":[{"food_id":"f1","name":"Pizza","quantity":1,"price":500}],"total_price":500,"address":{"street":"Main"}}'

2) Получить заказ по ID
- GET /order/{id}
- Заголовки: опционально X-User-ID или X-Bypass-Auth=true
- Ответ 200: объект заказа

Пример:
  curl http://localhost:8080/public/api/v1/order/ORDER_ID

3) Получить статус заказа
- GET /order/{id}/status
- Ответ 200: {"order_id":"...","status":"..."}

Пример:
  curl http://localhost:8080/public/api/v1/order/ORDER_ID/status

4) Список заказов с момента времени
- GET /orders?from=RFC3339
- Параметры: from — ISO/RFC3339-строка (по умолчанию 1970-01-01T00:00:00Z)
- Ответ 200: массив заказов

Пример:
  curl 'http://localhost:8080/public/api/v1/orders?from=1970-01-01T00:00:00Z'

5) Обновить заказ
- PUT /order/{id}
- Заголовки: X-User-ID или X-Bypass-Auth=true
- Тело (JSON): любые изменяемые поля (fio, items, total_price, address)
- Ответ 200: объект заказа

Пример:
  curl -X PUT http://localhost:8080/public/api/v1/order/ORDER_ID \
       -H 'Content-Type: application/json' \
       -H 'X-Bypass-Auth: true' \
       -d '{"fio":"Ivanov I.I."}'

6) Удалить заказ
- DELETE /order/{id}
- Заголовки: X-User-ID или X-Bypass-Auth=true
- Ответ 200: {"id":"...","status":"deleted"}

Пример:
  curl -X DELETE http://localhost:8080/public/api/v1/order/ORDER_ID -H 'X-Bypass-Auth: true'

7) Отладочное заполнение данными (seed)
- POST /debug/seed
- Заголовки: X-User-ID или X-Bypass-Auth=true
- Создаёт N=10 демо-заказов для текущего пользователя
- Ответ 201: массив созданных заказов

Пример:
  curl -X POST http://localhost:8080/public/api/v1/debug/seed -H 'X-Bypass-Auth: true'

Замечания по поведению
- Статусы заказов автоматически прогрессируют во времени фоновой задачей (см. cmd/service/main.go): created → pending → confirmed → cooking → delivering → completed с учебными интервалами.
- В In-Memory репозитории данные живут только в памяти процесса.

Файлы и полезные ссылки
- OpenAPI: api_openapi.yaml
- Тесты: internal/handlers/*_test.go, internal/usecase/order/*_test.go
- Makefile: цели run, logs, down, test, lint
- Docker Compose: docker-compose.yaml

Лицензия и ответственность
- Код предоставлен «как есть» исключительно для учебных целей. Используйте на свой страх и риск.