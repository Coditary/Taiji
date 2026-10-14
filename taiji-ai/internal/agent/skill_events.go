package agent

import (
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/skills"
	"github.com/coditary/taiji/internal/tools"
)

func emitSkillToolActivity(
	emit func(events.Event),
	step int,
	tc provider.ToolCall,
	outcome toolExecOutcome,
	mcpServer string,
	isMCP bool,
) []string {
	out := outcome.text
	var result tools.RunResult
	if outcome.duplicate {
		emitSingleSkillActivity(emit, step, tc, out, mcpServer, isMCP, outcome.durationMs, "", nil)
		if !strings.HasPrefix(strings.TrimSpace(out), "error:") {
			return []string{out}
		}
		return nil
	}

	result = outcome.result
	out = result.Text
	if outcome.err != nil {
		out = "error: " + outcome.err.Error()
	}
	if strings.HasPrefix(strings.TrimSpace(out), "error:") {
		emitSingleSkillActivity(emit, step, tc, out, mcpServer, isMCP, outcome.durationMs, "", result.Data)
		return nil
	}

	blocks := skills.BlocksFromInvokedOutput(out)
	if len(blocks) <= 1 {
		skillName := skills.SkillNameFromArgsJSON(tc.Args)
		if skillName == "" && len(blocks) == 1 {
			skillName = blocks[0].Name
		}
		emitSingleSkillActivity(emit, step, tc, out, mcpServer, isMCP, outcome.durationMs, skillName, result.Data)
		return []string{out}
	}

	perSkillData := func(name string) map[string]any {
		if result.Data == nil {
			return map[string]any{"skill": name}
		}
		data := map[string]any{}
		for k, v := range result.Data {
			data[k] = v
		}
		data["skill"] = name
		return data
	}

	observations := make([]string, 0, len(blocks))
	for i, block := range blocks {
		args := fmt.Sprintf(`{"name":%q}`, block.Name)
		toolID := tc.ID
		if len(blocks) > 1 {
			toolID = fmt.Sprintf("%s_skill_%d", tc.ID, i+1)
		}
		emit(events.Event{
			Type:      events.TypeToolCall,
			Step:      step,
			ToolName:  tc.Name,
			ToolID:    toolID,
			Args:      args,
			IsMCP:     isMCP,
			MCPServer: mcpServer,
			IsSkill:   true,
			SkillName: block.Name,
		})
		emit(events.Event{
			Type:       events.TypeObservation,
			Step:       step,
			ToolName:   tc.Name,
			ToolID:     toolID,
			Args:       args,
			Text:       block.Text,
			DurationMs: outcome.durationMs,
			IsMCP:      isMCP,
			MCPServer:  mcpServer,
			IsSkill:    true,
			SkillName:  block.Name,
			Data:       perSkillData(block.Name),
		})
		observations = append(observations, block.Text)
	}
	return observations
}

func emitSingleSkillActivity(
	emit func(events.Event),
	step int,
	tc provider.ToolCall,
	out string,
	mcpServer string,
	isMCP bool,
	durationMs int64,
	skillName string,
	data map[string]any,
) {
	if skillName == "" {
		skillName = skills.SkillNameFromArgsJSON(tc.Args)
	}
	emit(events.Event{
		Type:      events.TypeToolCall,
		Step:      step,
		ToolName:  tc.Name,
		ToolID:    tc.ID,
		Args:      tc.Args,
		IsMCP:     isMCP,
		MCPServer: mcpServer,
		IsSkill:   true,
		SkillName: skillName,
	})
	obs := events.Event{
		Type:       events.TypeObservation,
		Step:       step,
		ToolName:   tc.Name,
		ToolID:     tc.ID,
		Text:       out,
		DurationMs: durationMs,
		IsMCP:      isMCP,
		MCPServer:  mcpServer,
		IsSkill:    true,
		SkillName:  skillName,
	}
	if len(data) > 0 {
		obs.Data = data
		if resolved, ok := data["skill"].(string); ok && strings.TrimSpace(resolved) != "" {
			obs.SkillName = strings.TrimSpace(resolved)
		}
	}
	emit(obs)
}
