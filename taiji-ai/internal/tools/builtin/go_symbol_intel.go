package builtin

import (
	"context"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type GoSymbolIntel struct{}

func (GoSymbolIntel) Name() string { return "go_symbol_intel" }

func (GoSymbolIntel) Description() string {
	return "Go symbol navigation: definition (file + line range + signature), usages (AST references), pkg_summary (exported symbols in a package directory)."
}

func (GoSymbolIntel) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "definition | usages | pkg_summary",
			},
			"symbol": map[string]any{
				"type":        "string",
				"description": "Symbol name (e.g. Tool, Execute, New)",
			},
			"dir": map[string]any{
				"type":        "string",
				"description": "Package directory for pkg_summary or search root",
			},
			"pkg": map[string]any{
				"type":        "string",
				"description": "Alias for dir",
			},
		},
		"required": []string{"action"},
	}
}

func (GoSymbolIntel) Run(_ context.Context, args map[string]any) (string, error) {
	action := strings.ToLower(strings.TrimSpace(stringArg(args, "action")))
	dir := strings.TrimSpace(stringArg(args, "dir"))
	if dir == "" {
		dir = strings.TrimSpace(stringArg(args, "pkg"))
	}
	if dir == "" {
		dir = workspaceProjectRoot()
	}
	symbol := strings.TrimSpace(stringArg(args, "symbol"))

	switch action {
	case "pkg_summary":
		return goPkgSummary(dir)
	case "definition":
		if symbol == "" {
			return "", fmt.Errorf("symbol is required for definition")
		}
		return goFindDefinition(dir, symbol)
	case "usages":
		if symbol == "" {
			return "", fmt.Errorf("symbol is required for usages")
		}
		return goFindUsages(dir, symbol)
	default:
		return "", fmt.Errorf("unknown action %q", action)
	}
}

func goPkgSummary(dir string) (string, error) {
	files, pkgName, err := goPackageFiles(dir)
	if err != nil {
		return "", err
	}
	fset := token.NewFileSet()
	var astFiles []*ast.File
	for _, f := range files {
		af, err := parser.ParseFile(fset, f, nil, parser.ParseComments)
		if err != nil {
			continue
		}
		astFiles = append(astFiles, af)
	}
	if len(astFiles) == 0 {
		return "", fmt.Errorf("no parseable Go files in %s", dir)
	}
	pkg := &ast.Package{Name: pkgName, Files: map[string]*ast.File{}}
	for i, f := range files {
		pkg.Files[filepath.Base(f)] = astFiles[i]
	}
	docPkg := doc.New(pkg, pkgName, doc.AllDecls)
	var lines []string
	lines = append(lines, fmt.Sprintf("package %s (%d files)", pkgName, len(files)))
	for _, t := range docPkg.Types {
		lines = append(lines, fmt.Sprintf("type %s — %s", t.Name, strings.TrimSpace(t.Doc)))
	}
	for _, f := range docPkg.Funcs {
		lines = append(lines, fmt.Sprintf("func %s — %s", f.Name, strings.TrimSpace(f.Doc)))
	}
	for _, v := range docPkg.Vars {
		lines = append(lines, fmt.Sprintf("var %s", v.Names))
	}
	for _, c := range docPkg.Consts {
		lines = append(lines, fmt.Sprintf("const %s", c.Names))
	}
	return strings.Join(lines, "\n"), nil
}

func goFindDefinition(searchRoot, symbol string) (string, error) {
	symbol = strings.TrimSpace(symbol)
	symbol = strings.TrimPrefix(symbol, "*")
	if idx := strings.LastIndex(symbol, "."); idx >= 0 {
		symbol = symbol[idx+1:]
	}
	var hits []string
	_ = walkGoFiles(searchRoot, func(path string, fset *token.FileSet, file *ast.File) error {
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncDecl:
				if n.Name != nil && n.Name.Name == symbol {
					start, end := fset.Position(n.Pos()).Line, fset.Position(n.End()).Line
					hits = append(hits, fmt.Sprintf("%s L%d-L%d %s", path, start, end, formatFuncDecl(n)))
				}
			case *ast.TypeSpec:
				if n.Name != nil && n.Name.Name == symbol {
					start, end := fset.Position(n.Pos()).Line, fset.Position(n.End()).Line
					hits = append(hits, fmt.Sprintf("%s L%d-L%d type %s", path, start, end, n.Name.Name))
				}
			case *ast.GenDecl:
				if n.Tok == token.CONST || n.Tok == token.VAR {
					for _, spec := range n.Specs {
						vs, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}
						for _, name := range vs.Names {
							if name.Name == symbol {
								start, end := fset.Position(vs.Pos()).Line, fset.Position(vs.End()).Line
								hits = append(hits, fmt.Sprintf("%s L%d-L%d %s %s", path, start, end, n.Tok.String(), name.Name))
							}
						}
					}
				}
			}
			return true
		})
		return nil
	})
	if len(hits) == 0 {
		return "no definition found for " + symbol, nil
	}
	return strings.Join(hits, "\n"), nil
}

func goFindUsages(searchRoot, symbol string) (string, error) {
	symbol = strings.TrimSpace(symbol)
	if idx := strings.LastIndex(symbol, "."); idx >= 0 {
		symbol = symbol[idx+1:]
	}
	var hits []string
	_ = walkGoFiles(searchRoot, func(path string, fset *token.FileSet, file *ast.File) error {
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Ident:
				if n.Name != symbol {
					return true
				}
				if n.Obj != nil && n.Obj.Pos() == n.Pos() {
					return true
				}
				line := fset.Position(n.Pos()).Line
				hits = append(hits, fmt.Sprintf("%s L%d", path, line))
			case *ast.SelectorExpr:
				if n.Sel != nil && n.Sel.Name == symbol {
					line := fset.Position(n.Sel.Pos()).Line
					hits = append(hits, fmt.Sprintf("%s L%d", path, line))
				}
			}
			return true
		})
		return nil
	})
	if len(hits) == 0 {
		return "no usages found for " + symbol, nil
	}
	sort.Strings(hits)
	if len(hits) > 80 {
		extra := len(hits) - 80
		hits = hits[:80]
		hits = append(hits, fmt.Sprintf("... (%d more)", extra))
	}
	return strings.Join(hits, "\n"), nil
}

func goPackageFiles(dir string) ([]string, string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = "."
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", err
	}
	var files []string
	pkgName := ""
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		files = append(files, path)
		if pkgName == "" {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
			if err == nil && f.Name != nil {
				pkgName = f.Name.Name
			}
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, "", fmt.Errorf("no Go files in %s", dir)
	}
	return files, pkgName, nil
}

func walkGoFiles(root string, fn func(path string, fset *token.FileSet, file *ast.File) error) error {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
	}
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "vendor" || base == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}
		return fn(path, fset, file)
	})
}
