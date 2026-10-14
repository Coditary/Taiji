package skills

import "strings"

// InvokedBlock is one skill payload from invoke_skill output.
type InvokedBlock struct {
	Name string
	Text string
}

// BlocksFromInvokedOutput splits combined invoke_skill output into per-skill blocks.
func BlocksFromInvokedOutput(text string) []InvokedBlock {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasPrefix(strings.ToLower(text), "error:") {
		return nil
	}
	var blocks []InvokedBlock
	var current *InvokedBlock
	var body []string
	flush := func() {
		if current == nil {
			return
		}
		blockText := invokedHeaderPrefix + current.Name
		if len(body) > 0 {
			blockText += "\n\n" + strings.Join(body, "\n")
		}
		current.Text = strings.TrimSpace(blockText)
		blocks = append(blocks, *current)
		current = nil
		body = nil
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, invokedHeaderPrefix) {
			flush()
			current = &InvokedBlock{
				Name: strings.TrimSpace(strings.TrimPrefix(line, invokedHeaderPrefix)),
			}
			continue
		}
		if current != nil {
			body = append(body, line)
		}
	}
	flush()
	return blocks
}
