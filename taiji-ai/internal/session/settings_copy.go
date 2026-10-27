package session

// CopySettings returns a deep copy of chat settings (for council/template restore).
func CopySettings(src Settings) Settings {
	out := src
	if src.Temperature != nil {
		v := *src.Temperature
		out.Temperature = &v
	}
	if src.MaxTokens != nil {
		v := *src.MaxTokens
		out.MaxTokens = &v
	}
	if src.MaxSteps != nil {
		v := *src.MaxSteps
		out.MaxSteps = &v
	}
	if src.ThoughtMaxChars != nil {
		v := *src.ThoughtMaxChars
		out.ThoughtMaxChars = &v
	}
	if src.DecisionsEnabled != nil {
		v := *src.DecisionsEnabled
		out.DecisionsEnabled = &v
	}
	if src.MemoryEnabled != nil {
		v := *src.MemoryEnabled
		out.MemoryEnabled = &v
	}
	if src.ContextWindow != nil {
		v := *src.ContextWindow
		out.ContextWindow = &v
	}
	out.SkillsAllow = append([]string(nil), src.SkillsAllow...)
	out.SkillsDeny = append([]string(nil), src.SkillsDeny...)
	out.PacksAllow = append([]string(nil), src.PacksAllow...)
	out.DefaultPacks = append([]string(nil), src.DefaultPacks...)
	out.MCPServers = append([]string(nil), src.MCPServers...)
	out.MCPToolsDeny = append([]string(nil), src.MCPToolsDeny...)
	out.BuiltinToolsAllow = append([]string(nil), src.BuiltinToolsAllow...)
	out.BuiltinToolsDeny = append([]string(nil), src.BuiltinToolsDeny...)
	out.FewShotExamples = append([]ExampleTurn(nil), src.FewShotExamples...)
	if len(src.ActorState) > 0 {
		out.ActorState = map[string]string{}
		for k, v := range src.ActorState {
			out.ActorState[k] = v
		}
	} else {
		out.ActorState = nil
	}
	out.CouncilActors = append([]string(nil), src.CouncilActors...)
	out.FollowUpActor = src.FollowUpActor
	out.ChatTools = append([]ChatTool(nil), src.ChatTools...)
	return out
}
