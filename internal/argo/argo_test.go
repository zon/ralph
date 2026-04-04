package argo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestK8sContext(t *testing.T) {
	ctx := K8sContext{
		Name:      "test-context",
		Namespace: "test-namespace",
	}

	assert.Equal(t, "test-context", ctx.Name)
	assert.Equal(t, "test-namespace", ctx.Namespace)
}
