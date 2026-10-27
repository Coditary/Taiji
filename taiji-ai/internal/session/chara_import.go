package session

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type charaCardV2 struct {
	Data charaCardData `json:"data"`
}

type charaCardData struct {
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Personality    string          `json:"personality"`
	Scenario       string          `json:"scenario"`
	FirstMes       string          `json:"first_mes"`
	MesExample     string          `json:"mes_example"`
	CharacterBook json.RawMessage `json:"character_book"`
}

type charaLoreBook struct {
	Entries []charaLoreEntry `json:"entries"`
}

type charaLoreEntry struct {
	Name      string   `json:"name"`
	Keys      []string `json:"keys"`
	Content   string   `json:"content"`
	Enabled   *bool    `json:"enabled"`
}

// ImportCharaCard reads JSON or PNG (tEXt chara) and writes a .md template plus optional lore pack.
func ImportCharaCard(projectRoot, inputPath, templatesDir string) (templatePath string, lorePack string, err error) {
	raw, err := readCharaPayload(inputPath)
	if err != nil {
		return "", "", err
	}
	var card charaCardV2
	if err := json.Unmarshal(raw, &card); err != nil {
		return "", "", fmt.Errorf("parse chara card: %w", err)
	}
	if strings.TrimSpace(card.Data.Name) == "" {
		var flat charaCardData
		if err := json.Unmarshal(raw, &flat); err == nil && strings.TrimSpace(flat.Name) != "" {
			card.Data = flat
		}
	}
	id := slugTemplateID(card.Data.Name)
	if id == "" {
		id = "imported-character"
	}
	body := buildImportedTemplateBody(card.Data)
	examples := convertMesExample(card.Data.MesExample)
	for _, ex := range examples {
		body += "\n\n<example>\n"
		for _, turn := range ex {
			tag := "{{user}}"
			if turn.Role == "assistant" {
				tag = "{{actor}}"
			}
			body += tag + ": " + turn.Content + "\n"
		}
		body += "</example>"
	}

	var packsAllow []string
	lorePack = ""
	loreEntries := parseLoreEntries(card.Data.CharacterBook)
	if len(loreEntries) > 0 {
		lorePack = id + "-lore"
		packDir := filepath.Join(taijiPacksDir(projectRoot), lorePack)
		if err := os.MkdirAll(packDir, 0o755); err != nil {
			return "", "", err
		}
		for i, e := range loreEntries {
			name := strings.TrimSpace(e.Name)
			if name == "" {
				name = fmt.Sprintf("entry-%d", i+1)
			}
			file := slugTemplateID(name) + ".md"
			kw := e.Keys
			if len(kw) == 0 {
				kw = []string{name}
			}
			node := fmt.Sprintf("---\nname: %s\ninjection: keyword\nkeywords: [%s]\n---\n\n%s\n",
				name, strings.Join(quoteYAMLList(kw), ", "), strings.TrimSpace(e.Content))
			if err := os.WriteFile(filepath.Join(packDir, file), []byte(node), 0o644); err != nil {
				return "", "", err
			}
		}
		packsAllow = []string{lorePack}
	}

	if templatesDir == "" {
		templatesDir = filepath.Join(taijiGlobalTemplatesDir(projectRoot))
	}
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		return "", "", err
	}
	front := fmt.Sprintf(`---
id: %s
name: %q
description: Imported from SillyTavern chara_card_v2
avatar: "🎭"
presentation: narrative
packs_allow: [%s]
---

`, id, card.Data.Name, strings.Join(quoteYAMLList(packsAllow), ", "))
	outPath := filepath.Join(templatesDir, id+".md")
	if err := os.WriteFile(outPath, []byte(front+body), 0o644); err != nil {
		return "", "", err
	}
	return outPath, lorePack, nil
}

func readCharaPayload(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".png") {
		return parsePNGCharaText(raw)
	}
	return raw, nil
}

func parsePNGCharaText(data []byte) ([]byte, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("invalid png")
	}
	off := 8
	for off+8 <= len(data) {
		length := int(data[off])<<24 | int(data[off+1])<<16 | int(data[off+2])<<8 | int(data[off+3])
		chunkType := string(data[off+4 : off+8])
		off += 8
		if off+length > len(data) {
			break
		}
		chunk := data[off : off+length]
		off += length + 4 // CRC
		if chunkType != "tEXt" {
			continue
		}
		key, val, ok := splitPNGText(chunk)
		if ok && key == "chara" {
			trim := strings.TrimSpace(val)
			if strings.HasPrefix(trim, "{") {
				return []byte(trim), nil
			}
			dec, err := base64.StdEncoding.DecodeString(val)
			if err != nil {
				return []byte(val), nil
			}
			return dec, nil
		}
	}
	return nil, fmt.Errorf("png has no chara tEXt chunk")
}

func splitPNGText(chunk []byte) (key, val string, ok bool) {
	i := 0
	for i < len(chunk) && chunk[i] != 0 {
		i++
	}
	if i >= len(chunk) {
		return "", "", false
	}
	return string(chunk[:i]), string(chunk[i+1:]), true
}

func buildImportedTemplateBody(d charaCardData) string {
	var parts []string
	if p := strings.TrimSpace(d.Personality); p != "" {
		parts = append(parts, "# Persönlichkeit\n"+p)
	}
	if p := strings.TrimSpace(d.Description); p != "" {
		parts = append(parts, "# Beschreibung\n"+p)
	}
	if p := strings.TrimSpace(d.Scenario); p != "" {
		parts = append(parts, "# Szenario\n"+p)
	}
	if p := strings.TrimSpace(d.FirstMes); p != "" {
		parts = append(parts, "# Erste Nachricht\n"+p)
	}
	return strings.Join(parts, "\n\n")
}

func convertMesExample(raw string) [][]ExampleTurn {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	// ST format uses <START> blocks
	parts := strings.Split(raw, "<START>")
	var blocks [][]ExampleTurn
	for _, block := range parts {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		var turns []ExampleTurn
		for _, line := range strings.Split(block, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "{{user}}:") {
				turns = append(turns, ExampleTurn{Role: "user", Content: strings.TrimSpace(strings.TrimPrefix(line, "{{user}}:"))})
			} else if strings.HasPrefix(line, "{{char}}:") {
				turns = append(turns, ExampleTurn{Role: "assistant", Content: strings.TrimSpace(strings.TrimPrefix(line, "{{char}}:"))})
			}
		}
		if len(turns) > 0 {
			blocks = append(blocks, turns)
		}
	}
	return blocks
}

func slugTemplateID(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func quoteYAMLList(items []string) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		it = strings.TrimSpace(it)
		if it == "" {
			continue
		}
		out = append(out, fmt.Sprintf("%q", it))
	}
	return out
}

func taijiPacksDir(projectRoot string) string {
	return filepath.Join(strings.TrimSpace(projectRoot), ".taiji", "packs")
}

func taijiGlobalTemplatesDir(projectRoot string) string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "taiji", "templates")
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		return filepath.Join(home, ".config", "taiji", "templates")
	}
	return filepath.Join(projectRoot, ".taiji", "templates")
}
