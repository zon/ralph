package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/services"
)

// Example demonstrating service orchestration with dry-run support
func main() {
	// Set verbose logging
	logger.SetVerbose(true)

	// Define some example services
	exampleServices := []config.Service{
		{
			Name:    "database",
			Command: "sleep",
			Args:    []string{"5"},
			Port:    5432, // Example port (not actually used by sleep)
		},
		{
			Name:    "api-server",
			Command: "sleep",
			Args:    []string{"5"},
			Port:    3000,
		},
		{
			Name:    "worker",
			Command: "sleep",
			Args:    []string{"5"},
			// No port = no health check
		},
	}

	// Determine if we're in dry-run mode
	dryRun := len(os.Args) > 1 && os.Args[1] == "--dry-run"

	if dryRun {
		fmt.Println("=== DRY-RUN MODE ===")
		fmt.Println("Simulating service orchestration without executing")
		fmt.Println()
	} else {
		fmt.Println("=== REAL EXECUTION ===")
		fmt.Println("Starting services...")
		fmt.Println()
	}

	// Start all services
	var processes []*services.Process
	for _, svc := range exampleServices {
		process, err := services.StartService(svc, dryRun)
		if err != nil {
			logger.Error("Failed to start service %s: %v", svc.Name, err)
			services.StopAllServices(processes)
			os.Exit(1)
		}
		processes = append(processes, process)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if !dryRun {
		fmt.Println()
		logger.Info("Services running. Press Ctrl+C to stop...")
		fmt.Println()

		// Wait for signal or timeout
		select {
		case sig := <-sigChan:
			fmt.Println()
			logger.Info("Received signal: %v", sig)
		case <-time.After(3 * time.Second):
			fmt.Println()
			logger.Info("Example timeout reached")
		}
	} else {
		fmt.Println()
		logger.Info("In dry-run mode - services were not actually started")
	}

	// Stop all services
	fmt.Println()
	logger.Info("Stopping all services...")
	services.StopAllServices(processes)

	fmt.Println()
	logger.Success("Example completed successfully")
}
