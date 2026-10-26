# golm-gui

Оконная версия golm на [Wails v2](https://wails.io) + Svelte 5.
Логика проверок не дублируется: GUI вызывает те же пакеты `internal/*`, что и консольная утилита.

## Возможности

- **DataMobile** — то же, что `golm -dm`.
- **ЛМ ЧЗ** — «Проверить статус» (`golm -status`) и «Инициализировать» (`golm -token`).
- **Настройки** — редактирование `config.yaml`. Пароли и токен скрыты, пока не нажать «Показать».
  Комментарии в файле при сохранении остаются на месте.
- Ход проверки выводится в окно лога теми же строками, что CLI печатает в консоль.
- Под кнопками — список шагов проверки со статусами: ✓ успешно, ! замечание
  (например, медленный ответ), ✗ ошибка, – пропущен. Шаги объявляются в Go
  (`internal/report`), GUI только отображает то, что пришло событием `step`.

`config.yaml` ищется в таком порядке:
1. путь из переменной окружения `GOLM_CONFIG`;
2. рядом с `golm-gui.exe`;
3. в текущей папке.

Если файл не найден, форма настроек открывается со значениями по умолчанию, а «Сохранить» создает его рядом с exe.

## Разработка

Нужны Go 1.26+, Node.js 20+ и Wails CLI:

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails doctor   # проверит, что всё установлено (на Windows нужен WebView2 — в Windows 10/11 он уже есть)
```

Запуск с горячей перезагрузкой интерфейса (из папки `gui/`):

```sh
# PowerShell
$env:GOLM_CONFIG="..\config.yaml"; wails dev
# bash
GOLM_CONFIG=../config.yaml wails dev
```

## Сборка

```sh
cd gui
wails build                            # под текущую ОС
wails build -platform windows/amd64    # exe для Windows (собирается и из Linux/WSL)
```

Результат — `gui/build/bin/golm-gui.exe`, один файл. Рядом с ним нужно положить `config.yaml`.

## Тесты

```sh
cd gui
go test ./...
```

Тесты поднимают локальные HTTP-серверы вместо Apache/1С и ЛМ ЧЗ и проверяют
сохранение настроек, проверку DataMobile, статус и инициализацию ЛМ ЧЗ.

## Устройство

```
gui/
├── main.go          # запуск окна Wails, встраивание frontend/dist в exe
├── app.go           # методы, доступные из JS: LoadSettings, SaveSettings,
│                    # CheckDataMobile, LmczStatus, LmczInit
├── logwriter.go     # io.Writer: режет вывод клиентов golm на строки → событие "log"
├── app_test.go
├── go.mod           # отдельный модуль, чтобы `go build ./...` CLI не требовал Wails
└── frontend/
    ├── src/App.svelte       # вкладки, кнопки, итог проверки, лог
    ├── src/Settings.svelte  # форма config.yaml
    └── wailsjs/             # генерируется Wails (обертки над методами App)
```

GUI — отдельный Go-модуль (`github.com/dr2cc/golm/gui`), подключающий основной через
`replace github.com/dr2cc/golm => ../`. Так зависимости Wails не попадают в CLI,
а сборка CLI на Linux/WSL не требует библиотек GTK/WebKit.
