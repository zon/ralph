package cmd

import (
	"github.com/zon/ralph/internal/context"
)

func createExecutionContext() *context.Context {
	return context.NewContextFromEnv()
}
