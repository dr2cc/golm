package cmdutil

import "net/http"

// Factory инкапсулирует создание общих зависимостей, как в GitHub CLI
type Factory struct {
	// Поле-функция для генерации стандартного HTTP-клиента
	HttpClient func() (*http.Client, error)
}

// NewFactory — конструктор фабрики.
// Здесь мы один раз настраиваем, КAK именно будут создаваться http-клиенты в приложении.
func NewFactory() *Factory {
	return &Factory{
		HttpClient: func() (*http.Client, error) {
			// Настраиваем базовый сетевой движок Go (гарантируем, что Transport != nil)
			// Настройки таймаутов на установку соединения ( Dial / TLS ) берутся дефолтные
			return &http.Client{
				Transport: http.DefaultTransport,
				// Жесткий Timeout здесь не ставим (0 = no timeout),
				// так как вы управляете временем запроса через context.WithTimeout сверху!
			}, nil
		},
	}
}
