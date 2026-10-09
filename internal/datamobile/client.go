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
	// Создаем координатор проверок, передавая туда точечные конфиги и сам клиента
	suite := NewDiagnosticSuite(c.cfg.Apache, c.cfg.DataMobile, c)
	return suite.Run(ctx)
}
