package datamobile

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/dr2cc/golm/internal/config"
)

// DiagnosticSuite через свои методы управляет набором проверок
// и координирует процесс диагностики:
// - веб-сервера Apache,
// - публикации платформы 1С,
// - HTTP-сервиса DataMobile.
type DiagnosticSuite struct {
	apacheCfg     config.ApacheConfig
	dataMobileCfg config.DataMobileConfig
	client        *Client
}

// Главный смысл этого конструктора- создание экземпляра DiagnosticSuite с его методами.
// В конструктор передаем только то, что реально нужно для диагностики.
// Передавать можно по указателю (*ApacheConfig) или по значению,
// в зависимости от текущих предпочтений.
func NewDiagnosticSuite(apacheCfg config.ApacheConfig, dmCfg config.DataMobileConfig, client *Client) *DiagnosticSuite {
	return &DiagnosticSuite{
		apacheCfg:     apacheCfg,
		dataMobileCfg: dmCfg,
		client:        client,
	}
}

// Run — координатор
func (ds *DiagnosticSuite) Run(ctx context.Context) error {
	// 1. Сначала проверяем только сам Apache
	isAlive, err := ds.pingApache(ctx)
	if err != nil {
		return fmt.Errorf("Connection failure! %w", err)
	}

	// Если статус некорректный (например, 5xx), завершаем проверку досрочно без ошибки.
	// Ведь Apache ответил, но глубокая диагностика сейчас невозможна.
	if !isAlive {
		return nil
	}

	// 2. Если Apache "живой", формируем цепочку последующих проверок
	subSteps := []func(ctx context.Context) error{
		ds.checkLocalConfig,
		ds.checkDataMobileService,
	}

	for _, step := range subSteps {
		if err := step(ctx); err != nil {
			return err // Прерываемся при первой ошибке во вложенных шагах
		}
	}

	return nil
}

// pingApache возвращает
// (true, nil) если статус ок,
// (false, nil) если статус >= 500, и
// (false, err) при сетевой ошибке
func (ds *DiagnosticSuite) pingApache(ctx context.Context) (bool, error) {
	// Использование HEAD-запроса экономит трафик
	req, err := http.NewRequestWithContext(ctx, "HEAD", ds.apacheCfg.BaseURL(), nil)
	if err != nil {
		return false, fmt.Errorf("failed to create HEAD request: %w", err)
	}

	resp, err := ds.client.httpClient.Do(req)
	if err != nil {
		// //
		var opErr *net.OpError
		if errors.As(err, &opErr) {
			// 1. Ошибка DNS (неверное имя хоста)
			var dnsErr *net.DNSError
			if errors.As(opErr.Err, &dnsErr) {
				return false, fmt.Errorf("Сервер недоступен: хост %q не найден", dnsErr.Name)
			}

			// 2. Сервер не запущен (Connection Refused)
			// Проверяем как универсальный ECONNREFUSED, так и числовой код Windows 10061 (WSAECONNREFUSED)
			if errors.Is(opErr.Err, syscall.ECONNREFUSED) || isWindowsConnRefused(opErr.Err) {
				return false, fmt.Errorf("Web server not running at %s (connection refused)", ds.apacheCfg.BaseURL())
			}
		}

		// 3. Обработка таймаута контекста
		if errors.Is(err, context.DeadlineExceeded) {
			return false, fmt.Errorf("Время ожидания ответа от сервера %s истекло", ds.apacheCfg.BaseURL())
		}

		// 4. Все остальные сетевые ошибки (общий случай)
		// return false, fmt.Errorf("failed to execute HEAD request: %w", err)
		return false, fmt.Errorf("Сетевой запрос к %s завершился ошибкой: %w", ds.apacheCfg.BaseURL(), err)
	}
	defer resp.Body.Close()

	// Проверяем корректность статуса (200-499)
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		serverHeader := resp.Header.Get("Server")
		fmt.Printf("- It just works! %s - %s\n", ds.apacheCfg.BaseURL(), serverHeader)
		return true, nil // Разрешаем дальнейшие проверки
	}

	fmt.Printf("- Apache responded with server error status: %d\n", resp.StatusCode)
	return false, nil // Запрещаем дальнейшие проверки, но это не критическая ошибка самой диагностики
}

// Вспомогательная функция для отлова специфичного кода Windows connectex (WSAECONNREFUSED)
func isWindowsConnRefused(err error) bool {
	if err == nil {
		return false
	}
	// Разворачиваем возможный os.SyscallError
	var sysErr *os.SyscallError
	if errors.As(err, &sysErr) {
		err = sysErr.Err
	}
	// Проверяем внутренний числовой код ошибки (10061 для Windows)
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno == 10061 // WSAECONNREFUSED
	}
	return false
}

func (ds *DiagnosticSuite) checkLocalConfig(ctx context.Context) error {
	// Чисто, коротко и понятно, откуда берутся данные
	addr := ds.apacheCfg.BaseURL()
	if !strings.Contains(addr, "127.0.0.1") && !strings.Contains(addr, "localhost") {
		return nil
	}

	if err := ShowApache1CModule(ds.apacheCfg.ConfPath); err != nil {
		return fmt.Errorf("local apache config module error: %w", err)
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

// checkDataMobileService — Функция проверки доступности и скорости RESTful-сервиса
func (ds *DiagnosticSuite) checkDataMobileService(ctx context.Context) error {
	targetURL := ds.apacheCfg.BaseURL() + "/" + ds.apacheCfg.PublicationName + "/hs/DataMobileExch/"
	fmt.Printf("\n- GET to endpoint (DataMobile): %s\n", targetURL)

	// Время до создания запроса (чтобы замер отклика был точным)
	startTime := time.Now()

	// Используем Context для корректной отмены/таймаута запроса
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return fmt.Errorf("-- Не удалось создать HTTP-запрос. Детали: %w", err)
	}

	// Add Basic Auth
	req.SetBasicAuth(ds.dataMobileCfg.User, ds.dataMobileCfg.Pass)

	// Выполняем запрос через клиент
	resp, err := ds.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("-- Сервер не отвечает или упал!\n   Детали: %w\n", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)

	// Выводим скорость работы (тормоза) независимо от статуса, так как замер уже сделан
	fmt.Printf("-- Версии 1С в публикации и ИБ совпадают. Время ответа: %v", duration)
	if duration > ds.apacheCfg.WarningDuration {
		fmt.Printf(" Сервер сильно тормозит! Превышен лимит в %v\n", ds.apacheCfg.WarningDuration)
	} else {
		fmt.Printf(" Скорость работы в норме\n")
	}

	// ОБРАБОТКА ОШИБОК: Если HTTP статус-код НЕ 200 OK
	if resp.StatusCode != http.StatusOK {
		// Обыгрываем конфликт версий (HTTP 409)
		if resp.StatusCode == http.StatusConflict {
			bodyBytes, _ := io.ReadAll(resp.Body)
			bodyStr := string(bodyBytes)

			fmt.Println("-- status", resp.StatusCode, "Конфликт версий 1С:")

			re := regexp.MustCompile(`\((\d+\.\d+\.\d+\.\d+)\s*-\s*(\d+\.\d+\.\d+\.\d+)\)`)
			matches := re.FindStringSubmatch(bodyStr)

			if len(matches) == 3 {
				fmt.Printf("--- Версия wsap24: %s\n", matches[1])
				fmt.Printf("--- Кластер    1С: %s\n", matches[2])
				fmt.Println("-- Перепубликуйте базу или обновите путь к wsap24-модулю in httpd.conf.")
			} else {
				fmt.Printf("   Технические детали:\n%s\n", bodyStr)
			}
			return fmt.Errorf("1C version conflict (HTTP 409)")
		}

		// Другие ошибки (401, 404, 500 и т.д.)
		fmt.Printf("-- service error: status %d\n", resp.StatusCode)
		if resp.StatusCode == http.StatusNotFound {
			fmt.Printf("--- База %s не опубликована на %s\n", ds.apacheCfg.PublicationName, ds.apacheCfg.BaseURL())
		}
		if resp.StatusCode == http.StatusUnauthorized {
			fmt.Println("--- Неверный логин или пароль пользователя 1С.")
		}

		// ВАЖНО: Возвращаем ошибку здесь, чтобы код не шел дальше пытаться парсить возвращаемый 1С HTML как JSON
		return fmt.Errorf("mobile service responded with status %d", resp.StatusCode)
	}

	// Парсим JSON ответ от hs DataMobile (значит, 1С ни к чему не придралась и пропустила запрос).
	// Сюда код дойдет ТОЛЬКО при статусе 200 OK
	var result DataMobileResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return fmt.Errorf(" ОШИБКА JSON: Не удалось прочитать ответ. Детали: %w\n", err)
	}

	// Выводим статус, который прислал hs DataMobile
	fmt.Printf("-- data: %s\n", strings.TrimSpace(result.Data))

	return nil
}
