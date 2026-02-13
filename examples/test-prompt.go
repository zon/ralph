package main

import (
	"fmt"
	"os"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/prompt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run examples/test-prompt.go <project-file>")
		os.Exit(1)
	}

	projectFile := os.Args[1]
	ctx := context.NewContext(false, true, false, false)

	fmt.Println("=== Building Development Prompt ===\n")

	promptText, err := prompt.BuildDevelopPrompt(ctx, projectFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(promptText)
	fmt.Printf("\n=== Prompt Statistics ===\n")
	fmt.Printf("Total length: %d characters\n", len(promptText))
}
