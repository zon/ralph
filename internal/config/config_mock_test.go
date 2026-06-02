package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockLoader_LoadFnCalled(t *testing.T) {
	m := &MockLoader{
		LoadFn: func() (*RalphConfig, error) {
			return WithVariant("custom"), nil
		},
	}
	cfg, err := m.Load()
	require.NoError(t, err)
	assert.Equal(t, "custom", cfg.Variant)
}

func TestMockLoader_LoadFnReturnsError(t *testing.T) {
	expectedErr := errors.New("load error")
	m := &MockLoader{
		LoadFn: func() (*RalphConfig, error) {
			return nil, expectedErr
		},
	}
	cfg, err := m.Load()
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, cfg)
}

func TestMockLoader_LoadFnNilReturnsAny(t *testing.T) {
	m := &MockLoader{}
	cfg, err := m.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 10, cfg.MaxIterations)
	assert.Equal(t, "main", cfg.DefaultBranch)
}
