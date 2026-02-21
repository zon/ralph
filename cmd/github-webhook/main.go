package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/zon/ralph/internal/webhook"
)

func main() {
	var configPath string
	var secretsPath string

	flag.StringVar(&configPath, "config", "", "path to app config YAML file (overrides WEBHOOK_CONFIG env var)")
	flag.StringVar(&secretsPath, "secrets", "", "path to secrets YAML file (overrides WEBHOOK_SECRETS env var)")
	flag.Parse()

	cfg, err := webhook.LoadConfig(configPath, secretsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Wire up the invoker as the event handler so that comment events trigger
	// `ralph run` and approval events trigger `ralph merge`.
	inv := webhook.NewInvoker(false)
	handler := inv.HandleEvent(cfg)

	s := webhook.NewServer(cfg, handler)
	log.Printf("starting github-webhook service on port %d", cfg.App.Port)
	if err := s.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
