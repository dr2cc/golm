package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/dr2cc/golm/internal/client"
	"github.com/dr2cc/golm/internal/config"
)

// Константы для настройки проверки
const (
	// targetURL       = "http://localhost/tradeProfilDm/hs/DataMobileExch/" // "http://localhost/polyMark/hs/DataMobileExch/"
	publicationName = "polyMark"                    // polyMark / tradeProfilDm
	apacheAddress   = "192.168.0.75"                // 192.168.0.75 / localhost
	configPath      = `C:\Apache24\conf\httpd.conf` // ssh drk@192.168.0.75 cd /etc/apache2/ apache2.conf // Путь к конфигурационному файлу Apache (для Windows или Linux)
	requestTimeout  = 5 * time.Second               // Время, после которого считаем, что сервер "умер"
	warningDuration = 2 * time.Second               // Время, после которого считаем, что сервер "тормозит"
	// expectedVersion = "8.3.27.2214"                                       // Ожидаемая версия платформы 1С

	// НАСТРОЙКА АВТОРИЗАЦИИ 1С
	// Укажите имя пользователя и пароль, под которыми ТСД подключаются к 1С
	dbUser = "admin"
	dbPass = ""
)

// Структура для парсинга ответа DataMobile
type DataMobileResponse struct {
	Data string `json:"data"`
}

func apacheChecker(ctx context.Context) {

	apiClient := client.New(apacheAddress, dbUser, dbPass)

	if err := apiClient.GetApacheInfo(ctx); err != nil {
		fmt.Printf("ошибка получения информации от сервера: %v\n", err)
		return
	}
}

// 1. Функция чтения версии модуля 1С из Apache
func showApache1CModule() {
	// Простой вариант, без сравнения с expectedVersion
	file, err := os.Open(configPath)
	if err != nil {
		fmt.Printf("❌ ОШИБКА АПАЧА: Не удалось открыть httpd.conf: %v\n", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	found := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "_1cws_module") && strings.Contains(line, "LoadModule") {
			fmt.Printf("- Модуль 1С в конфигурации Apache:\n   %s\n", strings.TrimSpace(line))
			found = true
			break
		}
	}

	if !found {
		fmt.Println("❓ Модуль '_1cws_module' не найден или закомментирован в httpd.conf")
	}

}

// 2. Функция проверки доступности и скорости HTTP-сервиса
func checkDataMobileService() {
	client := &http.Client{
		Timeout: requestTimeout,
	}

	targetURL := "http://" + apacheAddress + "/" + publicationName + "/hs/DataMobileExch/"

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
		// Обыгрываем конфликт версий (HTTP 409)
		if resp.StatusCode == http.StatusConflict {
			bodyBytes, _ := io.ReadAll(resp.Body)
			bodyStr := string(bodyBytes)

			fmt.Println("❌ КРИТИЧЕСКАЯ ОШИБКА: Конфликт версий 1С!")

			// Регулярное выражение для поиска версий в круглых скобках
			re := regexp.MustCompile(`\((\d+\.\d+\.\d+\.\d+)\s*-\s*(\d+\.\d+\.\d+\.\d+)\)`)
			matches := re.FindStringSubmatch(bodyStr)

			if len(matches) == 3 {
				fmt.Printf("   📌 В Apache (wsap24-модуль): %s\n", matches[1])
				fmt.Printf("   📌 На сервере (кластер 1С): %s\n", matches[2])
				fmt.Println("   💡 Решение: Перепубликуйте базу или обновите путь к wsap24-модулю в httpd.conf!")
			} else {
				// Если текст ошибки изменился, выводим как есть
				fmt.Printf("   Технические детали:\n%s\n", bodyStr)
			}
			return
		}

		// Другие ошибки (401, 404, 500 и т.д.)
		fmt.Printf("-- ОШИБКА СЕРВИСА: Получен HTTP код %d вместо 200 OK\n", resp.StatusCode)
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Логика goapache
	apacheChecker(ctx)

	// Смотрим конфигурацию Apache (если это локальный компьютер)
	if apacheAddress == "localhost" {
		showApache1CModule()
	}

	// Затем тестируем живой сервис
	checkDataMobileService()

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
