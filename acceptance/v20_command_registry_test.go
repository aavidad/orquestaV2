package acceptance_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	commandcore "orquesta/internal/commands"
)

const v20FixturePath = "acceptance/fixtures/v20_command_registry.json"
const v20ContractBaseGitCommitOID = "42692512b0048f116d660a53fbe6650e243bb4e3"

func TestV20PreflightContractIsStructurallyValid(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v20Fixture](t, filepath.Join(repositoryRoot, v20FixturePath))
	v20AssertPreflightFixture(t, repositoryRoot, fixture)
	v20AssertRoadmapBoundary(t, repositoryRoot, fixture)
}

func TestAcceptanceV20CommandRegistry(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v20Fixture](t, filepath.Join(repositoryRoot, v20FixturePath))
	v20AssertPreflightFixture(t, repositoryRoot, fixture)
	v20AssertRoadmapBoundary(t, repositoryRoot, fixture)

	registryPath := filepath.Join(repositoryRoot, filepath.FromSlash(fixture.CanonicalRegistryPath))
	if _, err := os.Stat(registryPath); err != nil {
		if os.IsNotExist(err) {
			t.Fatal("V20_RED integrated command registry product is absent on the accredited V19 base")
		}
		t.Fatal(err)
	}

	v20AssertRegistryAndHandlers(t, repositoryRoot, fixture)
	v20AssertGeneratedBindings(t, repositoryRoot, fixture)
	v20AssertInterfacesContainNoLifecycle(t, repositoryRoot, fixture)
	v20AssertMigrationAndBehaviorGates(t, repositoryRoot, fixture)
}

func TestV20PrivateCommandRegistryBehavior(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v20Fixture](t, filepath.Join(repositoryRoot, v20FixturePath))
	v20AssertPreflightFixture(t, repositoryRoot, fixture)
	v20AssertRegistryAndHandlers(t, repositoryRoot, fixture)
	v20AssertGeneratedBindings(t, repositoryRoot, fixture)
	v20AssertPrivateInterfacesContainNoLifecycle(t, repositoryRoot, fixture)
}

func v20AssertPreflightFixture(t *testing.T, repositoryRoot string, fixture v20Fixture) {
	t.Helper()
	wantPreflightStatus := "integrated_implementation_in_progress"
	wantCatalogStatus := "integrated_catalog_in_progress"
	if fixture.ImplementationStatus != "development_unsealed" {
		wantPreflightStatus = "integrated_implementation_complete"
		wantCatalogStatus = "integrated_catalog_complete"
	}
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V20-COMMAND-REGISTRY" ||
		fixture.TrustedBaseGitCommitOID != v20ContractBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v20ContractBaseGitCommitOID ||
		fixture.PreflightStatus != wantPreflightStatus || fixture.PreflightNote == "" ||
		fixture.BlockingDependency != "none" ||
		fixture.PendingDependencyReceipt != "product/evidence/v19_council.json" ||
		fixture.Command == "" || len(fixture.ExecutionArgv) != 3 ||
		fixture.ExecutionArgv[0] != "sh" || fixture.ExecutionArgv[1] != "-c" ||
		fixture.Command != "sh -c '"+fixture.ExecutionArgv[2]+"'" ||
		fixture.OutputPath != "product/evidence/v20_command_registry.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v20_command_registry.json" ||
		fixture.SealNote == "" || fixture.PSEContract.P == "" ||
		fixture.PSEContract.S == "" || fixture.PSEContract.E == "" {
		t.Fatalf("invalid V20 preflight identity: %+v", fixture)
	}
	if fixture.DependencyActivation != (v20DependencyGate{
		Contract: "AC-V19-COUNCIL", Result: "PASS", SourceWorktreeState: "detached_clean",
		ProductOID: "cccfb4a64bd2cfb788a2d604ccd8d008e946e8a3", SealedOID: "f7a574e36528871ee07a5529a88be1f0a88511d7",
		EvidenceOID: "42692512b0048f116d660a53fbe6650e243bb4e3", IntegrationHeadOID: v20ContractBaseGitCommitOID,
	}) || fixture.CommandCatalogStatus != wantCatalogStatus ||
		fixture.RuntimeRegistryMode != "compiled_generated_only" || len(fixture.ConfigurationKeysAdded) != 0 ||
		!reflect.DeepEqual(fixture.BundledLocales, []string{"es", "en"}) ||
		fixture.I18NScope != "keys_only_v21_owns_formatting" {
		t.Fatalf("invalid V20 dependency/config/i18n boundary: %+v", fixture)
	}
	if !reflect.DeepEqual(fixture.OwnedCapabilityIDs, []string{"GOV-17", "UI-02"}) ||
		!reflect.DeepEqual(fixture.DependencyVerticals, []string{"config", "identity_projects_rbac", "test_attestor", "independent_reviews", "council"}) {
		t.Fatalf("invalid V20 ownership or dependency set: %+v", fixture)
	}
	if len(fixture.Commands) != 25 || len(fixture.RequiredApplicationTypes) != 5 ||
		len(fixture.GeneratedFiles) != 1 || len(fixture.RequiredBehaviorTests) != 34 ||
		len(fixture.StableErrorCodes) != 7 || len(fixture.RequiredDefinitionFields) != 15 {
		t.Fatalf("invalid V20 coverage counts: %+v", fixture)
	}
	if !reflect.DeepEqual(fixture.RequiredDefinitionFields, []string{
		"id", "version", "kind", "handler", "permission", "audience", "execution_bound", "replay_mode",
		"input_schema", "output_schema", "description_key", "error_codes", "http", "mcp", "cli",
	}) || !reflect.DeepEqual(fixture.StableErrorCodes, []string{
		"invalid_request", "unauthenticated", "forbidden", "not_found", "conflict", "unavailable", "internal",
	}) {
		t.Fatalf("invalid V20 registry/error vocabulary: %+v", fixture)
	}
	if fixture.SimplicityBudget != (v20SimplicityBudget{
		ManualProductLOC: 6400, CommandsApplicationLOC: 2600, BindingAdapterLOC: 2000,
		SDKLOC: 500, SQLiteRecoveryBootstrapLOC: 1300, GeneratedLOC: 3000,
		MigrationSQLLOC: 500, MaximumCommandDefinitions: 25,
		MaximumLifecycleWriters: 1, MaximumCommandRegistries: 1,
	}) {
		t.Fatalf("invalid V20 simplicity budget: %+v", fixture.SimplicityBudget)
	}
	seen := make(map[string]struct{}, len(fixture.Commands))
	seenHandlers := make(map[string]struct{}, len(fixture.Commands))
	for _, command := range fixture.Commands {
		wantReplay := "application_receipt"
		if command.Kind == "query" {
			wantReplay = "read_reexecute"
		}
		if !strings.HasPrefix(command.ID, "orquesta.") || command.Handler == "" || command.Permission == "" ||
			(command.Kind != "command" && command.Kind != "query") ||
			(command.Audience != "principal" && command.Audience != "execution") ||
			command.ExecutionBound != (command.Audience == "execution") || command.ReplayMode != wantReplay {
			t.Fatalf("invalid V20 command contract: %+v", command)
		}
		if _, duplicate := seen[command.ID]; duplicate {
			t.Fatalf("duplicate V20 command %q", command.ID)
		}
		if _, duplicate := seenHandlers[command.Handler]; duplicate {
			t.Fatalf("duplicate V20 application handler %q", command.Handler)
		}
		seen[command.ID] = struct{}{}
		seenHandlers[command.Handler] = struct{}{}
	}
	seenTests := make(map[string]struct{}, len(fixture.RequiredBehaviorTests))
	implementedTests := v20TestFunctionNames(t, repositoryRoot)
	for _, name := range fixture.RequiredBehaviorTests {
		if !strings.HasPrefix(name, "Test") || strings.TrimSpace(name) != name {
			t.Fatalf("invalid V20 behavior gate %q", name)
		}
		if _, duplicate := seenTests[name]; duplicate {
			t.Fatalf("duplicate V20 behavior gate %q", name)
		}
		if _, implemented := implementedTests[name]; !implemented {
			t.Fatalf("V20 behavior gate %q is declared but has no test implementation", name)
		}
		seenTests[name] = struct{}{}
	}
	if _, err := evidenceGitCanonicalCommit(repositoryRoot, fixture.TrustedBaseGitCommitOID); err != nil {
		t.Fatalf("invalid V20 trusted base: %v", err)
	}
	evidenceAssertReceiptV3(t, repositoryRoot, evidenceReceiptV3Expectation{Contract: "AC-V19-COUNCIL", FixturePath: "acceptance/" + v19FixturePath, ReceiptPath: fixture.PendingDependencyReceipt, ExecutedNotBefore: "2026-07-23T00:00:00Z", TrustedBaseGitCommitOID: "6f244a7594141c74dc28e095e0e5e32a05102826"})
}

func v20AssertRoadmapBoundary(t *testing.T, repositoryRoot string, fixture v20Fixture) {
	t.Helper()
	receiptPresent := v20ExecutionEvidencePresent(t, repositoryRoot, fixture)
	content, err := os.ReadFile(filepath.Join(repositoryRoot, "product", "roadmap.json"))
	if err != nil {
		t.Fatal(err)
	}
	var roadmap struct {
		AcceptanceContracts []struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			TestRef string `json:"test_ref"`
			Command string `json:"command"`
			Fixture string `json:"fixture"`
			Receipt string `json:"receipt"`
		} `json:"acceptance_contracts"`
		CapabilityEntries []struct {
			ID           string   `json:"id"`
			OwnerContext string   `json:"owner_context"`
			Status       string   `json:"status"`
			EvidenceRefs []string `json:"evidence_refs"`
		} `json:"capability_entries"`
	}
	if err := json.Unmarshal(content, &roadmap); err != nil {
		t.Fatal(err)
	}
	foundContract := false
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V20-COMMAND-REGISTRY" {
			continue
		}
		foundContract = true
		if !receiptPresent {
			if contract.Status != "planned" ||
				contract.Command != "planned:go test -mod=vendor -count=1 . ./internal/... ./cmd/orquesta -run '^TestAcceptance$'" ||
				contract.TestRef != "planned:acceptance/v20_command_registry_test.go" ||
				contract.Fixture != "planned:fixtures/v20_command_registry" || contract.Receipt != "" {
				t.Fatalf("invalid planned V20 roadmap boundary: %+v", contract)
			}
		} else {
			if contract.Status != "executable" ||
				contract.TestRef != "acceptance/v20_command_registry_test.go" ||
				contract.Command != fixture.Command ||
				contract.Fixture != v20FixturePath || contract.Receipt != fixture.ReceiptPath {
				t.Fatalf("invalid executable V20 roadmap boundary: %+v", contract)
			}
		}
	}
	if !foundContract {
		t.Fatal("AC-V20-COMMAND-REGISTRY missing")
	}
	owned := map[string]bool{"GOV-17": true, "UI-02": true}
	wantEvidence := []string{"acceptance/v20_command_registry_test.go", v20FixturePath, fixture.ReceiptPath}
	count := 0
	for _, capability := range roadmap.CapabilityEntries {
		if !owned[capability.ID] {
			continue
		}
		count++
		if capability.OwnerContext != "command_registry" {
			t.Fatalf("invalid V20 capability boundary: %+v", capability)
		}
		if !receiptPresent && (capability.Status != "declared" || len(capability.EvidenceRefs) != 0) {
			t.Fatalf("V20 capability has premature evidence: %+v", capability)
		}
		if receiptPresent && (capability.Status != "accredited" ||
			!reflect.DeepEqual(capability.EvidenceRefs, wantEvidence)) {
			t.Fatalf("V20 capability lacks exact E evidence: %+v", capability)
		}
	}
	if count != len(owned) {
		t.Fatalf("V20 owned capability count=%d, want %d", count, len(owned))
	}
}

func v20AssertRegistryAndHandlers(t *testing.T, repositoryRoot string, fixture v20Fixture) {
	t.Helper()
	registryPath := filepath.Join(repositoryRoot, filepath.FromSlash(fixture.CanonicalRegistryPath))
	content, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var registry v20RegistryDocument
	if err := decoder.Decode(&registry); err != nil {
		t.Fatalf("invalid V20 registry: %v", err)
	}
	if registry.SchemaVersion != 1 || strings.TrimSpace(registry.Revision) == "" ||
		len(registry.Commands) < len(fixture.Commands) {
		t.Fatalf("invalid V20 registry identity: %+v", registry)
	}
	want := make(map[string]v20Command, len(fixture.Commands))
	for _, command := range fixture.Commands {
		want[command.ID] = command
	}
	for _, definition := range registry.Commands {
		contract, ok := want[definition.ID]
		if !ok {
			continue
		}
		delete(want, definition.ID)
		wantDescription := "command." + strings.TrimPrefix(definition.ID, "orquesta.") + ".description"
		wantCLI := strings.Split(strings.TrimPrefix(definition.ID, "orquesta."), ".")
		if definition.Version != "1" || definition.Kind != contract.Kind || definition.Handler != contract.Handler ||
			definition.Permission != contract.Permission || definition.Audience != contract.Audience ||
			definition.ExecutionBound != contract.ExecutionBound || definition.ReplayMode != contract.ReplayMode ||
			definition.DescriptionKey != wantDescription ||
			!reflect.DeepEqual(definition.ErrorCodes, fixture.StableErrorCodes) ||
			definition.HTTP.Method != "POST" ||
			definition.HTTP.Path != "/api/v1/commands/"+definition.ID ||
			definition.MCP.Tool != definition.ID || !v20MCPAnnotationsAreCanonical(definition) ||
			!reflect.DeepEqual(definition.CLI.Path, wantCLI) ||
			len(definition.InputSchema) == 0 || len(definition.OutputSchema) == 0 ||
			!json.Valid(definition.InputSchema) || !json.Valid(definition.OutputSchema) {
			t.Errorf("V20 definition differs from canonical contract: %+v", definition)
		}
		v20AssertClosedSchema(t, definition.ID+" input", definition.InputSchema)
		v20AssertClosedSchema(t, definition.ID+" output", definition.OutputSchema)
		v20AssertRBACPermission(t, definition.Permission)
	}
	if len(want) != 0 {
		missing := make([]string, 0, len(want))
		for id := range want {
			missing = append(missing, id)
		}
		sort.Strings(missing)
		t.Errorf("V20 registry lacks commands: %v", missing)
	}
	compiled := commandcore.CanonicalDefinitions()
	if len(compiled) != len(registry.Commands) {
		t.Fatalf("compiled definitions=%d registry=%d", len(compiled), len(registry.Commands))
	}
	for index, definition := range registry.Commands {
		got := compiled[index]
		if got.ID != definition.ID || got.Handler != definition.Handler || got.Permission != definition.Permission ||
			!v20JSONEqual(got.InputSchema, definition.InputSchema) || !v20JSONEqual(got.OutputSchema, definition.OutputSchema) {
			t.Errorf("compiled definition %d drift: %+v", index, got)
		}
	}

	commandsDirectory := filepath.Join(repositoryRoot, "internal", "commands")
	for _, required := range fixture.RequiredApplicationTypes {
		fields, found := v20ProductionStructFields(t, commandsDirectory, required.Name)
		if !found {
			t.Errorf("V20 application-side type %s missing", required.Name)
			continue
		}
		for _, field := range required.Fields {
			if !fields[field] {
				t.Errorf("V20 type %s lacks %s", required.Name, field)
			}
		}
	}
	methods := v20ProductionMethods(t, commandsDirectory, "Dispatcher")
	for _, method := range fixture.RequiredDispatcherMethods {
		if !methods[method] {
			t.Errorf("V20 Dispatcher lacks %s", method)
		}
	}
	functions := v20ProductionFunctions(t, filepath.Join(repositoryRoot, filepath.FromSlash(fixture.ApplicationHandlerPath)))
	for _, command := range fixture.Commands {
		if !functions["handle"+command.Handler] {
			t.Errorf("V20 application handler handle%s missing", command.Handler)
		}
	}
	production := v20ReadProductionGo(t, commandsDirectory)
	for _, forbidden := range fixture.ForbiddenPrivateAuthorities {
		if strings.Contains(production, "type "+forbidden+" ") {
			t.Errorf("V20 private authority %s is forbidden", forbidden)
		}
	}
}

func v20MCPAnnotationsAreCanonical(definition v20RegistryDefinition) bool {
	annotations := definition.MCP.Annotations
	if annotations == nil || annotations.ReadOnly == nil || annotations.Destructive == nil ||
		annotations.Idempotent == nil || annotations.OpenWorld == nil {
		return false
	}
	wantReadOnly := definition.Kind == "query"
	wantDestructive := !wantReadOnly
	switch definition.ID {
	case "orquesta.mailbox.claim",
		"orquesta.mailbox.mark_delivered",
		"orquesta.mailbox.consume",
		"orquesta.mailbox.acknowledge":
		// These mailbox transitions are monotonic, not destructive effects.
		wantDestructive = false
	}
	return *annotations.ReadOnly == wantReadOnly &&
		*annotations.Destructive == wantDestructive &&
		*annotations.Idempotent &&
		!*annotations.OpenWorld
}

func v20AssertGeneratedBindings(t *testing.T, repositoryRoot string, fixture v20Fixture) {
	t.Helper()
	for _, relative := range fixture.GeneratedFiles {
		content, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(relative)))
		if err != nil {
			t.Errorf("V20 generated binding %s missing: %v", relative, err)
			continue
		}
		if !bytes.Contains(content, []byte("RegistrySourceSHA256")) || !bytes.Contains(content, []byte("Code generated")) {
			t.Errorf("V20 generated binding %s lacks source identity or generated marker", relative)
		}
	}
	if _, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(fixture.GeneratorPath))); err != nil {
		t.Errorf("V20 deterministic generator missing: %v", err)
	}
}

func v20AssertInterfacesContainNoLifecycle(t *testing.T, repositoryRoot string, fixture v20Fixture) {
	t.Helper()
	paths := append([]string(nil), v20CommandInterfaceFiles...)
	paths = append(paths, "sdk/commands")
	for _, relative := range paths {
		source := v20ReadProductionGo(t, filepath.Join(repositoryRoot, filepath.FromSlash(relative)))
		for _, forbidden := range fixture.ForbiddenInterfaceImports {
			if strings.Contains(source, `"`+forbidden+`"`) {
				t.Errorf("V20 interface %s imports forbidden application/lifecycle package %s", relative, forbidden)
			}
		}
		for _, forbidden := range fixture.ForbiddenInterfaceMarkers {
			if strings.Contains(source, forbidden) {
				t.Errorf("V20 interface %s contains lifecycle writer marker %q", relative, forbidden)
			}
		}
	}
}
