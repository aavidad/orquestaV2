package acceptance_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/text/language"
)

func TestV21BCP47MatrixAndSpanishFallbackContract(t *testing.T) {
	fixture := evidenceDecodeStrictJSON[v21Fixture](t,
		filepath.Join(evidenceRepositoryRoot(t), v21FixturePath))
	for _, raw := range append(append(append([]string{}, fixture.BCP47Matrix.Direct...),
		fixture.BCP47Matrix.Regional...), fixture.BCP47Matrix.UnsupportedFallback...) {
		if !v21ValidBCP47Input(raw) {
			t.Errorf("V21 valid BCP-47 example %q was rejected", raw)
		}
	}
	for _, raw := range fixture.BCP47Matrix.Invalid {
		if v21ValidBCP47Input(raw) {
			t.Errorf("V21 invalid BCP-47 example %q was accepted", raw)
		}
	}
	if fixture.FallbackLocale != "es" ||
		!reflect.DeepEqual(fixture.BCP47Matrix.UnsupportedFallback, []string{"ca-ES-valencia", "zh-Hans"}) {
		t.Fatalf("V21 explicit fallback contract drift: %+v", fixture.BCP47Matrix)
	}
}

func v21ValidBCP47Input(raw string) bool {
	if strings.TrimSpace(raw) != raw || strings.ContainsRune(raw, '_') {
		return false
	}
	_, err := language.Parse(raw)
	return err == nil
}

func TestV21RejectsDuplicateJSONKeysAndPlaceholderDriftInContractData(t *testing.T) {
	if err := v21RejectDuplicateJSONKeys([]byte(`{"a":"uno","a":"dos"}`)); err == nil {
		t.Fatal("V21 duplicate JSON key passed")
	}
	if err := v21RejectDuplicateJSONKeys([]byte(`{"a":{"b":"uno"},"c":["dos"]}`)); err != nil {
		t.Fatalf("V21 valid nested JSON failed: %v", err)
	}
	if reflect.DeepEqual(v21Placeholders("Hola {name}"), v21Placeholders("Hello {user}")) {
		t.Fatal("V21 placeholder drift was not detected")
	}
	values, err := v21DecodeCatalog([]byte(`{"count":{"one":"{count} item","other":"{count} items"},"title":"Title"}`))
	if err != nil || values["count"].Kind != "plural" || values["title"].Kind != "text" {
		t.Fatalf("V21 string/plural catalog decoding failed: values=%+v err=%v", values, err)
	}
	if err := v21ValidateCatalogMessage(v21CatalogMessage{Kind: "plural", Forms: map[string]string{"one": "one"}}); err == nil {
		t.Fatal("V21 plural without other form passed")
	}
	reference := map[string]v21CatalogMessage{
		"count": {Kind: "plural", Forms: map[string]string{"one": "{count} item", "other": "{count} items"}},
	}
	if err := v21CatalogParityError(reference, map[string]v21CatalogMessage{
		"count": {Kind: "text", Text: "{count} items"},
	}); err == nil {
		t.Fatal("V21 text/plural kind drift passed")
	}
	if err := v21CatalogParityError(reference, map[string]v21CatalogMessage{
		"count": {Kind: "plural", Forms: map[string]string{"other": "{count} items"}},
	}); err == nil {
		t.Fatal("V21 plural form drift passed")
	}
	if err := v21CatalogParityError(reference, map[string]v21CatalogMessage{
		"count": {Kind: "plural", Forms: map[string]string{"one": "{items} item", "other": "{count} items"}},
	}); err == nil {
		t.Fatal("V21 plural placeholder drift passed")
	}
}

func TestV21MachineProtocolFieldsRemainLocaleInvariant(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v21Fixture](t, filepath.Join(root, v21FixturePath))
	wantFields := []string{
		"audit_ref", "command_id", "command_version", "digest", "error.code",
		"error.message_key", "project_ref", "request_ref",
	}
	if !reflect.DeepEqual(fixture.MachineInvariantFields, wantFields) {
		t.Fatalf("V21 machine invariant fields=%v want=%v", fixture.MachineInvariantFields, wantFields)
	}
	content, err := os.ReadFile(filepath.Join(root, "internal", "commands", "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	var registry struct {
		Commands []struct {
			ID         string   `json:"id"`
			Version    string   `json:"version"`
			ErrorCodes []string `json:"error_codes"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(content, &registry); err != nil || len(registry.Commands) != 25 {
		t.Fatalf("V21 consumes invalid V20 registry: commands=%d err=%v", len(registry.Commands), err)
	}
	for _, command := range registry.Commands {
		if !strings.HasPrefix(command.ID, "orquesta.") || command.Version != "1" || len(command.ErrorCodes) == 0 {
			t.Errorf("V21 cannot localize unstable command protocol: %+v", command)
		}
	}
}

func TestV21ScopeDoesNotClaimFutureProductSurfaces(t *testing.T) {
	fixture := evidenceDecodeStrictJSON[v21Fixture](t,
		filepath.Join(evidenceRepositoryRoot(t), v21FixturePath))
	assertV21ExactSet(t, "current public scope", fixture.ActiveSurfaceIDs,
		[]string{"cli", "command_registry", "http", "mcp", "public_docs"})
	assertV21ExactSet(t, "future consumer surfaces", fixture.FutureSurfaceIDs,
		[]string{"notifications", "prompts", "web", "wizard"})
	for _, statement := range fixture.DeferredOwnership {
		if strings.TrimSpace(statement) == "" {
			t.Fatal("V21 empty deferred ownership boundary")
		}
	}
	if len(fixture.DeferredOwnership) != 3 {
		t.Fatalf("V21 deferred ownership=%v", fixture.DeferredOwnership)
	}
}

func TestTypedExtractorFindsRegistryCatalogAndPublicDocsKeys(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v21Fixture](t, filepath.Join(root, v21FixturePath))
	manifest := evidenceDecodeStrictJSON[v21Manifest](t,
		filepath.Join(root, filepath.FromSlash(fixture.ManifestPath)))
	claims := v21ManifestClaimedCatalogKeys(manifest)
	actual := v21ActualTypedPublicKeys(t, root)
	if !reflect.DeepEqual(v21SortedKeys(claims), v21SortedKeys(actual)) {
		t.Fatalf("V21 typed extraction drift: claims=%v actual=%v",
			v21SortedKeys(claims), v21SortedKeys(actual))
	}
	if len(manifest.PublicDocuments) != len(fixture.PublicDocumentPairs) {
		t.Fatalf("V21 public document manifest count=%d want=%d",
			len(manifest.PublicDocuments), len(fixture.PublicDocumentPairs))
	}
	for _, document := range manifest.PublicDocuments {
		for locale, relative := range document.Locales {
			if locale != "es" && locale != "en" {
				t.Errorf("V21 public document %q has undeclared locale %q", document.ID, locale)
			}
			v21AssertSafeExistingPath(t, root, relative)
		}
	}
}

func TestV21ExactGateRunsFormattingAndRealBindings(t *testing.T) {
	fixture := evidenceDecodeStrictJSON[v21Fixture](t,
		filepath.Join(evidenceRepositoryRoot(t), v21FixturePath))
	for _, marker := range []string{
		"./scripts/check_rebuild_write_set.sh " + v21ContractBaseGitCommitOID,
		"git diff --check " + v21ContractBaseGitCommitOID + " HEAD --",
		"./acceptance", "./internal/i18n", "./internal/interfaces/httpapi",
		"./internal/interfaces/mcp", "./internal/interfaces/cli", "./internal/bootstrap",
		"./cmd/orquesta", "./sdk/commands", "-race",
		"TestCatalogConcurrentReadsAreRaceFree",
		"TestRealHTTPMCPCLIAndI18NParityEndToEnd",
		"\\\"Action\\\":\\\"run\\\"", "\\\"Action\\\":\\\"pass\\\"",
		"GOFLAGS=-mod=vendor go vet",
	} {
		if !strings.Contains(fixture.Command, marker) {
			t.Errorf("V21 gate lacks %q", marker)
		}
	}
	if strings.Contains(fixture.Command, " ./...") {
		t.Fatalf("V21 gate opens the legacy/global package universe: %q", fixture.Command)
	}
}
