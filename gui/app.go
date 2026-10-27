package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	golm "github.com/dr2cc/golm"
	"github.com/dr2cc/golm/internal/cmdutil"
	"github.com/dr2cc/golm/internal/config"
	"github.com/dr2cc/golm/internal/datamobile"
	"github.com/dr2cc/golm/internal/lmcz"
	"github.com/dr2cc/golm/internal/report"
)

// App — мост между фронтендом и логикой golm.
// Каждый публичный метод App Wails делает доступным из JS (frontend/wailsjs/go/main/App.js).
// Сама логика проверок не дублируется: вызываются те же клиенты, что и в CLI (internal/app).
type App struct {
	ctx context.Context

	// configPath — файл, из которого читаются и в который сохраняются настройки.
	// [GUI-без-файла] Файла по этому пути может еще не быть: тогда используются
	// встроенные значения по умолчанию (см. loadConfig), а «Сохранить» создаст файл.
	configPath string

	// emit отправляет одну строку лога во фронтенд (событие "log").
	// Вынесено в поле, чтобы тесты могли подставить свой сборщик строк вместо Wails.
	emit func(line string)

	// emitStep отправляет во фронтенд изменение статуса шага (событие "step").
	// Как и emit, вынесено в поле ради тестов.
	emitStep report.Func

	// Одновременно выполняется только одна операция:
	// так лог в окне не перемешивается, а сеть не сканируется параллельно дважды.
	busy sync.Mutex
}

func NewApp() *App {
	exeDir := "."
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}
	// os.UserConfigDir на Windows возвращает %APPDATA% (C:\Users\<имя>\AppData\Roaming)
	userConfigDir, _ := os.UserConfigDir()

	return &App{configPath: resolveConfigPath(os.Getenv("GOLM_CONFIG"), exeDir, userConfigDir)}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.emit = func(line string) { runtime.EventsEmit(ctx, "log", line) }
	a.emitStep = func(s report.Step) { runtime.EventsEmit(ctx, "step", s) }
}

// [GUI-без-файла] resolveConfigPath выбирает файл настроек:
//  1. путь из переменной окружения GOLM_CONFIG — для разработки (GOLM_CONFIG=../config.yaml wails dev);
//  2. config.yaml рядом с exe, если он там есть, — «портативный» режим
//     (программа на флешке или раздается вместе с готовыми настройками);
//  3. %APPDATA%\golm\config.yaml — основной вариант: папка профиля всегда доступна на запись,
//     у каждого пользователя Windows свои настройки, и они переживают обновление exe.
//
// Раньше вариантом 3 была папка рядом с exe, а еще проверялась текущая папка.
// Текущая папка убрана: при запуске через ярлык она может оказаться любой.
//
// Функция не обращается к окружению сама — все берет из параметров, поэтому ее легко тестировать.
func resolveConfigPath(envPath, exeDir, userConfigDir string) string {
	if envPath != "" {
		return envPath
	}

	portable := filepath.Join(exeDir, config.DefaultFileName)
	if _, err := os.Stat(portable); err == nil {
		return portable
	}

	if userConfigDir != "" {
		return filepath.Join(userConfigDir, "golm", config.DefaultFileName)
	}
	// Папку профиля определить не удалось (на Windows такого практически не бывает)
	return portable
}

// [GUI-без-файла] loadConfig читает настройки из a.configPath.
// Если файла еще нет — это не ошибка: возвращаются значения из встроенного
// config.example.yaml, а saved = false (форма покажет, что настройки еще не сохранены).
func (a *App) loadConfig() (cfg *config.Config, saved bool, err error) {
	cfg, err = config.Load(a.configPath)
	if errors.Is(err, os.ErrNotExist) {
		cfg, err = config.Parse(golm.DefaultConfigYAML)
		return cfg, false, err
	}
	return cfg, err == nil, err
}

// ---------- Настройки ----------

// Settings — то, что видит и редактирует форма «Настройки».
// Отличие от config.Config: длительности передаются строками ("10s", "500ms"),
// потому что time.Duration в JSON превратился бы в число наносекунд.
type Settings struct {
	ConfigPath string `json:"configPath"`
	// [GUI-без-файла] ConfigSaved — файл настроек уже существует.
	// false: показаны встроенные значения по умолчанию, файл создастся при сохранении.
	ConfigSaved          bool               `json:"configSaved"`
	ClientRequestTimeout string             `json:"clientRequestTimeout"`
	Apache               ApacheSettings     `json:"apache"`
	DataMobile           DataMobileSettings `json:"datamobile"`
	LMCZ                 LmczSettings       `json:"lmcz"`
}

type ApacheSettings struct {
	PublicationName string `json:"publicationName"`
	Address         string `json:"address"`
	Port            string `json:"port"`
	ConfPath        string `json:"confPath"`
	WarningDuration string `json:"warningDuration"`
}

type DataMobileSettings struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

type LmczSettings struct {
	Host           string `json:"host"`
	Subnet         string `json:"subnet"`
	Port           string `json:"port"`
	ScannerTimeout string `json:"scannerTimeout"`
	User           string `json:"user"`
	Pass           string `json:"pass"`
	Token          string `json:"token"`
}

// LoadSettings читает config.yaml и отдает его форме.
// Если файла еще нет, форма получает значения по умолчанию — «Сохранить» создаст файл.
//
// [GUI-без-файла] Значения по умолчанию раньше были зашиты здесь в коде (только таймауты,
// адрес и порт). Теперь они берутся из встроенного config.example.yaml через loadConfig.
func (a *App) LoadSettings() (Settings, error) {
	cfg, saved, err := a.loadConfig()
	if err != nil {
		return Settings{ConfigPath: a.configPath}, err
	}
	return Settings{
		ConfigPath:           a.configPath,
		ConfigSaved:          saved,
		ClientRequestTimeout: cfg.ClientRequestTimeout.String(),
		Apache: ApacheSettings{
			PublicationName: cfg.Apache.PublicationName,
			Address:         cfg.Apache.Address,
			Port:            cfg.Apache.Port,
			ConfPath:        cfg.Apache.ConfPath,
			WarningDuration: cfg.Apache.WarningDuration.String(),
		},
		DataMobile: DataMobileSettings{
			User: cfg.DataMobile.User,
			Pass: cfg.DataMobile.Pass,
		},
		LMCZ: LmczSettings{
			Host:           cfg.LMCZ.Host,
			Subnet:         cfg.LMCZ.Subnet,
			Port:           cfg.LMCZ.Port,
			ScannerTimeout: cfg.LMCZ.ScannerTimeout.String(),
			User:           cfg.LMCZ.User,
			Pass:           cfg.LMCZ.Pass,
			Token:          cfg.LMCZ.TokenXAPIKEY,
		},
	}, nil
}

// SaveSettings проверяет значения формы и записывает их в config.yaml.
// Поля, которых нет в форме (env, launch_*), берутся из текущего файла и не меняются.
func (a *App) SaveSettings(s Settings) error {
	// [GUI-без-файла] Если файла еще нет, основой служат встроенные значения по умолчанию
	// (раньше — пустая структура: env и launch_* терялись бы).
	cfg, saved, err := a.loadConfig()
	if err != nil {
		return err
	}

	durations := []struct {
		name  string
		value string
		dst   *time.Duration
	}{
		{"Таймаут запроса", s.ClientRequestTimeout, &cfg.ClientRequestTimeout},
		{"Порог медленного ответа", s.Apache.WarningDuration, &cfg.Apache.WarningDuration},
		{"Таймаут сканера", s.LMCZ.ScannerTimeout, &cfg.LMCZ.ScannerTimeout},
	}
	for _, d := range durations {
		v, err := time.ParseDuration(strings.TrimSpace(d.value))
		if err != nil || v <= 0 {
			return fmt.Errorf("%s: %q — ожидается длительность больше нуля, например 10s или 500ms", d.name, d.value)
		}
		*d.dst = v
	}

	cfg.Apache.PublicationName = strings.TrimSpace(s.Apache.PublicationName)
	cfg.Apache.Address = strings.TrimSpace(s.Apache.Address)
	cfg.Apache.Port = strings.TrimSpace(s.Apache.Port)
	cfg.Apache.ConfPath = strings.TrimSpace(s.Apache.ConfPath)
	cfg.DataMobile.User = s.DataMobile.User
	cfg.DataMobile.Pass = s.DataMobile.Pass
	cfg.LMCZ.Host = strings.TrimSpace(s.LMCZ.Host)
	cfg.LMCZ.Subnet = strings.TrimSpace(s.LMCZ.Subnet)
	cfg.LMCZ.Port = strings.TrimSpace(s.LMCZ.Port)
	cfg.LMCZ.User = s.LMCZ.User
	cfg.LMCZ.Pass = s.LMCZ.Pass
	cfg.LMCZ.TokenXAPIKEY = strings.TrimSpace(s.LMCZ.Token)

	// [GUI-без-файла] Первое сохранение: создаем папку (%APPDATA%\golm) и кладем в файл
	// встроенный шаблон — тогда config.Save перенесет в него значения формы, а комментарии
	// из config.example.yaml останутся подсказками для того, кто откроет файл руками.
	if !saved {
		if err := os.MkdirAll(filepath.Dir(a.configPath), 0o700); err != nil {
			return fmt.Errorf("не удалось создать папку для настроек: %w", err)
		}
		if err := os.WriteFile(a.configPath, golm.DefaultConfigYAML, 0o600); err != nil {
			return fmt.Errorf("не удалось создать файл настроек %s: %w", a.configPath, err)
		}
	}

	return config.Save(a.configPath, cfg)
}

// ---------- Проверки ----------

// LmczStatusResult — ответ GET /api/v2/status для фронтенда.
type LmczStatusResult struct {
	Status   string `json:"status"`
	HttpCode int    `json:"httpCode"`
}

// CheckDataMobile — аналог `golm -dm`.
func (a *App) CheckDataMobile() error {
	return a.run("Запуск проверки DataMobile-Apache-1C...", func(ctx context.Context, cfg *config.Config, out *lineWriter) error {
		client := datamobile.NewDmClient(newHTTPClient(), datamobile.ClientConfig{
			Apache:     cfg.Apache,
			DataMobile: cfg.DataMobile,
		})
		client.SetOutput(out).SetStepHandler(a.emitStep)
		return client.Check(ctx)
	})
}

// LmczStatus — аналог `golm -status`.
func (a *App) LmczStatus() (LmczStatusResult, error) {
	var result LmczStatusResult
	err := a.run("Запуск проверки ЛМ ЧЗ...", func(ctx context.Context, cfg *config.Config, out *lineWriter) error {
		checkResult, err := lmcz.NewCzClient(cfg.LMCZ, newHTTPClient()).SetOutput(out).SetStepHandler(a.emitStep).Status(ctx)
		if err != nil {
			return err
		}
		result = LmczStatusResult{Status: checkResult.Status, HttpCode: checkResult.HttpCode}
		fmt.Fprintf(out, "Проверка завершена успешно, (%d) status: %s\n", checkResult.HttpCode, checkResult.Status)
		return nil
	})
	return result, err
}

// LmczInit — аналог `golm -token`.
func (a *App) LmczInit() error {
	return a.run("Отправка токена для инициализации ЛМ ЧЗ...", func(ctx context.Context, cfg *config.Config, out *lineWriter) error {
		return lmcz.NewCzClient(cfg.LMCZ, newHTTPClient()).SetOutput(out).SetStepHandler(a.emitStep).Init(ctx)
	})
}

// run — общая обвязка для всех проверок, повторяет то, что делает internal/app.Run для CLI:
// читает конфиг, ставит общий таймаут, направляет вывод клиентов в окно лога.
// Конфиг перечитывается при каждом запуске, чтобы сразу действовали сохраненные настройки.
func (a *App) run(title string, op func(ctx context.Context, cfg *config.Config, out *lineWriter) error) error {
	if !a.busy.TryLock() {
		return errors.New("другая операция еще выполняется, дождитесь ее завершения")
	}
	defer a.busy.Unlock()

	out := newLineWriter(a.emit)
	defer out.Flush()

	fmt.Fprintln(out, title)

	// [GUI-без-файла] Без файла настроек проверка все равно запускается — со встроенными
	// значениями по умолчанию (раньше здесь была ошибка «не удалось прочитать файл конфигурации»).
	cfg, saved, err := a.loadConfig()
	if err != nil {
		return err
	}
	if !saved {
		fmt.Fprintln(out, "Настройки еще не сохранены — используются значения по умолчанию.")
	}

	ctx, cancel := context.WithTimeout(a.ctx, cfg.ClientRequestTimeout)
	defer cancel()

	if err := op(ctx, cfg, out); err != nil {
		out.Flush()
		fmt.Fprintf(out, "Ошибка: %v\n", err)
		return err
	}
	return nil
}

// newHTTPClient создает новый HTTP-клиент на каждую операцию.
// Это важно: datamobile.NewDmClient изменяет Transport переданного клиента
// (добавляет Basic Auth), и общий клиент «обрастал» бы обертками при каждом запуске.
func newHTTPClient() *http.Client {
	client, _ := cmdutil.NewFactory().HttpClient()
	return client
}
