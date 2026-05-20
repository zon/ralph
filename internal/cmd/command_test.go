package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandCmd_MissingCommand(t *testing.T) {
	cmd := &CommandCmd{}
	err := cmd.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command required")
}