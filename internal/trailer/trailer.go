// Package trailer formats and parses ralph completion trailers in commit
// messages. Trailer lines are pure strings with no filesystem, git, or network
// access, so they are unit tested directly.
package trailer

import "fmt"

// Format renders the completion trailer line for an item index and its
// optional key: "Ralph item 2 completed" or "Ralph item 2 (export-endpoint)
// completed".
func Format(index int, key string) string {
	if key == "" {
		return fmt.Sprintf("Ralph item %d completed", index)
	}
	return fmt.Sprintf("Ralph item %d (%s) completed", index, key)
}
