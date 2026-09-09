package client

import (
	"net/http"
)

// Client инкапсулирует настройки для работы с удаленным сервером.
// Хранение http.Client внутри структуры — это Go-идиома, позволяющая переиспользовать соединения.
type Client struct {
	httpClient *http.Client
	baseURL    string
	// username   string
	// password   string
}

// New создает и возвращает настроенный экземпляр Client.
// Мы явно передаем таймаут, избегая глобальных дефолтов.
// func New(baseURL, username, password string, timeout time.Duration) *Client {
func New(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{
			// Timeout: timeout,
		},
		baseURL: baseURL,
		// username: username,
		// password: password,
	}
}
