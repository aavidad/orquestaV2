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

	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV09ScopeAndExecutableContract|TestV09EvidenceBelongsOnlyToRecoveryCapabilities|TestV09AcceptanceCommandRunsRecoveryConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV09RecoveryBackup|TestV09CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/application ./internal/ports ./internal/adapters/state/sqlite ./internal/adapters/artifact/filesystem ./internal/config ./internal/adapters/config/toml ./internal/adapters/config/jsonimport ./cmd/orquesta ./internal/bootstrap'"
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

func TestV09EvidenceBelongsOnlyToRecoveryCapabilities(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)

	owned := map[string]bool{"EVD-15": true, "OPS-14": true}
	v09Evidence := map[string]bool{
		"acceptance/v09_recovery_backup_test.go":       true,
		"acceptance/fixtures/v09_recovery_backup.json": true,
		"product/evidence/v09_recovery_backup.json":    true,
	}
	for _, entry := range roadmap.CapabilityEntries {
		if owned[entry.ID] {
			if entry.Status != "accredited" {
				t.Errorf("owned V09 capability %s status=%q, want accredited", entry.ID, entry.Status)
			}
			continue
		}
		for _, evidenceRef := range entry.EvidenceRefs {
			if v09Evidence[evidenceRef] {
				t.Errorf("unowned capability %s claims V09 evidence %q", entry.ID, evidenceRef)
			}
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

func TestProductRoadmapV10ScopeAndExecutableContract(t *testing.T) {
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
		if entry.OwnerContext == "identity_projects_rbac" && entry.Decision == "accept" {
			ownedAccepted = append(ownedAccepted, entry.ID)
		}
	}
	sort.Strings(ownedAccepted)
	wantOwned := []string{"GOV-19", "GOV-20", "GOV-22"}
	if !reflect.DeepEqual(ownedAccepted, wantOwned) {
		t.Fatalf("V10 accepted ownership = %v, want exact %v", ownedAccepted, wantOwned)
	}

	wantV10Evidence := []string{
		"acceptance/v10_identity_projects_rbac_test.go",
		"acceptance/fixtures/v10_identity_projects_rbac.json",
		"product/evidence/v10_identity_projects_rbac.json",
	}
	for _, id := range wantOwned {
		entry := entries[id]
		if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantV10Evidence) {
			t.Errorf("V10 capability %s accreditation=%q evidence=%v", id, entry.Status, entry.EvidenceRefs)
		}
	}

	fairness := entries["ORC-11"]
	wantV15 := verticals["budgets_effects"]
	wantV15Evidence := []string{
		"acceptance/v15_budgets_effects_test.go",
		"acceptance/fixtures/v15_budgets_effects.json",
		"product/evidence/v15_budgets_effects.json",
	}
	if fairness.OwnerContext != "budgets_effects" ||
		!reflect.DeepEqual(fairness.Dependencies, wantV15.DependsOn) ||
		!reflect.DeepEqual(fairness.AcceptanceContracts, wantV15.AcceptanceContracts) ||
		fairness.Status != "accredited" || !reflect.DeepEqual(fairness.EvidenceRefs, wantV15Evidence) {
		t.Fatalf("ORC-11 fairness must remain owned and accredited only by V15: %#v", fairness)
	}

	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV10ScopeAndExecutableContract|TestV10EvidenceBelongsOnlyToIdentityCapabilities|TestV10AcceptanceCommandRunsIdentityConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV10IdentityProjectsRBAC|TestV10CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/identity ./internal/application ./internal/config ./internal/i18n ./internal/adapters/auth/localtoken ./internal/adapters/state/sqlite ./internal/interfaces/mcp ./internal/bootstrap ./cmd/orquesta'"
	var contract roadmapAcceptanceContract
	for _, candidate := range roadmap.AcceptanceContracts {
		if candidate.ID == "AC-V10-IDENTITY-PROJECTS-RBAC" {
			contract = candidate
			break
		}
	}
	if contract.Status != "executable" || contract.TestRef != "acceptance/v10_identity_projects_rbac_test.go" ||
		contract.Fixture != "acceptance/fixtures/v10_identity_projects_rbac.json" ||
		contract.Receipt != "product/evidence/v10_identity_projects_rbac.json" || contract.Command != wantCommand ||
		len(contract.Assertions) != 12 {
		t.Fatalf("invalid V10 executable contract: %#v", contract)
	}
	for _, forbidden := range []string{
		"./...", "^testacceptance$", "/codex", "oidc", "ldap", "workspace", "git", "budget",
		"fairness", "/web", "postgres", "s3", "multihost",
	} {
		if strings.Contains(strings.ToLower(contract.Command), forbidden) {
			t.Fatalf("V10 command opens a broad or deferred surface %q: %q", forbidden, contract.Command)
		}
	}
}

func TestProductRoadmapV10DeferredFairnessUsesOnlyV15EvidenceAfterClosure(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)

	var fairness roadmapEntry
	var budgets roadmapVertical
	for _, entry := range roadmap.CapabilityEntries {
		if entry.ID == "ORC-11" {
			fairness = entry
		}
	}
	for _, vertical := range roadmap.Verticals {
		if vertical.ID == "budgets_effects" {
			budgets = vertical
		}
	}
	wantEvidence := []string{
		"acceptance/v15_budgets_effects_test.go",
		"acceptance/fixtures/v15_budgets_effects.json",
		"product/evidence/v15_budgets_effects.json",
	}
	if fairness.OwnerContext != budgets.ID ||
		!reflect.DeepEqual(fairness.Dependencies, budgets.DependsOn) ||
		!reflect.DeepEqual(fairness.AcceptanceContracts, budgets.AcceptanceContracts) ||
		fairness.Status != "accredited" || !reflect.DeepEqual(fairness.EvidenceRefs, wantEvidence) {
		t.Fatalf("V10-deferred ORC-11 does not carry only its V15 accreditation: %#v", fairness)
	}
}

func TestV10EvidenceBelongsOnlyToIdentityCapabilities(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)

	owned := map[string]bool{"GOV-19": true, "GOV-20": true, "GOV-22": true}
	v10Evidence := map[string]bool{
		"acceptance/v10_identity_projects_rbac_test.go":       true,
		"acceptance/fixtures/v10_identity_projects_rbac.json": true,
		"product/evidence/v10_identity_projects_rbac.json":    true,
	}
	for _, entry := range roadmap.CapabilityEntries {
		if owned[entry.ID] {
			if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, []string{
				"acceptance/v10_identity_projects_rbac_test.go",
				"acceptance/fixtures/v10_identity_projects_rbac.json",
				"product/evidence/v10_identity_projects_rbac.json",
			}) {
				t.Errorf("owned V10 capability %s lacks exact accreditation: status=%q evidence=%v", entry.ID, entry.Status, entry.EvidenceRefs)
			}
			continue
		}
		for _, evidenceRef := range entry.EvidenceRefs {
			if v10Evidence[evidenceRef] {
				t.Errorf("unowned capability %s claims V10 evidence %q", entry.ID, evidenceRef)
			}
		}
	}
}

func TestV10AcceptanceCommandRunsIdentityConsumers(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V10-IDENTITY-PROJECTS-RBAC" {
			continue
		}
		for _, required := range []string{
			"./acceptance",
			"./internal/goal",
			"./internal/identity",
			"./internal/application",
			"./internal/config",
			"./internal/i18n",
			"./internal/adapters/auth/localtoken",
			"./internal/adapters/state/sqlite",
			"./internal/interfaces/mcp",
			"./internal/bootstrap",
			"./cmd/orquesta",
		} {
			if !roadmapCommandHasArgument(contract.Command, required) {
				t.Errorf("V10 acceptance omits identity consumer package %q: %q", required, contract.Command)
			}
		}
		return
	}
	t.Fatal("AC-V10-IDENTITY-PROJECTS-RBAC missing")
}

func TestProductRoadmapV11ScopeAndExecutableContract(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)

	for _, entry := range roadmap.CapabilityEntries {
		if entry.OwnerContext == "oidc_ad" {
			t.Fatalf("V11 is a transverse integration vertical and must not invent a capability ID: %#v", entry)
		}
	}
	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV11ScopeAndExecutableContract|TestV11OwnsNoCapabilityIDs|TestV11AcceptanceCommandRunsIdentityConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV11OIDCAD|TestV11CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/identity ./internal/config ./internal/adapters/auth/bearer ./internal/adapters/auth/localtoken ./internal/adapters/auth/oidc ./internal/bootstrap ./cmd/orquesta && ./scripts/smoke_v11_dex_samba_ad.sh'"
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V11-OIDC-AD" {
			continue
		}
		if contract.Status != "executable" || contract.TestRef != "acceptance/v11_oidc_ad_test.go" ||
			contract.Fixture != "acceptance/fixtures/v11_oidc_ad.json" ||
			contract.Receipt != "product/evidence/v11_oidc_ad.json" || contract.Command != wantCommand ||
			len(contract.Assertions) != 16 {
			t.Fatalf("invalid V11 executable contract: %#v", contract)
		}
		return
	}
	t.Fatal("AC-V11-OIDC-AD missing")
}

func TestV11OwnsNoCapabilityIDs(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	v11Evidence := map[string]bool{
		"acceptance/v11_oidc_ad_test.go":       true,
		"acceptance/fixtures/v11_oidc_ad.json": true,
		"product/evidence/v11_oidc_ad.json":    true,
	}
	for _, entry := range roadmap.CapabilityEntries {
		if entry.OwnerContext == "oidc_ad" {
			t.Errorf("capability %s is incorrectly owned by transverse V11", entry.ID)
		}
		for _, evidenceRef := range entry.EvidenceRefs {
			if v11Evidence[evidenceRef] {
				t.Errorf("capability %s incorrectly claims transverse V11 evidence %q", entry.ID, evidenceRef)
			}
		}
	}
}

func TestV11AcceptanceCommandRunsIdentityConsumers(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V11-OIDC-AD" {
			continue
		}
		for _, required := range []string{
			"./acceptance", "./internal/identity", "./internal/config",
			"./internal/adapters/auth/bearer", "./internal/adapters/auth/localtoken",
			"./internal/adapters/auth/oidc", "./internal/bootstrap", "./cmd/orquesta",
			"./scripts/smoke_v11_dex_samba_ad.sh",
		} {
			if !roadmapCommandHasArgument(contract.Command, required) {
				t.Errorf("V11 acceptance omits identity consumer package %q: %q", required, contract.Command)
			}
		}
		return
	}
	t.Fatal("AC-V11-OIDC-AD missing")
}

func TestProductRoadmapV12ScopeAndExecutableContract(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	var owned []string
	entries := make(map[string]roadmapEntry, len(roadmap.CapabilityEntries))
	for _, entry := range roadmap.CapabilityEntries {
		entries[entry.ID] = entry
		if entry.OwnerContext == "director_lease" && entry.Decision == "accept" {
			owned = append(owned, entry.ID)
		}
	}
	sort.Strings(owned)
	wantOwned := []string{"GOV-08", "GOV-09", "GOV-10", "ORC-24"}
	if !reflect.DeepEqual(owned, wantOwned) {
		t.Fatalf("V12 accepted ownership = %v, want exact %v", owned, wantOwned)
	}
	wantEvidence := []string{
		"acceptance/v12_director_lease_test.go",
		"acceptance/fixtures/v12_director_lease.json",
		"product/evidence/v12_director_lease.json",
	}
	for _, id := range wantOwned {
		entry := entries[id]
		if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence) {
			t.Errorf("V12 capability %s accreditation=%q evidence=%v", id, entry.Status, entry.EvidenceRefs)
		}
	}
	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV12ScopeAndExecutableContract|TestV12EvidenceBelongsOnlyToDirectorCapabilities|TestV12AcceptanceCommandRunsDirectorConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV12DirectorLease|TestV12CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/identity ./internal/application ./internal/config ./internal/adapters/state/sqlite ./internal/bootstrap ./cmd/orquesta'"
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V12-DIRECTOR-LEASE" {
			continue
		}
		if contract.Status != "executable" || contract.TestRef != "acceptance/v12_director_lease_test.go" ||
			contract.Fixture != "acceptance/fixtures/v12_director_lease.json" ||
			contract.Receipt != "product/evidence/v12_director_lease.json" || contract.Command != wantCommand ||
			len(contract.Assertions) != 14 {
			t.Fatalf("invalid V12 executable contract: %#v", contract)
		}
		return
	}
	t.Fatal("AC-V12-DIRECTOR-LEASE missing")
}

func TestV12EvidenceBelongsOnlyToDirectorCapabilities(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	owned := map[string]bool{"GOV-08": true, "GOV-09": true, "GOV-10": true, "ORC-24": true}
	wantEvidence := []string{
		"acceptance/v12_director_lease_test.go",
		"acceptance/fixtures/v12_director_lease.json",
		"product/evidence/v12_director_lease.json",
	}
	v12Evidence := make(map[string]bool, len(wantEvidence))
	for _, ref := range wantEvidence {
		v12Evidence[ref] = true
	}
	for _, entry := range roadmap.CapabilityEntries {
		if owned[entry.ID] {
			if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence) {
				t.Errorf("owned V12 capability %s lacks exact accreditation: status=%q evidence=%v", entry.ID, entry.Status, entry.EvidenceRefs)
			}
			continue
		}
		for _, evidenceRef := range entry.EvidenceRefs {
			if v12Evidence[evidenceRef] {
				t.Errorf("unowned capability %s claims V12 evidence %q", entry.ID, evidenceRef)
			}
		}
	}
}

func TestV12AcceptanceCommandRunsDirectorConsumers(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V12-DIRECTOR-LEASE" {
			continue
		}
		for _, required := range []string{
			"./acceptance", "./internal/goal", "./internal/identity", "./internal/application",
			"./internal/config", "./internal/adapters/state/sqlite", "./internal/bootstrap", "./cmd/orquesta",
		} {
			if !roadmapCommandHasArgument(contract.Command, required) {
				t.Errorf("V12 acceptance omits Director consumer package %q: %q", required, contract.Command)
			}
		}
		return
	}
	t.Fatal("AC-V12-DIRECTOR-LEASE missing")
}

func TestProductRoadmapV13ScopeAndExecutableContract(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	verticals := make(map[string]roadmapVertical, len(roadmap.Verticals))
	entries := make(map[string]roadmapEntry, len(roadmap.CapabilityEntries))
	var owned []string
	for _, vertical := range roadmap.Verticals {
		verticals[vertical.ID] = vertical
	}
	for _, entry := range roadmap.CapabilityEntries {
		entries[entry.ID] = entry
		if entry.OwnerContext == "mailbox" && entry.Decision == "accept" {
			owned = append(owned, entry.ID)
		}
	}
	sort.Strings(owned)
	wantOwned := []string{"ORC-04", "ORC-05", "ORC-14"}
	if !reflect.DeepEqual(owned, wantOwned) {
		t.Fatalf("V13 accepted ownership = %v, want exact %v", owned, wantOwned)
	}
	wantEvidence := []string{
		"acceptance/v13_mailbox_test.go",
		"acceptance/fixtures/v13_mailbox.json",
		"product/evidence/v13_mailbox.json",
	}
	wantVertical := verticals["mailbox"]
	for _, id := range wantOwned {
		entry := entries[id]
		if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence) ||
			!reflect.DeepEqual(entry.Dependencies, wantVertical.DependsOn) ||
			!reflect.DeepEqual(entry.AcceptanceContracts, wantVertical.AcceptanceContracts) {
			t.Errorf("V13 capability %s lacks exact accreditation or causality: %#v", id, entry)
		}
	}

	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV13ScopeAndExecutableContract|TestV13EvidenceBelongsOnlyToMailboxCapabilities|TestV13AcceptanceCommandRunsMailboxConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV13Mailbox|TestV13CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/identity ./internal/config ./internal/application ./internal/ports ./internal/adapters/state/sqlite ./internal/bootstrap ./cmd/orquesta'"
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V13-MAILBOX" {
			continue
		}
		if contract.Status != "executable" || contract.TestRef != "acceptance/v13_mailbox_test.go" ||
			contract.Fixture != "acceptance/fixtures/v13_mailbox.json" ||
			contract.Receipt != "product/evidence/v13_mailbox.json" || contract.Command != wantCommand ||
			!reflect.DeepEqual(contract.Assertions, roadmapV13Assertions()) {
			t.Fatalf("invalid V13 executable contract: %#v", contract)
		}
		for _, forbidden := range []string{
			"./...", "/codex", "oidc", "ldap", "workspace", "git", "budget", "fairness",
			"/web", "postgres", "s3", "multihost", "pause", "resume", "cancel", "stop",
		} {
			if strings.Contains(strings.ToLower(contract.Command), forbidden) {
				t.Fatalf("V13 command opens broad or deferred surface %q: %q", forbidden, contract.Command)
			}
		}
		return
	}
	t.Fatal("AC-V13-MAILBOX missing")
}

func TestV13EvidenceBelongsOnlyToMailboxCapabilities(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	owned := map[string]bool{"ORC-04": true, "ORC-05": true, "ORC-14": true}
	wantEvidence := []string{
		"acceptance/v13_mailbox_test.go",
		"acceptance/fixtures/v13_mailbox.json",
		"product/evidence/v13_mailbox.json",
	}
	v13Evidence := make(map[string]bool, len(wantEvidence))
	for _, ref := range wantEvidence {
		v13Evidence[ref] = true
	}
	for _, entry := range roadmap.CapabilityEntries {
		if owned[entry.ID] {
			if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence) {
				t.Errorf("owned V13 capability %s lacks exact accreditation: status=%q evidence=%v",
					entry.ID, entry.Status, entry.EvidenceRefs)
			}
			continue
		}
		for _, evidenceRef := range entry.EvidenceRefs {
			if v13Evidence[evidenceRef] {
				t.Errorf("unowned capability %s claims V13 evidence %q", entry.ID, evidenceRef)
			}
		}
	}
}

func TestV13AcceptanceCommandRunsMailboxConsumers(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V13-MAILBOX" {
			continue
		}
		for _, required := range []string{
			"./acceptance", "./internal/goal", "./internal/identity", "./internal/application",
			"./internal/config", "./internal/ports", "./internal/adapters/state/sqlite", "./internal/bootstrap", "./cmd/orquesta",
		} {
			if !roadmapCommandHasArgument(contract.Command, required) {
				t.Errorf("V13 acceptance omits mailbox consumer package %q: %q", required, contract.Command)
			}
		}
		return
	}
	t.Fatal("AC-V13-MAILBOX missing")
}

func roadmapV13Assertions() []string {
	return []string{
		"admission is request-idempotent and atomically binds one immutable envelope to one action in the existing outbox without treating admission as delivery",
		"the envelope binds exact project Goal plan generation parent and child WorkItems plus source and recipient principal WorkItem and execution identities",
		"admitted claimed delivered consumed and acknowledged or blocked are separate causal facts with trusted timestamps and no text-derived lifecycle",
		"concurrent exact-recipient claims have one winner and use transaction-clock lease opaque token and one monotonic fence as the delivery-attempt ordinal rather than caller time",
		"wrong project Goal generation principal WorkItem execution sibling or successor cannot claim deliver consume acknowledge block or enumerate the message",
		"expired lease wrong token and stale fence cannot mutate while a post-expiry reclaim preserves the envelope and increments the single fence exactly once",
		"a crash after claim and before terminal recipient resolution permits safe reclaim after restart without message loss or partial acknowledgement",
		"exact claim replay returns the original claimed frontier without renewing its lease after expiry delivery or consumption and conflicts after a superseding fence or terminal mailbox",
		"exact delivery and consumption replay returns the original historical frontier after terminal state or a later fence without authorizing another mutation",
		"recipient mailbox listing is deterministic causal FIFO by admitted_at then message ref and applies its limit after exact recipient scope",
		"delivery consumption and recipient acknowledgement have distinct immutable receipts and an outbox consumption receipt alone is not recipient evidence",
		"exact acknowledgement replay returns the same receipt and acknowledged or blocked messages are never redelivered by outbox replay restart or another execution",
		"a successor execution cannot acknowledge a message addressed to its predecessor even when both executions use the same principal",
		"if the exact recipient execution fails before resolution the mailbox becomes retired in the same Goal failure transaction without replacement readdress ACK authorization or ChildHandoffResolution",
		"a contractual parent cannot succeed until every successful direct child has one acknowledged delivery or explicit recipient block; failed and dependency_failed skipped children are durable causal blocks",
		"an unrelated child terminal WorkItem admission ACK or delivery without recipient acknowledgement does not satisfy the parent closure barrier",
		"parent lineage is noncontractual by default and only an explicit HandoffRequired true edge activates the mailbox barrier so public V05 DAGs remain operable while public mailbox bindings are deferred",
		"child_delivery is the only V13 causal envelope; canonical mailbox.max_envelope_bytes bounds its summary and artifact refs while generic messages rich context resumable sessions and provider handoff stay deferred",
		"the existing outbox fence is the single mailbox attempt ordinal; mailbox adds no second delivery counter or private fence store",
		"V13 preserves the V02 application-only Goal writer the V05 contractual lineage gate and the V06 V09 V10 acceptance harness wiring",
		"Goal and application remain the only lifecycle authority and mailbox adds no private store database queue scheduler loop goroutine daemon provider policy or parallel lifecycle",
	}
}

func TestProductRoadmapV14ScopeAndExecutableContract(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	verticals := make(map[string]roadmapVertical, len(roadmap.Verticals))
	entries := make(map[string]roadmapEntry, len(roadmap.CapabilityEntries))
	contracts := make(map[string]roadmapAcceptanceContract, len(roadmap.AcceptanceContracts))
	var owned []string
	for _, vertical := range roadmap.Verticals {
		verticals[vertical.ID] = vertical
	}
	for _, entry := range roadmap.CapabilityEntries {
		entries[entry.ID] = entry
		if entry.OwnerContext == "controls" && entry.Decision == "accept" {
			owned = append(owned, entry.ID)
		}
	}
	for _, contract := range roadmap.AcceptanceContracts {
		contracts[contract.ID] = contract
	}
	sort.Strings(owned)
	wantOwned := []string{"GOV-07", "ORC-03", "ORC-16", "STG-15"}
	if !reflect.DeepEqual(owned, wantOwned) {
		t.Fatalf("V14 accepted ownership = %v, want exact %v", owned, wantOwned)
	}
	wantEvidence := []string{
		"acceptance/v14_controls_test.go",
		"acceptance/fixtures/v14_controls.json",
		"product/evidence/v14_controls.json",
	}
	wantVertical := verticals["controls"]
	for _, id := range wantOwned {
		entry := entries[id]
		if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence) ||
			!reflect.DeepEqual(entry.Dependencies, wantVertical.DependsOn) ||
			!reflect.DeepEqual(entry.AcceptanceContracts, wantVertical.AcceptanceContracts) {
			t.Errorf("V14 capability %s lacks exact accreditation: %#v", id, entry)
		}
	}

	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV14ScopeAndExecutableContract|TestV14EvidenceBelongsOnlyToControlCapabilities|TestV14AcceptanceCommandRunsControlConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV14Controls|TestV14CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/identity ./internal/config ./internal/credentials ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/bootstrap ./cmd/orquesta && go test -mod=vendor -race -count=1 ./internal/application ./internal/adapters/state/sqlite ./internal/adapters/agent/codex ./internal/bootstrap -run \"^(TestConcurrentIdenticalControlCASLoserReturnsExactReplay|TestControlsGoalAndWorkItemCancelCompletionCASBothOrders|TestControlsStopCompletionCASAndUnsupportedMode|TestControlsStopCrashReplayConvergesWithoutDuplicateEffect|TestClaimedRetryRevalidatesPauseBeforeLaunchPreparation|TestClaimedAutomaticReplacementRevalidatesPauseBeforeLaunchPreparation|TestSQLiteControlsRestartAndConcurrentCAS|TestSQLiteForcedStopSupersessionIsAtomicConcurrentAndRestartSafe|TestSQLiteTerminalStopSettlesAfterRestartWithReplacementAgentRouting|TestV14RecoveryAcceptsClaimedTerminalStopThenReclaimsAndSettlesOnce|TestCodexSelectiveStopPreservesSiblingProcessTrees|TestCodexSelectiveStopAdoptsAfterCrashAndRejectsReusedPID|TestCodexLaunchGateCrashNeverOrphansProcess|TestCodexOwnerLockIsExclusiveAndCLOEXEC|TestCodexRestoredDatabaseCannotAdoptSourceProcess|TestBuildBindsAgentToOpenedRepositoryIdentity|TestRealCodexControlsThroughProductionComposition|TestRealCodexCooperativeStopLeavesResidentSchedulerLive)$\"'"
	contract := contracts["AC-V14-CONTROLS"]
	if contract.Status != "executable" || contract.TestRef != "acceptance/v14_controls_test.go" ||
		contract.Fixture != "acceptance/fixtures/v14_controls.json" ||
		contract.Receipt != "product/evidence/v14_controls.json" || contract.Command != wantCommand ||
		!reflect.DeepEqual(contract.Assertions, roadmapV14Assertions()) {
		t.Fatalf("invalid V14 executable contract: %#v", contract)
	}
	for _, forbidden := range []string{
		"./...", "./internal/interfaces/mcp", "oidc", "ldap", "workspace", "git", "budget",
		"fairness", "effects", "/web", "postgres", "s3", "multihost", "claude", "gemini",
		"ollama", "hermes",
	} {
		if strings.Contains(strings.ToLower(contract.Command), forbidden) {
			t.Fatalf("V14 command opens broad or deferred surface %q: %q", forbidden, contract.Command)
		}
	}
	if next := contracts["AC-V15-BUDGETS-EFFECTS"]; next.Status != "executable" ||
		next.Receipt != "product/evidence/v15_budgets_effects.json" {
		t.Fatalf("V14 successor V15 must advance only through its executable contract: %#v", next)
	}
	for _, id := range []string{"AC-V20-COMMAND-REGISTRY", "AC-V31-POSTGRES-S3-MULTIHOST"} {
		if deferred := contracts[id]; deferred.Status != "planned" || deferred.Receipt != "" {
			t.Fatalf("V14 prematurely opens deferred contract %s: %#v", id, deferred)
		}
	}
}

func TestV14EvidenceBelongsOnlyToControlCapabilities(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	owned := map[string]bool{"GOV-07": true, "STG-15": true, "ORC-03": true, "ORC-16": true}
	v14Evidence := map[string]bool{
		"acceptance/v14_controls_test.go":          true,
		"acceptance/fixtures/v14_controls.json":    true,
		"product/evidence/v14_controls.json":       true,
		"product/evidence/v14_controls.output.txt": true,
	}
	wantEvidence := []string{
		"acceptance/v14_controls_test.go",
		"acceptance/fixtures/v14_controls.json",
		"product/evidence/v14_controls.json",
	}
	for _, entry := range roadmap.CapabilityEntries {
		if owned[entry.ID] {
			if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence) {
				t.Errorf("owned V14 capability %s lacks exact accreditation: status=%q evidence=%v",
					entry.ID, entry.Status, entry.EvidenceRefs)
			}
			continue
		}
		for _, evidenceRef := range entry.EvidenceRefs {
			if v14Evidence[evidenceRef] {
				t.Errorf("unowned capability %s claims V14 evidence %q", entry.ID, evidenceRef)
			}
		}
	}
	wantHistoricalCapabilities := map[string][]string{
		"BUG-ORQ-20260705-197":  {"GOV-07"},
		"BUG-ORQ-20260709-198":  {"STG-15"},
		"BUG-ORQ-20260710-208":  {"ORC-16"},
		"BUG-ORQ-20260710-208C": {"ORC-16"},
		"BUG-ORQ-20260710-208S": {"ORC-03"},
		"BUG-ORQ-20260711-208Z": {"GOV-07"},
		"BUG-ORQ-20260711-239":  {"STG-15"},
		"BUG-ORQ-20260711-240":  {"STG-15"},
		"BUG-ORQ-20260711-243":  {"STG-15"},
		"BUG-ORQ-20260711-260":  {"STG-15"},
		"BUG-ORQ-20260711-261":  {"ORC-16"},
		"BUG-ORQ-20260711-267":  {"ORC-16"},
		"BUG-ORQ-20260711-270":  {"STG-15"},
	}
	for _, bug := range traceReadHistoricalBugIDs(t, "product/traceability/historical_bug_ids.jsonl") {
		wantCapabilities, belongsToV14 := wantHistoricalCapabilities[bug.BugID]
		if !belongsToV14 {
			for _, evidenceRef := range bug.RebuildEvidenceRefs {
				if v14Evidence[evidenceRef] {
					t.Errorf("unowned historical bug %s claims V14 evidence %q", bug.BugID, evidenceRef)
				}
			}
			continue
		}
		wantClosure := "not_verified"
		if !reflect.DeepEqual(bug.VerifiedCapabilityIDs, wantCapabilities) ||
			!reflect.DeepEqual(bug.RebuildEvidenceRefs, wantEvidence) || bug.ClosureEvidence != wantClosure {
			t.Errorf("historical bug %s V14 evidence=%v/%v closure=%q, want %v/%v/%q",
				bug.BugID, bug.VerifiedCapabilityIDs, bug.RebuildEvidenceRefs, bug.ClosureEvidence,
				wantCapabilities, wantEvidence, wantClosure)
		}
		delete(wantHistoricalCapabilities, bug.BugID)
	}
	if len(wantHistoricalCapabilities) != 0 {
		t.Fatalf("historical V14 bugs missing evidence: %v", wantHistoricalCapabilities)
	}
}

func TestV14AcceptanceCommandRunsControlConsumers(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V14-CONTROLS" {
			continue
		}
		for _, required := range []string{
			"./acceptance", "./internal/goal", "./internal/identity", "./internal/config",
			"./internal/credentials", "./internal/application", "./internal/ports",
			"./internal/adapters/agent/fake", "./internal/adapters/agent/codex",
			"./internal/adapters/state/sqlite", "./internal/bootstrap", "./cmd/orquesta",
		} {
			if !roadmapCommandHasArgument(contract.Command, required) {
				t.Errorf("V14 acceptance omits control consumer package %q: %q", required, contract.Command)
			}
		}
		if !strings.Contains(contract.Command, "TestV14CandidateSubjectsCoverCommittedDelta") {
			t.Errorf("V14 acceptance omits candidate delta gate: %q", contract.Command)
		}
		return
	}
	t.Fatal("AC-V14-CONTROLS missing")
}

func roadmapV14Assertions() []string {
	return []string{
		"controls are authenticated request-idempotent application mutations bound to exact project Goal AppSpec generation and hash PlanGeneration WorkItem revision Execution attempt request ref and fingerprint",
		"pause and resume are reversible Goal or WorkItem dispatch gates whose effective union blocks new launch_agent claims while observe_agent mailbox and in-flight work continue",
		"pause before RecordLaunchPrepared invalidates the launch claim while a durable prepared launch remains in flight and records its exact acceptance or rejection before stop can claim",
		"resume reclaims the existing pending action without creating another Execution outbox action or effect",
		"stop targets one exact Execution and generation through the existing outbox scheduler and AgentController and separates stop requested from exact stop confirmed",
		"cooperative and forced stop execute only when adapter capabilities advertise them and unsupported never becomes stopped or invokes global Shutdown",
		"forced stop supersedes only the exact still-active cooperative stop for the same Execution; atomic lineage retires the old action before the new one while completed retired or quarantined owners reject without partial writes",
		"four disjoint A B C D executions prove that stopping B preserves A C D processes state and progress before and after crash restart without global Shutdown",
		"stop requested permits late V13 delivery consume and acknowledgement until confirmation; confirmation retires only unresolved exact-recipient mailbox without readdress synthetic ACK or ChildHandoffResolution",
		"completion and stop race through one CAS; already completed is observed rather than falsified as stopped and every terminal Execution remains immutable and never restarts",
		"execution retry after confirmed stop creates a new Execution ref idempotency key and attempt with replaces_execution_ref while terminal Goal retired mailbox or exhausted attempt policy rejects without effects",
		"V14 execution retry preserves V06 attempt receipts fences and idempotency but does not claim V15 external-effect retry approvals budgets quotas or fairness",
		"cancel is irreversible at Goal or WorkItem scope, blocks new launches and delivery claims after its CAS, cancels queued work locally and stops every exact prepared or running Execution before terminal publication",
		"Goal and WorkItem cancel race with completion by CAS and preserve exact terminal Execution evidence without converting it into another terminal state or leaving an orphan process or launch",
		"WorkItem cancel preserves independent work, derives dependency_canceled skips, fails the resolved Goal, and treats a canceled HandoffRequired endpoint as a causal failure without synthetic resolution",
		"ProposeDirectorPlan remains the only replan entry and requires live Director lease token fence authenticated principal exact revisions generations source WorkItem and causal Execution or assessment evidence",
		"split_pending atomically cancels the queued Execution consumes its launch marks the source superseded and appends one or more successors without a reclaimable source action",
		"exhausted execution attempts leave the last Execution failed and the WorkItem interrupted with cause execution_failed while the Goal stays open and dependents stay pending for replan or cancel",
		"append-only replan records only successor rework_of source relations and recursively derives logical source success or failure across nested successors without rewriting history",
		"replan rejects stale or revoked authority invalid target pairs cycles write-set conflicts already skipped descendants and every HandoffRequired endpoint with zero partial snapshot event outbox receipt or effect",
		"one StateRepository transaction persists control snapshot audit event outbox consumption and receipt while replay returns the same frontier and conflicting request semantics fail",
		"SQLite restart backup restore and races at pause claim cancel launch stop completion retry and replan preserve one fenced WorkItem lease and never repeat a terminal effect",
		"the neutral AgentController contract passes one fake suite and Codex proves exact process-tree stop crash adoption and PID PGID birth-identity checks while clean Shutdown remains separate",
		"Codex persists process identity only in its existing private WorkRoot journal behind an FD3 launch gate holds one CLOEXEC owner.lock and binds local runtime scope to non-backup-clonable StateRepository file identity while distributed ownership remains V31",
		"every SQLite physical connection validates the retained local file identity before configuration so path replacement lazy open missing-path recreation and alternate WAL namespaces fail closed",
		"control receipts expose opaque identities without PID argv environment prompt or secrets and V14 adds no undeclared configuration key",
		"V14 preserves the V02 single writer V05 DAG and dependency rules V06 atomic retries and receipts V07 config V08 credentials V09 recovery V10 RBAC V12 Director and V13 mailbox ratchets without another store scheduler loop daemon database or lifecycle",
		"V14 exposes application use cases only; HTTP MCP CLI command registry and full i18n bindings remain V20 and V21 while budgets effects workspace reviews provider parity generic messages UI and later surfaces remain deferred",
	}
}

func TestProductRoadmapV15ScopeAndExecutableContract(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	verticals := make(map[string]roadmapVertical, len(roadmap.Verticals))
	entries := make(map[string]roadmapEntry, len(roadmap.CapabilityEntries))
	contracts := make(map[string]roadmapAcceptanceContract, len(roadmap.AcceptanceContracts))
	var owned []string
	for _, vertical := range roadmap.Verticals {
		verticals[vertical.ID] = vertical
	}
	for _, entry := range roadmap.CapabilityEntries {
		entries[entry.ID] = entry
		if entry.OwnerContext == "budgets_effects" && entry.Decision == "accept" {
			owned = append(owned, entry.ID)
		}
	}
	for _, contract := range roadmap.AcceptanceContracts {
		contracts[contract.ID] = contract
	}
	sort.Strings(owned)
	wantOwned := []string{"EVD-03", "EVD-14", "GOV-15", "ORC-08", "ORC-09", "ORC-10", "ORC-11", "STG-09"}
	if !reflect.DeepEqual(owned, wantOwned) {
		t.Fatalf("V15 accepted ownership = %v, want exact %v", owned, wantOwned)
	}
	wantVertical := verticals["budgets_effects"]
	wantEvidence := []string{
		"acceptance/v15_budgets_effects_test.go",
		"acceptance/fixtures/v15_budgets_effects.json",
		"product/evidence/v15_budgets_effects.json",
	}
	for _, id := range wantOwned {
		entry := entries[id]
		if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence) ||
			!reflect.DeepEqual(entry.Dependencies, wantVertical.DependsOn) ||
			!reflect.DeepEqual(entry.AcceptanceContracts, wantVertical.AcceptanceContracts) {
			t.Errorf("V15 capability %s lacks exact accreditation: %#v", id, entry)
		}
	}

	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV15ScopeAndExecutableContract|TestV15EvidenceBelongsOnlyToBudgetsEffectsCapabilities|TestV15AcceptanceCommandRunsBudgetEffectConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestTraceabilityRebuildHistoricalBugIDs|TestTraceabilityRebuildHistoricalBugReviewBindings|TestTraceabilityRebuildSchemaValidatesCanonicalLedgers|TestHistoricalBugCapabilityCoverageNeverInfersLegacyClosure|TestAcceptanceV15BudgetsEffects|TestV15CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/governance ./internal/identity ./internal/config ./internal/credentials ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/bootstrap ./cmd/orquesta && timeout --kill-after=10s 150s go test -mod=vendor -race -count=1 -timeout=120s ./internal/application ./internal/adapters/state/sqlite ./internal/adapters/agent/fake ./internal/bootstrap -run \"^(TestBudgetContractUsesOneCanonicalEnvelopeAcrossLayers|TestConcurrentBudgetReservationsNeverExceedEnvelope|TestTemporaryQuotaParksActionWithoutTerminalFailure|TestHierarchicalFairnessBoundsProjectAndGoalStarvation|TestEffectRequiresExactLiveApprovalBeforeAdapterInvocation|TestEffectCrashAfterApplyBeforeReceiptReconcilesOnce|TestPendingStopBackoffPreservesFirstUrgencyThenYieldsAndCaps|TestRepositoryOpenAppliesPrivateModesMigrationsAndPragmas|TestFastSemanticRepositoryRestartPersistsCommittedData|TestSQLiteBudgetsEffectsRestartRaceAndReplay|TestV15RecoveryRejectsBudgetEffectCausalTampering|TestV15BackupRestorePreservesBudgetsAndEffects|TestRealCodexBudgetsAndEffectsThroughProductionComposition)$\"'"
	contract := contracts["AC-V15-BUDGETS-EFFECTS"]
	if contract.Status != "executable" || contract.TestRef != "acceptance/v15_budgets_effects_test.go" ||
		contract.Fixture != "acceptance/fixtures/v15_budgets_effects.json" ||
		contract.Receipt != "product/evidence/v15_budgets_effects.json" ||
		contract.Command != wantCommand || !reflect.DeepEqual(contract.Assertions, roadmapV15Assertions()) {
		t.Fatalf("invalid V15 executable contract: %#v", contract)
	}
	for _, id := range []string{"AC-V16-WORKSPACE-GIT", "AC-V20-COMMAND-REGISTRY", "AC-V31-POSTGRES-S3-MULTIHOST"} {
		if deferred := contracts[id]; deferred.Status != "planned" || deferred.Receipt != "" {
			t.Fatalf("V15 contract preparation prematurely opens deferred contract %s: %#v", id, deferred)
		}
	}
}

func TestV15AcceptanceCommandRunsBudgetEffectConsumers(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	var contract roadmapAcceptanceContract
	for _, candidate := range roadmap.AcceptanceContracts {
		if candidate.ID == "AC-V15-BUDGETS-EFFECTS" {
			contract = candidate
			break
		}
	}
	for _, required := range []string{
		"TestAcceptanceV15BudgetsEffects", "TestV15CandidateSubjectsCoverCommittedDelta",
		"TestTraceabilityRebuildHistoricalBugIDs", "TestTraceabilityRebuildHistoricalBugReviewBindings",
		"TestTraceabilityRebuildSchemaValidatesCanonicalLedgers",
		"TestHistoricalBugCapabilityCoverageNeverInfersLegacyClosure",
		"./internal/governance", "./internal/application", "./internal/adapters/state/sqlite",
		"./internal/adapters/agent/fake", "./internal/adapters/agent/codex", "./internal/bootstrap",
		"TestConcurrentBudgetReservationsNeverExceedEnvelope",
		"TestEffectCrashAfterApplyBeforeReceiptReconcilesOnce",
		"TestPendingStopBackoffPreservesFirstUrgencyThenYieldsAndCaps",
		"TestRepositoryOpenAppliesPrivateModesMigrationsAndPragmas",
		"TestFastSemanticRepositoryRestartPersistsCommittedData",
		"TestRealCodexBudgetsAndEffectsThroughProductionComposition",
		"timeout --kill-after=10s 150s", "-timeout=120s",
	} {
		if !strings.Contains(contract.Command, required) {
			t.Errorf("V15 command does not execute %q", required)
		}
	}
}

func TestV15EvidenceBelongsOnlyToBudgetsEffectsCapabilities(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)
	owned := map[string]bool{
		"GOV-15": true, "STG-09": true, "ORC-08": true, "ORC-09": true,
		"ORC-10": true, "ORC-11": true, "EVD-03": true, "EVD-14": true,
	}
	v15Evidence := map[string]bool{
		"acceptance/v15_budgets_effects_test.go":          true,
		"acceptance/fixtures/v15_budgets_effects.json":    true,
		"product/evidence/v15_budgets_effects.json":       true,
		"product/evidence/v15_budgets_effects.output.txt": true,
	}
	for _, entry := range roadmap.CapabilityEntries {
		if owned[entry.ID] {
			want := []string{
				"acceptance/v15_budgets_effects_test.go",
				"acceptance/fixtures/v15_budgets_effects.json",
				"product/evidence/v15_budgets_effects.json",
			}
			if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, want) {
				t.Errorf("owned V15 capability %s lacks exact accreditation: status=%q evidence=%v",
					entry.ID, entry.Status, entry.EvidenceRefs)
			}
			continue
		}
		for _, evidenceRef := range entry.EvidenceRefs {
			if v15Evidence[evidenceRef] {
				t.Errorf("unowned capability %s claims V15 evidence %q", entry.ID, evidenceRef)
			}
		}
	}
	wantHistoricalEvidence := []string{
		"acceptance/v15_budgets_effects_test.go",
		"acceptance/fixtures/v15_budgets_effects.json",
		"product/evidence/v15_budgets_effects.json",
	}
	foundHistorical := false
	for _, bug := range traceReadHistoricalBugIDs(t, "product/traceability/historical_bug_ids.jsonl") {
		if bug.BugID != "BUG-ORQ-20260706-BUDGET-CONTRACT-DESALINEADO" {
			continue
		}
		foundHistorical = true
		if !reflect.DeepEqual(bug.VerifiedCapabilityIDs, []string{"ORC-09"}) ||
			!reflect.DeepEqual(bug.RebuildEvidenceRefs, wantHistoricalEvidence) ||
			bug.ClosureEvidence != "not_verified" {
			t.Errorf("V15 historical budget evidence/closure invalid: %#v", bug)
		}
	}
	if !foundHistorical {
		t.Fatal("V15 historical budget bug binding missing")
	}
}

func roadmapV15Assertions() []string {
	return []string{
		"GOV-15 STG-09 ORC-08 ORC-09 ORC-10 ORC-11 EVD-03 and EVD-14 are the exact accepted V15 ownership; every other capability and V16 workspace Git collaboration remain deferred",
		"BudgetEnvelope BudgetDemand BudgetReservation BudgetSettlement EffectIntent EffectApproval EffectAttempt and EffectReceipt are distinct typed causal facts and neither an admission ACK nor agent text proves reserved capacity approval attempt or external effect",
		"budget dimensions are typed nonnegative tokens money in minor units wall time process slots and disk bytes with explicit deployment project and Goal scopes; every reservation is charged atomically against all applicable finite envelopes",
		"a Director plan declares WorkItem decomposition dependencies write sets typed budget demand model effort security criticality and intended effects, while authenticated application policy remains the only authority that validates and persists them",
		"concurrent reservations through the single StateRepository transaction never exceed any envelope and exact request replay returns the original reservation without double charging",
		"settlement records observed usage releases only unused reserved capacity never mints capacity and is idempotent across concurrent completion crash restart backup and restore",
		"temporary capacity or provider quota defers eligible work with a durable retry frontier and never marks its Goal WorkItem Execution or effect terminal failed merely because capacity is currently unavailable",
		"fair scheduling across projects and Goals is deterministic and bounded under contention so an eligible nonexhausted contender cannot starve while exhausted or temporarily limited contenders remain durably deferred",
		"the Codex composition exposes canonical configurable defaults of 70 parent executions per scheduler cycle and at most 6 direct children per parent, effective configuration reports both values, and restart preserves them",
		"the neutral core has no provider-named or hidden global concurrency cap; only explicit envelopes configured limits and observed provider runtime or OS limits may reduce dispatch and every reduction is visible as quota evidence rather than silent truncation",
		"security criticality and model reasoning effort are independent typed axes, all valid combinations survive plan persistence, and risk policy uses declared metadata rather than keyword rails or provider-specific heuristics",
		"permission risk budget and approval checks bind the exact authenticated principal project Goal generation WorkItem Execution effect kind target scope payload digest cost envelope revision request ref and idempotency key before an EffectAttempt can be claimed",
		"EffectIntent admission is not approval; approval and denial are immutable explicit decisions with approver authority policy revision expiry and exact scope, and stale revoked denied mismatched or absent approval produces zero adapter calls",
		"agent launch reserves its declared demand before launch preparation and exact cooperative or forced stop remains an authorized safety effect with its own intent attempt and receipt while never degrading to global Shutdown",
		"all agent launch stop and future external effects use the existing application writer single outbox action scheduler claim lease fence and StateRepository rather than a BudgetStore EffectDB private queue second scheduler goroutine daemon or provider policy",
		"an EffectAttempt is persisted before adapter invocation and its terminal EffectReceipt is persisted separately with exact intent approval attempt idempotency outcome and observed usage while receipts expose no credential secret prompt environment argv or local process identity",
		"a crash before or after adapter invocation retries only through the same effect idempotency key and fenced attempt; replay or reconciliation converges on one immutable EffectReceipt and never performs the external effect twice",
		"V15 preserves the V02 single Goal writer V05 DAG V06 atomic outbox V07 canonical configuration V08 credentials V09 recovery V10 RBAC V12 Director lease V13 mailbox and V14 controls without another lifecycle state authority scheduler store database or command surface",
		"V15 exposes application and neutral port contracts only; workspace Git forge artifacts reviews council HTTP MCP CLI web provider parity deploy notifications PostgreSQL S3 and multihost remain owned by their later verticals",
	}
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
