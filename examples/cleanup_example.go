package main

import (
	"fmt"
	"os"
	"time"

	"github.com/zon/ralph/internal/cleanup"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/services"
)

// Example demonstrating graceful service shutdown with signal handling
func main() {
	// Create cleanup manager
	manager := cleanup.NewManager()

	// Set up signal handlers for graceful shutdown
	manager.SetupSignalHandlers()

	// Ensure cleanup runs on normal exit too
	defer manager.Cleanup()

	// Example: Start some services
	exampleServices := []config.Service{
		{
			Name:    "example-server",
			Command: "python3",
			Args:    []string{"-m", "http.server", "8080"},
			Port:    8080,
		},
	}

	fmt.Println("Starting services...")
	processes, err := services.StartAllServices(exampleServices, false)
	if err != nil {
		fmt.Printf("Failed to start services: %v\n", err)
		os.Exit(1)
	}

	// Register processes with cleanup manager
	manager.RegisterProcesses(processes)

	fmt.Println("Services running. Press Ctrl+C to stop.")
	fmt.Println("Testing graceful shutdown:")
	fmt.Println("  - SIGTERM will trigger cleanup")
	fmt.Println("  - Services have 5 seconds to stop gracefully")
	fmt.Println("  - After timeout, SIGKILL is sent")

	// Keep running until interrupted
	time.Sleep(1000 * time.Hour)
}
