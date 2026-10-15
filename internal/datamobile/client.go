package datamobile

import (
	"net/http"

	"github.com/dr2cc/golm/internal/config"
)

// ClientConfig содержит строго то, что нужно для HTTP-запросов
type ClientConfig struct {
	Apache     config.ApacheConfig
	DataMobile config.DataMobileConfig
}

type DmClient struct {
	httpClient *http.Client
	cfg        ClientConfig
}

// The provided HTTP client will be mutated for use with Dm, so it should not
// be reused elsewhere.
// Переданный HTTP-клиент будет изменен для работы с Dm, поэтому его не следует
// использовать где-либо еще.
func NewDmClient(httpClient *http.Client, cfg ClientConfig) *DmClient {
	// 1. Проверяем: если транспорт nil, берем стандартный движок Go
	baseTransport := httpClient.Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}

	// 2. Оборачиваем его в ваш кастомный dmTransport
	httpClient.Transport = newDmTransport(cfg.DataMobile.User, cfg.DataMobile.Pass, baseTransport)

	// // Transport adds auth headers
	// // Добавлю в этот клиент
	// httpClient.Transport = newDmTransport(cfg.DataMobile.User, cfg.DataMobile.Pass, httpClient.Transport)
	return &DmClient{
		httpClient: httpClient,
		cfg:        cfg,
	}
}

type dmTransport struct {
	rp       http.RoundTripper
	username string
	password string
}

func newDmTransport(user, pass string, rp http.RoundTripper) *dmTransport {
	return &dmTransport{
		rp:       rp,
		username: user,
		password: pass,
	}
}

func (dt *dmTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// req.Header.Set("Authorization", "Bearer "+dt.token)
	req.SetBasicAuth(dt.username, dt.password)
	return dt.rp.RoundTrip(req)
}
