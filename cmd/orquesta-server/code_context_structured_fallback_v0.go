package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
)

type serverStructuredGoFileV0 struct {
	Path   string
	Rel    string
	Source []byte
	File   *ast.File
	Fset   *token.FileSet
}

func serverStructuredCodeContextHitsV0(
	root string,
	query orquestacontext.CodeContextQueryV0,
) ([]orquestacontext.CodeContextHitV0, error) {
	files, err := serverStructuredGoFilesV0(root, query)
	if err != nil {
		return nil, err
	}
	switch query.QueryKind {
	case orquestacontext.CodeContextQueryKindCallersV0:
		return serverStructuredCallersHitsV0(files, query), nil
	case orquestacontext.CodeContextQueryKindImportsV0:
		return serverStructuredImportHitsV0(files, query), nil
	case orquestacontext.CodeContextQueryKindModuleExportsV0:
		return serverStructuredModuleExportHitsV0(files, query), nil
	case orquestacontext.CodeContextQueryKindRelevantSnippetsV0:
		return serverStructuredRelevantSnippetHitsV0(files, query), nil
	default:
		return nil, nil
	}
}

func serverStructuredGoFilesV0(root string, query orquestacontext.CodeContextQueryV0) ([]serverStructuredGoFileV0, error) {
	root, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return nil, err
	}
	scopes := serverRGCodeContextScopesV0(query.Scope)
	if len(scopes) == 0 {
		scopes = []string{"cmd", "modulos"}
	}
	files := make([]serverStructuredGoFileV0, 0)
	for _, scope := range scopes {
		target := filepath.Join(root, filepath.Clean(scope))
		if !serverCodeContextPathWithinRootV0(root, target) {
			continue
		}
		info, err := os.Stat(target)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			if filepath.Ext(target) == ".go" {
				if file, ok := serverParseStructuredGoFileV0(root, target); ok {
					files = append(files, file)
				}
			}
			continue
		}
		if err := filepath.WalkDir(target, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || entry == nil {
				return nil
			}
			if entry.IsDir() {
				if serverGoCodeContextSkipDirV0(entry.Name()) && path != target {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") ||
				!serverCodeContextPathWithinRootV0(root, path) {
				return nil
			}
			if file, ok := serverParseStructuredGoFileV0(root, path); ok {
				files = append(files, file)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func serverParseStructuredGoFileV0(root string, path string) (serverStructuredGoFileV0, bool) {
	source, err := os.ReadFile(path)
	if err != nil {
		return serverStructuredGoFileV0{}, false
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
	if err != nil {
		return serverStructuredGoFileV0{}, false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = filepath.Base(path)
	}
	return serverStructuredGoFileV0{
		Path:   path,
		Rel:    filepath.ToSlash(filepath.Clean(rel)),
		Source: source,
		File:   file,
		Fset:   fset,
	}, true
}

func serverStructuredCallersHitsV0(
	files []serverStructuredGoFileV0,
	query orquestacontext.CodeContextQueryV0,
) []orquestacontext.CodeContextHitV0 {
	needle := strings.TrimSpace(query.Query)
	maxResults := serverCodeContextMaxResultsV0(query)
	hits := make([]orquestacontext.CodeContextHitV0, 0, maxResults)
	for _, file := range files {
		if len(hits) >= maxResults {
			break
		}
		for _, decl := range file.File.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			caller := serverStructuredFuncNameV0(fn)
			called := false
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				if called {
					return false
				}
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				callee := serverStructuredCalleeNameV0(call.Fun)
				if callee != "" && serverStructuredSymbolMatchesV0(callee, needle) {
					called = true
					return false
				}
				return true
			})
			if !called {
				continue
			}
			line := file.Fset.Position(fn.Pos()).Line
			hits = append(hits, orquestacontext.CodeContextHitV0{
				Kind:    "caller",
				Path:    file.Rel,
				Line:    line,
				Symbol:  caller,
				Summary: "caller " + caller + " -> " + needle,
				Snippet: serverStructuredNodeSnippetV0(file, fn),
				Score:   1,
			})
			if len(hits) >= maxResults {
				break
			}
		}
	}
	return serverStructuredNormalizeHitRefsV0(hits, "code-context-callers-hit-")
}

func serverStructuredImportHitsV0(
	files []serverStructuredGoFileV0,
	query orquestacontext.CodeContextQueryV0,
) []orquestacontext.CodeContextHitV0 {
	maxResults := serverCodeContextMaxResultsV0(query)
	hits := make([]orquestacontext.CodeContextHitV0, 0, maxResults)
	for _, file := range files {
		if len(hits) >= maxResults {
			break
		}
		for _, spec := range file.File.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			if !serverCodeContextSmartCaseMatchV0(importPath, query.Query) &&
				!serverCodeContextSmartCaseMatchV0(file.Rel, query.Query) {
				continue
			}
			line := file.Fset.Position(spec.Pos()).Line
			hits = append(hits, orquestacontext.CodeContextHitV0{
				Kind:    "import",
				Path:    file.Rel,
				Line:    line,
				Symbol:  importPath,
				Summary: "import " + importPath + " en " + file.Rel,
				Snippet: serverStructuredLineSnippetV0(file.Source, line),
				Score:   1,
			})
			if len(hits) >= maxResults {
				break
			}
		}
	}
	return serverStructuredNormalizeHitRefsV0(hits, "code-context-imports-hit-")
}

func serverStructuredModuleExportHitsV0(
	files []serverStructuredGoFileV0,
	query orquestacontext.CodeContextQueryV0,
) []orquestacontext.CodeContextHitV0 {
	maxResults := serverCodeContextMaxResultsV0(query)
	hits := make([]orquestacontext.CodeContextHitV0, 0, maxResults)
	for _, file := range files {
		if len(hits) >= maxResults {
			break
		}
		if !serverStructuredModuleMatchesV0(file.Rel, query.Query) {
			continue
		}
		for _, decl := range file.File.Decls {
			switch typed := decl.(type) {
			case *ast.FuncDecl:
				if typed.Name == nil || !ast.IsExported(typed.Name.Name) {
					continue
				}
				line := file.Fset.Position(typed.Pos()).Line
				hits = append(hits, orquestacontext.CodeContextHitV0{
					Kind:    "exported_function",
					Path:    file.Rel,
					Line:    line,
					Symbol:  serverStructuredFuncNameV0(typed),
					Summary: "export de modulo " + file.Rel,
					Snippet: serverStructuredNodeSnippetV0(file, typed),
					Score:   1,
				})
			case *ast.GenDecl:
				for _, spec := range typed.Specs {
					name := serverStructuredSpecExportedNameV0(spec)
					if name == "" {
						continue
					}
					line := file.Fset.Position(spec.Pos()).Line
					hits = append(hits, orquestacontext.CodeContextHitV0{
						Kind:    "exported_" + strings.ToLower(typed.Tok.String()),
						Path:    file.Rel,
						Line:    line,
						Symbol:  name,
						Summary: "export de modulo " + file.Rel,
						Snippet: serverStructuredNodeSnippetV0(file, spec),
						Score:   1,
					})
					if len(hits) >= maxResults {
						break
					}
				}
			}
			if len(hits) >= maxResults {
				break
			}
		}
	}
	return serverStructuredNormalizeHitRefsV0(hits, "code-context-module-exports-hit-")
}

func serverStructuredRelevantSnippetHitsV0(
	files []serverStructuredGoFileV0,
	query orquestacontext.CodeContextQueryV0,
) []orquestacontext.CodeContextHitV0 {
	maxResults := serverCodeContextMaxResultsV0(query)
	hits := make([]orquestacontext.CodeContextHitV0, 0, maxResults)
	for _, file := range files {
		if len(hits) >= maxResults {
			break
		}
		for _, decl := range file.File.Decls {
			symbol := serverStructuredDeclSymbolV0(decl)
			snippet := serverStructuredNodeSnippetV0(file, decl)
			if symbol == "" && snippet == "" {
				continue
			}
			if !serverStructuredSymbolMatchesV0(symbol, query.Query) &&
				!serverCodeContextSmartCaseMatchV0(snippet, query.Query) &&
				!serverCodeContextSmartCaseMatchV0(file.Rel, query.Query) {
				continue
			}
			line := file.Fset.Position(decl.Pos()).Line
			hits = append(hits, orquestacontext.CodeContextHitV0{
				Kind:    "relevant_snippet",
				Path:    file.Rel,
				Line:    line,
				Symbol:  symbol,
				Summary: "snippet relevante para " + strings.TrimSpace(query.Query),
				Snippet: snippet,
				Score:   1,
			})
			if len(hits) >= maxResults {
				break
			}
		}
	}
	return serverStructuredNormalizeHitRefsV0(hits, "code-context-snippet-hit-")
}

func serverCodeContextMaxResultsV0(query orquestacontext.CodeContextQueryV0) int {
	if query.MaxResults > 0 {
		return query.MaxResults
	}
	return 8
}

func serverStructuredFuncNameV0(fn *ast.FuncDecl) string {
	if fn == nil || fn.Name == nil {
		return ""
	}
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		return serverStructuredReceiverNameV0(fn.Recv.List[0].Type) + "." + fn.Name.Name
	}
	return fn.Name.Name
}

func serverStructuredReceiverNameV0(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return serverStructuredReceiverNameV0(typed.X)
	case *ast.SelectorExpr:
		return typed.Sel.Name
	default:
		return "receiver"
	}
}

func serverStructuredCalleeNameV0(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return typed.Sel.Name
	default:
		return ""
	}
}

func serverStructuredSpecExportedNameV0(spec ast.Spec) string {
	switch typed := spec.(type) {
	case *ast.TypeSpec:
		if typed.Name != nil && ast.IsExported(typed.Name.Name) {
			return typed.Name.Name
		}
	case *ast.ValueSpec:
		for _, name := range typed.Names {
			if name != nil && ast.IsExported(name.Name) {
				return name.Name
			}
		}
	}
	return ""
}

func serverStructuredDeclSymbolV0(decl ast.Decl) string {
	switch typed := decl.(type) {
	case *ast.FuncDecl:
		return serverStructuredFuncNameV0(typed)
	case *ast.GenDecl:
		for _, spec := range typed.Specs {
			if name := serverStructuredSpecExportedNameV0(spec); name != "" {
				return name
			}
		}
	}
	return ""
}

func serverStructuredSymbolMatchesV0(symbol string, query string) bool {
	symbol = strings.TrimSpace(symbol)
	query = strings.TrimSpace(query)
	if symbol == "" || query == "" {
		return false
	}
	if serverCodeContextSmartCaseMatchV0(symbol, query) {
		return true
	}
	if idx := strings.LastIndex(query, "."); idx >= 0 {
		return serverCodeContextSmartCaseMatchV0(symbol, query[idx+1:])
	}
	return false
}

func serverStructuredModuleMatchesV0(path string, query string) bool {
	query = strings.Trim(strings.TrimSpace(query), `/\`)
	if query == "" {
		return true
	}
	path = filepath.ToSlash(filepath.Clean(path))
	if strings.HasPrefix(path, query) {
		return true
	}
	return serverCodeContextSmartCaseMatchV0(path, query)
}

func serverStructuredNodeSnippetV0(file serverStructuredGoFileV0, node ast.Node) string {
	if node == nil {
		return ""
	}
	start := file.Fset.Position(node.Pos()).Offset
	end := file.Fset.Position(node.End()).Offset
	if start < 0 || start >= len(file.Source) || end <= start {
		return ""
	}
	if end > len(file.Source) {
		end = len(file.Source)
	}
	return serverGoCodeContextSnippetV0(string(file.Source[start:end]))
}

func serverStructuredLineSnippetV0(source []byte, line int) string {
	if line <= 0 {
		return ""
	}
	lines := strings.Split(string(source), "\n")
	if line > len(lines) {
		return ""
	}
	return serverGoCodeContextSnippetV0(lines[line-1])
}

func serverStructuredNormalizeHitRefsV0(
	hits []orquestacontext.CodeContextHitV0,
	prefix string,
) []orquestacontext.CodeContextHitV0 {
	for idx := range hits {
		hits[idx].HitRef = prefix + strconv.Itoa(idx+1)
	}
	return hits
}
