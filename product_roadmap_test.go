package orquesta_test

import (
	"encoding/json"
	"io"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type roadmapDocument struct {
	SchemaVersion       int                         `json:"schema_version"`
	Product             string                      `json:"product"`
	CatalogSize         int                         `json:"catalog_size"`
	SourceHashes        map[string]string           `json:"source_hashes"`
	StatusVocabulary    []string                    `json:"status_vocabulary"`
	DecisionVocabulary  []string                    `json:"decision_vocabulary"`
	ReleaseTargets      []string                    `json:"release_targets"`
	OperatorDecisions   map[string]json.RawMessage  `json:"operator_decisions"`
	Verticals           []roadmapVertical           `json:"verticals"`
	AcceptanceContracts []roadmapAcceptanceContract `json:"acceptance_contracts"`
	CapabilityEntries   []roadmapEntry              `json:"capability_entries"`
	DeferredMappings    []roadmapMapping            `json:"deferred_mappings"`
}

type roadmapVertical struct {
	ID                  string   `json:"id"`
	Title               string   `json:"title"`
	Sequence            int      `json:"sequence"`
	DependsOn           []string `json:"depends_on"`
	AcceptanceContracts []string `json:"acceptance_contracts"`
}

type roadmapAcceptanceContract struct {
	ID         string   `json:"id"`
	Vertical   string   `json:"vertical"`
	Status     string   `json:"status"`
	TestRef    string   `json:"test_ref"`
	Command    string   `json:"command"`
	Fixture    string   `json:"fixture"`
	Receipt    string   `json:"receipt,omitempty"`
	Assertions []string `json:"assertions"`
}

type roadmapEntry struct {
	ID                  string   `json:"id"`
	Title               string   `json:"title"`
	SourceProposal      string   `json:"source_proposal"`
	Decision            string   `json:"decision"`
	Kind                string   `json:"kind"`
	ReleaseTarget       string   `json:"release_target"`
	CutoverRequired     bool     `json:"cutover_required"`
	OwnerContext        string   `json:"owner_context"`
	Dependencies        []string `json:"dependencies"`
	AcceptanceContracts []string `json:"acceptance_contracts"`
	Status              string   `json:"status"`
	EvidenceRefs        []string `json:"evidence_refs"`
	Supersedes          []string `json:"supersedes"`
	AliasOf             string   `json:"alias_of,omitempty"`
}

type roadmapMapping struct {
	ID            string   `json:"id"`
	CapabilityIDs []string `json:"capability_ids"`
}

func TestProductRoadmapIsExhaustiveAndCausal(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	if roadmap.SchemaVersion != 2 || roadmap.Product != "orquesta" || roadmap.CatalogSize != 257 {
		t.Fatalf("invalid roadmap header: schema=%d product=%q catalog=%d", roadmap.SchemaVersion, roadmap.Product, roadmap.CatalogSize)
	}
	wantHashes := map[string]string{
		"catalog":          "sha256:351c2258562424fc5c0dab9e0dcce0b623f9eabf42f2485b6ae45432694d4fd3",
		"structural_route": "sha256:483d947532d81841fa9caf58a176400552d2ea5482e8f93be375c07e4ae62b73",
		"total_audit_cut":  "sha256:852568908c1754004f2e0acc04bf40e272e7b22a94192bedac04af6222423e99",
		"operator_order":   "sha256:252a84fdf5c68e61def6ae7d1e44d1039c1ca9297385af271d3fda7aed145d13",
	}
	if !reflect.DeepEqual(roadmap.SourceHashes, wantHashes) {
		t.Fatalf("source_hashes = %#v, want %#v", roadmap.SourceHashes, wantHashes)
	}
	if !reflect.DeepEqual(roadmap.StatusVocabulary, []string{"declared", "implemented", "wired", "exercised", "accredited"}) ||
		!reflect.DeepEqual(roadmap.DecisionVocabulary, []string{"accept", "reject", "conditional"}) ||
		!reflect.DeepEqual(roadmap.ReleaseTargets, []string{"total_v1", "conditional", "excluded"}) {
		t.Fatalf("invalid controlled vocabularies")
	}
	assertRoadmapOperatorDecisions(t, roadmap.OperatorDecisions)

	verticals := make(map[string]roadmapVertical, len(roadmap.Verticals))
	if len(roadmap.Verticals) != 34 {
		t.Fatalf("vertical count = %d, want 34", len(roadmap.Verticals))
	}
	for index, vertical := range roadmap.Verticals {
		if vertical.ID == "" || strings.TrimSpace(vertical.Title) == "" || vertical.Sequence != index+1 ||
			len(vertical.AcceptanceContracts) != 1 {
			t.Fatalf("incomplete or unordered vertical: %#v", vertical)
		}
		if _, duplicate := verticals[vertical.ID]; duplicate {
			t.Fatalf("duplicate vertical %q", vertical.ID)
		}
		verticals[vertical.ID] = vertical
	}
	for _, vertical := range roadmap.Verticals {
		for _, dependency := range vertical.DependsOn {
			dependencyVertical, exists := verticals[dependency]
			if !exists {
				t.Fatalf("vertical %q depends on missing %q", vertical.ID, dependency)
			}
			if dependencyVertical.Sequence >= vertical.Sequence {
				t.Fatalf("vertical %q has non-causal dependency %q", vertical.ID, dependency)
			}
		}
	}
	assertRoadmapAcyclic(t, "vertical", roadmapVerticalKeys(verticals), func(id string) []string {
		return verticals[id].DependsOn
	})

	contracts := make(map[string]roadmapAcceptanceContract, len(roadmap.AcceptanceContracts))
	if len(roadmap.AcceptanceContracts) != 34 {
		t.Fatalf("acceptance contract count = %d, want 34", len(roadmap.AcceptanceContracts))
	}
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID == "" || contract.Vertical == "" || contract.Command == "" || contract.Fixture == "" ||
			len(contract.Assertions) < 2 || !strings.Contains(contract.Command, "go test -mod=vendor") {
			t.Fatalf("incomplete acceptance contract: %#v", contract)
		}
		if _, duplicate := contracts[contract.ID]; duplicate {
			t.Fatalf("duplicate acceptance contract %q", contract.ID)
		}
		if _, exists := verticals[contract.Vertical]; !exists {
			t.Fatalf("contract %q has missing vertical %q", contract.ID, contract.Vertical)
		}
		switch contract.Status {
		case "executable":
			if strings.HasPrefix(contract.Command, "planned:") ||
				!strings.HasPrefix(contract.Receipt, "product/evidence/") || !strings.HasSuffix(contract.Receipt, ".json") {
				t.Fatalf("executable contract %q lacks runnable command or receipt", contract.ID)
			}
			requireRepositoryFile(t, ".", contract.TestRef)
			requireRepositoryFile(t, ".", contract.Fixture)
		case "planned":
			if !strings.HasPrefix(contract.TestRef, "planned:acceptance/") ||
				!strings.HasPrefix(contract.Fixture, "planned:fixtures/") ||
				!strings.HasPrefix(contract.Command, "planned:go test -mod=vendor") || contract.Receipt != "" {
				t.Fatalf("planned contract %q lacks explicit planned refs", contract.ID)
			}
		default:
			t.Fatalf("contract %q has invalid status %q", contract.ID, contract.Status)
		}
		contracts[contract.ID] = contract
	}
	for _, vertical := range roadmap.Verticals {
		ref := vertical.AcceptanceContracts[0]
		contract, exists := contracts[ref]
		if !exists || contract.Vertical != vertical.ID {
			t.Fatalf("vertical %q has invalid acceptance ref %q", vertical.ID, ref)
		}
	}

	wantFamilies := map[string][2]int{
		"GOV": {1, 22}, "WIZ": {1, 25}, "STG": {0, 20}, "ORC": {1, 29},
		"EVD": {1, 15}, "UI": {1, 18}, "AGT": {1, 12}, "EXT": {0, 22},
		"OPE": {1, 20}, "OPS": {1, 30}, "TLS": {1, 14}, "CTX": {1, 12}, "APP": {1, 16},
	}
	wantIDs := make(map[string]struct{}, 257)
	for family, bounds := range wantFamilies {
		for number := bounds[0]; number <= bounds[1]; number++ {
			wantIDs[family+"-"+roadmapTwoDigits(number)] = struct{}{}
		}
	}
	if len(wantIDs) != 257 || len(roadmap.CapabilityEntries) != 257 {
		t.Fatalf("capability count = %d, want %d", len(roadmap.CapabilityEntries), len(wantIDs))
	}

	statuses := roadmapSetOf(roadmap.StatusVocabulary...)
	kinds := roadmapSetOf(
		"governance", "intake", "stage_template", "orchestration", "evidence", "interface",
		"agent_protocol", "extension", "domain_composition", "operations", "tooling", "context",
		"generated_app_profile",
	)
	rejected := roadmapSetOf("WIZ-12", "WIZ-14", "ORC-18", "UI-09", "OPE-14")
	conditional := roadmapSetOf("CTX-09", "CTX-10", "CTX-11")
	entries := make(map[string]roadmapEntry, len(roadmap.CapabilityEntries))
	counts := map[string]int{}
	for _, entry := range roadmap.CapabilityEntries {
		if _, expected := wantIDs[entry.ID]; !expected {
			t.Fatalf("unexpected capability %q", entry.ID)
		}
		if _, duplicate := entries[entry.ID]; duplicate {
			t.Fatalf("duplicate capability %q", entry.ID)
		}
		vertical, verticalExists := verticals[entry.OwnerContext]
		_, kindValid := kinds[entry.Kind]
		_, statusValid := statuses[entry.Status]
		if strings.TrimSpace(entry.Title) == "" || strings.TrimSpace(entry.SourceProposal) == "" ||
			!kindValid || !statusValid || !verticalExists ||
			len(entry.AcceptanceContracts) != 1 || entry.AcceptanceContracts[0] != vertical.AcceptanceContracts[0] ||
			!reflect.DeepEqual(entry.Dependencies, vertical.DependsOn) || entry.EvidenceRefs == nil ||
			entry.Supersedes == nil {
			t.Fatalf("incomplete capability %q: %#v", entry.ID, entry)
		}
		switch entry.Decision {
		case "accept":
			if entry.ReleaseTarget != "total_v1" || !entry.CutoverRequired {
				t.Fatalf("accepted capability %q has inconsistent target/cutover", entry.ID)
			}
		case "reject":
			if entry.ReleaseTarget != "excluded" || entry.CutoverRequired {
				t.Fatalf("rejected capability %q has inconsistent target/cutover", entry.ID)
			}
		case "conditional":
			if entry.ReleaseTarget != "conditional" || entry.CutoverRequired {
				t.Fatalf("conditional capability %q has inconsistent target/cutover", entry.ID)
			}
		default:
			t.Fatalf("capability %q has invalid decision %q", entry.ID, entry.Decision)
		}
		assertRoadmapEvidenceCoherent(t, entry, contracts[entry.AcceptanceContracts[0]])
		counts[entry.Decision]++
		entries[entry.ID] = entry
	}
	if counts["accept"] != 249 || counts["reject"] != 5 || counts["conditional"] != 3 {
		t.Fatalf("decision counts = %#v, want accept=249 reject=5 conditional=3", counts)
	}
	for id := range wantIDs {
		if _, exists := entries[id]; !exists {
			t.Errorf("missing capability %q", id)
		}
	}
	for id := range rejected {
		if entries[id].Decision != "reject" {
			t.Errorf("capability %q decision = %q, want reject", id, entries[id].Decision)
		}
	}
	for id := range conditional {
		if entries[id].Decision != "conditional" {
			t.Errorf("capability %q decision = %q, want conditional", id, entries[id].Decision)
		}
	}
	for _, id := range []string{"GOV-20", "UI-17", "OPS-11"} {
		if entries[id].Decision != "accept" {
			t.Errorf("operator-required capability %q decision = %q, want accept", id, entries[id].Decision)
		}
	}

	wantAliases := map[string]string{"WIZ-14": "UI-09", "ORC-25": "OPS-18", "OPS-26": "OPS-01", "OPS-28": "OPS-06"}
	gotAliases := make(map[string]string)
	for _, entry := range roadmap.CapabilityEntries {
		if entry.AliasOf == "" {
			continue
		}
		if _, exists := entries[entry.AliasOf]; !exists {
			t.Fatalf("alias %q targets missing %q", entry.ID, entry.AliasOf)
		}
		gotAliases[entry.ID] = entry.AliasOf
	}
	if !reflect.DeepEqual(gotAliases, wantAliases) {
		t.Fatalf("aliases = %#v, want %#v", gotAliases, wantAliases)
	}
	assertRoadmapAcyclic(t, "alias", roadmapEntryKeys(entries), func(id string) []string {
		if entries[id].AliasOf == "" {
			return nil
		}
		return []string{entries[id].AliasOf}
	})
	assertRoadmapProgressCausality(t, roadmap.Verticals, verticals, contracts, entries)

	assertDeferredMappings(t, roadmap.DeferredMappings, entries)
}

func TestProductRoadmapAccreditationDoesNotExceedEvidence(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	entries := make(map[string]roadmapEntry, len(roadmap.CapabilityEntries))
	for _, entry := range roadmap.CapabilityEntries {
		entries[entry.ID] = entry
	}
	requiredAccreditedIDs := []string{
		"GOV-02", "GOV-03", "GOV-04", "GOV-16", "GOV-21",
		"ORC-01", "ORC-02", "ORC-06", "STG-00",
	}
	for _, id := range requiredAccreditedIDs {
		if entry := entries[id]; entry.Status != "accredited" || len(entry.EvidenceRefs) == 0 {
			t.Errorf("directly proven capability %s is not accredited with evidence: %#v", id, entry)
		}
	}
	wantDeferredOwners := map[string]string{
		"GOV-01": "generated_apps",
		"ORC-23": "operations_telemetry",
		"EXT-00": "domain_plugins",
	}
	for id, owner := range wantDeferredOwners {
		entry := entries[id]
		if entry.Status != "declared" || entry.OwnerContext != owner || len(entry.EvidenceRefs) != 0 {
			t.Errorf("partially evidenced capability %s is over-accredited or misrouted: %#v", id, entry)
		}
	}
}

func TestProductRoadmapV05ScopeAndExecutableContract(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)

	entries := make(map[string]roadmapEntry, len(roadmap.CapabilityEntries))
	var ownedAccepted []string
	for _, entry := range roadmap.CapabilityEntries {
		entries[entry.ID] = entry
		if entry.OwnerContext == "goal_dag_phases" && entry.Decision == "accept" {
			ownedAccepted = append(ownedAccepted, entry.ID)
		}
	}
	sort.Strings(ownedAccepted)
	wantAccepted := []string{
		"GOV-04",
		"ORC-01", "ORC-02", "ORC-06",
		"STG-00",
	}
	sort.Strings(wantAccepted)
	if !reflect.DeepEqual(ownedAccepted, wantAccepted) {
		t.Fatalf("V05 accepted ownership = %v, want exact %v", ownedAccepted, wantAccepted)
	}
	wantEvidence := []string{
		"acceptance/v05_goal_dag_phases_test.go",
		"acceptance/fixtures/v05_goal_dag_phases.json",
		"product/evidence/v05_goal_dag_phases.json",
	}
	for _, id := range wantAccepted {
		entry := entries[id]
		if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence) {
			t.Errorf("V05 capability %s lacks exact accreditation evidence: %#v", id, entry)
		}
	}

	wantMoved := map[string]string{
		"GOV-05": "atomic_state_outbox",
		"ORC-03": "controls",
		"ORC-04": "mailbox",
		"ORC-05": "mailbox",
		"ORC-12": "atomic_state_outbox",
		"ORC-13": "atomic_state_outbox",
		"ORC-17": "atomic_state_outbox",
		"STG-01": "wizard",
		"STG-02": "workspace_git",
		"STG-03": "wizard",
		"STG-04": "tools_skills_sdk",
		"STG-05": "context_rag_evals",
		"STG-06": "council",
		"STG-07": "wizard",
		"STG-09": "budgets_effects",
		"STG-10": "workspace_git",
		"STG-11": "codex_e2e",
		"STG-12": "tools_skills_sdk",
		"STG-16": "independent_reviews",
		"STG-20": "operations_telemetry",
	}
	for id, owner := range wantMoved {
		entry := entries[id]
		if entry.OwnerContext != owner || roadmapEvidenceContainsVertical(entry.EvidenceRefs, "v05_goal_dag_phases") {
			t.Errorf("moved capability %s = %#v, want owner %s without V05 evidence", id, entry, owner)
		}
	}
	if entry := entries["ORC-18"]; entry.Decision != "reject" || entry.ReleaseTarget != "excluded" || entry.CutoverRequired {
		t.Fatalf("ORC-18 must remain rejected and outside cutover: %#v", entry)
	}

	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV05ScopeAndExecutableContract|TestAcceptanceV05GoalDAGPhases|TestV05CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/interfaces/mcp ./internal/bootstrap'"
	var contract roadmapAcceptanceContract
	for _, candidate := range roadmap.AcceptanceContracts {
		if candidate.ID == "AC-V05-GOAL-DAG-PHASES" {
			contract = candidate
			break
		}
	}
	if contract.Status != "executable" || contract.TestRef != "acceptance/v05_goal_dag_phases_test.go" ||
		contract.Fixture != "acceptance/fixtures/v05_goal_dag_phases.json" ||
		contract.Receipt != "product/evidence/v05_goal_dag_phases.json" || contract.Command != wantCommand {
		t.Fatalf("invalid V05 executable contract: %#v", contract)
	}
	if !roadmapCommandHasArgument(contract.Command, "./acceptance") || strings.Contains(contract.Command, "^TestAcceptance$") {
		t.Fatalf("V05 command can omit real acceptance tests: %q", contract.Command)
	}
}

func TestProductRoadmapV06ScopeAndExecutableContract(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)

	entries := make(map[string]roadmapEntry, len(roadmap.CapabilityEntries))
	verticals := make(map[string]roadmapVertical, len(roadmap.Verticals))
	var ownedAccepted []string
	for _, vertical := range roadmap.Verticals {
		verticals[vertical.ID] = vertical
	}
	for _, entry := range roadmap.CapabilityEntries {
		entries[entry.ID] = entry
		if entry.OwnerContext == "atomic_state_outbox" && entry.Decision == "accept" {
			ownedAccepted = append(ownedAccepted, entry.ID)
		}
	}
	sort.Strings(ownedAccepted)
	wantOwned := []string{
		"EVD-02", "GOV-05", "GOV-06", "OPS-09", "OPS-10", "OPS-12", "ORC-12", "ORC-13", "ORC-17",
	}
	sort.Strings(wantOwned)
	if !reflect.DeepEqual(ownedAccepted, wantOwned) {
		t.Fatalf("V06 accepted ownership = %v, want exact %v", ownedAccepted, wantOwned)
	}
	wantV06Evidence := []string{
		"acceptance/v06_atomic_state_outbox_test.go",
		"acceptance/fixtures/v06_atomic_state_outbox.json",
		"product/evidence/v06_atomic_state_outbox.json",
	}
	for _, id := range wantOwned {
		entry := entries[id]
		switch entry.Status {
		case "declared":
			if len(entry.EvidenceRefs) != 0 {
				t.Errorf("declared V06 capability %s has premature evidence: %v", id, entry.EvidenceRefs)
			}
		case "accredited":
			if !reflect.DeepEqual(entry.EvidenceRefs, wantV06Evidence) {
				t.Errorf("accredited V06 capability %s has wrong evidence: %v", id, entry.EvidenceRefs)
			}
		default:
			t.Errorf("V06 capability %s has partial status %q; contract must remain declared or become fully accredited", id, entry.Status)
		}
	}

	wantMoved := map[string]string{
		"GOV-17": "command_registry",
		"EVD-01": "test_attestor",
		"OPS-13": "postgres_s3_multihost",
	}
	for id, owner := range wantMoved {
		entry := entries[id]
		vertical := verticals[owner]
		if entry.OwnerContext != owner || !reflect.DeepEqual(entry.Dependencies, vertical.DependsOn) ||
			!reflect.DeepEqual(entry.AcceptanceContracts, vertical.AcceptanceContracts) || entry.Status != "declared" ||
			len(entry.EvidenceRefs) != 0 {
			t.Errorf("deferred V06 capability %s = %#v, want exact owner %s", id, entry, owner)
		}
	}

	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV06ScopeAndExecutableContract|TestAcceptanceV06AtomicStateOutbox|TestV06CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/interfaces/mcp ./internal/bootstrap ./cmd/orquesta'"
	var contract roadmapAcceptanceContract
	for _, candidate := range roadmap.AcceptanceContracts {
		if candidate.ID == "AC-V06-ATOMIC-STATE-OUTBOX" {
			contract = candidate
			break
		}
	}
	if contract.Status != "executable" || contract.TestRef != "acceptance/v06_atomic_state_outbox_test.go" ||
		contract.Fixture != "acceptance/fixtures/v06_atomic_state_outbox.json" ||
		contract.Receipt != "product/evidence/v06_atomic_state_outbox.json" || contract.Command != wantCommand ||
		len(contract.Assertions) != 9 {
		t.Fatalf("invalid V06 executable contract: %#v", contract)
	}
	if !roadmapCommandHasArgument(contract.Command, "./acceptance") || strings.Contains(contract.Command, "./...") ||
		strings.Contains(contract.Command, "^TestAcceptance$") {
		t.Fatalf("V06 command is broad or can omit real gates: %q", contract.Command)
	}
}

func TestProductRoadmapV07ScopeAndExecutableContract(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)

	entries := make(map[string]roadmapEntry, len(roadmap.CapabilityEntries))
	verticals := make(map[string]roadmapVertical, len(roadmap.Verticals))
	var ownedAccepted []string
	for _, vertical := range roadmap.Verticals {
		verticals[vertical.ID] = vertical
	}
	for _, entry := range roadmap.CapabilityEntries {
		entries[entry.ID] = entry
		if entry.OwnerContext == "config" && entry.Decision == "accept" {
			ownedAccepted = append(ownedAccepted, entry.ID)
		}
	}
	sort.Strings(ownedAccepted)
	wantOwned := []string{
		"OPS-01", "OPS-02", "OPS-04", "OPS-05", "OPS-06",
		"OPS-26", "OPS-27", "OPS-28", "OPS-29", "OPS-30",
	}
	sort.Strings(wantOwned)
	if !reflect.DeepEqual(ownedAccepted, wantOwned) {
		t.Fatalf("V07 accepted ownership = %v, want exact %v", ownedAccepted, wantOwned)
	}
	wantV07Evidence := []string{
		"acceptance/v07_config_test.go",
		"acceptance/fixtures/v07_config.json",
		"product/evidence/v07_config.json",
	}
	for _, id := range wantOwned {
		entry := entries[id]
		switch entry.Status {
		case "declared":
			if len(entry.EvidenceRefs) != 0 {
				t.Errorf("declared V07 capability %s has premature evidence: %v", id, entry.EvidenceRefs)
			}
		case "accredited":
			if !reflect.DeepEqual(entry.EvidenceRefs, wantV07Evidence) {
				t.Errorf("accredited V07 capability %s has wrong evidence: %v", id, entry.EvidenceRefs)
			}
		default:
			t.Errorf("V07 capability %s has partial status %q; contract must remain declared or become fully accredited", id, entry.Status)
		}
	}

	deferred := entries["OPS-07"]
	wantDeferredVertical := verticals["web_admin"]
	if deferred.OwnerContext != "web_admin" ||
		!reflect.DeepEqual(deferred.Dependencies, wantDeferredVertical.DependsOn) ||
		!reflect.DeepEqual(deferred.AcceptanceContracts, wantDeferredVertical.AcceptanceContracts) ||
		deferred.Status != "declared" || len(deferred.EvidenceRefs) != 0 {
		t.Fatalf("OPS-07 must remain wholly deferred to V24 public bindings: %#v", deferred)
	}

	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV07ScopeAndExecutableContract|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV07Config|TestV07CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/config ./internal/adapters/config/effectivefile ./internal/adapters/config/toml ./internal/bootstrap ./cmd/orquesta'"
	var contract roadmapAcceptanceContract
	for _, candidate := range roadmap.AcceptanceContracts {
		if candidate.ID == "AC-V07-CONFIG" {
			contract = candidate
			break
		}
	}
	if contract.Status != "executable" || contract.TestRef != "acceptance/v07_config_test.go" ||
		contract.Fixture != "acceptance/fixtures/v07_config.json" ||
		contract.Receipt != "product/evidence/v07_config.json" || contract.Command != wantCommand ||
		len(contract.Assertions) != 13 {
		t.Fatalf("invalid V07 executable contract: %#v", contract)
	}
	if !roadmapCommandHasArgument(contract.Command, "./acceptance") || strings.Contains(contract.Command, "./...") ||
		strings.Contains(contract.Command, "^TestAcceptance$") || strings.Contains(contract.Command, "./internal/interfaces/mcp") {
		t.Fatalf("V07 command is broad, exposes premature public bindings, or can omit real gates: %q", contract.Command)
	}
}

func TestProductRoadmapV08ScopeAndExecutableContract(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)

	entries := make(map[string]roadmapEntry, len(roadmap.CapabilityEntries))
	verticals := make(map[string]roadmapVertical, len(roadmap.Verticals))
	var ownedAccepted []string
	for _, vertical := range roadmap.Verticals {
		verticals[vertical.ID] = vertical
	}
	for _, entry := range roadmap.CapabilityEntries {
		entries[entry.ID] = entry
		if entry.OwnerContext == "credentials" && entry.Decision == "accept" {
			ownedAccepted = append(ownedAccepted, entry.ID)
		}
	}
	sort.Strings(ownedAccepted)
	wantOwned := []string{"EVD-11", "EVD-12", "OPS-03", "OPS-08"}
	sort.Strings(wantOwned)
	if !reflect.DeepEqual(ownedAccepted, wantOwned) {
		t.Fatalf("V08 accepted ownership = %v, want exact %v", ownedAccepted, wantOwned)
	}
	wantV08Evidence := []string{
		"acceptance/v08_credentials_test.go",
		"acceptance/fixtures/v08_credentials.json",
		"product/evidence/v08_credentials.json",
	}
	for _, id := range wantOwned {
		entry := entries[id]
		switch entry.Status {
		case "declared":
			if len(entry.EvidenceRefs) != 0 {
				t.Errorf("declared V08 capability %s has premature evidence: %v", id, entry.EvidenceRefs)
			}
		case "accredited":
			if !reflect.DeepEqual(entry.EvidenceRefs, wantV08Evidence) {
				t.Errorf("accredited V08 capability %s has wrong evidence: %v", id, entry.EvidenceRefs)
			}
		default:
			t.Errorf("V08 capability %s has partial status %q; contract must remain declared or become fully accredited", id, entry.Status)
		}
	}

	deferred := entries["EVD-13"]
	wantDeferredVertical := verticals["test_attestor"]
	if deferred.OwnerContext != "test_attestor" ||
		!reflect.DeepEqual(deferred.Dependencies, wantDeferredVertical.DependsOn) ||
		!reflect.DeepEqual(deferred.AcceptanceContracts, wantDeferredVertical.AcceptanceContracts) ||
		deferred.Status != "declared" || len(deferred.EvidenceRefs) != 0 {
		t.Fatalf("EVD-13 must remain wholly deferred to V17 real sandbox enforcement: %#v", deferred)
	}

	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV08ScopeAndExecutableContract|TestV08AcceptanceCommandRunsCodexCredentialIntegration|TestCredentialRefsMatchConfigAndGoalOpaqueRefs|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV08Credentials|TestV08CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/credentials ./internal/adapters/credentials/local ./internal/adapters/agent/codex ./internal/config ./internal/goal ./internal/bootstrap ./cmd/orquesta'"
	var contract roadmapAcceptanceContract
	for _, candidate := range roadmap.AcceptanceContracts {
		if candidate.ID == "AC-V08-CREDENTIALS" {
			contract = candidate
			break
		}
	}
	if contract.Status != "executable" || contract.TestRef != "acceptance/v08_credentials_test.go" ||
		contract.Fixture != "acceptance/fixtures/v08_credentials.json" ||
		contract.Receipt != "product/evidence/v08_credentials.json" || contract.Command != wantCommand ||
		len(contract.Assertions) != 11 {
		t.Fatalf("invalid V08 executable contract: %#v", contract)
	}
	for _, forbidden := range []string{"./...", "^TestAcceptance$", "./internal/interfaces/mcp", "./internal/identity", "http", "web", "rbac"} {
		if strings.Contains(strings.ToLower(contract.Command), forbidden) {
			t.Fatalf("V08 command opens a broad or deferred surface %q: %q", forbidden, contract.Command)
		}
	}
	if !roadmapCommandHasArgument(contract.Command, "./acceptance") ||
		!roadmapCommandHasArgument(contract.Command, "./internal/credentials") ||
		!roadmapCommandHasArgument(contract.Command, "./internal/adapters/credentials/local") ||
		!roadmapCommandHasArgument(contract.Command, "./internal/adapters/agent/codex") {
		t.Fatalf("V08 command can omit contract or replaceable backend: %q", contract.Command)
	}
}

func TestV08AcceptanceCommandRunsCodexCredentialIntegration(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V08-CREDENTIALS" {
			continue
		}
		if !roadmapCommandHasArgument(contract.Command, "./internal/adapters/agent/codex") {
			t.Fatalf("V08 acceptance omits the real credential consumer package: %q", contract.Command)
		}
		return
	}
	t.Fatal("AC-V08-CREDENTIALS missing")
}

func TestProductRoadmapV09ScopeAndExecutableContract(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)

	entries := make(map[string]roadmapEntry, len(roadmap.CapabilityEntries))
	verticals := make(map[string]roadmapVertical, len(roadmap.Verticals))
	var ownedAccepted []string
	for _, vertical := range roadmap.Verticals {
		verticals[vertical.ID] = vertical
	}
	for _, entry := range roadmap.CapabilityEntries {
		entries[entry.ID] = entry
		if entry.OwnerContext == "recovery_backup" && entry.Decision == "accept" {
			ownedAccepted = append(ownedAccepted, entry.ID)
		}
	}
	sort.Strings(ownedAccepted)
	wantOwned := []string{"EVD-15", "OPS-14"}
	if !reflect.DeepEqual(ownedAccepted, wantOwned) {
		t.Fatalf("V09 accepted ownership = %v, want exact %v", ownedAccepted, wantOwned)
	}
	wantV09Evidence := []string{
		"acceptance/v09_recovery_backup_test.go",
		"acceptance/fixtures/v09_recovery_backup.json",
		"product/evidence/v09_recovery_backup.json",
	}
	for _, id := range wantOwned {
		entry := entries[id]
		switch entry.Status {
		case "declared":
			if len(entry.EvidenceRefs) != 0 {
				t.Errorf("declared V09 capability %s has premature evidence: %v", id, entry.EvidenceRefs)
			}
		case "accredited":
			if !reflect.DeepEqual(entry.EvidenceRefs, wantV09Evidence) {
				t.Errorf("accredited V09 capability %s has wrong evidence: %v", id, entry.EvidenceRefs)
			}
		default:
			t.Errorf("V09 capability %s has partial status %q; contract must remain declared or become fully accredited", id, entry.Status)
		}
	}

	deferred := entries["OPS-15"]
	wantDeferredVertical := verticals["operations_telemetry"]
	if deferred.OwnerContext != "operations_telemetry" ||
		!reflect.DeepEqual(deferred.Dependencies, wantDeferredVertical.DependsOn) ||
		!reflect.DeepEqual(deferred.AcceptanceContracts, wantDeferredVertical.AcceptanceContracts) ||
		deferred.Status != "declared" || len(deferred.EvidenceRefs) != 0 {
		t.Fatalf("OPS-15 must remain wholly deferred to V32 operation and rollback: %#v", deferred)
	}

	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV09ScopeAndExecutableContract|TestV09AcceptanceCommandRunsRecoveryConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV09RecoveryBackup|TestV09CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/application ./internal/ports ./internal/adapters/state/sqlite ./internal/adapters/artifact/filesystem ./internal/config ./internal/adapters/config/toml ./internal/adapters/config/jsonimport ./cmd/orquesta ./internal/bootstrap'"
	var contract roadmapAcceptanceContract
	for _, candidate := range roadmap.AcceptanceContracts {
		if candidate.ID == "AC-V09-RECOVERY-BACKUP" {
			contract = candidate
			break
		}
	}
	if contract.Status != "executable" || contract.TestRef != "acceptance/v09_recovery_backup_test.go" ||
		contract.Fixture != "acceptance/fixtures/v09_recovery_backup.json" ||
		contract.Receipt != "product/evidence/v09_recovery_backup.json" || contract.Command != wantCommand ||
		len(contract.Assertions) != 12 {
		t.Fatalf("invalid V09 executable contract: %#v", contract)
	}
	for _, forbidden := range []string{
		"./...", "^testacceptance$", "./internal/interfaces/mcp", "./internal/identity",
		"./internal/credentials", "./internal/adapters/credentials", "codex", "http", "web", "rbac",
		"postgres", "s3", "install", "update", "rollback",
	} {
		if strings.Contains(strings.ToLower(contract.Command), forbidden) {
			t.Fatalf("V09 command opens a broad or deferred surface %q: %q", forbidden, contract.Command)
		}
	}
}

func TestV09AcceptanceCommandRunsRecoveryConsumers(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V09-RECOVERY-BACKUP" {
			continue
		}
		for _, required := range []string{
			"./acceptance",
			"./internal/application",
			"./internal/ports",
			"./internal/adapters/state/sqlite",
			"./internal/adapters/artifact/filesystem",
			"./internal/config",
			"./internal/adapters/config/toml",
			"./internal/adapters/config/jsonimport",
			"./internal/bootstrap",
			"./cmd/orquesta",
		} {
			if !roadmapCommandHasArgument(contract.Command, required) {
				t.Errorf("V09 acceptance omits recovery consumer package %q: %q", required, contract.Command)
			}
		}
		return
	}
	t.Fatal("AC-V09-RECOVERY-BACKUP missing")
}

func TestV04AccreditsOnlyGOV02AndPreservesGOV01Deferred(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	entries := make(map[string]roadmapEntry, len(roadmap.CapabilityEntries))
	for _, entry := range roadmap.CapabilityEntries {
		entries[entry.ID] = entry
	}

	if entry := entries["GOV-01"]; entry.Status != "declared" || entry.OwnerContext != "generated_apps" || len(entry.EvidenceRefs) != 0 {
		t.Fatalf("GOV-01 must remain deferred to generated_apps without V04 evidence: %#v", entry)
	}
	wantEvidence := []string{
		"acceptance/v04_intent_appspec_test.go",
		"acceptance/fixtures/v04_intent_appspec.json",
		"product/evidence/v04_intent_appspec.json",
	}
	if entry := entries["GOV-02"]; entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence) {
		t.Fatalf("GOV-02 accreditation = %#v, want exact V04 evidence %v", entry, wantEvidence)
	}
}

func TestProductRoadmapPlannedContractsAreNonRunnable(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	for _, contract := range roadmap.AcceptanceContracts {
		switch contract.Status {
		case "planned":
			if !strings.HasPrefix(contract.Command, "planned:go test -mod=vendor") {
				t.Errorf("planned contract %s exposes a deceptively runnable command %q", contract.ID, contract.Command)
			}
		case "executable":
			if strings.HasPrefix(contract.Command, "planned:") {
				t.Errorf("executable contract %s retains planned command %q", contract.ID, contract.Command)
			}
		}
	}
}

func TestProductRoadmapExecutableContractsDeclareReceiptPaths(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	requiredExecutable := map[string]string{
		"AC-V01-SOURCE-INTEGRATION":  "product/evidence/v01_source_integration.json",
		"AC-V02-AUTHORITY-RULES":     "product/evidence/v02_authority_rules.json",
		"AC-V03-CANONICAL-LEDGERS":   "product/evidence/v03_canonical_ledgers.json",
		"AC-V04-INTENT-APPSPEC":      "product/evidence/v04_intent_appspec.json",
		"AC-V05-GOAL-DAG-PHASES":     "product/evidence/v05_goal_dag_phases.json",
		"AC-V06-ATOMIC-STATE-OUTBOX": "product/evidence/v06_atomic_state_outbox.json",
		"AC-V07-CONFIG":              "product/evidence/v07_config.json",
	}
	gotExecutable := make(map[string]string)
	for _, contract := range roadmap.AcceptanceContracts {
		switch contract.Status {
		case "executable":
			if !strings.HasPrefix(contract.Receipt, "product/evidence/") || !strings.HasSuffix(contract.Receipt, ".json") {
				t.Errorf("executable contract %s lacks canonical receipt path", contract.ID)
			}
			gotExecutable[contract.ID] = contract.Receipt
		case "planned":
			if contract.Receipt != "" {
				t.Errorf("planned contract %s has premature receipt %q", contract.ID, contract.Receipt)
			}
		}
	}
	for id, want := range requiredExecutable {
		if got := gotExecutable[id]; got != want {
			t.Errorf("executable receipt %s = %q, want %q", id, got, want)
		}
	}
}

func TestProductRoadmapExecutableCommandsRunDeclaredTestPackage(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.Status != "executable" {
			continue
		}
		directory := path.Dir(contract.TestRef)
		packageArgument := "."
		if directory != "." {
			packageArgument = "./" + directory
		}
		if !roadmapCommandHasArgument(contract.Command, packageArgument) {
			t.Errorf("executable contract %s command omits declared test package %q: %q", contract.ID, packageArgument, contract.Command)
		}
	}
}

func roadmapCommandHasArgument(command, wanted string) bool {
	for _, argument := range strings.Fields(command) {
		if strings.Trim(argument, "'\"") == wanted {
			return true
		}
	}
	return false
}

func roadmapEvidenceContainsVertical(evidence []string, vertical string) bool {
	for _, ref := range evidence {
		if strings.Contains(ref, vertical) {
			return true
		}
	}
	return false
}

func assertRoadmapEvidenceCoherent(t *testing.T, entry roadmapEntry, contract roadmapAcceptanceContract) {
	t.Helper()
	if entry.Status == "declared" {
		if len(entry.EvidenceRefs) != 0 {
			t.Fatalf("declared capability %q has premature evidence %v", entry.ID, entry.EvidenceRefs)
		}
		return
	}
	if contract.Status != "executable" {
		t.Fatalf("capability %q progressed to %q against %s contract %q", entry.ID, entry.Status, contract.Status, contract.ID)
	}
	if len(entry.EvidenceRefs) == 0 {
		t.Fatalf("progressed capability %q lacks evidence", entry.ID)
	}
	seen := make(map[string]struct{}, len(entry.EvidenceRefs))
	for _, evidence := range entry.EvidenceRefs {
		if _, duplicate := seen[evidence]; duplicate {
			t.Fatalf("capability %q duplicates evidence %q", entry.ID, evidence)
		}
		seen[evidence] = struct{}{}
		requireRepositoryFile(t, ".", evidence)
	}
	if entry.Status != "accredited" {
		return
	}
	if contract.Receipt == "" {
		t.Fatalf("accredited capability %q lacks contract receipt", entry.ID)
	}
	for _, required := range []string{contract.TestRef, contract.Fixture, contract.Receipt} {
		if _, exists := seen[required]; !exists {
			t.Fatalf("accredited capability %q lacks contract evidence %q", entry.ID, required)
		}
	}
}

func assertRoadmapProgressCausality(
	t *testing.T,
	ordered []roadmapVertical,
	verticals map[string]roadmapVertical,
	contracts map[string]roadmapAcceptanceContract,
	entries map[string]roadmapEntry,
) {
	t.Helper()
	ownedAccepted := make(map[string][]roadmapEntry, len(verticals))
	progressed := make(map[string]bool, len(verticals))
	for _, entry := range entries {
		if entry.Status != "declared" {
			progressed[entry.OwnerContext] = true
		}
		if entry.Decision == "accept" {
			ownedAccepted[entry.OwnerContext] = append(ownedAccepted[entry.OwnerContext], entry)
		}
	}
	accredited := make(map[string]bool, len(verticals))
	for _, vertical := range ordered {
		contract := contracts[vertical.AcceptanceContracts[0]]
		if contract.Status == "executable" {
			progressed[vertical.ID] = true
		}
		owned := ownedAccepted[vertical.ID]
		if contract.Status != "executable" || contract.Receipt == "" {
			continue
		}
		if len(owned) == 0 {
			accredited[vertical.ID] = true
			continue
		}
		accredited[vertical.ID] = true
		for _, entry := range owned {
			if entry.Status != "accredited" {
				accredited[vertical.ID] = false
				break
			}
		}
	}
	for _, vertical := range ordered {
		if !progressed[vertical.ID] {
			continue
		}
		for _, dependency := range vertical.DependsOn {
			if !accredited[dependency] {
				t.Fatalf("vertical %q progressed before dependency %q was accredited", vertical.ID, dependency)
			}
		}
	}
	for _, required := range []string{"source_integration", "authority_rules"} {
		if !accredited[required] {
			t.Fatalf("required foundation vertical %q is not causally accredited", required)
		}
	}
}

func assertRoadmapOperatorDecisions(t *testing.T, decisions map[string]json.RawMessage) {
	t.Helper()
	if len(decisions) != 13 {
		t.Fatalf("operator decision count = %d, want 13", len(decisions))
	}
	wantStrings := map[string]string{
		"architecture":        "modular_monolith_hexagonal",
		"lifecycle_authority": "goal",
		"lifecycle_writer":    "application_orchestrator",
		"director_model":      "replaceable_role_with_lease_and_fencing",
		"local_identity":      "local_token",
		"server_identity":     "oidc_or_ldap_ad_selected_at_install",
		"native_desktop":      "rejected_use_pwa",
		"declarative_a2ui":    "rejected_until_real_need",
		"time_travel":         "rejected_use_causal_generations_and_backup",
		"opes_games":          "rejected_outside_core_production",
		"advanced_retrieval":  "conditional_on_versioned_benchmark",
	}
	for key, want := range wantStrings {
		var got string
		if json.Unmarshal(decisions[key], &got) != nil || got != want {
			t.Fatalf("operator decision %q = %s, want %q", key, decisions[key], want)
		}
	}
	for _, key := range []string{"state_adapters", "artifact_adapters"} {
		var values []string
		if json.Unmarshal(decisions[key], &values) != nil || len(values) != 2 {
			t.Fatalf("operator decision %q = %s, want two adapters", key, decisions[key])
		}
	}
}

func assertDeferredMappings(t *testing.T, mappings []roadmapMapping, entries map[string]roadmapEntry) {
	t.Helper()
	var minimal productManifest
	decodeRoadmapStrictJSON(t, "product/capabilities.json", &minimal)
	want := append([]string(nil), minimal.Deferred...)
	sort.Strings(want)
	got := make([]string, 0, len(mappings))
	seen := make(map[string]struct{}, len(mappings))
	for _, mapping := range mappings {
		if _, duplicate := seen[mapping.ID]; duplicate {
			t.Fatalf("duplicate deferred mapping %q", mapping.ID)
		}
		if len(mapping.CapabilityIDs) == 0 {
			t.Fatalf("deferred mapping %q is empty", mapping.ID)
		}
		seen[mapping.ID] = struct{}{}
		got = append(got, mapping.ID)
		for _, id := range mapping.CapabilityIDs {
			if _, exists := entries[id]; !exists {
				t.Fatalf("deferred mapping %q targets missing %q", mapping.ID, id)
			}
		}
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("deferred mappings = %v, want %v", got, want)
	}
}

func decodeRoadmapStrictJSON(t *testing.T, path string, target any) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("decode trailing %s: %v", path, err)
	}
}

func assertRoadmapAcyclic(t *testing.T, kind string, ids []string, dependencies func(string) []string) {
	t.Helper()
	state := make(map[string]uint8, len(ids))
	var visit func(string)
	visit = func(id string) {
		if state[id] == 2 {
			return
		}
		if state[id] == 1 {
			t.Fatalf("%s dependency cycle at %q", kind, id)
		}
		state[id] = 1
		for _, dependency := range dependencies(id) {
			visit(dependency)
		}
		state[id] = 2
	}
	for _, id := range ids {
		visit(id)
	}
}

func roadmapVerticalKeys(values map[string]roadmapVertical) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func roadmapEntryKeys(values map[string]roadmapEntry) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func roadmapSetOf(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func roadmapTwoDigits(value int) string {
	return string([]byte{'0' + byte(value/10), '0' + byte(value%10)})
}
