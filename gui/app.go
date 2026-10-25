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

	"github.com/dr2cc/golm/internal/cmdutil"
	"github.com/dr2cc/golm/internal/config"
	"github.com/dr2cc/golm/internal/datamobile"
	"github.com/dr2cc/golm/internal/lmcz"
)

// App — мост между фронтендом и логикой golm.
// Каждый публичный метод App Wails делает доступным из JS (frontend/wailsjs/go/main/App.js).
// Сама логика проверок не дублируется: вызываются те же клиенты, что и в CLI (internal/app).
type App struct {
	ctx        context.Context
	configPath string

	// emit отправляет одну строку лога во фронтенд (событие "log").
	// Вынесено в поле, чтобы тесты могли подставить свой сборщик строк вместо Wails.
	emit func(line string)

	// Одновременно выполняется только одна операция:
	// так лог в окне не перемешивается, а сеть не сканируется параллельно дважды.
	busy sync.Mutex
}

func NewApp() *App {
	return &App{configPath: resolveConfigPath()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.emit = func(line string) { runtime.EventsEmit(ctx, "log", line) }
}

// resolveConfigPath ищет config.yaml:
//  1. путь из переменной окружения GOLM_CONFIG (удобно при разработке: GOLM_CONFIG=../config.yaml wails dev);
//  2. рядом с exe — основной вариант для пользователя;
//  3. в текущей папке — как у CLI.
//
// Если файл не найден нигде, используется путь рядом с exe: туда его создаст «Сохранить».
func resolveConfigPath() string {
	if p := os.Getenv("GOLM_CONFIG"); p != "" {
		return p
	}

	exeDir := "."
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}
	candidates := []string{
		filepath.Join(exeDir, config.DefaultFileName),
		config.DefaultFileName,
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if abs, err := filepath.Abs(p); err == nil {
				return abs
			}
			return p
		}
	}
	return candidates[0]
}

// ---------- Настройки ----------

// Settings — то, что видит и редактирует форма «Настройки».
// Отличие от config.Config: длительности передаются строками ("10s", "500ms"),
// потому что time.Duration в JSON превратился бы в число наносекунд.
type Settings struct {
	ConfigPath           string             `json:"configPath"`
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
func (a *App) LoadSettings() (Settings, error) {
	cfg, err := config.Load(a.configPath)
	if errors.Is(err, os.ErrNotExist) {
		cfg = &config.Config{
			ClientRequestTimeout: 10 * time.Second,
			Apache:               config.ApacheConfig{Address: "localhost", WarningDuration: 2 * time.Second},
			LMCZ:                 config.LmczConfig{Port: "5998", ScannerTimeout: 500 * time.Millisecond},
		}
	} else if err != nil {
		return Settings{ConfigPath: a.configPath}, err
	}
	return Settings{
		ConfigPath:           a.configPath,
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
	cfg, err := config.Load(a.configPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		cfg = &config.Config{}
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
		client.SetOutput(out)
		return client.Check(ctx)
	})
}

// LmczStatus — аналог `golm -status`.
func (a *App) LmczStatus() (LmczStatusResult, error) {
	var result LmczStatusResult
	err := a.run("Запуск проверки ЛМ ЧЗ...", func(ctx context.Context, cfg *config.Config, out *lineWriter) error {
		checkResult, err := lmcz.NewCzClient(cfg.LMCZ, newHTTPClient()).SetOutput(out).Status(ctx)
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
		return lmcz.NewCzClient(cfg.LMCZ, newHTTPClient()).SetOutput(out).Init(ctx)
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

	cfg, err := config.Load(a.configPath)
	if err != nil {
		return err
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
