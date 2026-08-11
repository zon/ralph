package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAny_ReturnsValidDefaultState(t *testing.T) {
	proj := Any()
	assert.NotNil(t, proj)
	assert.Equal(t, "test-project", proj.Slug)
	assert.NotEmpty(t, proj.Items)
}
