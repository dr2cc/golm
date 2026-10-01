package lmcz

// Структура для парсинга ответа сервера (например: {"status": "not_configured"})
type StatusResponse struct {
	Status string `json:"status"`
}

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
