package lmcz

import (
	"net/http"
	"strings"

	"github.com/dr2cc/golm/internal/config"
)

// Client инкапсулирует настройки для работы с удаленным сервером.
// Хранение http.Client внутри структуры — это Go-идиома, позволяющая переиспользовать соединения.
type Client struct {
	cfg        config.LmczConfig
	httpClient *http.Client
}

// New создает и возвращает настроенный экземпляр Client.
// Мы явно передаем таймаут, избегая глобальных дефолтов.
func NewClient(lmcz config.LmczConfig, httpClient *http.Client) *Client {
	// Добавляем схему при инициализации, если забыли указать
	if !strings.HasPrefix(lmcz.LmczHost, "http://") && !strings.HasPrefix(lmcz.LmczHost, "https://") {
		lmcz.LmczHost = "http://" + lmcz.LmczHost
	}
	return &Client{
		cfg:        lmcz,
		httpClient: httpClient,
	}
}
