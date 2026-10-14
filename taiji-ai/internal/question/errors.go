package question

import "errors"

var (
	// ErrRejected is returned when the user dismisses a question.
	ErrRejected = errors.New("user dismissed the question")
	// ErrNotFound is returned when replying to an unknown request id.
	ErrNotFound = errors.New("question request not found")
	// ErrNoPrompter is returned when no prompter is configured and no reply arrives.
	ErrNoPrompter = errors.New("no question prompter configured")
	// ErrInvalidQuestions is returned when the tool args are malformed.
	ErrInvalidQuestions = errors.New("at least one question is required")
)
