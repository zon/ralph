package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name        string
	debugging   bool
	msg         string
	format      string
	args        []any
	outContains string
	errContains string
}

func checkOutput(t *testing.T, gotOut, gotErr, wantOut, wantErr string) {
	t.Helper()
	if wantOut != "" {
		assert.Contains(t, gotOut, wantOut)
	} else {
		assert.Empty(t, gotOut)
	}
	if wantErr != "" {
		assert.Contains(t, gotErr, wantErr)
	} else {
		assert.Empty(t, gotErr)
	}
}

func TestClientDebug(t *testing.T) {
	tests := []testCase{
		{
			name:        "suppressed when debugging false",
			debugging:   false,
			msg:         "test debug",
			outContains: "",
			errContains: "",
		},
		{
			name:        "writes to out when debugging true",
			debugging:   true,
			msg:         "test debug",
			outContains: "test debug",
			errContains: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, err bytes.Buffer
			NewClient(&out, &err, tt.debugging).Debug(tt.msg)
			checkOutput(t, out.String(), err.String(), tt.outContains, tt.errContains)
		})
	}
}

func TestClientDebugf(t *testing.T) {
	tests := []testCase{
		{
			name:        "suppressed when debugging false",
			debugging:   false,
			format:      "test %s",
			args:        []any{"debug"},
			outContains: "",
			errContains: "",
		},
		{
			name:        "writes to out when debugging true",
			debugging:   true,
			format:      "test %s",
			args:        []any{"debug"},
			outContains: "test debug",
			errContains: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, err bytes.Buffer
			NewClient(&out, &err, tt.debugging).Debugf(tt.format, tt.args...)
			checkOutput(t, out.String(), err.String(), tt.outContains, tt.errContains)
		})
	}
}

func TestClientInfo(t *testing.T) {
	tests := []testCase{
		{
			name:        "always writes to out",
			debugging:   false,
			msg:         "test info",
			outContains: "test info",
			errContains: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, err bytes.Buffer
			NewClient(&out, &err, tt.debugging).Info(tt.msg)
			checkOutput(t, out.String(), err.String(), tt.outContains, tt.errContains)
		})
	}
}

func TestClientInfof(t *testing.T) {
	tests := []testCase{
		{
			name:        "always writes to out",
			debugging:   false,
			format:      "test %s",
			args:        []any{"info"},
			outContains: "test info",
			errContains: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, err bytes.Buffer
			NewClient(&out, &err, tt.debugging).Infof(tt.format, tt.args...)
			checkOutput(t, out.String(), err.String(), tt.outContains, tt.errContains)
		})
	}
}

func TestClientWarn(t *testing.T) {
	tests := []testCase{
		{
			name:        "always writes to out",
			debugging:   false,
			msg:         "test warning",
			outContains: "test warning",
			errContains: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, err bytes.Buffer
			NewClient(&out, &err, tt.debugging).Warn(tt.msg)
			checkOutput(t, out.String(), err.String(), tt.outContains, tt.errContains)
		})
	}
}

func TestClientWarnf(t *testing.T) {
	tests := []testCase{
		{
			name:        "always writes to out",
			debugging:   false,
			format:      "test %s",
			args:        []any{"warning"},
			outContains: "test warning",
			errContains: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, err bytes.Buffer
			NewClient(&out, &err, tt.debugging).Warnf(tt.format, tt.args...)
			checkOutput(t, out.String(), err.String(), tt.outContains, tt.errContains)
		})
	}
}

func TestClientError(t *testing.T) {
	tests := []testCase{
		{
			name:        "writes to err with nothing to out",
			debugging:   false,
			msg:         "test error",
			outContains: "",
			errContains: "test error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, err bytes.Buffer
			NewClient(&out, &err, tt.debugging).Error(tt.msg)
			checkOutput(t, out.String(), err.String(), tt.outContains, tt.errContains)
		})
	}
}

func TestClientErrorf(t *testing.T) {
	tests := []testCase{
		{
			name:        "writes to err with nothing to out",
			debugging:   false,
			format:      "test %s",
			args:        []any{"error"},
			outContains: "",
			errContains: "test error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, err bytes.Buffer
			NewClient(&out, &err, tt.debugging).Errorf(tt.format, tt.args...)
			checkOutput(t, out.String(), err.String(), tt.outContains, tt.errContains)
		})
	}
}

func TestClientSuccess(t *testing.T) {
	tests := []testCase{
		{
			name:        "writes checkmark and message to out",
			debugging:   false,
			msg:         "test success",
			outContains: "✓ test success",
			errContains: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, err bytes.Buffer
			NewClient(&out, &err, tt.debugging).Success(tt.msg)
			checkOutput(t, out.String(), err.String(), tt.outContains, tt.errContains)
		})
	}
}

func TestClientSuccessf(t *testing.T) {
	tests := []testCase{
		{
			name:        "writes checkmark and formatted message to out",
			debugging:   false,
			format:      "test %s",
			args:        []any{"success"},
			outContains: "✓ test success",
			errContains: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, err bytes.Buffer
			NewClient(&out, &err, tt.debugging).Successf(tt.format, tt.args...)
			checkOutput(t, out.String(), err.String(), tt.outContains, tt.errContains)
		})
	}
}
