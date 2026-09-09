package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
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

// New парсит флаги и возвращает готовую конфигурацию.
// Если в будущем будет нужно читать переменные окружения или .env файл,
// поменяется код только внутри этой функции.
func New() (*Config, error) {
	cfg := &Config{}

	// Проверяем, передал ли пользователь вообще команду
	if len(os.Args) < 2 {
		return nil, errors.New("expected 'get' or 'post' subcommands")
	}

	cfg.Command = os.Args[1]

	// timeout := flag.Duration("t", 500*time.Millisecond, "scanner timeout")
	// host := flag.String("h", "", "target host")
	// subnet := flag.String("s", "192.168.0", "target subnet")
	// port := flag.String("p", "5995", "target port")
	// username := flag.String("u", "admin", "username")
	// password := flag.String("pass", "admin", "password")
	// flag.Parse()

	switch cfg.Command {
	case "get":
		getCmd := flag.NewFlagSet("get", flag.ExitOnError)
		host := getCmd.String("h", "", "target host")
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
		host := postCmd.String("h", "localhost", "target host")
		port := postCmd.String("p", "5997", "target port")
		// Специфичный флаг только для post
		username := postCmd.String("u", "admin", "username")
		password := postCmd.String("pass", "admin", "password")
		token := postCmd.String("t", "", "auth token to send (required)")

		if err := postCmd.Parse(os.Args[2:]); err != nil {
			return nil, err
		}

		if *token == "" {
			return nil, errors.New("flag -token is required for 'post' command")
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
