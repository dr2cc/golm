package lmcz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/dr2cc/golm/internal/scanner"
)

// Init — публичный метод. Он координирует процесс: проверяет статус
// и при необходимости выполняет инициализацию, переиспользуя логику Check.
func (c *Client) Init(ctx context.Context) error {
	// 1. Делаем предварительную проверку статуса.
	// Метод Check сам разберется: сканировать сеть или брать жесткий Host,
	// очистит префиксы, сделает GET-запрос и вернет строковый статус.
	checkResult, err := c.Status(ctx)
	if err != nil {
		return fmt.Errorf("предварительная проверка статуса перед инициализацией неудачна: %w", err)
	}

	fmt.Printf("LMCZ server response: (%d) status: %s\n", checkResult.HttpCode, checkResult.Status)

	// 2. Бизнес-логика: отправляем токен только если сервер не настроен
	if checkResult.Status != "not_configured" {
		fmt.Println("Сервер LMCZ уже настроен. Отправка токена инициализации не требуется.")
		return nil
	}

	fmt.Println("Server not configured. Sending token....")

	// 3. Вызываем приватный метод для отправки токена.
	// Так как в Check мы уже гарантированно проверили адрес, мы можем вызвать
	// вспомогательную функцию для получения правильного hostPort, чтобы не сканировать сеть дважды.
	hostPort := c.getTargetHostPort()
	if err := c.sendToken(ctx, hostPort); err != nil {
		return err
	}

	return nil
}

// sendToken — приватный метод. Отвечает строго за техническую
// отправку POST-запроса на конкретный хост.
func (c *Client) sendToken(ctx context.Context, hostPort string) error {
	payload := InitRequestPayload{Token: c.cfg.TokenXAPIKEY}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload to JSON: %w", err)
	}

	targetURL := fmt.Sprintf("http://%s/api/v2/init", hostPort)

	// Формируем запрос:
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.cfg.User, c.cfg.Pass)

	// Обрабатывем ответ:
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Распаковываем ошибку http-клиента, теперь это работает без ошибок компилятора
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return fmt.Errorf("сервер недоступен по адресу %s: %w", hostPort, urlErr.Err)
		}
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем HTTP статус-код (все что вне диапазона 2xx — ошибка)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr InitErrorPayload

		// Пытаемся прочитать JSON-ответ от сервера, чтобы узнать точную причину ошибки
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Reason != "" {
			// Оборачиваем apiErr. Названия полей (errorCode, reason)
			// подставятся автоматически благодаря методу Error() у InitErrorPayload
			return fmt.Errorf("(%d) %w", resp.StatusCode, apiErr)
		}

		return fmt.Errorf("unexpected status code: (%d) %w", resp.StatusCode, apiErr)
		// // Если сервер прислал не JSON (например, ошибку nginx), отдаем стандартный статус
		// return fmt.Errorf("unexpected status codee: %d (%s)", resp.StatusCode, resp.Status)
	}

	// Если статус 2xx — успешно декодируем финальный ответ
	var result InitResponsePayload
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// При успешной инициализации возвращается пустое тело и StatusOK,
		if errors.Is(err, io.EOF) {
			fmt.Printf("Успех (%d)!", resp.StatusCode)
			return nil
		}
		// Вдруг что-то еще..
		return fmt.Errorf("failed to decode JSON response (HTTP %d): %w", resp.StatusCode, err)
	}

	fmt.Printf("Ответ сервера на инициализацию: %s (Сообщение: %s)\n", result.Status, result.Message)
	return nil
}

// Вспомогательный приватный метод, чтобы не дублировать логику склейки хоста и порта
func (c *Client) getTargetHostPort() string {
	if c.cfg.Host == "" {
		// Если хост пустой, значит на этапе Check мы сканировали сеть.
		// Чтобы не запускать ScanSubnet повторно (это долго), сканер должен уметь
		// возвращать кешированный или первый найденный результат, либо
		// (если ScanSubnet быстрый) вызываем его:
		return scanner.ScanSubnet(c.cfg.Subnet, c.cfg.Port, c.cfg.ScannerTimeout)
	}

	cleanHost := strings.TrimPrefix(c.cfg.Host, "http://")
	cleanHost = strings.TrimPrefix(cleanHost, "https://")
	return net.JoinHostPort(cleanHost, c.cfg.Port)
}
