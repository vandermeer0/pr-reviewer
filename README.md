### Запуск:

```bash
git clone https://github.com/vandermeer0/pr-reviewer.git
cd pr-reviewer

go test ./...
make lint

docker compose up --build
```

API: http://localhost:8080

health‑чеки: GET /health или GET /healthz

PostgreSQL: postgres://postgres:postgres@db:5432/pr-reviewer

## Структура проекта

* `cmd/app/main.go` - входная точка
* `internal/entity` - доменные сущности (User, Team, PullRequest)
* `internal/usecase` - доменные сервисы:
  * `TeamService` - создание и получение команд
  * `UserService` - активация / деактивация пользователя
  * `PullRequestService` - создание PR, merge, перевыбор ревьюеров, выборка по ревьюеру
  * `StatsService` - статистика по ревьюерам
  * `TeamMaintenanceService` - массовая деактивация и перераспределение ревью
* `internal/usecase/repo` - интерфейсы репозиториев и доменные ошибки
* `internal/infrastructure/repository/postgresql` - реализация репозиториев поверх PostgreSQL.
* `internal/transport/httpapi` - HTTP‑слой на gin, DTO и маппинг ошибок доменного слоя в HTTP‑ответы
* `internal/integration` - интеграционный тест, поднимающий PostgreSQL и гоняющий основные сценарии

### Возникшие вопросы:

#### 1. Если в команде автора меньше двух подходящих ревьюеров: 
0 - PR без ревьюеров;
1 - один ревьюер

#### 2. Если автор в команде один:
PR создаётся, но без ревьюеров

#### 3. Когда кандидатов на перевыбор ревьюера нет:
Возвращаю доменную ошибку NO_CANDIDATE и HTTP 409, PR со старым ревьюером 

#### 4. Перевыбор ревьюера, когда PR уже смержен:

Возвращаю доменную ошибку PR_MERGED HTTP 409, ревьюеры не трогаются

#### 5. Поведение merge, если PR уже смержен:

Возвращаю текущий PR без ошибок и без доп апдейта в базу

#### 6. Ошибки при создании команд:

Если команда с таким именем уже есть в репозитории TEAM_EXISTS и HTTP 409

#### 7. Массовая деактивация команды:

TeamMaintenanceService.DeactivateTeamMembers и один большой SQL в транзакции:
деактивирую всех пользователей,
удаляю их назначения ревью в открытых PR,
если ревьюеров в затронутом PR меньше двух, пытаюсь добрать новых,
смерженные PR не трогаю

#### 8. Когда пользователь существует, но ни разу не был ревьювером

Возвращаю пустой список PR и HTTP 200

#### 9. Что если команда создалась, а один из пользователей не сохранился?

В идеале бы обернуть оба шага в одну транзакцию БД Сейчас оставил простую реализацию где такая ситуация возможна

### Тесты

```bash
go test ./...
```

Покрытие:

* `internal/usecase/services_impl_test.go` - unit‑тесты доменных сервисов
* `internal/usecase/team_maintenance_test.go` - тесты `TeamMaintenanceService.DeactivateTeamMembers` поверх настоящей БД
* `internal/integration/integration_test.go` - интеграционный сценарий, который создаёт команды, пользователей, PR, делает переназначение и merge и проверяет, что все работает


### Нагрузочное тестирование k6

```bash
$ k6 run loadtest/pr_flow.js

  ✓ http_req_duration..............: p(95)=11.18ms (threshold: p(95)<300ms)
  ✓ http_req_failed................: 0.00% (threshold: rate<0.001)

  checks_total.......: 901
  checks_succeeded...: 100.00%
  checks_failed......: 0.00%

  http_req_duration..: avg=5.12ms min=1.6ms med=2.93ms max=94.84ms
  http_reqs..........: 901  15.47/s
```

### Линтер

Линтер настроен через `golangci-lint` и конфигурацию `.golangci.yml`:

```yaml
version: "2"

run:
  timeout: 5m
  tests: true

linters:
  disable-all: true
  enable:
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused
    - misspell
    - revive
    - bodyclose

linters-settings:
  revive:
    rules:
      - name: exported
        arguments: [disableStuttering]
```


errcheck - не даёт забывать проверять ошибки;
govet, staticcheck, ineffassign, unused - поиск баг‑паттернов;
misspell - ловит опечатки в комментариях;
revive - проверяет стиль, наличие комментариев к сущностям;

```bash
make lint
```
