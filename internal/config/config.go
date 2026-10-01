package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Функция для вывода общей справки по приложению
func printGlobalUsage() {
	exeName := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Использование: %s <команда> [флаги]\n\n", exeName)
	fmt.Fprintf(os.Stderr, "Команды:\n")
	fmt.Fprintf(os.Stderr, "  get   Запуск сканирования сети\n")
	fmt.Fprintf(os.Stderr, "  post  Отправка данных на сервер\n\n")
	fmt.Fprintf(os.Stderr, "Используйте \"%s <команда> -h\" для просмотра флагов конкретной команды.\n", exeName)
}

// Выносим в отдельный тип
type DataMobileConfig struct {
	PublicationName string        `yaml:"apache_publication_name"`
	Address         string        `yaml:"apache_address"`
	ConfPath        string        `yaml:"apache_conf_path"`
	WarningDuration time.Duration `yaml:"apache_warning_duration"`
	DmUser          string        `yaml:"dm_user"`
	DmPass          string        `yaml:"dm_pass"`
}

type LmczConfig struct {
	// LmczURL  string `yaml:"lmcz_url"`
	Host           string        `yaml:"lmcz_host"`
	Subnet         string        `yaml:"lmcz_subnet"`
	Port           string        `yaml:"lmcz_port"`
	ScannerTimeout time.Duration `yaml:"lmcz_scanner_timeout"`
	User           string        `yaml:"lmcz_user"`
	Pass           string        `yaml:"lmcz_pass"`
	TokenXAPIKEY   string        `yaml:"token_x_api_key"`
}

type Config struct {
	Env                  string           `yaml:"env"`
	ClientRequestTimeout time.Duration    `yaml:"client_request_timeout"`
	LaunchDataMobile     bool             `yaml:"launch_datamobile"`
	LaunchLmczCheck      bool             `yaml:"launch_lmcz_check"`
	LaunchLmczInit       bool             `yaml:"launch_lmcz_init"`
	DataMobile           DataMobileConfig `yaml:"datamobile"`
	LMCZ                 LmczConfig       `yaml:"lmcz"`
}

// switch cfg.Command {
// case "get":
// 	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
// case "post":
// 	postCmd := flag.NewFlagSet("post", flag.ExitOnError)

// New() парсит флаги и возвращает готовую конфигурацию.
// Когда будем читать переменные окружения или .env файл, код поменяется только тут.
func New() (*Config, error) {
	// Вариант 03.
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
	flag.BoolVar(&cfg.LaunchLmczCheck, "lm", cfg.LaunchLmczCheck, "Запустить тест ЛМ ЧЗ")
	flag.BoolVar(&cfg.LaunchLmczInit, "token", cfg.LaunchLmczInit, "Инициализировать ЛМ ЧЗ")

	// 5. Парсим флаги.
	// Если пользователь передаст флаг в терминале (-dm),
	// он перепишет значение, которое было в config.yaml
	flag.Parse()

	// Вызываем проверку проблем среды уже при создании конфига
	cfg.validateEnvironment()

	// // Вариант 02 (apache + 1С)
	// cfg := &Config{
	// 	// Инициализируем вложенную структуру DataMobile
	// 	DataMobile: DataMobileConfig{
	// 		PublicationName: "polyMark",
	// 		ApacheAddress:   "localhost",                   // 192.168.0.75 / localhost
	// 		ConfigPath:      `C:\Apache24\conf\httpd.conf`, // ssh drk@192.168.0.75 cd /etc/apache2/ apache2.conf // Путь к конфигурационному файлу Apache (для Windows или Linux)
	// 		RequestTimeout:  5 * time.Second,               // Используем тип time.Duration
	// 		WarningDuration: 2 * time.Second,
	// 		DbUser:          "admin",
	// 		DbPass:          "",
	// 	},
	// }

	// // Вызываем проверку сразу при создании конфига
	// cfg.validateEnvironment()

	// // Вариант 01 ❌ Текущая реализация функции config.New() нарушает принцип единственной ответственности (Single Responsibility Principle)
	// // и содержит архитектурный антипаттерн,
	// // так как конфигуратор берет на себя роль управления жизненным циклом приложения (os.Exit).

	// // Проверяем, передал ли пользователь вообще команду
	// if len(os.Args) < 2 {
	// 	// Перехватываем вызов справки на самом верхнем уровне (до подкоманд)
	// 	printGlobalUsage()
	// 	os.Exit(1) // Завершаем программу сразу с кодом ошибки
	// }

	// if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
	// 	printGlobalUsage()
	// 	os.Exit(0) // Успешный выход после печати справки
	// }

	// cfg.Command = os.Args[1]

	// switch cfg.Command {
	// case "get":
	// 	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	// 	host := getCmd.String("host", "", "target host")
	// 	subnet := getCmd.String("s", "192.168.0", "target subnet")
	// 	port := getCmd.String("p", "5995", "target port")
	// 	timeout := getCmd.Duration("t", 500*time.Millisecond, "scanner timeout")

	// 	// Парсим аргументы начиная со 2-го индекса (пропуская имя программы и само слово 'get')
	// 	if err := getCmd.Parse(os.Args[2:]); err != nil {
	// 		return nil, err
	// 	}

	// 	cfg.Host = *host
	// 	cfg.Subnet = *subnet
	// 	cfg.Port = *port
	// 	cfg.ScannerTimeout = *timeout

	// case "post":
	// 	postCmd := flag.NewFlagSet("post", flag.ExitOnError)
	// 	host := postCmd.String("host", "localhost", "target host")
	// 	port := postCmd.String("p", "5997", "target port")
	// 	username := postCmd.String("u", "admin", "username")
	// 	password := postCmd.String("pass", "admin", "password")
	// 	token := postCmd.String("token", "", "auth token to send (required)")

	// 	if err := postCmd.Parse(os.Args[2:]); err != nil {
	// 		return nil, err
	// 	}

	// 	if *token == "" {
	// 		// Если токена нет, принудительно покажем справку для post
	// 		fmt.Println("flag -token is required for 'post' command")
	// 		postCmd.Usage()
	// 		os.Exit(1)
	// 		// return nil, errors.New("flag -token is required for 'post' command")
	// 	}

	// 	cfg.Host = *host
	// 	cfg.Port = *port
	// 	cfg.Username = *username
	// 	cfg.Password = *password
	// 	cfg.Token = *token
	// 	cfg.ScannerTimeout = 500 * time.Millisecond // дефолт

	// default:
	// 	return nil, fmt.Errorf("unknown command: %s. Choose 'get' or 'post'", cfg.Command)
	// }

	return cfg, nil
}

// Внутренний метод для проверки потенциальных проблем среды
func (c *Config) validateEnvironment() {
	if !isWSL() {
		return
	}

	// Если мы в WSL и адрес локальный — выводим предупреждение
	if strings.Contains(c.DataMobile.Address, "127.0.0.1") || strings.Contains(c.DataMobile.Address, "localhost") {
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
