package cli

import (
	"flag"
	"fmt"
	"time"

	"github.com/dr2cc/golm/internal/scanner"
)

type Config struct {
	Subnet         string
	Port           string
	ScannerTimeout time.Duration
}

type CLI struct {
	cfg Config
}

func New() *CLI {
	timeout := 500 * time.Millisecond
	// Прямо здесь удобно парсить флаги командной строки, специфичные для клиента
	subnet := flag.String("s", "192.168.0", "target subnet") // После теста поменять на часто используемую
	port := flag.String("p", "5995", "target port")
	flag.Parse()

	return &CLI{
		cfg: Config{
			Subnet:         *subnet,
			Port:           *port,
			ScannerTimeout: timeout,
		},
	}
}

func (c *CLI) Run() error {
	// Находим компьютеры с сервисом ЛМ ЧЗ и получаем первый результат
	hostPort := scanner.ScanSubnet(c.cfg.Subnet, c.cfg.Port, c.cfg.ScannerTimeout)

	if hostPort != "" {
		fmt.Printf("Проверяем статус ЛМ ЧЗ по адресу: %s\n ", hostPort)
		getServerInfo(hostPort)
	} else {
		fmt.Printf("Хостов с открытым портом %s не найдено.\n", c.cfg.Port)
	}

	return nil
}
