package memory

import "errors"

var (
	errKindRequired   = errors.New("kind is required")
	errUnknownKind    = errors.New("unknown memory kind")
	errTitleRequired  = errors.New("title is required")
	errQueryRequired  = errors.New("query is required")
	errIDRequired     = errors.New("id is required")
	errEntryNotFound  = errors.New("memory entry not found")
	errRecallRequired = errors.New("recall_text is required")
	errFixRequired    = errors.New("verified_fix is required")
	errNoteRequired   = errors.New("revision_note is required")
	errDisabled       = errors.New("memory is disabled")
)
