package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// DecisionQuestion is the Wuji ProduceDecision question shape.
type DecisionQuestion struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Instructions string   `json:"instructions,omitempty"`
	Criteria     []string `json:"criteria,omitempty"`
}

// DecisionRequest is the Wuji ProduceDecision request body.
type DecisionRequest struct {
	Text      string             `json:"text"`
	Model     string             `json:"model,omitempty"`
	Questions []DecisionQuestion `json:"questions"`
}

// ProduceDecision runs structured decisions through Wuji core.
func (c *WujiCoreClient) ProduceDecision(ctx context.Context, driverID string, req DecisionRequest) (json.RawMessage, error) {
	driverID = strings.TrimSpace(driverID)
	out, err := c.call(ctx, "ProduceDecision", map[string]any{
		"driver_id": driverID,
		"request":   req,
	})
	if err != nil {
		return nil, err
	}
	var doc struct {
		Answers json.RawMessage `json:"answers"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, err
	}
	if len(doc.Answers) == 0 {
		return nil, fmt.Errorf("empty decision response")
	}
	return out, nil
}
