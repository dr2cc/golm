package datamobile

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

	client "github.com/dr2cc/golm/internal/lmcz"
)

// Константы для настройки проверки
const (
	// targetURL       = "http://localhost/tradeProfilDm/hs/DataMobileExch/" // "http://localhost/polyMark/hs/DataMobileExch/"
	publicationName = "tradeProfilDm"               // polyMark / tradeProfilDm
	ApacheAddress   = "192.168.0.75"                // 192.168.0.75 / localhost
	configPath      = `C:\Apache24\conf\httpd.conf` // ssh drk@192.168.0.75 cd /etc/apache2/ apache2.conf // Путь к конфигурационному файлу Apache (для Windows или Linux)
	requestTimeout  = 5 * time.Second               // Время, после которого считаем, что сервер "умер"
	warningDuration = 2 * time.Second               // Время, после которого считаем, что сервер "тормозит"
	// expectedVersion = "8.3.27.2214"                                       // Ожидаемая версия платформы 1С

	// НАСТРОЙКА АВТОРИЗАЦИИ 1С
	// Укажите имя пользователя и пароль, под которыми ТСД подключаются к 1С
	dbUser = "admin"
	dbPass = ""
)

func ApacheChecker(ctx context.Context) {

	apiClient := client.New(ApacheAddress, dbUser, dbPass)

	if err := apiClient.GetApacheInfo(ctx); err != nil {
		fmt.Printf("ошибка получения информации от сервера: %v\n", err)
		return
	}
}

// Функция чтения версии модуля 1С из Apache
func ShowApache1CModule() {
	// Простой вариант, без сравнения с expectedVersion
	file, err := os.Open(configPath)
	if err != nil {
		fmt.Printf("apache error: failed to open httpd.conf: %v\n", err)
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
		fmt.Println("Модуль '_1cws_module' не найден или закомментирован в httpd.conf")
	}

}

// Функция проверки доступности и скорости RESTful-сервиса
func CheckDataMobileService() {
	client := &http.Client{
		Timeout: requestTimeout,
	}

	targetURL := "http://" + ApacheAddress + "/" + publicationName + "/hs/DataMobileExch/"

	fmt.Printf("\n- Endpoint address (DataMobile): %s\n", targetURL)

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

			fmt.Println("-- status", resp.StatusCode, "Конфликт версий 1С:")

			// Регулярное выражение для поиска версий в круглых скобках
			re := regexp.MustCompile(`\((\d+\.\d+\.\d+\.\d+)\s*-\s*(\d+\.\d+\.\d+\.\d+)\)`)
			matches := re.FindStringSubmatch(bodyStr)

			if len(matches) == 3 {
				fmt.Printf("--- Версия wsap24: %s\n", matches[1])
				fmt.Printf("--- Кластер    1С: %s\n", matches[2])
				fmt.Println("-- Перепубликуйте базу или обновите путь к wsap24-модулю в httpd.conf.")
			} else {
				// Если текст ошибки изменился, выводим как есть
				fmt.Printf("   Технические детали:\n%s\n", bodyStr)
			}
			return
		}

		// Другие ошибки (401, 404, 500 и т.д.)
		fmt.Printf("-- service error: status %d\n", resp.StatusCode)
		if resp.StatusCode == http.StatusNotFound {
			fmt.Printf("--- База %s не опубликована на web-сервере %s\n", publicationName, ApacheAddress)
		}
		if resp.StatusCode == http.StatusUnauthorized {
			fmt.Println("--- Неверный логин или пароль пользователя 1С.")
		}
		return
	}

	// Проверяем скорость работы (тормоза)
	fmt.Printf("-- Время ответа сервера: %v", duration)
	if duration > warningDuration {
		fmt.Printf(" ⚠️ ВНИМАНИЕ: Сервер сильно тормозит! Превышен лимит в %v\n", warningDuration)
	} else {
		fmt.Printf(" Скорость работы в норме\n")
	}

	// Парсим JSON ответ от 1С
	var result DataMobileResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		fmt.Printf("❌ ОШИБКА JSON: Не удалось прочитать ответ от 1С. Детали: %v\n", err)
		return
	}

	// Выводим статус, который прислала сама 1С
	fmt.Printf("-- Response: %s\n", strings.TrimSpace(result.Data))
}
