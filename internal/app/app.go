package app

import (
	"context"
	"log"
	"net/http"

	"github.com/dr2cc/golm/internal/config"
	"github.com/dr2cc/golm/internal/datamobile"
)

func Run(cfg config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DataMobile.RequestTimeout)
	defer cancel()

	// 1. Создаем общий HTTP-клиент с таймаутами
	httpClient := &http.Client{
		Timeout: cfg.DataMobile.RequestTimeout,
	}

	// 2. Initialize domain clients (инициализируем доменные клиенты)
	if cfg.LaunchDataMobile {
		log.Println("Инициализация клиента DataMobile...")
		dmClient := datamobile.NewClient(cfg.DataMobile, httpClient)
		// Передаем этот контекст «вглубь» по цепочке вызовов
		dmClient.ApacheChecker(ctx, cfg.DataMobile)
	}

	if cfg.LaunchLMCZ {
		log.Println("Инициализация клиента ЛМ ЧЗ...")
		// czClient := lmcz.NewClient(cfg.LMCZ.BaseURL, httpClient)
		// _ = czClient
	}

	if cfg.LaunchDataMobile || cfg.LaunchLMCZ {
		select {}
	} else {
		log.Println("Hи один клиент не был выбран через флаги запуска.")
	}

	// // Логика golm
	// var hostPort string

	// switch cfg.Command {
	// case "get":

	// 	if cfg.Host == "" {
	// 		// Находим компьютеры с портом сервиса ЛМ ЧЗ и получаем первый результат
	// 		hostPort = scanner.ScanSubnet(cfg.Subnet, cfg.Port, cfg.ScannerTimeout)
	// 	} else {
	// 		hostPort = net.JoinHostPort(cfg.Host, cfg.Port)
	// 	}

	// 	if hostPort == "" {
	// 		return fmt.Errorf("хостов с открытым портом %s не найдено", cfg.Port)
	// 	}

	// 	fmt.Printf("Проверяем статус ЛМ ЧЗ по адресу: %s\n", hostPort)

	// 	apiClient := client.New(hostPort, "", "")

	// 	if err := apiClient.GetServerInfo(hostPort); err != nil {
	// 		return fmt.Errorf("ошибка получения информации от сервера: %w", err)
	// 	}
	// case "post":
	// 	hostPort = net.JoinHostPort(cfg.Host, cfg.Port)
	// 	apiClient := client.New(hostPort, cfg.Username, cfg.Password)

	// 	fmt.Printf("Отправляем токен на %s...\n", hostPort)
	// 	res, err := apiClient.SendToken(ctx, cfg.Token)
	// 	if err != nil {
	// 		return fmt.Errorf("post command failed: %w", err)
	// 	}
	// 	fmt.Printf("Успешно! Ответ сервера: %s\n", res.Status)
	// }
	return nil
}
