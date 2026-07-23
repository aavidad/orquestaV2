package acceptance_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var v21PlaceholderPattern = regexp.MustCompile(`\{[A-Za-z][A-Za-z0-9_.-]*\}`)

type v21CatalogMessage struct {
	Kind  string
	Text  string
	Forms map[string]string
}

var v21PluralForms = map[string]struct{}{
	"zero": {}, "one": {}, "two": {}, "few": {}, "many": {}, "other": {},
}

func v21ExecutionEvidencePresent(t *testing.T, root string, fixture v21Fixture) bool {
	t.Helper()
	present := make([]bool, 0, 2)
	for _, relative := range []string{fixture.ReceiptPath, fixture.OutputPath} {
		if err := evidenceValidateRepositoryPath(relative); err != nil {
			t.Fatal(err)
		}
		_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		present = append(present, err == nil)
	}
	if present[0] != present[1] {
		t.Fatal("V21 execution evidence must contain both receipt and output or neither")
	}
	return present[0]
}

func v21AssertSafeExistingPath(t *testing.T, root, relative string) {
	t.Helper()
	if err := evidenceValidateRepositoryPath(relative); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatalf("V21 source %q: %v", relative, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("V21 source %q must not be a symlink", relative)
	}
}

func v21AssertCatalogSources(t *testing.T, root string, fixture v21Fixture, manifest v21Manifest) {
	t.Helper()
	catalogs := make(map[string]map[string]v21CatalogMessage, len(manifest.Catalogs))
	for _, catalog := range manifest.Catalogs {
		if _, duplicate := catalogs[catalog.Locale]; duplicate {
			t.Fatalf("V21 duplicate catalog locale %q", catalog.Locale)
		}
		relative := filepath.Join(filepath.Dir(fixture.ManifestPath), filepath.FromSlash(catalog.Path))
		path := filepath.Join(root, relative)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := v21RejectDuplicateJSONKeys(content); err != nil {
			t.Fatalf("V21 catalog %s: %v", catalog.Locale, err)
		}
		values, err := v21DecodeCatalog(content)
		if err != nil {
			t.Fatalf("V21 catalog %s: %v", catalog.Locale, err)
		}
		if len(values) == 0 || len(values) > fixture.SimplicityBudget.MaximumCatalogKeys {
			t.Fatalf("V21 catalog %s key count=%d", catalog.Locale, len(values))
		}
		for key, message := range values {
			if strings.TrimSpace(key) != key || key == "" {
				t.Fatalf("V21 catalog %s has invalid key %q", catalog.Locale, key)
			}
			if err := v21ValidateCatalogMessage(message); err != nil {
				t.Fatalf("V21 catalog %s key %q: %v", catalog.Locale, key, err)
			}
		}
		catalogs[catalog.Locale] = values
	}
	reference, exists := catalogs[fixture.DefaultLocale]
	if !exists {
		t.Fatalf("V21 default catalog %q is absent", fixture.DefaultLocale)
	}
	for locale, values := range catalogs {
		if err := v21CatalogParityError(reference, values); err != nil {
			t.Fatalf("V21 locale %s: %v", locale, err)
		}
	}

	claims := v21ManifestClaimedCatalogKeys(manifest)
	used := v21ActualTypedPublicKeys(t, root)
	if !reflect.DeepEqual(v21SortedKeys(claims), v21SortedKeys(used)) {
		t.Errorf("V21 manifest claims and typed sources differ: claims=%v used=%v",
			v21SortedKeys(claims), v21SortedKeys(used))
	}
	for key := range reference {
		if _, ok := used[key]; !ok {
			t.Errorf("V21 unused catalog key %q", key)
		}
	}
	for key, source := range used {
		if _, ok := reference[key]; !ok {
			t.Errorf("V21 public key %q from %s is missing in catalogs", key, source)
		}
	}
	for _, document := range manifest.PublicDocuments {
		for _, relative := range []string{document.Locales["es"], document.Locales["en"]} {
			content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
			if err != nil {
				t.Fatal(err)
			}
			if len(bytes.TrimSpace(content)) == 0 {
				t.Errorf("V21 public document %q is empty", relative)
			}
		}
	}
}

func v21RejectDuplicateJSONKeys(content []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	if err := v21ConsumeJSONValue(decoder); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("trailing JSON: %v", err)
	}
	return nil
}

func v21ConsumeJSONValue(decoder *json.Decoder) error {
	tokenValue, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, composite := tokenValue.(json.Delim)
	if !composite {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			rawKey, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := rawKey.(string)
			if !ok {
				return fmt.Errorf("object key is not a string")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate JSON key %q", key)
			}
			seen[key] = struct{}{}
			if err := v21ConsumeJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return fmt.Errorf("invalid object terminator: %v", err)
		}
	case '[':
		for decoder.More() {
			if err := v21ConsumeJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return fmt.Errorf("invalid array terminator: %v", err)
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

func v21DecodeCatalog(content []byte) (map[string]v21CatalogMessage, error) {
	var rawValues map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&rawValues); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("trailing JSON: %v", err)
	}
	values := make(map[string]v21CatalogMessage, len(rawValues))
	for key, raw := range rawValues {
		var text string
		if err := json.Unmarshal(raw, &text); err == nil {
			values[key] = v21CatalogMessage{Kind: "text", Text: text}
			continue
		}
		var forms map[string]string
		if err := json.Unmarshal(raw, &forms); err != nil {
			return nil, fmt.Errorf("key %q must be string or plural forms: %w", key, err)
		}
		values[key] = v21CatalogMessage{Kind: "plural", Forms: forms}
	}
	return values, nil
}

func v21ValidateCatalogMessage(message v21CatalogMessage) error {
	switch message.Kind {
	case "text":
		if strings.TrimSpace(message.Text) == "" || len(message.Forms) != 0 {
			return fmt.Errorf("text message must contain non-empty text only")
		}
	case "plural":
		if len(message.Forms) == 0 || message.Text != "" {
			return fmt.Errorf("plural message must contain forms only")
		}
		if _, ok := message.Forms["other"]; !ok {
			return fmt.Errorf("plural message lacks other form")
		}
		for form, text := range message.Forms {
			if _, ok := v21PluralForms[form]; !ok || strings.TrimSpace(text) == "" {
				return fmt.Errorf("invalid plural form %q", form)
			}
		}
	default:
		return fmt.Errorf("unknown message kind %q", message.Kind)
	}
	return nil
}

func v21CatalogParityError(reference, candidate map[string]v21CatalogMessage) error {
	if !reflect.DeepEqual(v21SortedCatalogKeys(reference), v21SortedCatalogKeys(candidate)) {
		return fmt.Errorf("lacks exact catalog key parity")
	}
	for key, want := range reference {
		got := candidate[key]
		if got.Kind != want.Kind {
			return fmt.Errorf("message kind drift at %q: %q want %q", key, got.Kind, want.Kind)
		}
		switch want.Kind {
		case "text":
			if !reflect.DeepEqual(v21Placeholders(got.Text), v21Placeholders(want.Text)) {
				return fmt.Errorf("placeholder drift at %q", key)
			}
		case "plural":
			if !reflect.DeepEqual(v21SortedKeys(got.Forms), v21SortedKeys(want.Forms)) {
				return fmt.Errorf("plural form drift at %q", key)
			}
			for form, wantText := range want.Forms {
				if !reflect.DeepEqual(v21Placeholders(got.Forms[form]), v21Placeholders(wantText)) {
					return fmt.Errorf("placeholder drift at %q/%q", key, form)
				}
			}
		}
	}
	return nil
}

func v21SortedCatalogKeys(values map[string]v21CatalogMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func v21SortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func v21Placeholders(value string) []string {
	seen := map[string]struct{}{}
	for _, placeholder := range v21PlaceholderPattern.FindAllString(value, -1) {
		seen[placeholder] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for placeholder := range seen {
		result = append(result, placeholder)
	}
	sort.Strings(result)
	return result
}

func v21ManifestClaimedCatalogKeys(manifest v21Manifest) map[string]string {
	result := map[string]string{}
	for _, surface := range manifest.Surfaces {
		if surface.State != "active" {
			continue
		}
		for _, source := range surface.KeySources {
			if strings.HasPrefix(source, "catalog:") {
				result[strings.TrimPrefix(source, "catalog:")] = "manifest:" + surface.ID
			}
		}
	}
	return result
}

func v21ActualTypedPublicKeys(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	v21CollectRegistryKeys(t, root, filepath.Join(root, "internal", "commands", "registry.json"), result)
	for _, relative := range []string{
		"cmd/orquesta",
		"internal/bootstrap",
		"internal/interfaces/cli",
		"internal/interfaces/mcp",
		"internal/interfaces/httpapi",
	} {
		absolute := filepath.Join(root, filepath.FromSlash(relative))
		err := filepath.WalkDir(absolute, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			v21CollectGoKeys(t, root, path, result)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func v21CollectRegistryKeys(t *testing.T, root, path string, target map[string]string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var registry struct {
		Commands []struct {
			DescriptionKey string   `json:"description_key"`
			ErrorCodes     []string `json:"error_codes"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(content, &registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Commands) == 0 {
		t.Fatal("V21 command registry has no typed public keys")
	}
	source, _ := filepath.Rel(root, path)
	source = filepath.ToSlash(source)
	for _, command := range registry.Commands {
		target[command.DescriptionKey] = source
		for _, code := range command.ErrorCodes {
			target["error."+code] = source
		}
	}
}

func v21CollectGoKeys(t *testing.T, root, path string, target map[string]string) {
	t.Helper()
	tree, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	source, _ := filepath.Rel(root, path)
	source = filepath.ToSlash(source)
	constants := map[string]string{}
	for _, declaration := range tree.Decls {
		generic, ok := declaration.(*ast.GenDecl)
		if !ok || generic.Tok != token.CONST {
			continue
		}
		for _, specification := range generic.Specs {
			values, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for index, name := range values.Names {
				if index >= len(values.Values) {
					continue
				}
				literal, ok := values.Values[index].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(literal.Value)
				if err == nil {
					constants[name.Name] = value
				}
			}
		}
	}
	addExpression := func(expression ast.Expr) {
		switch value := expression.(type) {
		case *ast.BasicLit:
			if value.Kind != token.STRING {
				return
			}
			key, err := strconv.Unquote(value.Value)
			if err == nil && key != "" {
				target[key] = source
			}
		case *ast.Ident:
			if key := constants[value.Name]; key != "" {
				target[key] = source
			}
		}
	}
	ast.Inspect(tree, func(node ast.Node) bool {
		composite, ok := node.(*ast.CompositeLit)
		if ok {
			for _, element := range composite.Elts {
				pair, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				field, ok := pair.Key.(*ast.Ident)
				if ok && (field.Name == "messageKey" || field.Name == "MessageKey") {
					addExpression(pair.Value)
				}
			}
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		index := -1
		switch function := call.Fun.(type) {
		case *ast.SelectorExpr:
			if (function.Sel.Name == "Text" || function.Sel.Name == "Plural") &&
				v21CatalogReceiver(function.X) {
				index = 1
			}
		case *ast.Ident:
			if function.Name == "writeCatalogText" {
				index = 3
			}
		}
		if index < 0 || len(call.Args) <= index {
			return true
		}
		addExpression(call.Args[index])
		return true
	})
}

func v21CatalogReceiver(expression ast.Expr) bool {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name == "catalog"
	case *ast.SelectorExpr:
		return value.Sel.Name == "Catalog"
	default:
		return false
	}
}

func v21AssertRequiredCatalogAPI(t *testing.T, root string, fixture v21Fixture) {
	t.Helper()
	found := map[string]struct{}{}
	directory := filepath.Join(root, "internal", "i18n")
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		tree, err := parser.ParseFile(token.NewFileSet(), filepath.Join(directory, entry.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range tree.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			name := function.Name.Name
			if function.Recv != nil && len(function.Recv.List) == 1 {
				if v21ReceiverName(function.Recv.List[0].Type) == "Catalog" {
					name = "Catalog." + name
				}
			}
			found[name] = struct{}{}
		}
	}
	for _, required := range fixture.RequiredCatalogAPI {
		if _, ok := found[required]; !ok {
			t.Errorf("V21 required catalog API %q is absent", required)
		}
	}
}

func v21ReceiverName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return v21ReceiverName(value.X)
	default:
		return ""
	}
}

func v21AssertRequiredProductTests(t *testing.T, root string, fixture v21Fixture) {
	t.Helper()
	found := map[string]string{}
	for _, relative := range []string{"internal/i18n", "internal/interfaces", "internal/bootstrap"} {
		absolute := filepath.Join(root, filepath.FromSlash(relative))
		err := filepath.WalkDir(absolute, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
				return nil
			}
			tree, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			for _, declaration := range tree.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if ok && function.Recv == nil && strings.HasPrefix(function.Name.Name, "Test") {
					source, _ := filepath.Rel(root, path)
					found[function.Name.Name] = filepath.ToSlash(source)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, required := range fixture.RequiredBehaviorTests {
		if _, ok := found[required]; !ok {
			t.Errorf("V21 required product test %q is absent", required)
		}
	}
	if source := found["TestRealHTTPMCPCLIAndI18NParityEndToEnd"]; source != "internal/bootstrap/command_surfaces_i18n_e2e_test.go" {
		t.Errorf("V21 real binding E2E lives at %q", source)
	}
}

func v21CountPhysicalLines(t *testing.T, paths []string) int {
	t.Helper()
	total := 0
	for _, path := range paths {
		handle, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(handle)
		for scanner.Scan() {
			total++
		}
		closeErr := handle.Close()
		if err := scanner.Err(); err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	return total
}
