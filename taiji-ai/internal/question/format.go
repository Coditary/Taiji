package question

import (
	"fmt"
	"strings"
)

// FormatToolOutput builds the observation text returned to the model.
func FormatToolOutput(questions []Info, answers []Answer) string {
	parts := make([]string, 0, len(questions))
	for i, q := range questions {
		answer := "Unanswered"
		if i < len(answers) && len(answers[i]) > 0 {
			answer = strings.Join(answers[i], ", ")
		}
		parts = append(parts, fmt.Sprintf("%q=%q", q.Question, answer))
	}
	return "User has answered your questions: " + strings.Join(parts, ", ") + ". Continue with these answers in mind."
}

// AnswersData returns structured answer metadata for events.
func AnswersData(questions []Info, answers []Answer) map[string]any {
	items := make([]map[string]any, 0, len(questions))
	for i, q := range questions {
		item := map[string]any{
			"id":       q.ID,
			"question": q.Question,
			"answers":  []string{},
		}
		if i < len(answers) {
			item["answers"] = append([]string(nil), answers[i]...)
		}
		items = append(items, item)
	}
	return map[string]any{"answers": items}
}
