package main

import (
	"log"

	"github.com/dr2cc/golm/internal/cli"
)

func main() {
	// Initialize client (parse flags, read config)
	client := cli.New() // Не обрабатываем ошибки
	// Внутри функции cli.New() нет кода, который может завершиться сбоем (функция flag.String и чтение констант не возвращают ошибок)

	// Run
	if err := client.Run(); err != nil {
		log.Fatalf("error: %s", err)
	}
}
