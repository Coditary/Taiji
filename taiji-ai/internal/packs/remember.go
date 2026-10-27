package packs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AppendToNode appends markdown content to a pack node under allowed pack roots.
func AppendToNode(packDirs []string, packName, nodeRel, content string) (string, error) {
	packName = strings.TrimSpace(packName)
	nodeRel = strings.TrimSpace(nodeRel)
	content = strings.TrimSpace(content)
	if packName == "" || nodeRel == "" || content == "" {
		return "", fmt.Errorf("pack_name, node_name, and content are required")
	}
	nodeRel = filepath.Clean(nodeRel)
	if nodeRel == "." || strings.HasPrefix(nodeRel, "..") {
		return "", fmt.Errorf("invalid node path")
	}
	if !strings.HasSuffix(strings.ToLower(nodeRel), ".md") {
		nodeRel += ".md"
	}

	for _, root := range packDirs {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		packDir := filepath.Join(root, packName)
		target := filepath.Join(packDir, nodeRel)
		if !pathWithinDir(packDir, target) {
			continue
		}
		if _, err := os.Stat(packDir); err != nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		block := "\n\n" + content + "\n"
		if _, err := os.Stat(target); err != nil {
			if os.IsNotExist(err) {
				initial := fmt.Sprintf("---\nname: %s\ninjection: keyword\n---\n\n%s\n", filepath.Base(nodeRel), content)
				if err := os.WriteFile(target, []byte(initial), 0o644); err != nil {
					return "", err
				}
				return target, nil
			}
			return "", err
		}
		f, err := os.OpenFile(target, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return "", err
		}
		_, err = f.WriteString(block)
		closeErr := f.Close()
		if err != nil {
			return "", err
		}
		if closeErr != nil {
			return "", closeErr
		}
		return target, nil
	}
	return "", fmt.Errorf("pack %q not found under configured pack dirs", packName)
}

func pathWithinDir(dir, target string) bool {
	dir = filepath.Clean(dir)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(dir, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
