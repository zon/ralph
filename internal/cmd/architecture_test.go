package cmd

import (
	"testing"

	"github.com/alecthomas/kong"
)

func TestArchitectureCmdFlagParsing(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedOut   string
		expectedVerb  bool
		expectedModel string
		wantParseErr  bool
	}{
		{
			name:          "review architecture default values",
			args:          []string{"review", "architecture"},
			expectedOut:   "architecture.yaml",
			expectedVerb:  false,
			expectedModel: "",
		},
		{
			name:          "review architecture custom output path",
			args:          []string{"review", "architecture", "--output", "custom.yaml"},
			expectedOut:   "custom.yaml",
			expectedVerb:  false,
			expectedModel: "",
		},
		{
			name:          "review architecture verbose flag",
			args:          []string{"review", "architecture", "--verbose"},
			expectedOut:   "architecture.yaml",
			expectedVerb:  true,
			expectedModel: "",
		},
		{
			name:          "review architecture model override",
			args:          []string{"review", "architecture", "--model", "deepseek/deepseek-chat"},
			expectedOut:   "architecture.yaml",
			expectedVerb:  false,
			expectedModel: "deepseek/deepseek-chat",
		},
		{
			name:          "review architecture all flags",
			args:          []string{"review", "architecture", "--output", "out.yaml", "--verbose", "--model", "gpt-4"},
			expectedOut:   "out.yaml",
			expectedVerb:  true,
			expectedModel: "gpt-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Cmd{}
			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}),
			)
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}

			_, err = parser.Parse(tt.args)
			if tt.wantParseErr {
				if err == nil {
					t.Error("expected parse error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("failed to parse args: %v", err)
			}

			if cmd.Review.Architecture.Output != tt.expectedOut {
				t.Errorf("expected Output=%q, got %q", tt.expectedOut, cmd.Review.Architecture.Output)
			}
			if cmd.Review.Architecture.Verbose != tt.expectedVerb {
				t.Errorf("expected Verbose=%v, got %v", tt.expectedVerb, cmd.Review.Architecture.Verbose)
			}
			if cmd.Review.Architecture.Model != tt.expectedModel {
				t.Errorf("expected Model=%q, got %q", tt.expectedModel, cmd.Review.Architecture.Model)
			}
		})
	}
}
