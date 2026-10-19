package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItemPickRefIsTheCompletionHash(t *testing.T) {
	item := NewItems([]any{"plain item"})[0]

	assert.Equal(t, item.Hash(), item.PickRef())
	assert.Equal(t, "ref: "+item.Hash(), item.PickRefLine())
}

func TestDecodePickRef(t *testing.T) {
	items := NewItems([]any{
		"plain item",
		map[string]any{"slug": "exporter", "description": "export endpoint"},
		`tables = ["AuditLogs"]`,
	})
	escaped := `tables = [\"AuditLogs\"]`

	tests := []struct {
		name   string
		report string
		want   int
		ok     bool
	}{
		{name: "reference line", report: items[1].PickRefLine(), want: 1, ok: true},
		{name: "bare reference", report: items[1].PickRef(), want: 1, ok: true},
		{name: "completion trailer", report: "csv-export-" + items[1].PickRef(), want: 1, ok: true},
		{name: "JSON reference", report: `{"ref":"` + items[1].PickRef() + `"}`, want: 1, ok: true},
		{name: "reference in prose", report: "Selected ref: " + items[1].PickRef(), want: 1, ok: true},
		{name: "fenced reference", report: "```\n" + items[1].PickRefLine() + "\n```", want: 1, ok: true},
		{name: "item text", report: items[0].PickText(), want: 0, ok: true},
		{name: "canonical item text", report: items[1].Text(), want: 1, ok: true},
		{name: "text with YAML escaping", report: escaped, want: 2, ok: true},
		{name: "quoted text with YAML escaping", report: `"` + escaped + `"`, want: 2, ok: true},
		{name: "unrelated text", report: "not a listed item", ok: false},
		{name: "empty report", report: "", ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			item, ok := DecodePickRef(items, tc.report)
			require.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.want, item.Index)
			}
		})
	}
}

func TestDecodePickRefOnlyMatchesTheGivenItems(t *testing.T) {
	all := NewItems([]any{"one", "two"})

	_, ok := DecodePickRef([]Item{all[1]}, all[0].PickRef())
	assert.False(t, ok, "a reference to an item outside the incomplete set is not usable")
}
