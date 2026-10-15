package lmcz

import (
	"net/http"

	"github.com/dr2cc/golm/internal/config"
)

// CzClient инкапсулирует настройки для работы с удаленным сервером.
// Хранение http.CzClient внутри структуры — это Go-идиома, позволяющая переиспользовать соединения.
type CzClient struct {
	cfg        config.LmczConfig
	httpClient *http.Client
}

// New создает и возвращает настроенный экземпляр Client
func NewCzClient(lmcz config.LmczConfig, httpClient *http.Client) *CzClient {
	return &CzClient{
		cfg:        lmcz,
		httpClient: httpClient,
	}
}
