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

	// The event handler is a no-op for now; workflow invocation will be wired
	// up as part of the workflow-invocation requirement.
	handler := func(eventType string, payload map[string]interface{}) {
		log.Printf("received %s event for repository %v/%v",
			eventType,
			nestedStr(payload, "repository", "owner", "login"),
			nestedStr(payload, "repository", "name"),
		)
	}

	s := webhook.NewServer(cfg, handler)
	log.Printf("starting github-webhook service on port %d", cfg.App.Port)
	if err := s.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

// nestedStr walks a nested map[string]interface{} by key path and returns the
// final string value, or "" if any step is missing.
func nestedStr(m map[string]interface{}, keys ...string) string {
	var cur interface{} = m
	for _, k := range keys {
		mp, ok := cur.(map[string]interface{})
		if !ok {
			return ""
		}
		cur = mp[k]
	}
	s, _ := cur.(string)
	return s
}
