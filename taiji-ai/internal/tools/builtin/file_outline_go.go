package builtin

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"regexp"
	"strings"
)

func outlineGoSource(filename string, src []byte) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		return "", err
	}
	var lines []string
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			start := fset.Position(d.Pos()).Line
			end := fset.Position(d.End()).Line
			lines = append(lines, fmt.Sprintf("L%d-L%d %s", start, end, formatFuncDecl(d)))
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					start := fset.Position(s.Pos()).Line
					end := fset.Position(s.End()).Line
					kind := d.Tok.String()
					name := s.Name.Name
					extra := ""
					switch s.Type.(type) {
					case *ast.StructType:
						extra = " struct"
					case *ast.InterfaceType:
						extra = " interface"
					}
					lines = append(lines, fmt.Sprintf("L%d-L%d %s %s%s", start, end, kind, name, extra))
				case *ast.ValueSpec:
					start := fset.Position(s.Pos()).Line
					end := fset.Position(s.End()).Line
					names := make([]string, 0, len(s.Names))
					for _, n := range s.Names {
						names = append(names, n.Name)
					}
					lines = append(lines, fmt.Sprintf("L%d-L%d %s %s", start, end, d.Tok.String(), strings.Join(names, ", ")))
				}
			}
		}
	}
	if len(lines) == 0 {
		return "no top-level symbols found", nil
	}
	return strings.Join(lines, "\n"), nil
}

func formatFuncDecl(fd *ast.FuncDecl) string {
	var b strings.Builder
	b.WriteString("func ")
	if fd.Recv != nil && len(fd.Recv.List) > 0 {
		b.WriteString(formatRecv(fd.Recv))
	}
	if fd.Name != nil {
		b.WriteString(fd.Name.Name)
	}
	b.WriteString(formatFieldList(fd.Type.Params))
	if fd.Type.Results != nil {
		b.WriteString(formatFieldList(fd.Type.Results))
	}
	return b.String()
}

func formatRecv(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	f := recv.List[0]
	typ := exprString(f.Type)
	if len(f.Names) > 0 {
		return "(" + f.Names[0].Name + " " + typ + ") "
	}
	return "(" + typ + ") "
}

func formatFieldList(fl *ast.FieldList) string {
	if fl == nil {
		return "()"
	}
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, token.NewFileSet(), fl)
	return buf.String()
}

func exprString(e ast.Expr) string {
	if e == nil {
		return ""
	}
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, token.NewFileSet(), e)
	return buf.String()
}

var reOutlineHeader = regexp.MustCompile(`(?m)^(?:export\s+)?(?:pub\s+)?(?:async\s+)?(?:func|fn|def|class|interface|struct|type|enum)\b`)

func outlineHeuristic(src []byte) string {
	text := strings.ReplaceAll(string(src), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	var out []string
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "#") {
			continue
		}
		if reOutlineHeader.MatchString(line) || reOutlineHeader.MatchString(trim) {
			if len(trim) > 140 {
				trim = trim[:137] + "..."
			}
			out = append(out, fmt.Sprintf("L%d %s", i+1, trim))
		}
	}
	if len(out) == 0 {
		return "no symbol headers detected (heuristic)"
	}
	return strings.Join(out, "\n")
}
