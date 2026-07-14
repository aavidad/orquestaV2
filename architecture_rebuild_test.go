package orquesta_test

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const rebuildArchitectureEnvLoader = "internal/config/env_loader.go"

type rebuildArchitectureImport struct {
	path string
	pos  token.Pos
}

type rebuildArchitectureGoFile struct {
	path      string
	fileSet   *token.FileSet
	syntax    *ast.File
	imports   []rebuildArchitectureImport
	osAliases map[string]struct{}
	dotOS     bool
}

func TestRebuildArchitecture(t *testing.T) {
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	files := rebuildArchitectureLoadGoFiles(t, repoRoot, "internal", "cmd/orquesta", "cmd-orquesta")

	t.Run("new_product_does_not_import_legacy_modules", func(t *testing.T) {
		for _, file := range files {
			for _, imported := range file.imports {
				if imported.path == "orquesta/modulos" || strings.HasPrefix(imported.path, "orquesta/modulos/") {
					rebuildArchitectureImportError(t, file, imported, "legacy module imports are forbidden")
				}
			}
		}
	})

	t.Run("goal_is_domain_only", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "internal/goal") {
			for _, imported := range file.imports {
				if reason := rebuildArchitectureGoalImportReason(imported.path); reason != "" {
					rebuildArchitectureImportError(t, file, imported, reason)
				}
			}
		}
	})

	t.Run("application_has_no_delivery_or_concrete_runtime_dependencies", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "internal/application") {
			for _, imported := range file.imports {
				if reason := rebuildArchitectureApplicationImportReason(imported.path); reason != "" {
					rebuildArchitectureImportError(t, file, imported, reason)
				}
			}
		}
	})

	t.Run("ports_and_identity_depend_only_inward", func(t *testing.T) {
		for _, root := range []string{"internal/ports", "internal/identity"} {
			for _, file := range rebuildArchitectureFilesUnder(files, root) {
				for _, imported := range file.imports {
					if reason := rebuildArchitectureOnlyInternalPackages(imported.path, "orquesta/internal/goal"); reason != "" {
						rebuildArchitectureImportError(t, file, imported, root+" "+reason)
					}
				}
			}
		}
	})

	t.Run("adapters_depend_only_on_inward_contracts", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "internal/adapters") {
			for _, imported := range file.imports {
				if reason := rebuildArchitectureOnlyInternalPackages(
					imported.path,
					"orquesta/internal/application",
					"orquesta/internal/goal",
					"orquesta/internal/ports",
				); reason != "" {
					rebuildArchitectureImportError(t, file, imported, "internal/adapters "+reason)
				}
			}
		}
	})

	t.Run("interfaces_do_not_import_adapters_or_bootstrap", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "internal/interfaces") {
			for _, imported := range file.imports {
				if layer := rebuildArchitectureForbiddenInternalLayer(imported.path, "adapter", "adapters", "bootstrap"); layer != "" {
					rebuildArchitectureImportError(t, file, imported, "internal/interfaces must not depend on internal/"+layer)
				}
			}
		}
	})

	t.Run("environment_is_loaded_only_by_config_adapter", func(t *testing.T) {
		for _, file := range files {
			if file.path == rebuildArchitectureEnvLoader {
				continue
			}
			if file.dotOS {
				t.Errorf("%s: dot-import of os is forbidden outside %s", file.path, rebuildArchitectureEnvLoader)
			}
			ast.Inspect(file.syntax, func(node ast.Node) bool {
				selector, ok := node.(*ast.SelectorExpr)
				if !ok || !rebuildArchitectureEnvFunction(selector.Sel.Name) {
					return true
				}
				identifier, ok := selector.X.(*ast.Ident)
				if !ok {
					return true
				}
				if _, importedOS := file.osAliases[identifier.Name]; importedOS {
					position := file.fileSet.Position(selector.Pos())
					t.Errorf("%s:%d: os.%s is only allowed in %s", file.path, position.Line, selector.Sel.Name, rebuildArchitectureEnvLoader)
				}
				return true
			})
		}
	})

	t.Run("command_is_thin_bootstrap", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "cmd/orquesta") {
			for _, imported := range file.imports {
				if rebuildArchitectureCommandImportAllowed(imported.path) {
					continue
				}
				rebuildArchitectureImportError(t, file, imported, "cmd/orquesta may import only the standard library and internal/bootstrap, internal/config, or internal/i18n")
			}
		}
	})

	t.Run("human_editable_config_is_toml_not_yaml", func(t *testing.T) {
		for _, file := range files {
			for _, imported := range file.imports {
				if rebuildArchitectureIsYAMLImport(imported.path) {
					rebuildArchitectureImportError(t, file, imported, "YAML configuration dependencies are forbidden; human-editable configuration uses TOML")
				}
			}
		}
		rebuildArchitectureRejectYAMLFiles(t, repoRoot, "config", "internal", "cmd/orquesta", "cmd-orquesta", "product")
		rebuildArchitectureRequireTOMLHumanConfig(t, repoRoot, "config")
	})
}

func rebuildArchitectureLoadGoFiles(t *testing.T, repoRoot string, roots ...string) []rebuildArchitectureGoFile {
	t.Helper()
	byPath := make(map[string]rebuildArchitectureGoFile)
	for _, root := range roots {
		absoluteRoot := filepath.Join(repoRoot, filepath.FromSlash(root))
		info, err := os.Stat(absoluteRoot)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("stat %s: %v", root, err)
		}
		if !info.IsDir() {
			t.Fatalf("architecture root %s is not a directory", root)
		}

		err = filepath.WalkDir(absoluteRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&fs.ModeSymlink != 0 {
				return &rebuildArchitectureScanError{path: path, reason: "symlinks are not allowed in architecture roots"}
			}
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
				return nil
			}

			relative, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			if _, alreadyLoaded := byPath[relative]; alreadyLoaded {
				return nil
			}

			fileSet := token.NewFileSet()
			syntax, err := parser.ParseFile(fileSet, path, nil, parser.AllErrors)
			if err != nil {
				return &rebuildArchitectureScanError{path: relative, reason: err.Error()}
			}
			parsed := rebuildArchitectureGoFile{
				path:      relative,
				fileSet:   fileSet,
				syntax:    syntax,
				osAliases: make(map[string]struct{}),
			}
			for _, spec := range syntax.Imports {
				importPath, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					return &rebuildArchitectureScanError{path: relative, reason: "invalid import literal"}
				}
				parsed.imports = append(parsed.imports, rebuildArchitectureImport{path: importPath, pos: spec.Pos()})
				if importPath != "os" {
					continue
				}
				alias := "os"
				if spec.Name != nil {
					alias = spec.Name.Name
				}
				switch alias {
				case ".":
					parsed.dotOS = true
				case "_":
				default:
					parsed.osAliases[alias] = struct{}{}
				}
			}
			byPath[relative] = parsed
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
	}

	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	files := make([]rebuildArchitectureGoFile, 0, len(paths))
	for _, path := range paths {
		files = append(files, byPath[path])
	}
	return files
}

type rebuildArchitectureScanError struct {
	path   string
	reason string
}

func (err *rebuildArchitectureScanError) Error() string {
	return err.path + ": " + err.reason
}

func rebuildArchitectureFilesUnder(files []rebuildArchitectureGoFile, root string) []rebuildArchitectureGoFile {
	prefix := strings.TrimSuffix(filepath.ToSlash(root), "/") + "/"
	result := make([]rebuildArchitectureGoFile, 0)
	for _, file := range files {
		if strings.HasPrefix(file.path, prefix) {
			result = append(result, file)
		}
	}
	return result
}

func rebuildArchitectureImportError(t *testing.T, file rebuildArchitectureGoFile, imported rebuildArchitectureImport, reason string) {
	t.Helper()
	position := file.fileSet.Position(imported.pos)
	t.Errorf("%s:%d: forbidden import %q: %s", file.path, position.Line, imported.path, reason)
}

func rebuildArchitectureGoalImportReason(importPath string) string {
	if layer := rebuildArchitectureForbiddenInternalLayer(importPath, "adapter", "adapters", "interface", "interfaces", "bootstrap"); layer != "" {
		return "internal/goal must not depend on internal/" + layer
	}
	if strings.HasPrefix(importPath, "orquesta/internal/") && importPath != "orquesta/internal/goal" {
		return "internal/goal must not depend on another internal package"
	}
	switch {
	case importPath == "database/sql":
		return "internal/goal must not depend on database/sql"
	case importPath == "net/http" || strings.HasPrefix(importPath, "net/http/"):
		return "internal/goal must not depend on HTTP"
	case importPath == "os/exec" || strings.HasPrefix(importPath, "os/exec/"):
		return "internal/goal must not execute processes"
	case rebuildArchitectureIsMCPImport(importPath):
		return "internal/goal must not depend on MCP"
	case rebuildArchitectureIsProviderImport(importPath):
		return "internal/goal must not depend on a provider SDK"
	case rebuildArchitectureIsSQLiteImport(importPath):
		return "internal/goal must not depend on SQLite"
	default:
		return ""
	}
}

func rebuildArchitectureApplicationImportReason(importPath string) string {
	if layer := rebuildArchitectureForbiddenInternalLayer(importPath, "adapter", "adapters", "interface", "interfaces", "bootstrap"); layer != "" {
		return "internal/application must not depend on internal/" + layer
	}
	if strings.HasPrefix(importPath, "orquesta/internal/") &&
		importPath != "orquesta/internal/goal" &&
		importPath != "orquesta/internal/ports" {
		return "internal/application may depend only on internal/goal and internal/ports"
	}
	switch {
	case importPath == "net/http" || strings.HasPrefix(importPath, "net/http/"):
		return "internal/application must not depend on HTTP"
	case rebuildArchitectureIsMCPImport(importPath):
		return "internal/application must not depend on MCP"
	case rebuildArchitectureIsProviderImport(importPath):
		return "internal/application must not depend on a provider SDK"
	case rebuildArchitectureIsSQLiteImport(importPath):
		return "internal/application must not depend on SQLite"
	default:
		return ""
	}
}

func rebuildArchitectureForbiddenInternalLayer(importPath string, layers ...string) string {
	const internalPrefix = "orquesta/internal/"
	if !strings.HasPrefix(importPath, internalPrefix) {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(importPath, internalPrefix), "/")
	for _, part := range parts {
		for _, layer := range layers {
			if part == layer {
				return layer
			}
		}
	}
	return ""
}

func rebuildArchitectureOnlyInternalPackages(importPath string, allowed ...string) string {
	if !strings.HasPrefix(importPath, "orquesta/internal/") {
		return ""
	}
	for _, prefix := range allowed {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return ""
		}
	}
	return "may not depend on " + importPath
}

func rebuildArchitectureIsMCPImport(importPath string) bool {
	lower := strings.ToLower(importPath)
	if strings.Contains(lower, "modelcontextprotocol") {
		return true
	}
	for _, part := range strings.Split(lower, "/") {
		if part == "mcp" || strings.HasPrefix(part, "mcp-") || strings.HasSuffix(part, "-mcp") {
			return true
		}
	}
	return false
}

func rebuildArchitectureIsProviderImport(importPath string) bool {
	lower := strings.ToLower(importPath)
	for _, part := range strings.Split(lower, "/") {
		if part == "provider" || part == "providers" || strings.HasPrefix(part, "provider-") || strings.HasSuffix(part, "-provider") {
			return true
		}
	}
	for _, marker := range []string{"openai", "anthropic", "codex", "gemini", "ollama", "generative-ai", "/genai"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func rebuildArchitectureIsSQLiteImport(importPath string) bool {
	return strings.Contains(strings.ToLower(importPath), "sqlite")
}

func rebuildArchitectureEnvFunction(name string) bool {
	switch name {
	case "Getenv", "LookupEnv", "Environ":
		return true
	default:
		return false
	}
}

func rebuildArchitectureCommandImportAllowed(importPath string) bool {
	for _, allowed := range []string{
		"orquesta/internal/bootstrap",
		"orquesta/internal/config",
		"orquesta/internal/i18n",
	} {
		if importPath == allowed || strings.HasPrefix(importPath, allowed+"/") {
			return true
		}
	}
	if importPath == "C" {
		return false
	}
	pkg, err := build.Default.Import(importPath, ".", build.FindOnly)
	return err == nil && pkg.Goroot
}

func rebuildArchitectureIsYAMLImport(importPath string) bool {
	lower := strings.ToLower(importPath)
	for _, part := range strings.Split(lower, "/") {
		if part == "yaml" || strings.HasPrefix(part, "yaml.") || strings.HasPrefix(part, "yaml-") || strings.HasSuffix(part, "-yaml") {
			return true
		}
	}
	return false
}

func rebuildArchitectureRejectYAMLFiles(t *testing.T, repoRoot string, roots ...string) {
	t.Helper()
	for _, root := range roots {
		absoluteRoot := filepath.Join(repoRoot, filepath.FromSlash(root))
		if _, err := os.Stat(absoluteRoot); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("stat %s: %v", root, err)
		}
		err := filepath.WalkDir(absoluteRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&fs.ModeSymlink != 0 {
				return &rebuildArchitectureScanError{path: path, reason: "symlinks are not allowed in architecture roots"}
			}
			if entry.IsDir() {
				return nil
			}
			extension := strings.ToLower(filepath.Ext(entry.Name()))
			if extension != ".yaml" && extension != ".yml" {
				return nil
			}
			relative, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			t.Errorf("%s: YAML configuration is forbidden; human-editable configuration uses TOML", filepath.ToSlash(relative))
			return nil
		})
		if err != nil {
			t.Fatalf("scan configuration root %s: %v", root, err)
		}
	}
}

func rebuildArchitectureRequireTOMLHumanConfig(t *testing.T, repoRoot string, root string) {
	t.Helper()
	absoluteRoot := filepath.Join(repoRoot, filepath.FromSlash(root))
	if _, err := os.Stat(absoluteRoot); err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("stat %s: %v", root, err)
	}
	err := filepath.WalkDir(absoluteRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return &rebuildArchitectureScanError{path: path, reason: "symlinks are not allowed in architecture roots"}
		}
		if entry.IsDir() {
			return nil
		}
		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".toml") || strings.HasSuffix(name, ".toml.example") ||
			strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".schema.json") ||
			name == "registry.json" || name == ".gitkeep" {
			return nil
		}
		relative, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		t.Errorf("%s: human-editable configuration must use TOML", filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		t.Fatalf("scan human-editable configuration root %s: %v", root, err)
	}
}
