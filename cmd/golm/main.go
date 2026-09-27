package main

import (
	"log"

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
		log.Fatalf("error: %s", err)
	}
}
