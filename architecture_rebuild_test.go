package orquesta_test

import (
	"encoding/json"
	"fmt"
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
	"time"
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

type rebuildArchitectureConfigRegistry struct {
	Aliases []struct {
		Name string `json:"name"`
	} `json:"aliases"`
	Keys []rebuildArchitectureConfigRegistryKey `json:"keys"`
}

type rebuildArchitectureConfigRegistryKey struct {
	Key      string          `json:"key"`
	GoName   string          `json:"go_name"`
	Type     string          `json:"type"`
	Default  json.RawMessage `json:"default"`
	EnvAlias string          `json:"env_alias"`
}

type rebuildArchitectureConfigLiteral struct {
	kind string
	key  string
}

type rebuildArchitectureConfigDefault struct {
	key       string
	goName    string
	valueType string
	value     any
}

type rebuildArchitectureConfigPolicy struct {
	literals map[string]rebuildArchitectureConfigLiteral
	defaults []rebuildArchitectureConfigDefault
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
	configPolicy := rebuildArchitectureLoadConfigPolicy(t, repoRoot)

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
			allowed := []string{
				"orquesta/internal/application",
				"orquesta/internal/goal",
				"orquesta/internal/ports",
			}
			if rebuildArchitecturePathUnder(file.path, "internal/adapters/config") {
				allowed = append(allowed, "orquesta/internal/config")
			}
			for _, imported := range file.imports {
				if reason := rebuildArchitectureOnlyInternalPackages(imported.path, allowed...); reason != "" {
					rebuildArchitectureImportError(t, file, imported, "internal/adapters "+reason)
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

func rebuildArchitectureLoadConfigPolicy(t *testing.T, repoRoot string) rebuildArchitectureConfigPolicy {
	t.Helper()
	path := filepath.Join(repoRoot, "config", "registry.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config registry: %v", err)
	}
	policy, err := rebuildArchitectureConfigPolicyFromJSON(content)
	if err != nil {
		t.Fatalf("build architecture policy from config/registry.json: %v", err)
	}
	return policy
}

func rebuildArchitectureConfigPolicyFromJSON(content []byte) (rebuildArchitectureConfigPolicy, error) {
	var source rebuildArchitectureConfigRegistry
	if err := json.Unmarshal(content, &source); err != nil {
		return rebuildArchitectureConfigPolicy{}, err
	}
	if len(source.Keys) == 0 {
		return rebuildArchitectureConfigPolicy{}, fmt.Errorf("registry has no keys")
	}

	policy := rebuildArchitectureConfigPolicy{
		literals: make(map[string]rebuildArchitectureConfigLiteral, len(source.Keys)*2+len(source.Aliases)),
		defaults: make([]rebuildArchitectureConfigDefault, 0, len(source.Keys)),
	}
	addLiteral := func(value, kind string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("empty %s", kind)
		}
		if previous, exists := policy.literals[value]; exists {
			return fmt.Errorf("literal %q is both %s and %s", value, previous.kind, kind)
		}
		policy.literals[value] = rebuildArchitectureConfigLiteral{kind: kind, key: value}
		return nil
	}

	goNames := make(map[string]string, len(source.Keys))
	for _, key := range source.Keys {
		if strings.TrimSpace(key.Key) == "" || strings.TrimSpace(key.GoName) == "" || strings.TrimSpace(key.Type) == "" {
			return rebuildArchitectureConfigPolicy{}, fmt.Errorf("incomplete key definition for %q", key.Key)
		}
		normalizedGoName := rebuildArchitectureNormalizeIdentifier(key.GoName)
		if normalizedGoName == "" {
			return rebuildArchitectureConfigPolicy{}, fmt.Errorf("invalid go_name %q", key.GoName)
		}
		if previous, exists := goNames[normalizedGoName]; exists {
			return rebuildArchitectureConfigPolicy{}, fmt.Errorf("go_name %q collides with %q", key.GoName, previous)
		}
		goNames[normalizedGoName] = key.GoName

		if err := addLiteral(key.Key, rebuildArchitectureViolationCanonicalKey); err != nil {
			return rebuildArchitectureConfigPolicy{}, err
		}
		if err := addLiteral(key.EnvAlias, rebuildArchitectureViolationEnvAlias); err != nil {
			return rebuildArchitectureConfigPolicy{}, err
		}
		value, err := rebuildArchitectureDecodeConfigDefault(key.Type, key.Default)
		if err != nil {
			return rebuildArchitectureConfigPolicy{}, fmt.Errorf("default for %s: %w", key.Key, err)
		}
		policy.defaults = append(policy.defaults, rebuildArchitectureConfigDefault{
			key:       key.Key,
			goName:    key.GoName,
			valueType: key.Type,
			value:     value,
		})
	}
	for _, alias := range source.Aliases {
		if err := addLiteral(alias.Name, rebuildArchitectureViolationAlias); err != nil {
			return rebuildArchitectureConfigPolicy{}, err
		}
	}
	return policy, nil
}

func rebuildArchitectureDecodeConfigDefault(valueType string, raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("missing value")
	}
	switch valueType {
	case "string", "path", "credential_ref":
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		return value, nil
	case "integer":
		decoder := json.NewDecoder(strings.NewReader(string(raw)))
		decoder.UseNumber()
		var value json.Number
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		integer, err := value.Int64()
		if err != nil {
			return nil, err
		}
		return integer, nil
	case "duration":
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		duration, err := time.ParseDuration(value)
		if err != nil {
			return nil, err
		}
		return duration, nil
	case "string_list":
		var value []string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		return value, nil
	case "boolean":
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported type %q", valueType)
	}
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
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		definition, forbidden := policy.literals[value]
		if forbidden {
			violations = append(violations, rebuildArchitectureConfigViolation{
				kind: definition.kind, subject: definition.key, pos: literal.Pos(),
			})
		}
		return true
	})

	type defaultOccurrence struct {
		key string
		pos token.Pos
	}
	seenDefaults := make(map[defaultOccurrence]struct{})
	checkContext := func(name string, expressions ...ast.Expr) {
		for _, definition := range policy.defaults {
			if !rebuildArchitectureConfigContextMatches(name, definition.goName) {
				continue
			}
			for _, expression := range expressions {
				if expression == nil {
					continue
				}
				position, found := rebuildArchitectureFindDefaultExpression(expression, definition)
				if !found {
					continue
				}
				occurrence := defaultOccurrence{key: definition.key, pos: position}
				if _, duplicate := seenDefaults[occurrence]; duplicate {
					continue
				}
				seenDefaults[occurrence] = struct{}{}
				violations = append(violations, rebuildArchitectureConfigViolation{
					kind: rebuildArchitectureViolationDefault, subject: definition.key, pos: position,
				})
			}
		}
	}

	ast.Inspect(file.syntax, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.ValueSpec:
			for index, name := range node.Names {
				expressions := node.Values
				if len(node.Names) == len(node.Values) {
					expressions = node.Values[index : index+1]
				}
				checkContext(name.Name, expressions...)
			}
		case *ast.AssignStmt:
			for index, left := range node.Lhs {
				name := rebuildArchitectureExpressionName(left)
				if name == "" {
					continue
				}
				expressions := node.Rhs
				if len(node.Lhs) == len(node.Rhs) {
					expressions = node.Rhs[index : index+1]
				}
				checkContext(name, expressions...)
			}
		case *ast.KeyValueExpr:
			if name := rebuildArchitectureExpressionName(node.Key); name != "" {
				checkContext(name, node.Value)
			}
		case *ast.CallExpr:
			if name := rebuildArchitectureExpressionName(node.Fun); name != "" {
				checkContext(name, node.Args...)
			}
		case *ast.FuncDecl:
			if node.Body == nil {
				break
			}
			ast.Inspect(node.Body, func(child ast.Node) bool {
				result, ok := child.(*ast.ReturnStmt)
				if ok {
					checkContext(node.Name.Name, result.Results...)
				}
				return true
			})
		}
		return true
	})
	return violations
}

func rebuildArchitectureExpressionName(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name
	case *ast.SelectorExpr:
		return expression.Sel.Name
	case *ast.IndexExpr:
		return rebuildArchitectureExpressionName(expression.X)
	case *ast.IndexListExpr:
		return rebuildArchitectureExpressionName(expression.X)
	case *ast.ParenExpr:
		return rebuildArchitectureExpressionName(expression.X)
	default:
		return ""
	}
}

func rebuildArchitectureConfigContextMatches(name, goName string) bool {
	name = rebuildArchitectureNormalizeIdentifier(name)
	goName = rebuildArchitectureNormalizeIdentifier(goName)
	if name == "" || goName == "" {
		return false
	}
	return name == goName || strings.Contains(name, goName) && strings.Contains(name, "default")
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

func rebuildArchitectureFindDefaultExpression(expression ast.Expr, definition rebuildArchitectureConfigDefault) (token.Pos, bool) {
	var position token.Pos
	ast.Inspect(expression, func(node ast.Node) bool {
		candidate, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		value, ok := rebuildArchitectureEvaluateDefaultExpression(candidate, definition.valueType)
		if !ok || !reflect.DeepEqual(value, definition.value) {
			return true
		}
		position = candidate.Pos()
		return false
	})
	return position, position.IsValid()
}

func rebuildArchitectureEvaluateDefaultExpression(expression ast.Expr, valueType string) (any, bool) {
	switch valueType {
	case "string", "path", "credential_ref":
		return rebuildArchitectureEvaluateString(expression)
	case "integer":
		return rebuildArchitectureEvaluateInteger(expression)
	case "duration":
		return rebuildArchitectureEvaluateDuration(expression)
	case "string_list":
		return rebuildArchitectureEvaluateStringList(expression)
	case "boolean":
		identifier, ok := expression.(*ast.Ident)
		if !ok || identifier.Name != "true" && identifier.Name != "false" {
			return nil, false
		}
		return identifier.Name == "true", true
	default:
		return nil, false
	}
}

func rebuildArchitectureEvaluateString(expression ast.Expr) (string, bool) {
	switch expression := expression.(type) {
	case *ast.BasicLit:
		if expression.Kind != token.STRING {
			return "", false
		}
		value, err := strconv.Unquote(expression.Value)
		return value, err == nil
	case *ast.ParenExpr:
		return rebuildArchitectureEvaluateString(expression.X)
	case *ast.BinaryExpr:
		if expression.Op != token.ADD {
			return "", false
		}
		left, leftOK := rebuildArchitectureEvaluateString(expression.X)
		right, rightOK := rebuildArchitectureEvaluateString(expression.Y)
		return left + right, leftOK && rightOK
	default:
		return "", false
	}
}

func rebuildArchitectureEvaluateInteger(expression ast.Expr) (int64, bool) {
	switch expression := expression.(type) {
	case *ast.BasicLit:
		if expression.Kind != token.INT {
			return 0, false
		}
		value, err := strconv.ParseInt(expression.Value, 0, 64)
		return value, err == nil
	case *ast.ParenExpr:
		return rebuildArchitectureEvaluateInteger(expression.X)
	case *ast.UnaryExpr:
		value, ok := rebuildArchitectureEvaluateInteger(expression.X)
		if !ok {
			return 0, false
		}
		switch expression.Op {
		case token.ADD:
			return value, true
		case token.SUB:
			return -value, true
		default:
			return 0, false
		}
	case *ast.BinaryExpr:
		left, leftOK := rebuildArchitectureEvaluateInteger(expression.X)
		right, rightOK := rebuildArchitectureEvaluateInteger(expression.Y)
		if !leftOK || !rightOK {
			return 0, false
		}
		switch expression.Op {
		case token.ADD:
			return left + right, true
		case token.SUB:
			return left - right, true
		case token.MUL:
			return left * right, true
		case token.QUO:
			if right != 0 {
				return left / right, true
			}
		case token.REM:
			if right != 0 {
				return left % right, true
			}
		case token.SHL:
			if right >= 0 && right < 64 {
				return left << uint(right), true
			}
		case token.SHR:
			if right >= 0 && right < 64 {
				return left >> uint(right), true
			}
		}
		return 0, false
	default:
		return 0, false
	}
}

func rebuildArchitectureEvaluateDuration(expression ast.Expr) (time.Duration, bool) {
	if value, ok := rebuildArchitectureEvaluateString(expression); ok {
		duration, err := time.ParseDuration(value)
		return duration, err == nil
	}
	switch expression := expression.(type) {
	case *ast.BasicLit:
		value, ok := rebuildArchitectureEvaluateInteger(expression)
		return time.Duration(value), ok
	case *ast.ParenExpr:
		return rebuildArchitectureEvaluateDuration(expression.X)
	case *ast.UnaryExpr:
		value, ok := rebuildArchitectureEvaluateDuration(expression.X)
		if !ok {
			return 0, false
		}
		switch expression.Op {
		case token.ADD:
			return value, true
		case token.SUB:
			return -value, true
		default:
			return 0, false
		}
	case *ast.SelectorExpr:
		switch expression.Sel.Name {
		case "Nanosecond":
			return time.Nanosecond, true
		case "Microsecond":
			return time.Microsecond, true
		case "Millisecond":
			return time.Millisecond, true
		case "Second":
			return time.Second, true
		case "Minute":
			return time.Minute, true
		case "Hour":
			return time.Hour, true
		default:
			return 0, false
		}
	case *ast.BinaryExpr:
		switch expression.Op {
		case token.ADD, token.SUB:
			left, leftOK := rebuildArchitectureEvaluateDuration(expression.X)
			right, rightOK := rebuildArchitectureEvaluateDuration(expression.Y)
			if !leftOK || !rightOK {
				return 0, false
			}
			if expression.Op == token.ADD {
				return left + right, true
			}
			return left - right, true
		case token.MUL:
			if scalar, scalarOK := rebuildArchitectureEvaluateInteger(expression.X); scalarOK {
				if duration, durationOK := rebuildArchitectureEvaluateDuration(expression.Y); durationOK {
					return time.Duration(scalar) * duration, true
				}
			}
			if duration, durationOK := rebuildArchitectureEvaluateDuration(expression.X); durationOK {
				if scalar, scalarOK := rebuildArchitectureEvaluateInteger(expression.Y); scalarOK {
					return duration * time.Duration(scalar), true
				}
			}
		case token.QUO:
			duration, durationOK := rebuildArchitectureEvaluateDuration(expression.X)
			scalar, scalarOK := rebuildArchitectureEvaluateInteger(expression.Y)
			if durationOK && scalarOK && scalar != 0 {
				return duration / time.Duration(scalar), true
			}
		}
		return 0, false
	default:
		return 0, false
	}
}

func rebuildArchitectureEvaluateStringList(expression ast.Expr) ([]string, bool) {
	composite, ok := expression.(*ast.CompositeLit)
	if !ok {
		if parenthesized, ok := expression.(*ast.ParenExpr); ok {
			return rebuildArchitectureEvaluateStringList(parenthesized.X)
		}
		return nil, false
	}
	result := make([]string, 0, len(composite.Elts))
	for _, element := range composite.Elts {
		if keyed, ok := element.(*ast.KeyValueExpr); ok {
			element = keyed.Value
		}
		expression, ok := element.(ast.Expr)
		if !ok {
			return nil, false
		}
		value, ok := rebuildArchitectureEvaluateString(expression)
		if !ok {
			return nil, false
		}
		result = append(result, value)
	}
	return result, true
}

func rebuildArchitectureAssertConfigGuardMutants(t *testing.T) {
	t.Helper()
	policy, err := rebuildArchitectureConfigPolicyFromJSON([]byte(`{
		"aliases":[{"name":"server.bind"}],
		"keys":[
			{"key":"server.listen","go_name":"ServerListen","type":"string","default":"127.0.0.1:8080","env_alias":"ORQUESTA_SERVER_LISTEN"},
			{"key":"api.locale","go_name":"APILocale","type":"string","default":"es","env_alias":"ORQUESTA_API_LOCALE"},
			{"key":"runtime.codex.reasoning","go_name":"RuntimeCodexReasoning","type":"string","default":"medium","env_alias":"ORQUESTA_RUNTIME_CODEX_REASONING"},
			{"key":"runtime.codex.env_allowlist","go_name":"RuntimeCodexEnvAllowlist","type":"string_list","default":["PATH"],"env_alias":"ORQUESTA_RUNTIME_CODEX_ENV_ALLOWLIST"}
		]
	}`))
	if err != nil {
		t.Fatal(err)
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
