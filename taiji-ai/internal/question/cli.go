package question

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// CLIPrompter renders questions on a terminal and reads answers from stdin.
type CLIPrompter struct {
	In  io.Reader
	Out io.Writer
}

// Prompt asks the user each question in order.
func (p CLIPrompter) Prompt(ctx context.Context, req Request) ([]Answer, error) {
	if len(req.Questions) == 0 {
		return nil, ErrInvalidQuestions
	}
	in := p.In
	out := p.Out
	if in == nil {
		in = io.Reader(nil)
	}
	if out == nil {
		out = io.Discard
	}
	reader := bufio.NewReader(in)

	answers := make([]Answer, len(req.Questions))
	for i, q := range req.Questions {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		answer, err := promptOne(ctx, reader, out, q, i+1, len(req.Questions))
		if err != nil {
			return nil, err
		}
		answers[i] = answer
	}

	if len(req.Questions) > 1 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Summary:")
		for i, q := range req.Questions {
			fmt.Fprintf(out, "  %d. %s → %s\n", i+1, q.Header, strings.Join(answers[i], ", "))
		}
		fmt.Fprint(out, "Press Enter to confirm, or type 'back' to restart: ")
		line, err := readLine(ctx, reader)
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(strings.TrimSpace(line), "back") {
			return p.Prompt(ctx, req)
		}
	}
	return answers, nil
}

func promptOne(ctx context.Context, reader *bufio.Reader, out io.Writer, q Info, index, total int) ([]string, error) {
	fmt.Fprintln(out)
	if total > 1 {
		fmt.Fprintf(out, "=== %s (%d/%d) ===\n", q.Header, index, total)
	} else {
		fmt.Fprintf(out, "=== %s ===\n", q.Header)
	}
	fmt.Fprintf(out, "%s\n\n", q.Question)

	options := q.Options
	customIndex := -1
	if q.AllowsCustom() {
		customIndex = len(options)
	}

	for i, opt := range options {
		fmt.Fprintf(out, "  %d. %s\n", i+1, opt.Label)
		if opt.Description != "" {
			fmt.Fprintf(out, "     %s\n", opt.Description)
		}
	}
	if customIndex >= 0 {
		fmt.Fprintf(out, "  %d. Type your own answer\n", customIndex+1)
	}

	if q.Multiple {
		fmt.Fprint(out, "\nSelect one or more (comma-separated numbers")
		if customIndex >= 0 {
			fmt.Fprint(out, ", or enter custom text after choosing the custom option")
		}
		fmt.Fprint(out, "): ")
	} else {
		fmt.Fprint(out, "\nSelect a number")
		if customIndex >= 0 {
			fmt.Fprint(out, " or choose the custom option and enter text")
		}
		fmt.Fprint(out, ": ")
	}

	line, err := readLine(ctx, reader)
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("no answer provided")
	}
	if strings.EqualFold(line, "cancel") || strings.EqualFold(line, "reject") {
		return nil, ErrRejected
	}

	if q.Multiple {
		return parseMultiSelection(line, options, customIndex, reader, ctx, out)
	}
	return parseSingleSelection(line, options, customIndex, reader, ctx, out)
}

func parseSingleSelection(line string, options []Option, customIndex int, reader *bufio.Reader, ctx context.Context, out io.Writer) ([]string, error) {
	n, err := strconv.Atoi(line)
	if err != nil {
		if customIndex >= 0 {
			return []string{line}, nil
		}
		return nil, fmt.Errorf("invalid selection %q", line)
	}
	if n < 1 || n > len(options)+(func() int {
		if customIndex >= 0 {
			return 1
		}
		return 0
	}()) {
		return nil, fmt.Errorf("selection out of range: %d", n)
	}
	if customIndex >= 0 && n == customIndex+1 {
		fmt.Fprint(out, "Enter your answer: ")
		custom, err := readLine(ctx, reader)
		if err != nil {
			return nil, err
		}
		custom = strings.TrimSpace(custom)
		if custom == "" {
			return nil, fmt.Errorf("custom answer is empty")
		}
		return []string{custom}, nil
	}
	return []string{options[n-1].Label}, nil
}

func parseMultiSelection(line string, options []Option, customIndex int, reader *bufio.Reader, ctx context.Context, out io.Writer) ([]string, error) {
	parts := strings.FieldsFunc(line, func(r rune) bool {
		return r == ',' || r == ' '
	})
	selected := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			if _, ok := seen[part]; !ok {
				selected = append(selected, part)
				seen[part] = struct{}{}
			}
			continue
		}
		if customIndex >= 0 && n == customIndex+1 {
			fmt.Fprint(out, "Enter your custom answer: ")
			custom, err := readLine(ctx, reader)
			if err != nil {
				return nil, err
			}
			custom = strings.TrimSpace(custom)
			if custom == "" {
				return nil, fmt.Errorf("custom answer is empty")
			}
			if _, ok := seen[custom]; !ok {
				selected = append(selected, custom)
				seen[custom] = struct{}{}
			}
			continue
		}
		if n < 1 || n > len(options) {
			return nil, fmt.Errorf("selection out of range: %d", n)
		}
		label := options[n-1].Label
		if _, ok := seen[label]; !ok {
			selected = append(selected, label)
			seen[label] = struct{}{}
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("no valid selections")
	}
	return selected, nil
}

func readLine(ctx context.Context, reader *bufio.Reader) (string, error) {
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := reader.ReadString('\n')
		ch <- result{line: line, err: err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-ch:
		return res.line, res.err
	}
}

// RejectPrompter always rejects prompts (non-interactive runs).
type RejectPrompter struct{}

func (RejectPrompter) Prompt(context.Context, Request) ([]Answer, error) {
	return nil, ErrRejected
}
