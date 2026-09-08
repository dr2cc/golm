package config

import (
	"flag"
	"time"
)

type Config struct {
	ScannerTimeout time.Duration
	Host           string
	Subnet         string
	Port           string
}

// New парсит флаги и возвращает готовую конфигурацию.
// Если в будущем будет нужно читать переменные окружения или .env файл,
// поменяется код только внутри этой функции.
func New() (*Config, error) {
	host := flag.String("h", "", "target host")
	subnet := flag.String("s", "192.168.0", "target subnet")
	port := flag.String("p", "5995", "target port")
	timeout := flag.Duration("t", 500*time.Millisecond, "scanner timeout")
	flag.Parse()

	return &Config{
		ScannerTimeout: *timeout,
		Host:           *host,
		Subnet:         *subnet,
		Port:           *port,
	}, nil
}
