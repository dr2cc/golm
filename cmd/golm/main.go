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
	if err := app.Run(*cfg); err != nil {
		log.Fatalf("error: %s", err)
	}
}
