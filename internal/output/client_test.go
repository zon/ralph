package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	require.NotNil(t, NewClient(nil, nil, false))
}

func TestClientDebug_SuppressedWhenDebuggingFalse(t *testing.T) {
	var out, err bytes.Buffer
	NewClient(&out, &err, false).Debug("test debug")
	assert.Empty(t, out.String())
	assert.Empty(t, err.String())
}

func TestClientDebug_WritesToOutWhenDebuggingTrue(t *testing.T) {
	var out, err bytes.Buffer
	NewClient(&out, &err, true).Debug("test debug")
	assert.Contains(t, out.String(), "test debug")
	assert.Empty(t, err.String())
}

func TestClientDebugf_SuppressedWhenDebuggingFalse(t *testing.T) {
	var out, err bytes.Buffer
	NewClient(&out, &err, false).Debugf("test %s", "debug")
	assert.Empty(t, out.String())
	assert.Empty(t, err.String())
}

func TestClientDebugf_WritesToOutWhenDebuggingTrue(t *testing.T) {
	var out, err bytes.Buffer
	NewClient(&out, &err, true).Debugf("test %s", "debug")
	assert.Contains(t, out.String(), "test debug")
	assert.Empty(t, err.String())
}

func TestClientInfo_WritesToOut(t *testing.T) {
	var out, err bytes.Buffer
	NewClient(&out, &err, false).Info("test info")
	assert.Contains(t, out.String(), "test info")
	assert.Empty(t, err.String())
}

func TestClientInfof_WritesToOut(t *testing.T) {
	var out, err bytes.Buffer
	NewClient(&out, &err, false).Infof("test %s", "info")
	assert.Contains(t, out.String(), "test info")
	assert.Empty(t, err.String())
}

func TestClientWarn_WritesToOut(t *testing.T) {
	var out, err bytes.Buffer
	NewClient(&out, &err, false).Warn("test warning")
	assert.Contains(t, out.String(), "test warning")
	assert.Empty(t, err.String())
}

func TestClientWarnf_WritesToOut(t *testing.T) {
	var out, err bytes.Buffer
	NewClient(&out, &err, false).Warnf("test %s", "warning")
	assert.Contains(t, out.String(), "test warning")
	assert.Empty(t, err.String())
}

func TestClientError_WritesToErr(t *testing.T) {
	var out, err bytes.Buffer
	NewClient(&out, &err, false).Error("test error")
	assert.Contains(t, err.String(), "test error")
	assert.Empty(t, out.String())
}

func TestClientErrorf_WritesToErr(t *testing.T) {
	var out, err bytes.Buffer
	NewClient(&out, &err, false).Errorf("test %s", "error")
	assert.Contains(t, err.String(), "test error")
	assert.Empty(t, out.String())
}
