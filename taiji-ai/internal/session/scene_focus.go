package session

import (
	"regexp"
	"strings"
)

var sceneMovementRE = regexp.MustCompile(`(?i)(ich\s+)?(gehe|ging|laufe|renne|streife|wandere|verlasse|verließ|betrete|komme\s+an|reise\s+(ab|nach)|head\s+to|go\s+to|leave|enter|walk\s+into|travel\s+to)`)

// GoalImpliesSceneChange reports whether the user message likely moves to a new place or scene.
func GoalImpliesSceneChange(goal string) bool {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return false
	}
	return sceneMovementRE.MatchString(goal)
}

// BuildTurnSceneReminder returns a short instruction placed after chat history (high recency for local models).
func BuildTurnSceneReminder(projectRoot, goal string, settings Settings) string {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return ""
	}

	mentionID, _ := ResolveActorMention(projectRoot, goal)
	presentation := strings.ToLower(strings.TrimSpace(settings.Presentation))
	preset := strings.ToLower(strings.TrimSpace(settings.Preset))
	templateID := strings.ToLower(strings.TrimSpace(settings.TemplateID))
	narrative := presentation == "narrative" || presentation == "story" || preset == "story" || templateID == "story"

	stateLine := FormatActorState(settings.ActorState)
	var parts []string

	switch mentionID {
	case "story":
		parts = append(parts,
			"Der Nutzer hat @story gesetzt: Antworte als neutraler Spielleiter/Erzähler (Game Master), nicht als ein einzelner NPC.",
			"Die letzten Assistenten-Nachrichten können noch einen NPC zeigen — ignoriere dieses Muster für diesen Turn.",
			"Beschreibe die Welt aus der Ich-Perspektive des Erzählers oder in der dritten Person; NPCs nur, wenn der Spieler bei ihnen ist.",
		)
	case "":
		if narrative && GoalImpliesSceneChange(goal) {
			parts = append(parts,
				"Der Spieler hat den Ort gewechselt oder eine neue Szene angedeutet.",
				"NPCs aus der vorherigen Szene (z. B. Schmied/Werkstatt) dürfen nicht mehr sprechen oder handeln, es sei denn, der Spieler ist noch bei ihnen.",
				"Antworte als Erzähler: beschreibe die neue Umgebung und treibe die Handlung am neuen Ort voran.",
			)
		} else if narrative {
			parts = append(parts,
				"Story-/Erzählmodus: Du bist Spielleiter, nicht dauerhaft ein einzelner NPC. Halte Orte und anwesende Figuren konsistent.",
			)
		}
	default:
		parts = append(parts,
			"Der Nutzer hat @"+mentionID+" adressiert: Spiele diesen Charakter nur, wenn der Spieler sich am selben Ort befindet und mit ihm interagiert.",
			"Wenn der Spieler woanders ist, antworte als Erzähler über die Szene — nicht als dieser NPC aus der Ferne.",
		)
	}

	if stateLine != "" {
		parts = append(parts, "Aktueller Actor-State (verbindlich für diesen Turn): "+stateLine)
	}

	if len(parts) == 0 {
		return ""
	}
	return "[Turn focus]\n" + strings.Join(parts, "\n")
}
