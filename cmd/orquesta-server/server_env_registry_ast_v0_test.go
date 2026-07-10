package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type serverEnvRegistryASTHitV0 struct {
	Env      string
	Location string
}

const serverEnvRegistryASTMissingBaselineV0 = 36

func TestServerEnvRegistryASTV0LecturasORQUESTARegistradas(t *testing.T) {
	root := findRepoRootForServerEnvRegistryASTV0(t)
	dir := filepath.Join(root, "cmd", "orquesta-server")
	files, err := serverEnvRegistryASTProductionFilesV0(dir)
	if err != nil {
		t.Fatalf("listar go produccion: %v", err)
	}
	fset := token.NewFileSet()
	parsed := make([]*ast.File, 0, len(files))
	for _, path := range files {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		parsed = append(parsed, file)
	}
	literals := serverEnvRegistryASTStringLiteralsV0(parsed)
	hits := serverEnvRegistryASTFindReadsV0(fset, parsed, literals)
	allow := serverEnvRegistryASTAllowlistV0()
	var missing []serverEnvRegistryASTHitV0
	for _, hit := range hits {
		if _, ok := serverEffectiveEnvRegistryV0[hit.Env]; ok {
			continue
		}
		if allow[hit.Env] {
			continue
		}
		missing = append(missing, hit)
	}
	sort.Slice(missing, func(i, j int) bool {
		if missing[i].Env == missing[j].Env {
			return missing[i].Location < missing[j].Location
		}
		return missing[i].Env < missing[j].Env
	})
	if len(missing) > 0 {
		t.Logf(
			"diagnostico: lecturas ORQUESTA_* sin serverEffectiveEnvRegistryV0 ni allowlist child/test harness: total=%d baseline=%d primeros=%s",
			len(missing),
			serverEnvRegistryASTMissingBaselineV0,
			serverEnvRegistryASTFormatHitsV0(missing, 20),
		)
	}
	if len(missing) > serverEnvRegistryASTMissingBaselineV0 {
		t.Fatalf(
			"lecturas ORQUESTA_* sin registry aumentaron: total=%d baseline=%d primeros=%s",
			len(missing),
			serverEnvRegistryASTMissingBaselineV0,
			serverEnvRegistryASTFormatHitsV0(missing, 20),
		)
	}
}

func serverEnvRegistryASTProductionFilesV0(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	sort.Strings(files)
	return files, err
}

func serverEnvRegistryASTStringLiteralsV0(files []*ast.File) map[string]string {
	exprs := map[string]ast.Expr{}
	values := map[string]string{}
	for _, file := range files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || (gen.Tok != token.CONST && gen.Tok != token.VAR) {
				continue
			}
			for _, spec := range gen.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range valueSpec.Names {
					if i >= len(valueSpec.Values) {
						continue
					}
					exprs[name.Name] = valueSpec.Values[i]
				}
			}
		}
	}
	for name := range exprs {
		if value, ok := serverEnvRegistryASTResolveStringV0(name, exprs, map[string]bool{}); ok {
			values[name] = value
		}
	}
	return values
}

func serverEnvRegistryASTResolveStringV0(name string, exprs map[string]ast.Expr, visiting map[string]bool) (string, bool) {
	if visiting[name] {
		return "", false
	}
	expr, ok := exprs[name]
	if !ok {
		return "", false
	}
	if value, ok := serverEnvRegistryASTLiteralStringV0(expr); ok {
		return value, true
	}
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return "", false
	}
	visiting[name] = true
	value, ok := serverEnvRegistryASTResolveStringV0(ident.Name, exprs, visiting)
	delete(visiting, name)
	return value, ok
}

func serverEnvRegistryASTFindReadsV0(fset *token.FileSet, files []*ast.File, literals map[string]string) []serverEnvRegistryASTHitV0 {
	helperNames := map[string]bool{
		"envOrDefaultV0":      true,
		"intEnvOrDefaultV0":   true,
		"int64EnvOrDefaultV0": true,
		"boolEnvOrDefaultV0":  true,
		"csvEnvOrDefaultV0":   true,
	}
	seen := map[string]bool{}
	var hits []serverEnvRegistryASTHitV0
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			if !serverEnvRegistryASTIsTrackedCallV0(call, helperNames) {
				return true
			}
			env, ok := serverEnvRegistryASTEnvValueV0(call.Args[0], literals)
			if !ok || !strings.HasPrefix(env, "ORQUESTA_") {
				return true
			}
			pos := fset.Position(call.Lparen)
			location := serverEnvRegistryASTRelLocationV0(pos.Filename) + ":" + strconv.Itoa(pos.Line)
			key := env + "\x00" + location
			if seen[key] {
				return true
			}
			seen[key] = true
			hits = append(hits, serverEnvRegistryASTHitV0{Env: env, Location: location})
			return true
		})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Env == hits[j].Env {
			return hits[i].Location < hits[j].Location
		}
		return hits[i].Env < hits[j].Env
	})
	return hits
}

func serverEnvRegistryASTRelLocationV0(path string) string {
	root, err := os.Getwd()
	if err != nil {
		return filepath.ToSlash(path)
	}
	for {
		if hasDirServerEnvRegistryASTV0(root, "cmd") &&
			hasDirServerEnvRegistryASTV0(root, "modulos") {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			return filepath.ToSlash(path)
		}
		root = parent
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func serverEnvRegistryASTIsTrackedCallV0(call *ast.CallExpr, helperNames map[string]bool) bool {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return helperNames[fun.Name]
	case *ast.SelectorExpr:
		pkg, ok := fun.X.(*ast.Ident)
		return ok && pkg.Name == "os" && (fun.Sel.Name == "Getenv" || fun.Sel.Name == "LookupEnv")
	default:
		return false
	}
}

func serverEnvRegistryASTEnvValueV0(expr ast.Expr, literals map[string]string) (string, bool) {
	if value, ok := serverEnvRegistryASTLiteralStringV0(expr); ok {
		return value, true
	}
	if ident, ok := expr.(*ast.Ident); ok {
		value, ok := literals[ident.Name]
		return value, ok
	}
	return "", false
}

func serverEnvRegistryASTLiteralStringV0(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

func serverEnvRegistryASTAllowlistV0() map[string]bool {
	return map[string]bool{
		"ORQUESTA_GUARDIAN_BUILD_COMMAND":     true,
		"ORQUESTA_GUARDIAN_CANDIDATE_BIN":     true,
		"ORQUESTA_GUARDIAN_RUNNER_ENV_POLICY": true,
		"ORQUESTA_GUARDIAN_WORKTREE_REF":      true,
	}
}

func serverEnvRegistryASTFormatHitsV0(hits []serverEnvRegistryASTHitV0, limit int) string {
	if len(hits) == 0 {
		return ""
	}
	if limit <= 0 || limit > len(hits) {
		limit = len(hits)
	}
	parts := make([]string, 0, limit)
	for _, hit := range hits[:limit] {
		parts = append(parts, hit.Env+"@"+hit.Location)
	}
	if len(hits) > limit {
		parts = append(parts, "...+"+strconv.Itoa(len(hits)-limit))
	}
	return strings.Join(parts, ", ")
}

func findRepoRootForServerEnvRegistryASTV0(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if hasDirServerEnvRegistryASTV0(dir, "cmd") &&
			hasDirServerEnvRegistryASTV0(dir, "modulos") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root no encontrado desde %s", dir)
		}
		dir = parent
	}
}

func hasDirServerEnvRegistryASTV0(root string, name string) bool {
	info, err := os.Stat(filepath.Join(root, name))
	return err == nil && info.IsDir()
}
