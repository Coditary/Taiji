package question

import (
	"strings"
	"testing"
)

func TestFormatToolOutputAndAnswersData(t *testing.T) {
	qs := []Info{{ID: "q1", Question: "Color?"}, {ID: "q2", Question: "Size?"}}
	ans := []Answer{{"Red"}, {}}
	out := FormatToolOutput(qs, ans)
	for _, sub := range []string{"Color?", "Red", "Unanswered"} {
		if !strings.Contains(out, sub) {
			t.Fatalf("out=%q missing %q", out, sub)
		}
	}
	data := AnswersData(qs, ans)
	items, _ := data["answers"].([]map[string]any)
	if len(items) != 2 {
		t.Fatalf("data=%v", data)
	}
}
