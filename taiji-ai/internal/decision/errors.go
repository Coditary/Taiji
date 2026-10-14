package decision

import "errors"

var (
	// ErrDisabled is returned when the decision layer is not configured.
	ErrDisabled = errors.New("decisions are disabled")
)
