package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Command        string // "get" или "post"
	ScannerTimeout time.Duration
	Host           string
	Port           string
	Subnet         string // get
	Username       string // post
	Password       string // post
	Token          string // post
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
	cfg := &Config{}

	// Проверяем, передал ли пользователь вообще команду
	if len(os.Args) < 2 {
		// Перехватываем вызов справки на самом верхнем уровне (до подкоманд)
		printGlobalUsage()
		os.Exit(1) // Завершаем программу сразу с кодом ошибки
	}

	if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
		printGlobalUsage()
		os.Exit(0) // Успешный выход после печати справки
	}

	cfg.Command = os.Args[1]

	switch cfg.Command {
	case "get":
		getCmd := flag.NewFlagSet("get", flag.ExitOnError)
		host := getCmd.String("host", "", "target host")
		subnet := getCmd.String("s", "192.168.0", "target subnet")
		port := getCmd.String("p", "5995", "target port")
		timeout := getCmd.Duration("t", 500*time.Millisecond, "scanner timeout")

		// Парсим аргументы начиная со 2-го индекса (пропуская имя программы и само слово 'get')
		if err := getCmd.Parse(os.Args[2:]); err != nil {
			return nil, err
		}

		cfg.Host = *host
		cfg.Subnet = *subnet
		cfg.Port = *port
		cfg.ScannerTimeout = *timeout

	case "post":
		postCmd := flag.NewFlagSet("post", flag.ExitOnError)
		host := postCmd.String("host", "localhost", "target host")
		port := postCmd.String("p", "5997", "target port")
		username := postCmd.String("u", "admin", "username")
		password := postCmd.String("pass", "admin", "password")
		token := postCmd.String("token", "", "auth token to send (required)")

		if err := postCmd.Parse(os.Args[2:]); err != nil {
			return nil, err
		}

		if *token == "" {
			// Если токена нет, принудительно покажем справку для post
			fmt.Println("flag -token is required for 'post' command")
			postCmd.Usage()
			os.Exit(1)
			// return nil, errors.New("flag -token is required for 'post' command")
		}

		cfg.Host = *host
		cfg.Port = *port
		cfg.Username = *username
		cfg.Password = *password
		cfg.Token = *token
		cfg.ScannerTimeout = 500 * time.Millisecond // дефолт

	default:
		return nil, fmt.Errorf("unknown command: %s. Choose 'get' or 'post'", cfg.Command)
	}

	return cfg, nil
}

// return &Config{
// 	ScannerTimeout: *timeout,
// 	Host:           *host,
// 	Subnet:         *subnet,
// 	Port:           *port,
// 	Username:       *username,
// 	Password:       *password,
// }, nil
