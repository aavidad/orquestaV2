package orquesta_test

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"orquesta/internal/config"
)

const (
	rebuildArchitectureEnvLoader        = "internal/config/env_loader.go"
	rebuildArchitectureLauncherContract = "orquesta/internal/testattestorprotocol/launcher"
	rebuildArchitectureRawDriveProtocol = "orquesta/internal/testattestorprotocol/rawdrive"
)

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

type rebuildArchitectureConfigLiteral struct {
	kind string
	key  string
}

type rebuildArchitectureConfigPolicy struct {
	literals map[string]rebuildArchitectureConfigLiteral
	defaults map[string]string
}

type rebuildArchitectureConfigViolation struct {
	kind    string
	subject string
	pos     token.Pos
}

const (
	rebuildArchitectureViolationCanonicalKey = "canonical_key"
	rebuildArchitectureViolationEnvAlias     = "environment_alias"
	rebuildArchitectureViolationAlias        = "alias"
	rebuildArchitectureViolationDefault      = "typed_default"
	rebuildArchitectureViolationEnvFunction  = "environment_function"
)

func TestRebuildArchitecture(t *testing.T) {
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	files := rebuildArchitectureLoadGoFiles(t, repoRoot, "internal", "cmd/orquesta", "cmd-orquesta")
	configPolicy := rebuildArchitectureLoadConfigPolicy(t)

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

	t.Run("governance_is_shared_inward_domain", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "internal/governance") {
			for _, imported := range file.imports {
				if reason := rebuildArchitectureOnlyInternalPackages(imported.path, "orquesta/internal/governance"); reason != "" {
					rebuildArchitectureImportError(t, file, imported, "internal/governance "+reason)
				}
			}
		}
	})

	t.Run("council_is_shared_inward_domain", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "internal/council") {
			for _, imported := range file.imports {
				if reason := rebuildArchitectureOnlyInternalPackages(imported.path,
					"orquesta/internal/council", "orquesta/internal/governance"); reason != "" {
					rebuildArchitectureImportError(t, file, imported, "internal/council "+reason)
				}
			}
		}
	})

	t.Run("review_is_pure_inward_domain", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "internal/review") {
			for _, imported := range file.imports {
				if reason := rebuildArchitectureOnlyInternalPackages(imported.path, "orquesta/internal/goal", "orquesta/internal/governance", "orquesta/internal/identity"); reason != "" {
					rebuildArchitectureImportError(t, file, imported, "internal/review "+reason)
				}
			}
		}
	})

	t.Run("intake_is_pure_inward_domain", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "internal/intake") {
			for _, imported := range file.imports {
				if reason := rebuildArchitectureIntakeImportReason(imported.path); reason != "" {
					rebuildArchitectureImportError(t, file, imported, "internal/intake "+reason)
				}
			}
		}
	})

	t.Run("application_has_no_delivery_or_concrete_runtime_dependencies", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "internal/application") {
			for _, imported := range file.imports {
				if imported.path == "orquesta/internal/review" {
					continue
				}
				if reason := rebuildArchitectureApplicationImportReason(imported.path); reason != "" {
					rebuildArchitectureImportError(t, file, imported, reason)
				}
			}
		}
	})

	t.Run("ports_depend_only_inward", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "internal/ports") {
			for _, imported := range file.imports {
				// Ports may share pure identity value objects; providers, policy
				// decisions and every concrete identity adapter remain outside.
				if reason := rebuildArchitectureOnlyInternalPackages(imported.path,
					"orquesta/internal/goal", "orquesta/internal/governance", "orquesta/internal/identity"); reason != "" {
					rebuildArchitectureImportError(t, file, imported, "internal/ports "+reason)
				}
			}
		}
	})

	t.Run("identity_depends_only_inward", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "internal/identity") {
			for _, imported := range file.imports {
				if reason := rebuildArchitectureOnlyInternalPackages(imported.path, "orquesta/internal/goal"); reason != "" {
					rebuildArchitectureImportError(t, file, imported, "internal/identity "+reason)
				}
			}
		}
	})

	t.Run("test_attestor_shared_protocols_are_standard_library_only", func(t *testing.T) {
		for _, root := range []string{
			"internal/testattestorprotocol/launcher",
			"internal/testattestorprotocol/rawdrive",
		} {
			for _, file := range rebuildArchitectureFilesUnder(files, root) {
				for _, imported := range file.imports {
					if !rebuildArchitectureIsStandardLibraryImport(imported.path) {
						rebuildArchitectureImportError(
							t,
							file,
							imported,
							"shared test-attestor protocols must depend only on the standard library",
						)
					}
				}
			}
		}
	})

	t.Run("adapters_depend_only_on_inward_contracts", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "internal/adapters") {
			allowed := []string{
				"orquesta/internal/application",
				"orquesta/internal/council",
				"orquesta/internal/credentials",
				"orquesta/internal/goal",
				"orquesta/internal/governance",
				"orquesta/internal/identity",
				"orquesta/internal/intake",
				"orquesta/internal/ports",
				"orquesta/internal/review",
			}
			if rebuildArchitecturePathUnder(file.path, "internal/adapters/config") {
				allowed = append(allowed, "orquesta/internal/config")
			}
			for _, imported := range file.imports {
				if rebuildArchitectureIsSharedAdapterProtocol(imported.path) {
					continue
				}
				if reason := rebuildArchitectureOnlyInternalPackages(imported.path, allowed...); reason != "" {
					rebuildArchitectureImportError(t, file, imported, "internal/adapters "+reason)
				}
			}
		}
		if !rebuildArchitectureIsSharedAdapterProtocol(rebuildArchitectureLauncherContract) ||
			!rebuildArchitectureIsSharedAdapterProtocol(rebuildArchitectureRawDriveProtocol) ||
			rebuildArchitectureIsSharedAdapterProtocol("orquesta/internal/adapters/attestor/firecracker") ||
			rebuildArchitectureIsSharedAdapterProtocol(rebuildArchitectureLauncherContract+"/mutant") ||
			rebuildArchitectureIsSharedAdapterProtocol(rebuildArchitectureRawDriveProtocol+"/mutant") {
			t.Fatal("shared protocol exceptions must remain exact and outside internal/adapters")
		}
	})

	t.Run("credentials_contract_depends_only_inward", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "internal/credentials") {
			for _, imported := range file.imports {
				if reason := rebuildArchitectureOnlyInternalPackages(imported.path); reason != "" {
					rebuildArchitectureImportError(t, file, imported, "internal/credentials "+reason)
				}
			}
		}
	})

	t.Run("config_does_not_import_adapters", func(t *testing.T) {
		for _, file := range rebuildArchitectureFilesUnder(files, "internal/config") {
			for _, imported := range file.imports {
				if imported.path == "orquesta/internal/adapters" || strings.HasPrefix(imported.path, "orquesta/internal/adapters/") {
					rebuildArchitectureImportError(t, file, imported, "internal/config owns policy and ports; concrete adapters must point inward")
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
			for _, violation := range rebuildArchitectureFindEnvironmentViolations(file) {
				position := file.fileSet.Position(violation.pos)
				t.Errorf("%s:%d: os.%s is only allowed in %s", file.path, position.Line, violation.subject, rebuildArchitectureEnvLoader)
			}
		}
	})

	t.Run("configuration_names_and_defaults_are_registry_owned", func(t *testing.T) {
		for _, file := range files {
			if !rebuildArchitectureConfigGuardApplies(file) {
				continue
			}
			for _, violation := range rebuildArchitectureFindConfigViolations(file, configPolicy) {
				position := file.fileSet.Position(violation.pos)
				t.Errorf("%s:%d: %s %q is owned by config/registry.json", file.path, position.Line, violation.kind, violation.subject)
			}
		}
	})

	t.Run("configuration_guard_rejects_nested_and_last_argument_mutants", func(t *testing.T) {
		rebuildArchitectureAssertConfigGuardMutants(t)
	})

	t.Run("inward_domain_allowlist_keeps_concrete_boundaries_forbidden", func(t *testing.T) {
		if reason := rebuildArchitectureGoalImportReason("orquesta/internal/council"); reason != "" {
			t.Errorf("Goal must accept pure council domain: %s", reason)
		}
		if reason := rebuildArchitectureApplicationImportReason("orquesta/internal/council"); reason != "" {
			t.Errorf("application must accept pure council domain: %s", reason)
		}
		if reason := rebuildArchitectureApplicationImportReason("orquesta/internal/intake"); reason != "" {
			t.Errorf("application must accept pure intake domain: %s", reason)
		}
		if reason := rebuildArchitectureIntakeImportReason("orquesta/internal/intake"); reason != "" {
			t.Errorf("intake must accept only its own inward package: %s", reason)
		}
		for name, reason := range map[string]string{
			"goal_adapter":        rebuildArchitectureGoalImportReason("orquesta/internal/adapters/state/sqlite"),
			"goal_intake":         rebuildArchitectureGoalImportReason("orquesta/internal/intake"),
			"goal_http":           rebuildArchitectureGoalImportReason("net/http"),
			"intake_application":  rebuildArchitectureIntakeImportReason("orquesta/internal/application"),
			"intake_adapter":      rebuildArchitectureIntakeImportReason("orquesta/internal/adapters/state/sqlite"),
			"intake_provider":     rebuildArchitectureIntakeImportReason("github.com/openai/client"),
			"application_adapter": rebuildArchitectureApplicationImportReason("orquesta/internal/adapters/state/sqlite"),
			"application_http":    rebuildArchitectureApplicationImportReason("net/http"),
		} {
			if reason == "" {
				t.Errorf("%s concrete dependency escaped architecture guard", name)
			}
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
			syntax, err := parser.ParseFile(fileSet, path, nil, parser.AllErrors|parser.ParseComments)
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

func rebuildArchitectureLoadConfigPolicy(t *testing.T) rebuildArchitectureConfigPolicy {
	t.Helper()
	definitions, aliases := config.Definitions(), config.Aliases()
	if len(definitions) == 0 {
		t.Fatal("config registry has no definitions")
	}
	policy := rebuildArchitectureConfigPolicy{
		literals: make(map[string]rebuildArchitectureConfigLiteral, len(definitions)*2+len(aliases)),
		defaults: make(map[string]string, len(definitions)),
	}
	addLiteral := func(value, kind string) {
		if previous, exists := policy.literals[value]; exists {
			t.Fatalf("config literal %q is both %s and %s", value, previous.kind, kind)
		}
		policy.literals[value] = rebuildArchitectureConfigLiteral{kind: kind, key: value}
	}
	for _, definition := range definitions {
		normalizedGoName := rebuildArchitectureNormalizeIdentifier(definition.GoName)
		if previous, exists := policy.defaults[normalizedGoName]; exists {
			t.Fatalf("config go_name %q collides with %q", definition.GoName, previous)
		}
		policy.defaults[normalizedGoName] = string(definition.Key)
		addLiteral(string(definition.Key), rebuildArchitectureViolationCanonicalKey)
		addLiteral(definition.EnvAlias, rebuildArchitectureViolationEnvAlias)
	}
	for _, alias := range aliases {
		addLiteral(alias.Name, rebuildArchitectureViolationAlias)
	}
	return policy
}

func rebuildArchitecturePathUnder(path, root string) bool {
	path = filepath.ToSlash(path)
	root = strings.TrimSuffix(filepath.ToSlash(root), "/")
	return path == root || strings.HasPrefix(path, root+"/")
}

func rebuildArchitectureConfigGuardApplies(file rebuildArchitectureGoFile) bool {
	if strings.HasSuffix(file.path, "_test.go") {
		return false
	}
	if !rebuildArchitecturePathUnder(file.path, "internal") && !rebuildArchitecturePathUnder(file.path, "cmd/orquesta") {
		return false
	}
	if rebuildArchitecturePathUnder(file.path, "internal/config") || ast.IsGenerated(file.syntax) {
		return false
	}
	return true
}

func rebuildArchitectureFindConfigViolations(file rebuildArchitectureGoFile, policy rebuildArchitectureConfigPolicy) []rebuildArchitectureConfigViolation {
	violations := make([]rebuildArchitectureConfigViolation, 0)
	ast.Inspect(file.syntax, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.BasicLit:
			if node.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(node.Value)
			definition, forbidden := policy.literals[value]
			if err == nil && forbidden {
				violations = append(violations, rebuildArchitectureConfigViolation{
					kind: definition.kind, subject: definition.key, pos: node.Pos(),
				})
			}
		case *ast.Ident:
			name := rebuildArchitectureNormalizeIdentifier(node.Name)
			if !strings.Contains(name, "default") {
				return true
			}
			for goName, key := range policy.defaults {
				index := strings.Index(name, goName)
				if index >= 0 && strings.Contains(name[:index]+name[index+len(goName):], "default") {
					violations = append(violations, rebuildArchitectureConfigViolation{
						kind: rebuildArchitectureViolationDefault, subject: key, pos: node.Pos(),
					})
					break
				}
			}
		}
		return true
	})
	return violations
}

func rebuildArchitectureNormalizeIdentifier(value string) string {
	var result strings.Builder
	for _, character := range strings.ToLower(value) {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' {
			result.WriteRune(character)
		}
	}
	return result.String()
}

func rebuildArchitectureAssertConfigGuardMutants(t *testing.T) {
	t.Helper()
	policy := rebuildArchitectureConfigPolicy{
		literals: map[string]rebuildArchitectureConfigLiteral{
			"server.listen":          {kind: rebuildArchitectureViolationCanonicalKey, key: "server.listen"},
			"ORQUESTA_SERVER_LISTEN": {kind: rebuildArchitectureViolationEnvAlias, key: "ORQUESTA_SERVER_LISTEN"},
			"server.bind":            {kind: rebuildArchitectureViolationAlias, key: "server.bind"},
		},
		defaults: map[string]string{"apilocale": "api.locale"},
	}

	mutant := rebuildArchitectureParseGoSource(t, "internal/adapters/mutant.go", `package mutant
import envos "os"
func wrap(values ...any) any { return values[len(values)-1] }
func sink(values ...any) {}
var APILocaleDefault = wrap("ignored", "es")
func violate() {
	sink("ignored", wrap("server.listen"))
	sink("ignored", "ORQUESTA_SERVER_LISTEN")
	sink("ignored", wrap("server.bind"))
	sink("ignored", wrap(envos.Getenv("X")))
	sink("ignored", wrap(envos.LookupEnv("X")))
	sink("ignored", wrap(envos.Environ()))
	sink("ignored", wrap(envos.ExpandEnv("$X")))
	sink("ignored", wrap(envos.Expand("$X", nil)))
	sink("ignored", wrap(envos.Setenv("X", "Y")))
	sink("ignored", wrap(envos.Unsetenv("X")))
	sink("ignored", wrap(envos.Clearenv()))
}`)
	if !rebuildArchitectureConfigGuardApplies(mutant) {
		t.Fatal("configuration guard does not apply to production mutant")
	}
	gotConfig := make(map[string]int)
	for _, violation := range rebuildArchitectureFindConfigViolations(mutant, policy) {
		gotConfig[violation.kind+":"+violation.subject]++
	}
	wantConfig := map[string]int{
		rebuildArchitectureViolationCanonicalKey + ":server.listen":      1,
		rebuildArchitectureViolationEnvAlias + ":ORQUESTA_SERVER_LISTEN": 1,
		rebuildArchitectureViolationAlias + ":server.bind":               1,
		rebuildArchitectureViolationDefault + ":api.locale":              1,
	}
	if !reflect.DeepEqual(gotConfig, wantConfig) {
		t.Fatalf("configuration mutant escaped or produced unstable matches: got %#v want %#v", gotConfig, wantConfig)
	}

	gotEnvironment := make(map[string]int)
	for _, violation := range rebuildArchitectureFindEnvironmentViolations(mutant) {
		gotEnvironment[violation.subject]++
	}
	wantEnvironment := map[string]int{
		"Getenv": 1, "LookupEnv": 1, "Environ": 1, "ExpandEnv": 1,
		"Expand": 1, "Setenv": 1, "Unsetenv": 1, "Clearenv": 1,
	}
	if !reflect.DeepEqual(gotEnvironment, wantEnvironment) {
		t.Fatalf("environment mutant escaped: got %#v want %#v", gotEnvironment, wantEnvironment)
	}

	benign := rebuildArchitectureParseGoSource(t, "internal/benign/defaults.go", `package benign
const DefaultLocale = "es"
var reasoning = "medium"
var childEnvironment = []string{"PATH"}
`)
	if violations := rebuildArchitectureFindConfigViolations(benign, policy); len(violations) != 0 {
		t.Fatalf("generic literals were treated as configuration defaults: %#v", violations)
	}

	generated := rebuildArchitectureParseGoSource(t, "internal/generated_config.go", `// Code generated by configgen. DO NOT EDIT.
package generated
const key = "server.listen"
`)
	if rebuildArchitectureConfigGuardApplies(generated) {
		t.Fatal("configuration guard scans generated Go")
	}
	ingress := rebuildArchitectureParseGoSource(t, "internal/config/ingress.go", `package config
const key = "server.listen"
`)
	if rebuildArchitectureConfigGuardApplies(ingress) {
		t.Fatal("configuration guard scans canonical config ingress")
	}
}

func rebuildArchitectureParseGoSource(t *testing.T, path, source string) rebuildArchitectureGoFile {
	t.Helper()
	fileSet := token.NewFileSet()
	syntax, err := parser.ParseFile(fileSet, path, source, parser.AllErrors|parser.ParseComments)
	if err != nil {
		t.Fatalf("parse architecture mutant %s: %v", path, err)
	}
	result := rebuildArchitectureGoFile{
		path: path, fileSet: fileSet, syntax: syntax, osAliases: make(map[string]struct{}),
	}
	for _, spec := range syntax.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatalf("parse architecture mutant import: %v", err)
		}
		result.imports = append(result.imports, rebuildArchitectureImport{path: importPath, pos: spec.Pos()})
		if importPath != "os" {
			continue
		}
		alias := "os"
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		switch alias {
		case ".":
			result.dotOS = true
		case "_":
		default:
			result.osAliases[alias] = struct{}{}
		}
	}
	return result
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
	if strings.HasPrefix(importPath, "orquesta/internal/") &&
		importPath != "orquesta/internal/goal" &&
		importPath != "orquesta/internal/council" &&
		importPath != "orquesta/internal/governance" {
		return "internal/goal may depend only on internal/council and internal/governance"
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
		importPath != "orquesta/internal/council" &&
		importPath != "orquesta/internal/governance" &&
		importPath != "orquesta/internal/identity" &&
		importPath != "orquesta/internal/intake" &&
		importPath != "orquesta/internal/ports" {
		return "internal/application may depend only on internal/council, internal/goal, internal/governance, internal/identity, internal/intake and internal/ports"
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

func rebuildArchitectureIntakeImportReason(importPath string) string {
	if importPath == "orquesta/internal/intake" || rebuildArchitectureIsStandardLibraryImport(importPath) {
		return ""
	}
	return "may depend only on the standard library and orquesta/internal/intake"
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

func rebuildArchitectureIsStandardLibraryImport(importPath string) bool {
	if importPath == "" || strings.HasPrefix(importPath, "orquesta/") {
		return false
	}
	first, _, _ := strings.Cut(importPath, "/")
	return !strings.Contains(first, ".")
}

func rebuildArchitectureIsSharedAdapterProtocol(importPath string) bool {
	return importPath == rebuildArchitectureLauncherContract ||
		importPath == rebuildArchitectureRawDriveProtocol
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

func rebuildArchitectureFindEnvironmentViolations(file rebuildArchitectureGoFile) []rebuildArchitectureConfigViolation {
	violations := make([]rebuildArchitectureConfigViolation, 0)
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
			violations = append(violations, rebuildArchitectureConfigViolation{
				kind: rebuildArchitectureViolationEnvFunction, subject: selector.Sel.Name, pos: selector.Pos(),
			})
		}
		return true
	})
	return violations
}

func rebuildArchitectureEnvFunction(name string) bool {
	switch name {
	case "Getenv", "LookupEnv", "Environ", "ExpandEnv", "Expand", "Setenv", "Unsetenv", "Clearenv":
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
		relative, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if strings.HasSuffix(name, ".toml") || strings.HasSuffix(name, ".toml.example") ||
			strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".schema.json") ||
			name == "registry.json" || name == ".gitkeep" || relative == "config/generated/ui.json" {
			return nil
		}
		t.Errorf("%s: human-editable configuration must use TOML", filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		t.Fatalf("scan human-editable configuration root %s: %v", root, err)
	}
}
