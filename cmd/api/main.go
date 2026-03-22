package main

import (
	"log"

	"pos-backend/internal/app"
	"pos-backend/internal/config"
)

func main() {
	cfg := config.Load()

	server, err := app.NewServer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("starting %s on %s", cfg.AppName, cfg.HTTPAddress())
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
