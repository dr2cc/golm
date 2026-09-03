package main

import (
	"log"

	"github.com/dr2cc/golm/internal/app"
)

func main() {
	// Run
	if err := app.Run(); err != nil {
		log.Fatalf("error: %s", err)
	}
}
