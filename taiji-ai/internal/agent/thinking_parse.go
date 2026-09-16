package agent

import (
	"strings"

	"github.com/coditary/taiji/internal/provider"
)

type streamSection int

const (
	sectionBeforeThinking streamSection = 0
	sectionInsideThinking streamSection = 1
	sectionAfterThinking  streamSection = 2
)

// thinkingStreamParser routes streamed model output into thinking vs answer channels.
type thinkingStreamParser struct {
	section        streamSection
	buf            strings.Builder
	thoughtNum     int
	thoughtOffset  int
	onePerCall     bool
	thoughtDone    bool
	onThoughtStart func(int) error
	onThink        func(string) error
	onAnswer       func(string) error
}

func newThinkingStreamParser(
	thoughtOffset int,
	onePerCall bool,
	onThoughtStart func(int) error,
	onThink func(string) error,
	onAnswer func(string) error,
) *thinkingStreamParser {
	return &thinkingStreamParser{
		thoughtOffset:  thoughtOffset,
		onePerCall:       onePerCall,
		onThoughtStart: onThoughtStart,
		onThink:        onThink,
		onAnswer:       onAnswer,
	}
}

func (p *thinkingStreamParser) Write(delta string) error {
	if delta == "" || p.thoughtDone {
		return nil
	}
	p.buf.WriteString(delta)
	return p.drain(false)
}

func (p *thinkingStreamParser) Flush() error {
	return p.drain(true)
}

func (p *thinkingStreamParser) drain(flush bool) error {
	for {
		if p.buf.Len() > 0 {
			normalized := sanitizeChainStream(p.buf.String())
			p.buf.Reset()
			p.buf.WriteString(normalized)
		}
		s := p.buf.String()
		if s == "" {
			return nil
		}

		switch p.section {
		case sectionBeforeThinking:
			idx := strings.Index(s, thinkingOpenTag)
			if idx < 0 {
				if flush {
					if err := p.emitAnswerIfReal(s); err != nil {
						return err
					}
					p.buf.Reset()
					return nil
				}
				hold := partialTagSuffix(s, thinkingOpenTag)
				if hold > 0 {
					if emit := s[:len(s)-hold]; emit != "" {
						if err := p.emitAnswerIfReal(emit); err != nil {
							return err
						}
					}
					rest := s[len(s)-hold:]
					p.buf.Reset()
					p.buf.WriteString(rest)
					return nil
				}
				if err := p.emitAnswerIfReal(s); err != nil {
					return err
				}
				p.buf.Reset()
				return nil
			}
			if idx > 0 {
				if err := p.emitAnswerIfReal(s[:idx]); err != nil {
					return err
				}
			}
			s = s[idx+len(thinkingOpenTag):]
			p.section = sectionInsideThinking
			if err := p.startThinking(); err != nil {
				return err
			}
			p.buf.Reset()
			p.buf.WriteString(s)

		case sectionInsideThinking:
			idx := strings.Index(s, thinkingCloseTag)
			if idx < 0 {
				if flush {
					if err := p.emitThink(s); err != nil {
						return err
					}
					p.buf.Reset()
					return nil
				}
				hold := partialTagSuffix(s, thinkingCloseTag)
				if hold > 0 {
					if emit := s[:len(s)-hold]; emit != "" {
						if err := p.emitThink(emit); err != nil {
							return err
						}
					}
					rest := s[len(s)-hold:]
					p.buf.Reset()
					p.buf.WriteString(rest)
					return nil
				}
				if len(s) > 0 {
					if err := p.emitThink(s); err != nil {
						return err
					}
					p.buf.Reset()
				}
				return nil
			}
			if idx > 0 {
				if err := p.emitThink(s[:idx]); err != nil {
					return err
				}
			}
			s = s[idx+len(thinkingCloseTag):]
			p.section = sectionAfterThinking
			p.buf.Reset()
			p.buf.WriteString(s)

		case sectionAfterThinking:
			if p.onePerCall {
				s = sanitizeChainStream(s)
				if strings.Contains(s, thinkingOpenTag) {
					// Extra think blocks in the same message — ignore; auto-continue handles the next step.
					p.thoughtDone = true
					p.buf.Reset()
					return nil
				}
				cleaned := stripContinueMarkers(s)
				if strings.Contains(cleaned, "<|continue") || strings.TrimSpace(cleaned) == "" {
					p.thoughtDone = true
					p.buf.Reset()
					return nil
				}
				if flush || cleaned != "" {
					if err := p.emitAnswerIfReal(cleaned); err != nil {
						return err
					}
				}
				p.thoughtDone = true
				p.buf.Reset()
				return nil
			}
			s = sanitizeChainStream(s)
			next := strings.Index(s, thinkingOpenTag)
			if next >= 0 {
				if next > 0 {
					if err := p.emitAnswerIfReal(s[:next]); err != nil {
						return err
					}
				}
				s = s[next+len(thinkingOpenTag):]
				p.section = sectionInsideThinking
				if err := p.startThinking(); err != nil {
					return err
				}
				p.buf.Reset()
				p.buf.WriteString(s)
				continue
			}
			if flush || s != "" {
				if err := p.emitAnswerIfReal(s); err != nil {
					return err
				}
			}
			p.buf.Reset()
			return nil
		}
	}
}

func (p *thinkingStreamParser) startThinking() error {
	p.thoughtNum++
	if p.onThoughtStart != nil {
		return p.onThoughtStart(p.thoughtNum + p.thoughtOffset)
	}
	return nil
}

func (p *thinkingStreamParser) emitThink(s string) error {
	if s == "" || p.onThink == nil {
		return nil
	}
	return p.onThink(s)
}

func (p *thinkingStreamParser) emitAnswerIfReal(s string) error {
	if strings.TrimSpace(s) == "" || p.onAnswer == nil {
		return nil
	}
	cleaned := stripContinueMarkers(s)
	if strings.TrimSpace(cleaned) == "" || isChainNoise(cleaned) {
		return nil
	}
	if p.section == sectionAfterThinking {
		cleaned = strings.TrimSpace(cleaned)
	}
	return p.onAnswer(cleaned)
}

func partialTagSuffix(s, tag string) int {
	if strings.HasPrefix(tag, "<") {
		lt := strings.LastIndex(s, "<")
		if lt >= 0 {
			suffix := s[lt:]
			if suffix != tag && strings.HasPrefix(tag, suffix) {
				return len(suffix)
			}
		}
	}
	max := len(tag) - 1
	if max > len(s) {
		max = len(s)
	}
	for n := max; n > 0; n-- {
		suffix := s[len(s)-n:]
		if strings.HasPrefix(tag, suffix) {
			return n
		}
	}
	return 0
}

func parseAllThinkingBlocks(text string) []string {
	text = sanitizeChainStream(strings.TrimSpace(text))
	var blocks []string
	for {
		start := strings.Index(text, thinkingOpenTag)
		if start < 0 {
			break
		}
		text = text[start+len(thinkingOpenTag):]
		end := strings.Index(text, thinkingCloseTag)
		if end < 0 {
			if rest := strings.TrimSpace(text); rest != "" {
				blocks = append(blocks, rest)
			}
			break
		}
		if block := strings.TrimSpace(text[:end]); block != "" {
			blocks = append(blocks, block)
		}
		text = text[end+len(thinkingCloseTag):]
	}
	return blocks
}

func stripThinkingBlocks(text string) string {
	text = sanitizeChainStream(text)
	for {
		start := strings.Index(text, thinkingOpenTag)
		if start < 0 {
			break
		}
		end := strings.Index(text[start:], thinkingCloseTag)
		if end < 0 {
			text = strings.TrimSpace(text[:start])
			break
		}
		end = start + end + len(thinkingCloseTag)
		text = text[:start] + text[end:]
	}
	return strings.TrimSpace(text)
}

func extractAnswerFromChained(text string) string {
	text = sanitizeChainStream(strings.TrimSpace(text))
	lastClose := strings.LastIndex(text, thinkingCloseTag)
	if lastClose < 0 {
		return strings.TrimSpace(stripContinueMarkers(text))
	}
	answer := strings.TrimSpace(text[lastClose+len(thinkingCloseTag):])
	return strings.TrimSpace(stripContinueMarkers(answer))
}

// truncateToFirstChainStep keeps only the first completed thought and an optional continue marker.
func truncateToFirstChainStep(text, trigger string) string {
	text = sanitizeChainStream(strings.TrimSpace(text))
	start := strings.Index(text, thinkingOpenTag)
	if start < 0 {
		return text
	}
	rest := text[start+len(thinkingOpenTag):]
	end := strings.Index(rest, thinkingCloseTag)
	if end < 0 {
		return text
	}
	out := text[:start] + thinkingOpenTag + rest[:end] + thinkingCloseTag
	if containsContinueTrigger(text, trigger) {
		out += "\n" + trigger
	}
	return out
}

func parseThinkingTags(text string) (thinking, answer string) {
	text = sanitizeChainStream(strings.TrimSpace(text))
	if text == "" {
		return "", ""
	}

	blocks := parseAllThinkingBlocks(text)
	if len(blocks) == 0 {
		return "", text
	}

	answer = extractAnswerFromChained(text)
	start := strings.Index(text, thinkingOpenTag)
	if start > 0 {
		prefix := strings.TrimSpace(text[:start])
		if prefix != "" && !isChainNoise(prefix) {
			if answer != "" {
				answer = prefix + "\n\n" + answer
			} else {
				answer = prefix
			}
		}
	}
	return strings.Join(blocks, "\n\n"), answer
}

func extractAnswerText(resp *provider.CompletionResponse, streamed string, useTags bool) string {
	content := strings.TrimSpace(resp.Content)
	if streamed != "" && content == "" {
		content = strings.TrimSpace(streamed)
	}

	if useTags {
		_, answer := parseThinkingTags(content)
		if answer != "" && !isChainNoise(answer) {
			return answer
		}
		blocks := parseAllThinkingBlocks(content)
		if len(blocks) > 0 {
			return strings.Join(blocks, "\n\n")
		}
	}

	if think := strings.TrimSpace(provider.JoinThinkingBlocks(resp.ThinkingBlocks)); think != "" && content != "" {
		return content
	}
	if content != "" {
		return content
	}
	return strings.TrimSpace(provider.JoinThinkingBlocks(resp.ThinkingBlocks))
}
