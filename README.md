Структура проекта:
golm/
├── cmd/
│   └── golm/
│       └── main.go       # Вызывает client.Run()
└── internal/
    ├── client/
    │   ├── client.go     # Функция Run(), конструктор NewClient() и общая структура
    │   ├── server.go     # Эндпоинт получения информации о сервере
    │   └── regLm.go      # Эндпоинт регистрации ЛМ ЧЗ
    └── scanner/
        └── scanner.go    # Вспомогательный сканер (находит IP)