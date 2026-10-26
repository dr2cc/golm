package lmcz

import (
	"io"
	"net/http"
	"os"

	"github.com/dr2cc/golm/internal/config"
	"github.com/dr2cc/golm/internal/report"
)

// CzClient инкапсулирует настройки для работы с удаленным сервером.
// Хранение http.CzClient внутри структуры — это Go-идиома, позволяющая переиспользовать соединения.
type CzClient struct {
	cfg        config.LmczConfig
	httpClient *http.Client

	// [GUI-рефакторинг] out — куда клиент пишет сообщения о ходе работы.
	// Раньше методы печатали напрямую через fmt.Printf (то есть всегда в консоль).
	// Теперь печать идет через fmt.Fprintf(c.out, ...): для CLI это по-прежнему os.Stdout,
	// а GUI подставляет свой writer и показывает те же строки в окне лога.
	out io.Writer

	// [GUI-статусы] onStep получает изменения состояния шагов (поиск сервера, статус, токен).
	// В CLI не задается (nil) — тогда шаги никуда не отправляются.
	onStep report.Func
}

// New создает и возвращает настроенный экземпляр Client
func NewCzClient(lmcz config.LmczConfig, httpClient *http.Client) *CzClient {
	return &CzClient{
		cfg:        lmcz,
		httpClient: httpClient,
		// [GUI-рефакторинг] По умолчанию — стандартный вывод, поэтому CLI работает как раньше,
		// а сигнатура конструктора не изменилась.
		out: os.Stdout,
	}
}

// [GUI-рефакторинг] SetOutput перенаправляет вывод клиента (используется GUI).
// Возвращает сам клиент, чтобы можно было писать цепочкой:
// lmcz.NewCzClient(cfg, httpClient).SetOutput(w)
func (c *CzClient) SetOutput(w io.Writer) *CzClient {
	c.out = w
	return c
}

// [GUI-статусы] SetStepHandler подключает получателя статусов шагов (используется GUI).
func (c *CzClient) SetStepHandler(f report.Func) *CzClient {
	c.onStep = f
	return c
}
