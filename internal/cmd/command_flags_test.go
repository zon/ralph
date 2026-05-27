package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandFlags_FollowWithLocal(t *testing.T) {
	flags := CommandFlags{Follow: true, Local: true}
	err := flags.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--follow")
}