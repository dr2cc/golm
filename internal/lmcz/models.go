package lmcz

// Эндпойнт возвращает только статус (например: {"status": "not_configured"})
type StatusResponse struct {
	Status string `json:"status"`
}

// Эндпойнт инициализирует систему и возвращает, например, ID сессии
type InitResponse struct {
	ErrorCode string `json:"errorCode"`
	Reason    string `json:"reason"`
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
