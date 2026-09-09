package app

import (
	"fmt"
	"net"

	"github.com/dr2cc/golm/internal/client"
	"github.com/dr2cc/golm/internal/config"
	"github.com/dr2cc/golm/internal/scanner"
)

func Run(cfg config.Config) error {
	var hostPort string
	if cfg.Host == "" {
		// Находим компьютеры с портом сервиса ЛМ ЧЗ и получаем первый результат
		hostPort = scanner.ScanSubnet(cfg.Subnet, cfg.Port, cfg.ScannerTimeout)
	} else {
		hostPort = net.JoinHostPort(cfg.Host, cfg.Port)
	}

	if hostPort == "" {
		return fmt.Errorf("хостов с открытым портом %s не найдено", cfg.Port)
	}

	fmt.Printf("Проверяем статус ЛМ ЧЗ по адресу: %s\n", hostPort)

	apiClient := client.New(hostPort)

	if err := apiClient.GetServerInfo(hostPort); err != nil {
		return fmt.Errorf("ошибка получения информации от сервера: %w", err)
	}

	return nil
}
