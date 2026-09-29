# Архитектура MasterBook

## 1. Общая схема

```text
                         WEB CLIENT
                    HTML / CSS / JavaScript
                              |
                              | HTTP / JSON
                              v
+------------------------------------------------+
|              PRESENTATION LAYER               |
|                                                |
|  HTTP Handlers                                 |
|  - чтение request                              |
|  - валидация формата                          |
|  - вызов Service                               |
|  - формирование HTTP response                  |
+---------------------------+--------------------+
                            |
                            v
+------------------------------------------------+
|                BUSINESS LAYER                  |
|                                                |
|  Services                                      |
|  - правила предметной области                  |
|  - сценарии операций                           |
|  - проверки                                    |
|  - расчёт времени                              |
|  - изменение состояния записи                  |
+---------------------------+--------------------+
                            |
                     interfaces
                            |
                            v
+------------------------------------------------+
|                 DATA ACCESS LAYER              |
|                                                |
|  Repository interfaces + PostgreSQL           |
|  - SQL-запросы                                 |
|  - транзакции                                  |
|  - получение и сохранение данных               |
+---------------------------+--------------------+
                            |
                            v
                       PostgreSQL
```

## 2. Назначение слоёв

### Handler

Handler является входной точкой HTTP-запроса. Он:

- получает параметры пути, query-параметры и JSON;
- получает идентификатор пользователя из authentication context;
- вызывает метод Service;
- преобразует ошибку Service в HTTP-статус;
- формирует JSON-ответ.

Handler не содержит SQL и не знает деталей PostgreSQL.

### Service

Service содержит бизнес-логику. Здесь находятся правила, которые описывают поведение системы независимо от HTTP и PostgreSQL.

Например, для создания записи Service:

1. получает длительность выбранной услуги;
2. рассчитывает время окончания;
3. проверяет рабочий интервал мастера;
4. вызывает Repository для атомарного создания записи;
5. инициирует уведомление мастеру.

### Repository

Repository инкапсулирует доступ к данным. Интерфейс объявляется рядом с бизнес-логикой, а реализация PostgreSQL находится в `postgres_repository.go`.

Это позволяет Service зависеть от абстракции, а не от конкретного драйвера PostgreSQL.

## 3. Компоновка зависимостей

Все реализации связываются в `cmd/server/main.go`:

```text
PostgreSQL connection
       |
       +--> UserRepository --> AuthService --> AuthHandler
       |
       +--> MasterRepository --> MasterService --> MasterHandler
       |
       +--> ServiceRepository --> ServiceService --> ServiceHandler
       |
       +--> ScheduleRepository --> ScheduleService --> ScheduleHandler
       |
       +--> AppointmentRepository --> AppointmentService --> AppointmentHandler
       |
       +--> NotificationRepository --> NotificationService --> NotificationHandler
       |
       +--> AdminService(AuthService, MasterService) --> AdminHandler
```

Модуль `admin` не имеет своего репозитория — он оркестрирует уже существующие
`AuthService` (создание пользователя с ролью master) и `MasterService` (создание
профиля) через узкие интерфейсы, объявленные в `admin/service.go`.

## 4. Сценарий создания записи

```text
Client
  |
  | POST /api/v1/appointments
  v
AppointmentHandler
  |
  | Create(clientID, masterID, serviceID, startTime)
  v
AppointmentService
  |
  +--> получает длительность услуги
  |
  +--> рассчитывает endTime
  |
  +--> проверяет рабочие часы
  |
  v
AppointmentRepository
  |
  +--> BEGIN
  +--> SELECT master FOR UPDATE
  +--> проверяет услугу мастера
  +--> проверяет пересечение интервалов
  +--> INSERT appointment
  +--> COMMIT
  |
  v
AppointmentService
  |
  +--> NotificationService.Create(...)
  |
  v
AppointmentHandler
  |
  +--> HTTP 201 + JSON
```

Блокировка `FOR UPDATE` и повторная проверка пересечения выполняются непосредственно перед вставкой. Это защищает сценарий от конкурентного бронирования одного и того же временного интервала.

## 5. Почему выбран модульный монолит

Проект реализован как один Go-сервис с функциональными модулями. Для дипломной системы это позволяет сохранить простоту развёртывания и одновременно разделить ответственность внутри backend.

Отдельные сервисы и брокеры сообщений не добавляются, потому что они не требуются текущему пользовательскому сценарию и усложнили бы проект без существенной пользы.
