package run

import "errors"

var ErrBlocked = errors.New("Ralph stopped because of blocked.md.")

// blockedError carries the message Ralph prints as its final output before it
// closes. It matches ErrBlocked so callers can still test for a blocked stop.
type blockedError struct {
	message string
}

func (e *blockedError) Error() string {
	return e.message
}

func (e *blockedError) Is(target error) bool {
	return target == ErrBlocked
}
