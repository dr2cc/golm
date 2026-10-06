package lmcz

import (
	"net/http"

	"github.com/dr2cc/golm/internal/config"
)

// Client инкапсулирует настройки для работы с удаленным сервером.
// Хранение http.Client внутри структуры — это Go-идиома, позволяющая переиспользовать соединения.
type Client struct {
	cfg        config.LmczConfig
	httpClient *http.Client
}

// New создает и возвращает настроенный экземпляр Client
func NewClient(lmcz config.LmczConfig, httpClient *http.Client) *Client {
	return &Client{
		cfg:        lmcz,
		httpClient: httpClient,
	}
}
