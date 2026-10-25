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
)

// testApp создает App с config.yaml во временной папке и сборщиком строк лога вместо Wails.
func testApp(t *testing.T, configYAML string) (*App, *[]string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if configYAML != "" {
		if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var (
		mu    sync.Mutex
		lines []string
	)
	app := &App{
		ctx:        context.Background(),
		configPath: path,
		emit: func(line string) {
			mu.Lock()
			lines = append(lines, line)
			mu.Unlock()
		},
	}
	return app, &lines
}

func hostPort(t *testing.T, srv *httptest.Server) (string, string) {
	t.Helper()
	host, port, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	return host, port
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

func TestLoadSettingsWithoutFile(t *testing.T) {
	app, _ := testApp(t, "")
	s, err := app.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.LMCZ.Port != "5998" || s.ClientRequestTimeout != "10s" {
		t.Errorf("ожидались значения по умолчанию: %+v", s)
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

	app, lines := testApp(t, fmt.Sprintf(`client_request_timeout: "5s"
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
	requireLine(t, *lines, "Проверка завершена успешно, (200) status: not_configured")

	if err := app.LmczInit(); err != nil {
		t.Fatal(err)
	}
	if gotToken != "tok-123" || gotUser != "admin" {
		t.Errorf("сервер получил token=%q user=%q", gotToken, gotUser)
	}
	requireLine(t, *lines, "Server not configured. Sending token....")
	requireLine(t, *lines, "Успех (200)!")
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

	app, lines := testApp(t, fmt.Sprintf(`client_request_timeout: "5s"
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
		t.Fatalf("%v\n%s", err, strings.Join(*lines, "\n"))
	}
	if gotUser != "tur" {
		t.Errorf("Basic Auth не передан, user=%q", gotUser)
	}
	requireLine(t, *lines, "Модуль 1С в конфигурации Apache")
	requireLine(t, *lines, "-- data: ok from 1C")

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
