package decisions

import "github.com/coditary/taiji/internal/decision"

// Profile is a saved decision schema loaded from DECISION.md.
type Profile struct {
	Name                   string
	Description            string
	When                   string
	Model                  string
	Driver                 string
	Questions              []decision.Question
	Body                   string
	Path                   string
	DisableModelInvocation bool
}
