package main

import (
	"flag"
	"log"
	"os"

	"pine-ai/config"
	"pine-ai/router"
)

func main() {
	configPath := flag.String("config", "config.json", "local config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	r := router.New(cfg)

	addr := ":" + cfg.Port
	if cfg.Port == "" {
		addr = ":8080"
	}
	if portFromEnv := os.Getenv("PORT"); portFromEnv != "" {
		addr = ":" + portFromEnv
	}

	log.Printf("server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server start failed: %v", err)
	}
}
