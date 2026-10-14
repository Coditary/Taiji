package builtin

func memoryRecallTextArg(args map[string]any) string {
	return stringArgFirst(args, "recall_text", "text", "recall", "content", "query", "memory", "description")
}

func memoryVerifiedFixArg(args map[string]any) string {
	return stringArgFirst(args, "verified_fix", "fix", "value", "answer", "fact")
}

func stringArgFirst(args map[string]any, keys ...string) string {
	for _, key := range keys {
		if v := stringArg(args, key); v != "" {
			return v
		}
	}
	return ""
}
