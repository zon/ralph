package notify

import (
	"testing"
)

func TestRealNotifierImplementsNotifier(t *testing.T) {
	var _ Notifier = (*realNotifier)(nil)
}
