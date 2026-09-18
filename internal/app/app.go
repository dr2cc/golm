package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dr2cc/golm/internal/config"
)

// Константы для настройки проверки
const (
	// Замените на ваш URL публикации DataMobile
	targetURL       = "http://localhost/polyMark/hs/DataMobileExch/"
	requestTimeout  = 5 * time.Second               // Время, после которого считаем, что сервер "умер"
	warningDuration = 2 * time.Second               // Время, после которого считаем, что сервер "тормозит"
	configPath      = `C:\Apache24\conf\httpd.conf` // Путь к конфигурационному файлу Apache (для Windows или Linux)
	expectedVersion = "8.3.27.2214"                 // Ожидаемая версия платформы 1С

	// НАСТРОЙКА АВТОРИЗАЦИИ 1С
	// Укажите имя пользователя и пароль, под которыми ТСД подключаются к 1С
	dbUser = "admin"
	dbPass = ""
)

// Структура для парсинга ответа DataMobile
type DataMobileResponse struct {
	Data string `json:"data"`
}

// 1. Функция чтения версии модуля 1С из Apache
func showApache1CModule() {
	// // Более простой вариант, без сравнения с expectedVersion
	// file, err := os.Open(configPath)
	// if err != nil {
	// 	fmt.Printf("❌ ОШИБКА АПАЧА: Не удалось открыть httpd.conf: %v\n", err)
	// 	return
	// }
	// defer file.Close()

	// scanner := bufio.NewScanner(file)
	// found := false

	// for scanner.Scan() {
	// 	line := scanner.Text()
	// 	if strings.Contains(line, "_1cws_module") && strings.Contains(line, "LoadModule") {
	// 		fmt.Printf("📦 Модуль 1С в конфигурации Apache:\n   %s\n", strings.TrimSpace(line))
	// 		found = true
	// 		break
	// 	}
	// }

	// if !found {
	// 	fmt.Println("❓ Модуль '_1cws_module' не найден или закомментирован в httpd.conf")
	// }

	file, err := os.Open(configPath)
	if err != nil {
		fmt.Printf("ОШИБКА: Не удалось открыть файл httpd.conf: %v\n", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	found := false

	// Построчно читаем файл конфигурации
	for scanner.Scan() {
		line := scanner.Text()

		// Ищем строку, содержащую подключение модуля 1С
		if strings.Contains(line, "_1cws_module") && strings.Contains(line, "LoadModule") {
			fmt.Printf("- Найден активный модуль 1С в Apache: %s\n", strings.TrimSpace(line))
			// fmt.Println("- Найден активный модуль 1С в Apache:", strings.TrimSpace(line))
			// fmt.Println(strings.TrimSpace(line))

			// Пример валидации: проверяем, содержит ли строка ожидаемую версию
			if !strings.Contains(line, expectedVersion) {
				fmt.Printf("ВНИМАНИЕ: Версия модуля 1С отличается от целевой (%s)!\n", expectedVersion)
			} else {
				fmt.Printf("-- Целевая версия-%s Версия wsap24-модуля соответствует целевой!\n", expectedVersion)
			}

			found = true
			break
		}
	}

	if !found {
		fmt.Println("ПРЕДУПРЕЖДЕНИЕ: Модуль '_1cws_module' не найден в httpd.conf. Возможно, 1С не опубликована через этот Apache.")
	}
}

// 2. Функция проверки доступности и скорости HTTP-сервиса
func checkDataMobileService() {
	client := &http.Client{
		Timeout: requestTimeout,
	}

	fmt.Printf("\n- Проверка сервиса DataMobile: %s\n", targetURL)

	// Засекаем время до создания запроса, чтобы замер был точным
	startTime := time.Now()

	// 1. Создаем объект HTTP-запроса (метод GET)
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		fmt.Printf("❌ ОШИБКА: Не удалось создать HTTP-запрос. Детали: %v\n", err)
		return
	}

	// 2. Добавляем Базовую Авторизацию (Basic Auth)
	req.SetBasicAuth(dbUser, dbPass)

	// 3. Выполняем запрос через клиент
	resp, err := client.Do(req)
	duration := time.Since(startTime)

	// Проверка на полное падение сервера или таймаут
	if err != nil {
		fmt.Printf("❌ КРИТИЧЕСКАЯ ОШИБКА: Сервер не отвечает или упал!\n   Детали: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Проверяем HTTP статус-код (теперь должен быть 200)
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ ОШИБКА СЕРВЕРА: Получен HTTP код %d вместо 200 OK\n", resp.StatusCode)
		if resp.StatusCode == http.StatusUnauthorized {
			fmt.Println("   💡 Подсказка: Неверный логин или пароль пользователя 1С.")
		}
		return
	}

	// Проверяем скорость работы (тормоза)
	fmt.Printf("-- Время ответа сервера: %v\n", duration)
	if duration > warningDuration {
		fmt.Printf("⚠️ ВНИМАНИЕ: Сервер сильно тормозит! Превышен лимит в %v\n", warningDuration)
	} else {
		fmt.Printf("-- Скорость работы в норме\n")
	}

	// Парсим JSON ответ от 1С
	var result DataMobileResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		fmt.Printf("❌ ОШИБКА JSON: Не удалось прочитать ответ от 1С. Детали: %v\n", err)
		return
	}

	// Выводим статус, который прислала сама 1С
	fmt.Printf("-- Ответ от 1С: %s\n", strings.TrimSpace(result.Data))
}

func Run(cfg config.Config) error {
	// Сначала смотрим конфигурацию Apache
	showApache1CModule()

	// Затем тестируем живой сервис
	checkDataMobileService()

	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()

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
