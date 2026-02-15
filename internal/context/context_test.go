package context

import "testing"

func TestShouldNotify(t *testing.T) {
	tests := []struct {
		name         string
		noNotify     bool
		remote       bool
		watch        bool
		expectNotify bool
		description  string
	}{
		{
			name:         "default settings should notify",
			noNotify:     false,
			remote:       false,
			watch:        false,
			expectNotify: true,
			description:  "normal mode with notifications enabled",
		},
		{
			name:         "no-notify flag disables notifications",
			noNotify:     true,
			remote:       false,
			watch:        false,
			expectNotify: false,
			description:  "user explicitly disabled notifications",
		},
		{
			name:         "remote mode without watch disables notifications",
			noNotify:     false,
			remote:       true,
			watch:        false,
			expectNotify: false,
			description:  "remote execution without watching should not notify",
		},
		{
			name:         "remote mode with watch enables notifications",
			noNotify:     false,
			remote:       true,
			watch:        true,
			expectNotify: true,
			description:  "remote execution with watch should notify",
		},
		{
			name:         "remote with watch but no-notify flag disables notifications",
			noNotify:     true,
			remote:       true,
			watch:        true,
			expectNotify: false,
			description:  "explicit no-notify flag overrides watch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{
				NoNotify: tt.noNotify,
				Remote:   tt.remote,
				Watch:    tt.watch,
			}

			result := ctx.ShouldNotify()
			if result != tt.expectNotify {
				t.Errorf("%s: expected ShouldNotify()=%v, got %v",
					tt.description, tt.expectNotify, result)
			}
		})
	}
}

func TestIsRemote(t *testing.T) {
	tests := []struct {
		name   string
		remote bool
	}{
		{
			name:   "remote mode enabled",
			remote: true,
		},
		{
			name:   "remote mode disabled",
			remote: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{
				Remote: tt.remote,
			}

			result := ctx.IsRemote()
			if result != tt.remote {
				t.Errorf("expected IsRemote()=%v, got %v", tt.remote, result)
			}
		})
	}
}

func TestShouldWatch(t *testing.T) {
	tests := []struct {
		name  string
		watch bool
	}{
		{
			name:  "watch mode enabled",
			watch: true,
		},
		{
			name:  "watch mode disabled",
			watch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{
				Watch: tt.watch,
			}

			result := ctx.ShouldWatch()
			if result != tt.watch {
				t.Errorf("expected ShouldWatch()=%v, got %v", tt.watch, result)
			}
		})
	}
}
