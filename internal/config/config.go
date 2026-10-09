package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dr2cc/golm/internal/util/urlutil"
	"go.yaml.in/yaml/v3"
)

type ApacheConfig struct {
	PublicationName string        `yaml:"publication_name"`
	Address         string        `yaml:"address"`
	Port            string        `yaml:"port"`
	ConfPath        string        `yaml:"conf_path"`
	WarningDuration time.Duration `yaml:"warning_duration"`
}

type DataMobileConfig struct {
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

type LmczConfig struct {
	Host           string        `yaml:"host"`
	Subnet         string        `yaml:"subnet"`
	Port           string        `yaml:"port"`
	ScannerTimeout time.Duration `yaml:"scanner_timeout"`
	User           string        `yaml:"user"`
	Pass           string        `yaml:"pass"`
	TokenXAPIKEY   string        `yaml:"token_x_api_key"`
}

type Config struct {
	Env                  string           `yaml:"env"`
	ClientRequestTimeout time.Duration    `yaml:"client_request_timeout"`
	LaunchDataMobile     bool             `yaml:"launch_datamobile"`
	LaunchLmczStatus     bool             `yaml:"launch_lmcz_check"`
	LaunchLmczInit       bool             `yaml:"launch_lmcz_init"`
	Apache               ApacheConfig     `yaml:"apache"`
	DataMobile           DataMobileConfig `yaml:"datamobile"`
	LMCZ                 LmczConfig       `yaml:"lmcz"`
}

// Рефакторинг добавления port в конфигурацию.
// BaseURL собирает полный префикс адреса (включая протокол и порт) для HTTP-запросов.
// Метод не нужно прописывать в YAML, он работает на основе существующих полей.
func (a ApacheConfig) BaseURL() string {
	// Ваша функция уберет http:// и красиво склеит хост с портом
	cleanHostPort := urlutil.JoinHostPort(a.Address, a.Port)

	// Возвращаем готовую строку, которую сразу поймет http.Client
	return "http://" + cleanHostPort
}

// New() парсит флаги и возвращает готовую конфигурацию.
// Когда будем читать переменные окружения или .env файл, код поменяется только тут.
func New() (*Config, error) {
	// 1. Инициализируем пустую структуру
	cfg := &Config{}

	// 2. Читаем файл конфигурации
	filename := "config.yaml"
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл конфигурации %s: %w", filename, err)
	}

	// 3. Парсим YAML в структуру.
	// Библиотека yaml.v3 превращает строки вида "10s" или "5m" в тип time.Duration
	if err := yaml.Unmarshal(fileBytes, cfg); err != nil {
		return nil, fmt.Errorf("ошибка парсинга YAML: %w", err)
	}

	// 4. Привязываем флаги командной строки к полям созданного объекта.
	// В качестве дефолтных значений передаем то, что прочитано из файла YAML.
	flag.BoolVar(&cfg.LaunchDataMobile, "dm", cfg.LaunchDataMobile, "Запустить проверку DataMobile-Apache-1C")
	flag.BoolVar(&cfg.LaunchLmczStatus, "status", cfg.LaunchLmczStatus, "Запустить тест ЛМ ЧЗ")
	flag.BoolVar(&cfg.LaunchLmczInit, "token", cfg.LaunchLmczInit, "Инициализировать ЛМ ЧЗ")

	// Переопределяем стандартный вывод ошибок и подсказок, чтобы не получить странную строку при неправильном флаге.
	flag.Usage = func() {
		// Подменяем os.Args[0] на красивое имя
		os.Args[0] = "golm"

		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	// 5. Парсим флаги.
	// Если пользователь передаст флаг в терминале (-dm),
	// он перепишет значение, которое было в config.yaml
	flag.Parse()

	// Вызываем проверку проблем среды уже при создании конфига
	cfg.validateEnvironment()

	return cfg, nil
}

// Внутренний метод для проверки потенциальных проблем среды
func (c *Config) validateEnvironment() {
	if !isWSL() {
		return
	}

	// Если мы в WSL и адрес локальный — выводим предупреждение
	if strings.Contains(c.Apache.Address, "127.0.0.1") || strings.Contains(c.Apache.Address, "localhost") {
		fmt.Println("[CONFIG WARNING]: Вы запускаете код внутри WSL2 и запрашиваете localhost (127.0.0.1).")
		fmt.Println("Если сетевой режим WSL не изменен на 'mirrored', запрос завершится ошибкой 'connection refused'.")
	}
}

// Сама функция проверки среды (не экспортируется наружу, так как нужна только здесь)
func isWSL() bool {
	version, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	content := strings.ToLower(string(version))
	return strings.Contains(content, "microsoft") || strings.Contains(content, "wsl")
}
