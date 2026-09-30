package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dr2cc/golm/internal/config"
	"github.com/dr2cc/golm/internal/datamobile"
	"github.com/dr2cc/golm/internal/lmcz"
)

func Run(cfg config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ClientRequestTimeout)
	// ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Создаем общий HTTP-клиент с таймаутами
	httpClient := &http.Client{
		Timeout: cfg.ClientRequestTimeout,
	}

	// 2. Initialize domain clients

	if cfg.LaunchDataMobile {
		log.Println("Инициализация клиента DataMobile...")
		dmClient := datamobile.NewClient(cfg.DataMobile, httpClient)

		// Передаем контекст «вглубь» по цепочке вызовов
		if err := dmClient.ApacheChecker(ctx); err != nil {
			return err
		}
		// Завершаем работу
		return nil
	}

	// Логика golm
	if cfg.LaunchLMCZ {
		log.Println("Инициализация клиента ЛМ ЧЗ...")
		czClient := lmcz.NewClient(cfg.LMCZ, httpClient)
		// Кроме входа в блок LMCZ, нужно сдесь сделать разделение на запросы GET (информация о ЛМЧЗ) и POST (инициализация ЛМЧЗ)
		_ = czClient

		// if err := czClient.SomethingCheck(ctx); err != nil {
		// 	return err
		// }

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
		// return nil

		// Завершаем работу
		return nil
	}

	exeName := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Hи один клиент не был выбран через флаги запуска.\nИспользуйте \"%s <команда> -h\" для просмотра флагов конкретной команды.\n", exeName)
	// log.Println("Hи один клиент не был выбран через флаги запуска.")
	return nil
}
