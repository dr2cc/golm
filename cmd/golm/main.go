package main

import (
	"fmt"
	"log"
	"os"

	"github.com/dr2cc/golm/internal/app"
	"github.com/dr2cc/golm/internal/config"
)

func main() {
	// Configuration
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	// Run
	// Явное лучше неявного (Explicit over Implicit):
	// Функция app.Run(cfg) декларирует: «Мне для работы нужен cfg»
	if err := app.Run(*cfg); err != nil {
		// Выводим ТОЛЬКО текст ошибки в поток ошибок, без даты и системных приписок
		fmt.Fprintf(os.Stderr, "Server response: %v\n", err)
		os.Exit(1) // Завершаем программу чисто с кодом 1
	}
}
