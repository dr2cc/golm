package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// RequestPayload описывает структуру тела POST-запроса.
type RequestPayload struct {
	Token string `json:"token"`
}

// ResponsePayload описывает JSON, который мы получаем ОТ сервера в ответ.
type ResponsePayload struct {
	// OperationMode string `json:"operationMode"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// type Client struct {
// 	httpClient *http.Client
// 	baseURL    string
// 	username   string
// 	password   string
// }

// // New создает и возвращает настроенный экземпляр Client.
// // Мы явно передаем таймаут, избегая глобальных дефолтов.
// func New(baseURL, username, password string, timeout time.Duration) *Client {
// 	return &Client{
// 		httpClient: &http.Client{
// 			Timeout: timeout,
// 		},
// 		baseURL:  baseURL,
// 		username: username,
// 		password: password,
// 	}
// }

// SendToken отправляет POST-запрос с JSON-телом и Basic Auth.
// Метод принимает context.Context, что позволяет отменять запрос извне (например, при выходе из CLI).
func (c *Client) SendToken(ctx context.Context, token string) (*ResponsePayload, error) {
	// 1. Формируем структуру данных для JSON
	payload := RequestPayload{Token: token}

	// 2. Сериализуем структуру в JSON-байты
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload to JSON: %w", err)
	}

	url := "http://" + c.baseURL + "/api/v2/init"

	// 3. Создаем запрос с поддержкой контекста (для избежания зависания запросов)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// 4. Устанавливаем заголовки (Headers)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 5. Добавляем Basic Authentication
	req.SetBasicAuth(c.username, c.password)

	// 6. Выполняем сетевой запрос через переиспользуемый httpClient
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			// Если мы знаем, что сервер обрабатывает запрос, несмотря на таймаут
			return &ResponsePayload{
				Status:  "timeout_accepted",
				Message: "Запрос отправлен, но ответ не был получен в отведенное время (сервер думает слишком долго)",
			}, nil
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	// Важно: закрываем тело ответа, только если err == nil
	defer resp.Body.Close()

	// 7. Проверяем HTTP статус-код (200 OK или 201 Created и т.д.)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d (%s)", resp.StatusCode, resp.Status)
	}

	// 8. Максимально эффективно декодируем ответ «на лету» без выделения лишней памяти
	var result ResponsePayload
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return &result, nil
}
