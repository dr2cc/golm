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

	"github.com/dr2cc/golm/internal/config"
)

type Client struct {
	cfg        config.DataMobileConfig
	httpClient *http.Client
}

func NewClient(datamobile config.DataMobileConfig, httpClient *http.Client) *Client {
	// Добавляем схему при инициализации, если забыли указать
	if !strings.HasPrefix(datamobile.ApacheAddress, "http://") && !strings.HasPrefix(datamobile.ApacheAddress, "https://") {
		datamobile.ApacheAddress = "http://" + datamobile.ApacheAddress
	}
	return &Client{
		cfg:        datamobile,
		httpClient: httpClient,
	}
}

func (c *Client) ApacheChecker(ctx context.Context) error {
	// Использование HEAD-запроса экономит трафик
	req, err := http.NewRequestWithContext(ctx, "HEAD", c.cfg.ApacheAddress, nil)
	if err != nil {
		return fmt.Errorf("failed HEAD request: %w", err)
	}

	// Формат ответа в рамках HTTP это *Response
	resp, err := c.httpClient.Do(req) //http.Get("http://" + hostPort)
	if err != nil {
		return fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем, что сервер ответил корректным HTTP-статусом (например, 200 OK или 403/404, что тоже подтверждает работу Apache)
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		// Дополнительно можно проверить заголовок "Server"
		serverHeader := resp.Header.Get("Server") // например, "Apache/2.4.41 (Ubuntu)"
		fmt.Printf("- It just works! %s\n", c.cfg.ApacheAddress+" - "+serverHeader)

		// Смотрим конфигурацию Apache (если это локальный компьютер)
		if strings.Contains(c.cfg.ApacheAddress, "127.0.0.1") || strings.Contains(c.cfg.ApacheAddress, "localhost") { // "localhost""192.168.0.13" {
			if err := ShowApache1CModule(c.cfg.ApacheConfPath); err != nil {
				return err
			}
		}
		// Тестируем наш RESTful-сервис ("РЕСТный" сервис)
		if err := c.CheckDataMobileService(); err != nil {
			return err
		}
	}

	return nil
}

// Функция чтения версии модуля 1С из локального Apache
func ShowApache1CModule(configPath string) error {
	file, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("- apache error: failed to open httpd.conf: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	found := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "_1cws_module") && strings.Contains(line, "LoadModule") {
			fmt.Printf("- Модуль 1С в конфигурации Apache:%s\n", strings.TrimSpace(line))
			found = true
			break
		}
	}
	// Обязательно проверяем, почему остановился цикл scanner.Scan()
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("apache error: scanner encountered an error while reading httpd.conf: %w", err)
	}

	if !found {
		return fmt.Errorf("Модуль '_1cws_module' не найден или закомментирован в httpd.conf")
	}

	return nil
}

// Функция проверки доступности и скорости RESTful-сервиса
func (c *Client) CheckDataMobileService() error {
	targetURL := c.cfg.ApacheAddress + "/" + c.cfg.PublicationName + "/hs/DataMobileExch/"
	fmt.Printf("\n- GET to endpoint (DataMobile): %s\n", targetURL)

	// Время до создания запроса (чтобы замер отклика был точным)
	startTime := time.Now()

	// Создаем объект HTTP-запроса (метод GET)
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return fmt.Errorf("-- Не удалось создать HTTP-запрос. Детали: %w", err)
	}

	// Add Basic Auth
	req.SetBasicAuth(c.cfg.DmUser, c.cfg.DmPass)

	// Выполняем запрос через клиент
	resp, err := c.httpClient.Do(req)
	// Проверка на полное падение сервера или таймаут
	if err != nil {
		return fmt.Errorf("-- Сервер не отвечает или упал!\n   Детали: %w\n", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)
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
		}

		// Другие ошибки (401, 404, 500 и т.д.)
		fmt.Printf("-- service error: status %d\n", resp.StatusCode)
		if resp.StatusCode == http.StatusNotFound {
			fmt.Printf("--- База %s не опубликована на web-сервере %s\n", c.cfg.PublicationName, c.cfg.ApacheAddress)
		}
		if resp.StatusCode == http.StatusUnauthorized {
			fmt.Println("--- Неверный логин или пароль пользователя 1С.")
		}
	}

	// Проверяем скорость работы (тормоза)
	fmt.Printf("-- Время ответа сервера: %v", duration)
	if duration > c.cfg.ApacheWarningDuration {
		fmt.Printf(" Сервер сильно тормозит! Превышен лимит в %v\n", c.cfg.ApacheWarningDuration)
	} else {
		fmt.Printf(" Скорость работы в норме\n")
	}

	// Парсим JSON ответ от 1С
	var result DataMobileResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return fmt.Errorf(" ОШИБКА JSON: Не удалось прочитать ответ от 1С. Детали: %w\n", err)
	}

	// Выводим статус, который прислала сама 1С
	fmt.Printf("-- data: %s\n", strings.TrimSpace(result.Data))

	return nil
}
