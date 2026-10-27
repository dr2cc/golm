# golm

Консольная утилита на Go для диагностики связки DataMobile–Apache–1С и проверки/инициализации локального модуля «Честного знака» (ЛМ ЧЗ).

## Возможности

Настройки читаются из `config.yaml`, флаги командной строки могут их переопределить. За один запуск выполняется одна команда:

| Флаг | Что делает |
|---|---|
| `-dm` | Пошаговая диагностика DataMobile: HEAD-запрос к Apache с разбором типичных ошибок (DNS, connection refused, в том числе код Windows 10061, таймаут), затем проверка локальной публикации 1С в `httpd.conf` и HTTP-сервиса обмена DataMobile (Basic Auth). |
| `-status` | Запрос состояния ЛМ ЧЗ через `GET /api/v2/status`. |
| `-token` | Инициализация ЛМ ЧЗ: сначала проверяется статус, и только если сервер в состоянии `not_configured`, токен отправляется на `POST /api/v2/init`. |

Если хост ЛМ ЧЗ в конфиге не указан, встроенный сканер сам ищет его в подсети `/24`: параллельно проверяет порт (по умолчанию 5998) и берёт первый ответивший адрес.

## Запуск

Сначала создайте свой `config.yaml` из шаблона и заполните его. Файл в `.gitignore`, поэтому логины, пароли и токен в репозиторий не попадут:

```sh
cp config.example.yaml config.yaml      # Windows: copy config.example.yaml config.yaml
```

```sh
go run ./cmd/golm -dm
go run ./cmd/golm -status
go run ./cmd/golm -token
```

## Графический интерфейс

В папке [`gui/`](gui/README.md) лежит оконная версия на Wails + Svelte с теми же проверками и редактором `config.yaml`.
Консольная утилита от GUI не зависит и собирается как раньше.

```sh
cd gui
wails build -platform windows/amd64   # → gui/build/bin/golm-gui.exe
```

## Разработка

Логика проверок живет в `internal/*` и общая для CLI и GUI: изменение в ней сразу работает в обоих.

```
internal/datamobile, internal/lmcz, ...  ← логика проверок (общая)
internal/config                          ← config.yaml: Load / Save / New (+ флаги CLI)
internal/report                          ← шаги и их статусы для GUI
cmd/golm + internal/app                  ← CLI: флаги → вызов клиента
gui/app.go                               ← GUI: кнопка → вызов того же клиента
```

`gui/` — отдельный Go-модуль. Он подключает основной через `replace github.com/dr2cc/golm => ../`,
поэтому правки в `internal/*` видны в GUI сразу, без публикации и смены версий.

### Каждый день

```sh
# CLI (из корня, config.yaml лежит в корне)
go run ./cmd/golm -dm
go vet ./... && go test ./...

# GUI с автоперезапуском: wails dev следит и за .go, и за .svelte
cd gui
GOLM_CONFIG=../config.yaml wails dev            # bash
$env:GOLM_CONFIG="..\config.yaml"; wails dev    # PowerShell
go test ./...
```

`wails dev` при изменении Go пересобирает приложение и заново генерирует `gui/frontend/wailsjs`
(JS-обертки над методами `App`) — руками их не правим.

VS Code с расширением Go видит оба модуля, если открыть корень репозитория. `go.work` создавать не нужно:
с ним `go build ./...` в корне начнет собирать и GUI, а под Linux/WSL для этого нужны GTK/WebKit.

### Правила для кода в `internal/*`

1. **Никакого `fmt.Printf` / `fmt.Println`** — только `fmt.Fprintf(c.out, ...)`. Иначе строка появится в консоли CLI, но не в окне лога GUI.
2. **Никаких `log.Fatal`, `os.Exit`, `flag.*`** — они закроют окно GUI. Ошибку возвращаем (`return err`), решение о выходе принимает `cmd/golm`.
3. **Ошибку возвращать, а не только печатать** — по ней GUI ставит ✗.
4. **Новое поле в конфиге** добавляется:
   - в структуру `internal/config/config.go`;
   - в `config.example.yaml`;
   - если поле должно редактироваться в GUI — в `Settings` и `LoadSettings` / `SaveSettings` (`gui/app.go`) и в список `groups` в `gui/frontend/src/Settings.svelte`.

### Как добавить новую проверку

Пример — новый эндпоинт ЛМ ЧЗ.

**1. Логика** в `internal/lmcz`:

```go
func (c *CzClient) Version(ctx context.Context) (string, error) {
	steps := report.NewTracker(c.onStep, stepFind, report.Step{ID: "version", Title: "Версия сервера"})

	hostPort, err := c.findServer(steps) // готовый шаг «Поиск сервера»
	if err != nil {
		return "", err
	}

	steps.Start("version")
	fmt.Fprintf(c.out, "Запрашиваем версию: %s\n", hostPort) // в консоль или в окно GUI
	v, err := c.getVersion(ctx, hostPort)                    // обычный HTTP-запрос
	if err != nil {
		steps.Done("version", report.Fail, err.Error())
		return "", err
	}
	steps.Done("version", report.OK, v)
	return v, nil
}
```

Если статусы по шагам не нужны, `Tracker` можно не использовать: в GUI будут лог и общий итог.

**2. CLI** — флаг в `config.New()` (`flag.BoolVar(&cfg.LaunchLmczVersion, "version", ...)`)
и ветка `if cfg.LaunchLmczVersion { ... }` в `internal/app/app.go`.

**3. GUI** — метод в `gui/app.go`:

```go
func (a *App) LmczVersion() (string, error) {
	var v string
	err := a.run("Запрос версии ЛМ ЧЗ...", func(ctx context.Context, cfg *config.Config, out *lineWriter) error {
		var err error
		v, err = lmcz.NewCzClient(cfg.LMCZ, newHTTPClient()).SetOutput(out).SetStepHandler(a.emitStep).Version(ctx)
		return err
	})
	return v, err
}
```

и кнопка в `gui/frontend/src/App.svelte`: `exec('version', LmczVersion, (v) => v)`, а рядом
`{@render result('version')}` и `{@render stepList('version')}`. Функцию `LmczVersion` импортируем
из `../wailsjs/go/main/App.js` — ее сгенерирует `wails dev`.

**4. Тесты** — сервер подменяется `httptest.NewServer`, по образцу `TestLmczStatusAndInit` в `gui/app_test.go`.
Так проверяются сразу логика, строки лога и статусы шагов: `requireSteps(t, rec, "find=ok version=ok")`.

## Структура проекта

```
golm/
├── config.example.yaml   # Шаблон настроек (скопировать в config.yaml)
├── config.yaml           # Ваши настройки (в .gitignore, в репозиторий не попадает)
├── gui/                  # Оконная версия (Wails + Svelte), отдельный Go-модуль
│
├── cmd/
│   └── golm/
│       └── main.go       # Вызывает app.Run()
├── internal/
│   ├── app/
│   │   └── app.go        # Инициализирует клиенты и запускает логику (Run)
│   │
│   ├── config/
│   │   └── config.go     # Парсинг конфигурации
│   │
│   │── datamobile/       # Область ответственности: DataMobile-Apache-1C
│   │   ├── client.go     # Логика запросов, методы (например, GetProducts)
│   │   └── models.go     # Структуры данных (DTO) для этого сервиса
│   │
│   ├── lmcz/             # Область ответственности: ЛМ ЧЗ
│   │   ├── client.go     # Логика запросов к ЧЗ
│   │   └── models.go     # Структуры данных для ЧЗ
│   │
│   ├── scanner/
│   │   └── scanner.go    # Вспомогательный сканер (находит IP)
│   │
│   └── transport/        # (Опционально) Общий HTTP-клиент, если нужен
│       └── http.go       # Настройки таймаутов, логирования, retry-логики

    ├── client/
    │   ├── client.go     # Функция Run(), конструктор NewClient() и общая структура
    │   ├── server.go     # Эндпоинт получения информации о сервере
    │   └── regLm.go      # Эндпоинт регистрации ЛМ ЧЗ
    │

```
