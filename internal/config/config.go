package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Выносим в отдельный тип
type DataMobileConfig struct {
	PublicationName string
	ApacheAddress   string
	ConfigPath      string
	RequestTimeout  time.Duration
	WarningDuration time.Duration
	DbUser          string
	DbPass          string
}

// type LMCZ struct{
// 	BaseURL string `env:"LMCZ_BASE_URL" env-required:"true"`
// }

type Config struct {
	// Тоже оформить структурой типа
	// LMCZ LMCZ
	Command        string // "get" или "post"
	ScannerTimeout time.Duration
	Host           string
	Port           string
	Subnet         string // get
	Username       string // post
	Password       string // post
	Token          string // post
	//
	DataMobile DataMobileConfig
}

// Функция для вывода общей справки по приложению
func printGlobalUsage() {
	exeName := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Использование: %s <команда> [флаги]\n\n", exeName)
	fmt.Fprintf(os.Stderr, "Команды:\n")
	fmt.Fprintf(os.Stderr, "  get   Запуск сканирования сети\n")
	fmt.Fprintf(os.Stderr, "  post  Отправка данных на сервер\n\n")
	fmt.Fprintf(os.Stderr, "Используйте \"%s <команда> -h\" для просмотра флагов конкретной команды.\n", exeName)
}

// New парсит флаги и возвращает готовую конфигурацию.
// Если в будущем будет нужно читать переменные окружения или .env файл,
// поменяется код только внутри этой функции.
func New() (*Config, error) {
	cfg := &Config{
		// Инициализируем вложенную структуру DataMobile
		DataMobile: DataMobileConfig{
			PublicationName: "polyMark",
			ApacheAddress:   "localhost",                   // 192.168.0.75 / localhost
			ConfigPath:      `C:\Apache24\conf\httpd.conf`, // ssh drk@192.168.0.75 cd /etc/apache2/ apache2.conf // Путь к конфигурационному файлу Apache (для Windows или Linux)
			RequestTimeout:  5 * time.Second,               // Используем тип time.Duration
			WarningDuration: 2 * time.Second,
			DbUser:          "admin",
			DbPass:          "",
		},
	}

	// Вызываем проверку сразу при создании конфига
	cfg.validateEnvironment()

	// // ❌ Текущая реализация функции config.New() нарушает принцип единственной ответственности (Single Responsibility Principle)
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
	if strings.Contains(c.DataMobile.ApacheAddress, "127.0.0.1") || strings.Contains(c.DataMobile.ApacheAddress, "localhost") {
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
