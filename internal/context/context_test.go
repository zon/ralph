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

func TestAddNote(t *testing.T) {
	ctx := &Context{}

	// Initially should have no notes
	if ctx.HasNotes() {
		t.Error("New context should not have notes")
	}

	// Add first note
	ctx.AddNote("First note")
	if !ctx.HasNotes() {
		t.Error("Context should have notes after adding one")
	}
	if len(ctx.Notes) != 1 {
		t.Errorf("Expected 1 note, got %d", len(ctx.Notes))
	}
	if ctx.Notes[0] != "First note" {
		t.Errorf("Expected note 'First note', got '%s'", ctx.Notes[0])
	}

	// Add second note
	ctx.AddNote("Second note")
	if len(ctx.Notes) != 2 {
		t.Errorf("Expected 2 notes, got %d", len(ctx.Notes))
	}
	if ctx.Notes[1] != "Second note" {
		t.Errorf("Expected note 'Second note', got '%s'", ctx.Notes[1])
	}
}

func TestHasNotes(t *testing.T) {
	tests := []struct {
		name      string
		notes     []string
		expectHas bool
	}{
		{
			name:      "no notes",
			notes:     nil,
			expectHas: false,
		},
		{
			name:      "empty slice",
			notes:     []string{},
			expectHas: false,
		},
		{
			name:      "one note",
			notes:     []string{"note"},
			expectHas: true,
		},
		{
			name:      "multiple notes",
			notes:     []string{"note1", "note2"},
			expectHas: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{
				Notes: tt.notes,
			}

			result := ctx.HasNotes()
			if result != tt.expectHas {
				t.Errorf("expected HasNotes()=%v, got %v", tt.expectHas, result)
			}
		})
	}
}
