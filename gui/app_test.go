package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dr2cc/golm/internal/report"
)

// recorder собирает то, что App отправил бы во фронтенд: строки лога и статусы шагов.
type recorder struct {
	mu    sync.Mutex
	lines []string
	steps map[string]report.Step // последнее состояние каждого шага
	order []string               // порядок шагов, как их объявил Tracker
}

// reset очищает собранные шаги перед следующей операцией.
func (r *recorder) reset() {
	r.mu.Lock()
	r.steps, r.order = map[string]report.Step{}, nil
	r.mu.Unlock()
}

func (r *recorder) finalSteps() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var parts []string
	for _, id := range r.order {
		parts = append(parts, id+"="+string(r.steps[id].State))
	}
	return strings.Join(parts, " ")
}

// testApp создает App с config.yaml во временной папке и сборщиком событий вместо Wails.
func testApp(t *testing.T, configYAML string) (*App, *recorder) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if configYAML != "" {
		if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rec := &recorder{steps: map[string]report.Step{}}
	app := &App{
		ctx:        context.Background(),
		configPath: path,
		emit: func(line string) {
			rec.mu.Lock()
			rec.lines = append(rec.lines, line)
			rec.mu.Unlock()
		},
		emitStep: func(st report.Step) {
			rec.mu.Lock()
			if _, seen := rec.steps[st.ID]; !seen {
				rec.order = append(rec.order, st.ID)
			}
			rec.steps[st.ID] = st
			rec.mu.Unlock()
		},
	}
	return app, rec
}

func hostPort(t *testing.T, srv *httptest.Server) (string, string) {
	t.Helper()
	host, port, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	return host, port
}

func requireSteps(t *testing.T, rec *recorder, want string) {
	t.Helper()
	if got := rec.finalSteps(); got != want {
		t.Errorf("шаги: got %q, want %q", got, want)
	}
}

func requireLine(t *testing.T, lines []string, want string) {
	t.Helper()
	for _, l := range lines {
		if strings.Contains(l, want) {
			return
		}
	}
	t.Errorf("в логе нет строки с %q:\n%s", want, strings.Join(lines, "\n"))
}

func TestLineWriterSplitsLines(t *testing.T) {
	var got []string
	w := newLineWriter(func(l string) { got = append(got, l) })

	fmt.Fprint(w, "-- Время ответа: 1s")
	fmt.Fprint(w, " в норме\r\nвторая\n")
	fmt.Fprint(w, "без перевода строки")
	w.Flush()

	want := []string{"-- Время ответа: 1s в норме", "вторая", "без перевода строки"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	app, _ := testApp(t, `client_request_timeout: "10s" # комментарий
launch_datamobile: true
apache:
  address: "localhost"
  warning_duration: "2s"
lmcz:
  port: "5998"
  scanner_timeout: "500ms"
`)
	s, err := app.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.ClientRequestTimeout != "10s" || s.LMCZ.ScannerTimeout != "500ms" {
		t.Fatalf("неверно прочитаны длительности: %+v", s)
	}

	s.LMCZ.Host = " 10.0.0.5 "
	s.ClientRequestTimeout = "30s"
	if err := app.SaveSettings(s); err != nil {
		t.Fatal(err)
	}

	raw, _ := os.ReadFile(app.configPath)
	if !strings.Contains(string(raw), "# комментарий") || !strings.Contains(string(raw), "launch_datamobile: true") {
		t.Errorf("комментарий или поле вне формы потеряно:\n%s", raw)
	}
	s2, err := app.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s2.LMCZ.Host != "10.0.0.5" || s2.ClientRequestTimeout != "30s" {
		t.Errorf("значения не сохранились: %+v", s2)
	}

	s2.Apache.WarningDuration = "две секунды"
	if err := app.SaveSettings(s2); err == nil {
		t.Error("ожидалась ошибка для неверной длительности")
	}
}

// [GUI-без-файла] Без файла форма получает значения из встроенного config.example.yaml.
func TestLoadSettingsWithoutFile(t *testing.T) {
	app, _ := testApp(t, "")
	s, err := app.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.ConfigSaved {
		t.Error("файла нет, а ConfigSaved = true")
	}
	if s.LMCZ.Port != "5998" || s.ClientRequestTimeout != "10s" || s.Apache.PublicationName != "tradeProfil11" {
		t.Errorf("ожидались значения из config.example.yaml: %+v", s)
	}
}

// [GUI-без-файла] Первое сохранение создает папку и файл на основе шаблона — с его комментариями.
func TestFirstSaveCreatesFileFromTemplate(t *testing.T) {
	app, _ := testApp(t, "")
	app.configPath = filepath.Join(t.TempDir(), "AppData", "Roaming", "golm", "config.yaml")

	s, err := app.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	s.LMCZ.Host = "192.168.4.220"
	if err := app.SaveSettings(s); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(app.configPath)
	if err != nil {
		t.Fatalf("файл не создан: %v", err)
	}
	for _, want := range []string{"# Схемы не нужны!", `host: "192.168.4.220"`, "launch_datamobile: false"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("в созданном файле нет %q:\n%s", want, raw)
		}
	}

	s2, err := app.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !s2.ConfigSaved || s2.LMCZ.Host != "192.168.4.220" {
		t.Errorf("после сохранения ожидался файл с новым хостом: %+v", s2)
	}
}

// [GUI-без-файла] Проверка запускается и без файла — со значениями по умолчанию.
func TestRunWithoutFileUsesDefaults(t *testing.T) {
	app, rec := testApp(t, "")
	app.LmczStatus() // localhost:5998 обычно закрыт — важна не удача запроса, а что он был сделан

	requireLine(t, rec.lines, "используются значения по умолчанию")
	requireLine(t, rec.lines, "Проверяем статус ЛМ ЧЗ по адресу: http://localhost:5998/api/v2/status")
	for _, l := range rec.lines {
		if strings.Contains(l, "файл конфигурации") {
			t.Errorf("проверка не должна требовать файл: %q", l)
		}
	}
}

func TestResolveConfigPath(t *testing.T) {
	exeDir := t.TempDir()
	appData := filepath.Join(t.TempDir(), "Roaming")

	// Нет ни переменной, ни файла рядом с exe — профиль пользователя
	if got, want := resolveConfigPath("", exeDir, appData), filepath.Join(appData, "golm", "config.yaml"); got != want {
		t.Errorf("профиль: got %q, want %q", got, want)
	}

	// Файл рядом с exe — портативный режим
	portable := filepath.Join(exeDir, "config.yaml")
	os.WriteFile(portable, []byte("env: x\n"), 0o600)
	if got := resolveConfigPath("", exeDir, appData); got != portable {
		t.Errorf("портативный: got %q, want %q", got, portable)
	}

	// GOLM_CONFIG важнее всего
	if got := resolveConfigPath("/dev/golm.yaml", exeDir, appData); got != "/dev/golm.yaml" {
		t.Errorf("GOLM_CONFIG: got %q", got)
	}

	// Папку профиля определить не удалось — рядом с exe
	if got := resolveConfigPath("", t.TempDir(), ""); filepath.Base(got) != "config.yaml" {
		t.Errorf("без профиля: got %q", got)
	}
}

func TestLmczStatusAndInit(t *testing.T) {
	var gotToken, gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/status":
			fmt.Fprint(w, `{"status":"not_configured"}`)
		case "/api/v2/init":
			var body struct{ Token string }
			json.NewDecoder(r.Body).Decode(&body)
			gotToken = body.Token
			gotUser, _, _ = r.BasicAuth()
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	host, port := hostPort(t, srv)

	app, rec := testApp(t, fmt.Sprintf(`client_request_timeout: "5s"
lmcz:
  host: "%s"
  port: "%s"
  user: "admin"
  pass: "secret"
  token_x_api_key: "tok-123"
`, host, port))

	res, err := app.LmczStatus()
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "not_configured" || res.HttpCode != 200 {
		t.Errorf("неверный статус: %+v", res)
	}
	requireLine(t, rec.lines, "Проверка завершена успешно, (200) status: not_configured")
	requireSteps(t, rec, "find=ok status=ok")

	rec.reset()
	if err := app.LmczInit(); err != nil {
		t.Fatal(err)
	}
	if gotToken != "tok-123" || gotUser != "admin" {
		t.Errorf("сервер получил token=%q user=%q", gotToken, gotUser)
	}
	requireLine(t, rec.lines, "Server not configured. Sending token....")
	requireLine(t, rec.lines, "Успех (200)!")
	requireSteps(t, rec, "find=ok status=ok token=ok")
}

func TestCheckDataMobile(t *testing.T) {
	var gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, _, _ = r.BasicAuth()
		if r.URL.Path == "/base/hs/DataMobileExch/" {
			fmt.Fprint(w, `{"data":"ok from 1C"}`)
		}
	}))
	defer srv.Close()
	host, port := hostPort(t, srv)

	httpdConf := filepath.Join(t.TempDir(), "httpd.conf")
	os.WriteFile(httpdConf, []byte("LoadModule _1cws_module \"C:/1cv8/bin/wsap24.dll\"\n"), 0o600)

	app, rec := testApp(t, fmt.Sprintf(`client_request_timeout: "5s"
datamobile:
  user: "tur"
  pass: "pw"
apache:
  publication_name: "base"
  address: "%s"
  port: "%s"
  conf_path: %q
  warning_duration: "2s"
`, host, port, httpdConf))

	if err := app.CheckDataMobile(); err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(rec.lines, "\n"))
	}
	if gotUser != "tur" {
		t.Errorf("Basic Auth не передан, user=%q", gotUser)
	}
	requireLine(t, rec.lines, "Модуль 1С в конфигурации Apache")
	requireLine(t, rec.lines, "-- data: ok from 1C")
	requireSteps(t, rec, "apache=ok module1c=ok exchange=ok")

	// Повторный запуск: клиент создается заново, Basic Auth не «накапливается»
	if err := app.CheckDataMobile(); err != nil {
		t.Fatal(err)
	}
}

func TestOnlyOneOperationAtATime(t *testing.T) {
	app, _ := testApp(t, `client_request_timeout: "5s"`)
	app.busy.Lock()
	defer app.busy.Unlock()

	if err := app.LmczInit(); err == nil || !strings.Contains(err.Error(), "другая операция") {
		t.Errorf("ожидалась ошибка о занятости, получено %v", err)
	}
}

// dmApp поднимает тестовый «Apache + 1С» с заданным обработчиком и App, настроенный на него.
func dmApp(t *testing.T, warning string, h http.HandlerFunc) (*App, *recorder) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	host, port := hostPort(t, srv)

	httpdConf := filepath.Join(t.TempDir(), "httpd.conf")
	os.WriteFile(httpdConf, []byte("LoadModule _1cws_module \"x.dll\"\n"), 0o600)

	return testApp(t, fmt.Sprintf(`client_request_timeout: "5s"
apache:
  publication_name: "base"
  address: "%s"
  port: "%s"
  conf_path: %q
  warning_duration: %q
`, host, port, httpdConf, warning))
}

func TestDataMobileSteps(t *testing.T) {
	tests := []struct {
		name    string
		warning string
		handler http.HandlerFunc
		wantErr bool
		want    string
	}{
		{
			name:    "Apache отвечает 5xx — CLI считает это не ошибкой, но шаг Apache красный",
			warning: "2s",
			handler: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) },
			want:    "apache=fail module1c=skipped exchange=skipped",
		},
		{
			name:    "база не опубликована",
			warning: "2s",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					http.NotFound(w, r)
				}
			},
			wantErr: true,
			want:    "apache=ok module1c=ok exchange=fail",
		},
		{
			name:    "медленный ответ",
			warning: "1ms",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					time.Sleep(30 * time.Millisecond)
					fmt.Fprint(w, `{"data":"ok"}`)
				}
			},
			want: "apache=ok module1c=ok exchange=warn",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, rec := dmApp(t, tt.warning, tt.handler)
			err := app.CheckDataMobile()
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			requireSteps(t, rec, tt.want)
		})
	}
}

func TestLmczInitSteps(t *testing.T) {
	t.Run("сервер уже настроен — токен не отправляется", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v2/init" {
				t.Error("токен не должен отправляться")
			}
			fmt.Fprint(w, `{"status":"ready"}`)
		}))
		defer srv.Close()
		host, port := hostPort(t, srv)
		app, rec := testApp(t, fmt.Sprintf("client_request_timeout: \"5s\"\nlmcz: {host: %q, port: %q}\n", host, port))

		if err := app.LmczInit(); err != nil {
			t.Fatal(err)
		}
		requireSteps(t, rec, "find=ok status=ok token=skipped")
	})

	t.Run("сервер недоступен", func(t *testing.T) {
		l, _ := net.Listen("tcp", "127.0.0.1:0")
		_, port, _ := net.SplitHostPort(l.Addr().String())
		l.Close() // порт заведомо закрыт
		app, rec := testApp(t, fmt.Sprintf("client_request_timeout: \"5s\"\nlmcz: {host: \"127.0.0.1\", port: %q}\n", port))

		if err := app.LmczInit(); err == nil {
			t.Fatal("ожидалась ошибка")
		}
		requireSteps(t, rec, "find=ok status=fail token=skipped")
	})
}
