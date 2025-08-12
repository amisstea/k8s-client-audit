package main

import (
	"context"
	"log"
	"os"

	"cursor-experiment/internal/app"
)

func main() {
	if err := app.Run(context.Background(), os.Args[1:]); err != nil {
		log.Fatalf("error: %v", err)
	}
}
