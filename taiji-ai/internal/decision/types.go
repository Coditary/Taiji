package decision

// QuestionType is a structured decision question kind.
type QuestionType string

const (
	QuestionChoice QuestionType = "choice"
	QuestionNoul   QuestionType = "noul"
	QuestionScore  QuestionType = "score"
)

// Question describes one decision to produce.
type Question struct {
	ID           string
	Type         QuestionType
	Instructions string
	Criteria     []string
}

// Request is backend-neutral input for a decision run.
type Request struct {
	Text      string
	Model     string
	Questions []Question
}

// ChoiceAnswer is the result of a choice question.
type ChoiceAnswer struct {
	Choice        string
	Probabilities map[string]float64
	Confidence    float64
}

// NoulAnswer is the result of a yes/no-likelihood question (0–1).
type NoulAnswer struct {
	Value float64
}

// ScoreAnswer is the result of a scored question.
type ScoreAnswer struct {
	Score         float64
	Probabilities map[string]float64
}

// Answer holds one resolved question result.
type Answer struct {
	ID     string
	Type   QuestionType
	Choice *ChoiceAnswer
	Noul   *NoulAnswer
	Score  *ScoreAnswer
}

// Response is the normalized decision output.
type Response struct {
	Model   string
	Driver  string
	Backend string
	Answers map[string]Answer
}
