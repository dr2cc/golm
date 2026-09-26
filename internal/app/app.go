package app

import (
	"context"
	"net/http"
	"time"

	"github.com/dr2cc/golm/internal/config"
	"github.com/dr2cc/golm/internal/datamobile"
)

// // isWSL проверяет, запущен ли код внутри подсистемы WSL
// func isWSL() bool {
// 	version, err := os.ReadFile("/proc/version")
// 	if err != nil {
// 		return false
// 	}
// 	// Переводим в нижний регистр для надежности
// 	content := strings.ToLower(string(version))
// 	return strings.Contains(content, "microsoft") || strings.Contains(content, "wsl")
// }

func Run(cfg config.Config) error {
	// 1. Создаем общий HTTP-клиент с таймаутами
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// // Простая проверка перед отправкой запроса
	// if isWSL() && (strings.Contains(datamobile.ApacheAddress, "127.0.0.1") || strings.Contains(datamobile.ApacheAddress, "localhost")) {
	// 	fmt.Println("   Внимание: Вы запускаете код внутри WSL2 и обращаетесь к локальному интерфейсу (localhost/127.0.0.1).")
	// 	fmt.Println("   Если веб-сервер запущен на Windows, запрос завершится ошибкой 'connection refused'.")
	// }

	// 2. Initialize domain clients (инициализируем доменные клиенты)
	dmClient := datamobile.NewClient(cfg.DataMobile, httpClient)
	// czClient := lmcz.NewClient("http://localhost:8080", httpClient)

	// // 3. Описываем бизнес-логику взаимодействия между ними
	// log.Println("Приложение golm запущено...")

	// Пример вызова:
	// data, err := dmClient.FetchNewData()
	// err = czClient.SendMark(data.Mark)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Передаем этот контекст «вглубь» по цепочке вызовов
	dmClient.ApacheChecker(ctx, cfg.DataMobile)

	// err = czClient.VerifyMark(ctx, data.Barcode)
	// if err != nil {
	// 	log.Printf("Ошибка проверки марки: %v", err)
	// 	return
	// }

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
