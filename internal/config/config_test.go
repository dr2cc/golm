package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// [GUI-рефакторинг] Тест для Save: значения должны сохраняться,
// а комментарии в config.yaml — оставаться на месте.
func TestSaveKeepsCommentsAndValues(t *testing.T) {
	const original = `env: "development"

# Флаги запуска
launch_datamobile: false
client_request_timeout: "10s"   # таймаут

apache:
  publication_name: "tradeProfil11"
  # tradeProfilDm / polyMark
  address: "192.168.0.75"
  port: ""
  conf_path: C:/Apache24/conf/httpd.conf
  warning_duration: "2s"

lmcz:
  host: "localhost"
  # "localhost" "sidingkas"
  port: "5998"
  scanner_timeout: "500ms"
`
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ClientRequestTimeout = 15 * time.Second
	cfg.Apache.Address = "localhost: weird # value"
	cfg.Apache.Port = "8080"
	cfg.LMCZ.Host = ""
	cfg.LMCZ.User = "new-user" // ключа не было в файле — должен добавиться

	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}

	saved, _ := os.ReadFile(path)
	for _, comment := range []string{"# Флаги запуска", "# таймаут", "# tradeProfilDm / polyMark", `# "localhost" "sidingkas"`} {
		if !strings.Contains(string(saved), comment) {
			t.Errorf("комментарий %q потерян:\n%s", comment, saved)
		}
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("сохраненный файл не читается: %v\n%s", err, saved)
	}
	if !reflect.DeepEqual(cfg, reloaded) {
		t.Errorf("после сохранения значения отличаются:\nбыло  %+v\nстало %+v\n%s", cfg, reloaded, saved)
	}
}

func TestSaveCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &Config{Env: "test", ClientRequestTimeout: 3 * time.Second}

	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, reloaded) {
		t.Errorf("было %+v, стало %+v", cfg, reloaded)
	}
}
