package context

import "testing"

func TestShouldNotify(t *testing.T) {
	tests := []struct {
		name         string
		noNotify     bool
		local        bool
		watch        bool
		expectNotify bool
		description  string
	}{
		{
			name:         "default settings should not notify (remote workflow without watch)",
			noNotify:     false,
			local:        false,
			watch:        false,
			expectNotify: false,
			description:  "remote workflow without watch should not notify",
		},
		{
			name:         "local mode notifies by default",
			noNotify:     false,
			local:        true,
			watch:        false,
			expectNotify: true,
			description:  "local mode with notifications enabled",
		},
		{
			name:         "no-notify flag disables notifications",
			noNotify:     true,
			local:        true,
			watch:        false,
			expectNotify: false,
			description:  "user explicitly disabled notifications",
		},
		{
			name:         "remote workflow with watch enables notifications",
			noNotify:     false,
			local:        false,
			watch:        true,
			expectNotify: true,
			description:  "remote workflow with watch should notify",
		},
		{
			name:         "remote with watch but no-notify flag disables notifications",
			noNotify:     true,
			local:        false,
			watch:        true,
			expectNotify: false,
			description:  "explicit no-notify flag overrides watch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{
				NoNotify: tt.noNotify,
				Local:    tt.local,
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

func TestIsLocal(t *testing.T) {
	tests := []struct {
		name  string
		local bool
	}{
		{
			name:  "local mode enabled",
			local: true,
		},
		{
			name:  "local mode disabled",
			local: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{
				Local: tt.local,
			}

			result := ctx.IsLocal()
			if result != tt.local {
				t.Errorf("expected IsLocal()=%v, got %v", tt.local, result)
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
