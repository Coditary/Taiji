package builtin

import (
	"strings"

	"golang.org/x/net/html"
)

var htmlSkipTags = map[string]bool{
	"script": true,
	"style":  true,
	"nav":    true,
	"footer": true,
	"svg":    true,
}

func htmlToMarkdown(raw string) (string, error) {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return "", err
	}
	var b strings.Builder
	var listDepth int
	var render func(*html.Node)
	render = func(n *html.Node) {
		if n.Type == html.ElementNode && htmlSkipTags[n.Data] {
			return
		}
		switch n.Type {
		case html.DocumentNode, html.DoctypeNode:
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				render(c)
			}
			return
		case html.TextNode:
			text := strings.TrimSpace(n.Data)
			if text != "" {
				b.WriteString(text)
			}
		case html.ElementNode:
			switch n.Data {
			case "h1", "h2", "h3", "h4", "h5", "h6":
				level := n.Data[1] - '0'
				b.WriteString("\n\n" + strings.Repeat("#", int(level)) + " ")
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					render(c)
				}
				b.WriteString("\n")
				return
			case "p", "div", "section", "article", "main":
				b.WriteString("\n\n")
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					render(c)
				}
				b.WriteString("\n")
				return
			case "br":
				b.WriteString("\n")
				return
			case "li":
				b.WriteString("\n")
				b.WriteString(strings.Repeat("  ", listDepth) + "- ")
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					render(c)
				}
				return
			case "ul", "ol":
				listDepth++
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					render(c)
				}
				listDepth--
				b.WriteString("\n")
				return
			case "pre":
				b.WriteString("\n\n```\n")
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					render(c)
				}
				b.WriteString("\n```\n")
				return
			case "code":
				b.WriteString("`")
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					render(c)
				}
				b.WriteString("`")
				return
			case "a":
				href := attrVal(n, "href")
				b.WriteString("[")
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					render(c)
				}
				b.WriteString("](" + href + ")")
				return
			default:
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					render(c)
				}
				return
			}
		}
	}
	render(doc)
	out := strings.TrimSpace(b.String())
	out = collapseBlankLines(out)
	return out, nil
}

func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func collapseBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	blank := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blank++
			if blank > 2 {
				continue
			}
		} else {
			blank = 0
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
