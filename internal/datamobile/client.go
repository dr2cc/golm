package datamobile

import (
	"context"
	"net/http"

	"github.com/dr2cc/golm/internal/config"
)

// ClientConfig содержит строго то, что нужно для HTTP-запросов
type ClientConfig struct {
	Apache     config.ApacheConfig
	DataMobile config.DataMobileConfig
}

type Client struct {
	cfg        ClientConfig
	httpClient *http.Client
}

func NewClient(cfg ClientConfig, httpClient *http.Client) *Client {
	// После добавления
	// cfg.Apache.BaseURL()
	// Никаких склеек и проверок строк здесь больше не нужно! Конфиг сохраняется в своем первозданном виде.
	return &Client{
		cfg:        cfg,
		httpClient: httpClient,
	}
}

func (c *Client) Check(ctx context.Context) error {
	// Создаем координатор проверок, передавая туда точечные конфиги и самого клиента
	suite := NewDiagnosticSuite(c.cfg.Apache, c.cfg.DataMobile, c)
	return suite.Run(ctx)
}

// func (c *Client) Check(ctx context.Context) error {
// 	// Использование HEAD-запроса экономит трафик
// 	req, err := http.NewRequestWithContext(ctx, "HEAD", c.cfg.Apache.Address, nil)
// 	if err != nil {
// 		return fmt.Errorf("failed create HEAD request: %w", err)
// 	}

// 	// Формат ответа в рамках HTTP это *Response
// 	resp, err := c.httpClient.Do(req) //http.Get("http://" + hostPort)
// 	if err != nil {
// 		// return fmt.Errorf("network request failed: %w", err)
// 		// ТУТ
// 		var opErr *net.OpError
// 		if errors.As(err, &opErr) {
// 			// 1. Ошибка DNS (неверное имя хоста)
// 			var dnsErr *net.DNSError
// 			if errors.As(opErr.Err, &dnsErr) {
// 				return fmt.Errorf("сервер недоступен: хост %q не найден", dnsErr.Name)
// 			}

// 			// 2. Сервер не запущен (Connection Refused)
// 			// Проверяем как универсальный ECONNREFUSED, так и числовой код Windows 10061 (WSAECONNREFUSED)
// 			if errors.Is(opErr.Err, syscall.ECONNREFUSED) || isWindowsConnRefused(opErr.Err) {
// 				return fmt.Errorf("сервер не запущен по адресу %s (подключение отклонено)", c.cfg.Apache.Address)
// 			}
// 		}

// 		// 3. Обработка таймаута контекста
// 		if errors.Is(err, context.DeadlineExceeded) {
// 			return fmt.Errorf("время ожидания ответа от сервера %s истекло", c.cfg.Apache.Address)
// 		}

// 		// 4. Все остальные сетевые ошибки (общий случай)
// 		return fmt.Errorf("сетевой запрос к %s завершился ошибкой: %w", c.cfg.Apache.Address, err)
// 	}
// 	defer resp.Body.Close()

// 	// Проверяем, что сервер ответил корректным HTTP-статусом (например, 200 OK или 403/404, что тоже подтверждает работу Apache)
// 	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
// 		// Дополнительно можно проверить заголовок "Server"
// 		serverHeader := resp.Header.Get("Server") // например, "Apache/2.4.41 (Ubuntu)"
// 		fmt.Printf("- It just works! %s\n", c.cfg.Apache.Address+" - "+serverHeader)

// 		// Смотрим конфигурацию Apache (если это локальный компьютер)
// 		if strings.Contains(c.cfg.Apache.Address, "127.0.0.1") || strings.Contains(c.cfg.Apache.Address, "localhost") { // "localhost""192.168.0.13" {
// 			if err := ShowApache1CModule(c.cfg.Apache.ConfPath); err != nil {
// 				return err
// 			}
// 		}
// 		// Тестируем наш RESTful-сервис ("РЕСТный" сервис)
// 		if err := c.CheckDataMobileService(); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// // Вспомогательная функция для отлова специфичного кода Windows connectex (WSAECONNREFUSED)
// func isWindowsConnRefused(err error) bool {
// 	if err == nil {
// 		return false
// 	}
// 	// Разворачиваем возможный os.SyscallError
// 	var sysErr *os.SyscallError
// 	if errors.As(err, &sysErr) {
// 		err = sysErr.Err
// 	}
// 	// Проверяем внутренний числовой код ошибки (10061 для Windows)
// 	var errno syscall.Errno
// 	if errors.As(err, &errno) {
// 		return errno == 10061 // WSAECONNREFUSED
// 	}
// 	return false
// }

// // Функция чтения версии модуля 1С из локального Apache
// func ShowApache1CModule(configPath string) error {
// 	file, err := os.Open(configPath)
// 	if err != nil {
// 		return fmt.Errorf("- apache error: failed to open httpd.conf: %w", err)
// 	}
// 	defer file.Close()

// 	scanner := bufio.NewScanner(file)
// 	found := false

// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		if strings.Contains(line, "_1cws_module") && strings.Contains(line, "LoadModule") {
// 			fmt.Printf("- Модуль 1С в конфигурации Apache:%s\n", strings.TrimSpace(line))
// 			found = true
// 			break
// 		}
// 	}
// 	// Обязательно проверяем, почему остановился цикл scanner.Scan()
// 	if err := scanner.Err(); err != nil {
// 		return fmt.Errorf("apache error: scanner encountered an error while reading httpd.conf: %w", err)
// 	}

// 	if !found {
// 		return fmt.Errorf("Модуль '_1cws_module' не найден или закомментирован в httpd.conf")
// 	}

// 	return nil
// }

// // Функция проверки доступности и скорости RESTful-сервиса
// func (c *Client) CheckDataMobileService() error {
// 	targetURL := c.cfg.Apache.Address + "/" + c.cfg.Apache.PublicationName + "/hs/DataMobileExch/"
// 	fmt.Printf("\n- GET to endpoint (DataMobile): %s\n", targetURL)

// 	// Время до создания запроса (чтобы замер отклика был точным)
// 	startTime := time.Now()

// 	// Создаем объект HTTP-запроса (метод GET)
// 	req, err := http.NewRequest("GET", targetURL, nil)
// 	if err != nil {
// 		return fmt.Errorf("-- Не удалось создать HTTP-запрос. Детали: %w", err)
// 	}

// 	// Add Basic Auth
// 	req.SetBasicAuth(c.cfg.DataMobile.User, c.cfg.DataMobile.Pass)

// 	// Выполняем запрос через клиент
// 	resp, err := c.httpClient.Do(req)
// 	// Проверка на полное падение сервера или таймаут
// 	if err != nil {
// 		return fmt.Errorf("-- Сервер не отвечает или упал!\n   Детали: %w\n", err)
// 	}
// 	defer resp.Body.Close()

// 	duration := time.Since(startTime)
// 	// Проверяем HTTP статус-код (теперь должен быть 200)
// 	if resp.StatusCode != http.StatusOK {
// 		// Обыгрываем конфликт версий (HTTP 409)
// 		if resp.StatusCode == http.StatusConflict {
// 			bodyBytes, _ := io.ReadAll(resp.Body)
// 			bodyStr := string(bodyBytes)

// 			fmt.Println("-- status", resp.StatusCode, "Конфликт версий 1С:")

// 			// Регулярное выражение для поиска версий в круглых скобках
// 			re := regexp.MustCompile(`\((\d+\.\d+\.\d+\.\d+)\s*-\s*(\d+\.\d+\.\d+\.\d+)\)`)
// 			matches := re.FindStringSubmatch(bodyStr)

// 			if len(matches) == 3 {
// 				fmt.Printf("--- Версия wsap24: %s\n", matches[1])
// 				fmt.Printf("--- Кластер    1С: %s\n", matches[2])
// 				fmt.Println("-- Перепубликуйте базу или обновите путь к wsap24-модулю в httpd.conf.")
// 			} else {
// 				// Если текст ошибки изменился, выводим как есть
// 				fmt.Printf("   Технические детали:\n%s\n", bodyStr)
// 			}
// 		}

// 		// Другие ошибки (401, 404, 500 и т.д.)
// 		fmt.Printf("-- service error: status %d\n", resp.StatusCode)
// 		if resp.StatusCode == http.StatusNotFound {
// 			fmt.Printf("--- База %s не опубликована на web-сервере %s\n", c.cfg.Apache.PublicationName, c.cfg.Apache.Address)
// 		}
// 		if resp.StatusCode == http.StatusUnauthorized {
// 			fmt.Println("--- Неверный логин или пароль пользователя 1С.")
// 		}
// 	}

// 	// Проверяем скорость работы (тормоза)
// 	fmt.Printf("-- Время ответа сервера: %v", duration)
// 	if duration > c.cfg.Apache.WarningDuration {
// 		fmt.Printf(" Сервер сильно тормозит! Превышен лимит в %v\n", c.cfg.Apache.WarningDuration)
// 	} else {
// 		fmt.Printf(" Скорость работы в норме\n")
// 	}

// 	// Парсим JSON ответ от 1С
// 	var result DataMobileResponse
// 	err = json.NewDecoder(resp.Body).Decode(&result)
// 	if err != nil {
// 		return fmt.Errorf(" ОШИБКА JSON: Не удалось прочитать ответ от 1С. Детали: %w\n", err)
// 	}

// 	// Выводим статус, который прислала сама 1С
// 	fmt.Printf("-- data: %s\n", strings.TrimSpace(result.Data))

// 	return nil
// }
