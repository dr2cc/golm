package golm

import (
	"testing"

	"github.com/dr2cc/golm/internal/config"
)

// Шаблон встроен в GUI как настройки по умолчанию, поэтому он должен
// всегда оставаться корректным YAML с ненулевыми таймаутами.
func TestDefaultConfigIsValid(t *testing.T) {
	cfg, err := config.Parse(DefaultConfigYAML)
	if err != nil {
		t.Fatalf("config.example.yaml не разбирается: %v", err)
	}
	if cfg.ClientRequestTimeout <= 0 || cfg.Apache.WarningDuration <= 0 || cfg.LMCZ.ScannerTimeout <= 0 {
		t.Errorf("в config.example.yaml должны быть заданы все таймауты: %+v", cfg)
	}
	if cfg.LMCZ.Port == "" {
		t.Error("в config.example.yaml не задан порт ЛМ ЧЗ")
	}
}
