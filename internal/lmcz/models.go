package lmcz

import "fmt"

type CheckResult struct {
	Status   string
	HttpCode int
}

// Эндпойнт GET /status только возвращает состояние сервера ЛМЧЗ,
// нам нужно поле status (например: {"status": "not_configured"})
type StatusEndpointResponse struct {
	Status string `json:"status"`
}

// InitRequestPayload описывает структуру тела POST-запроса.
type InitRequestPayload struct {
	Token string `json:"token"`
}

// InitErrorPayload описывает JSON ответ ошибки от /init
type InitErrorPayload struct {
	ErrorCode int    `json:"errorCode"`
	Reason    string `json:"reason"`
}

// Реализуем интерфейс error. Теперь InitErrorPayload — это тоже error!
func (e InitErrorPayload) Error() string {
	return fmt.Sprintf("errorCode: %d, reason: %s", e.ErrorCode, e.Reason)
}

// InitResponsePayload описывает успешный JSON ответ от /init
type InitResponsePayload struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
