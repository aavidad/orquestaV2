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
	SchemaVersion           int                             `json:"schema_version"`
	Product                 string                          `json:"product"`
	CatalogSize             int                             `json:"catalog_size"`
	SourceHashes            map[string]string               `json:"source_hashes"`
	StatusVocabulary        []string                        `json:"status_vocabulary"`
	DecisionVocabulary      []string                        `json:"decision_vocabulary"`
	ReleaseTargets          []string                        `json:"release_targets"`
	OperatorDecisions       map[string]json.RawMessage      `json:"operator_decisions"`
	ImplementationDecisions []roadmapImplementationDecision `json:"implementation_decisions"`
	Verticals               []roadmapVertical               `json:"verticals"`
	AcceptanceContracts     []roadmapAcceptanceContract     `json:"acceptance_contracts"`
	CapabilityEntries       []roadmapEntry                  `json:"capability_entries"`
	DeferredMappings        []roadmapMapping                `json:"deferred_mappings"`
}

type roadmapImplementationDecision struct {
	ID                    string   `json:"id"`
	Status                string   `json:"status"`
	CapabilityRefs        []string `json:"capability_refs"`
	Transport             string   `json:"transport"`
	AllowedServices       []string `json:"allowed_services"`
	ForbiddenConnectivity []string `json:"forbidden_connectivity"`
	Authentication        string   `json:"authentication"`
	TestAttestorScope     string   `json:"test_attestor_scope"`
	TestRefs              []string `json:"test_refs"`
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
	assertRoadmapImplementationDecisions(t, roadmap.ImplementationDecisions)

	verticals := make(map[string]roadmapVertical, len(roadmap.Verticals))
	if len(roadmap.Verticals) != 38 {
		t.Fatalf("vertical count = %d, want 38", len(roadmap.Verticals))
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
	if len(roadmap.AcceptanceContracts) != 38 {
		t.Fatalf("acceptance contract count = %d, want 38", len(roadmap.AcceptanceContracts))
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

func TestProductRoadmapV38OwnsElasticAgentRuntimeWithoutReopeningPrerequisites(t *testing.T) {
	var roadmap roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &roadmap)

	var vertical roadmapVertical
	var contract roadmapAcceptanceContract
	for _, candidate := range roadmap.Verticals {
		if candidate.ID == "agent_runtime_elastic" {
			vertical = candidate
		}
		if candidate.ID == "V39" {
			t.Fatal("V39 no debe existir")
		}
	}
	for _, candidate := range roadmap.AcceptanceContracts {
		if candidate.ID == "AC-V38-AGENT-RUNTIME-ELASTIC" {
			contract = candidate
		}
		if strings.HasPrefix(candidate.ID, "AC-V39") {
			t.Fatal("no debe existir un contrato V39")
		}
	}
	wantDependencies := []string{
		"config", "credentials", "recovery_backup", "controls",
		"budgets_effects", "workspace_git", "test_attestor", "codex_e2e",
	}
	if vertical.Sequence != 38 || vertical.Title != "Runtime elástico de agentes" ||
		!reflect.DeepEqual(vertical.DependsOn, wantDependencies) ||
		!reflect.DeepEqual(vertical.AcceptanceContracts, []string{"AC-V38-AGENT-RUNTIME-ELASTIC"}) {
		t.Fatalf("V38 no conserva la autoridad prioritaria: %#v", vertical)
	}
	if contract.Vertical != vertical.ID || contract.Status != "planned" ||
		contract.Receipt != "" ||
		contract.TestRef != "planned:acceptance/v38_agent_runtime_elastic_plan_test.go" ||
		contract.Fixture != "planned:fixtures/v38_agent_runtime_elastic_plan.json" {
		t.Fatalf("el contrato V38 anticipa ejecución o evidencia: %#v", contract)
	}
	wantAssertions := []string{
		"ORC-28 trata la capacidad física reservable y la cuota del proveedor como hechos separados; una cuota desconocida o agotada cierra la admisión de nuevas reclamaciones y nunca se convierte en un hueco físico liberado",
		"la cuenta y la colocación opacas se eligen antes de la reclamación atómica; la reserva fija cuenta colocación capacidad cuota lease y fence y el lanzador debe obedecerla sin reselección",
		"runtime.codex.max_concurrent_executions es solo un guardarraíl local del conector Codex y no gobierna el presupuesto ni el despacho global; el techo global es neutral al proveedor y la admisión es atómica",
		"con aislamiento microVM todos los hilos turnos y agentes Codex viven dentro de la microVM; en el anfitrión solo puede persistir un app-server mínimo para identidad y cuota sin hilo turno agente ni autoridad de ejecución",
		"el núcleo persiste por el puerto StateRepository con una sola fuente transaccional activa elegida por composición; SQLite queda para local desarrollo y pruebas PostgreSQL es el adaptador productivo futuro y se prohíbe dual write",
		"la demanda completa conserva cohortes lógicas exactas de 1 16 70 y 500 sin techo oculto; las olas físicas obligatorias progresan por 1 5 10 16 y 20 sin atribuir 70 o 500 físicos sin recursos medidos",
		"el subgate A acredita solo el núcleo elástico neutral sin KVM; B conecta Firecracker opt-in sin fallback y una microVM por agente; C exige ola física del mismo candidato y solo A+B+C permiten acreditar V38",
		"un único despachador global prioriza stop permite progreso de observe con launch saturado y paraleliza únicamente launch_agent sin selector privado solo-launch ni goroutines ociosas",
		"la continuidad de mensajes la parada exacta y la conservación del entorno son comportamientos estrechos de V38 que no acreditan ORC-15 OPS-16 ni OPS-17",
		"antes de desmontar se sellan e inventarían los datos y el entorno queda preserved_pending_review sin borrado automático",
		"la retirada de datos es una decisión posterior, separada y autorizada",
		"restart y recovery cubren antes del intento intento antes del efecto efecto sin recibo recibo antes de observación y stop con recibo perdido; unknown_applied queda en cuarentena y solo definitely_not_applied se reintenta",
		"las latencias de solicitud, aprovisionamiento, arranque, disponibilidad, parada y sellado son medibles",
		"el gate exige eventos JSON run y pass de cada prueba exacta y rechaza paquetes verdes sin tests; este contrato planificado no crea receipt ni evidence",
	}
	if !reflect.DeepEqual(contract.Assertions, wantAssertions) {
		t.Fatalf("las afirmaciones V38 derivaron: %#v", contract.Assertions)
	}
	for _, marker := range []string{
		"TestV38AgentRuntimeElasticPlanMatchesCanonicalRoadmap",
		"TestV38AgentRuntimeElasticPlanRejectsSemanticDrift",
		"TestV38AgentRuntimeElasticPlanRejectsInvalidJSON",
		"TestV38AgentRuntimeElasticPlanRequiresExactRunPassEvidence",
		`"Action":"run"`, `"Action":"pass"`, "for v38_test in",
	} {
		if !strings.Contains(contract.Command, marker) {
			t.Fatalf("el comando V38 no prueba ejecución focal exacta %q: %q", marker, contract.Command)
		}
	}

	owners := map[string]bool{"ORC-28": false}
	for _, entry := range roadmap.CapabilityEntries {
		_, selected := owners[entry.ID]
		if !selected {
			continue
		}
		if entry.OwnerContext != vertical.ID ||
			!reflect.DeepEqual(entry.Dependencies, wantDependencies) ||
			!reflect.DeepEqual(entry.AcceptanceContracts, vertical.AcceptanceContracts) ||
			entry.Status != "declared" || len(entry.EvidenceRefs) != 0 {
			t.Fatalf("%s no pertenece únicamente a V38: %#v", entry.ID, entry)
		}
		owners[entry.ID] = true
	}
	for id, found := range owners {
		if !found {
			t.Fatalf("falta la capacidad V38 %s", id)
		}
	}

	restored := map[string]roadmapEntry{
		"ORC-15": {
			ID: "ORC-15", Title: "Mensajes y handoff entre agentes/sesiones", SourceProposal: "MANTENER",
			Decision: "accept", Kind: "orchestration", ReleaseTarget: "total_v1", CutoverRequired: true,
			OwnerContext: "context_rag_evals", Dependencies: []string{"test_attestor", "provider_adapters", "tools_skills_sdk"},
			AcceptanceContracts: []string{"AC-V27-CONTEXT-RAG-EVALS"}, Status: "declared", EvidenceRefs: []string{}, Supersedes: []string{},
		},
		"OPS-16": {
			ID: "OPS-16", Title: "Shutdown cooperativo y cero procesos propios residuales", SourceProposal: "MANTENER, V1",
			Decision: "accept", Kind: "operations", ReleaseTarget: "total_v1", CutoverRequired: true,
			OwnerContext:        "operations_telemetry",
			Dependencies:        []string{"config", "credentials", "recovery_backup", "identity_projects_rbac", "oidc_ad", "codex_e2e", "command_registry", "deploy_notifications", "postgres_s3_multihost"},
			AcceptanceContracts: []string{"AC-V32-OPERATIONS-TELEMETRY"}, Status: "declared", EvidenceRefs: []string{}, Supersedes: []string{},
		},
		"OPS-17": {
			ID: "OPS-17", Title: "Retención de logs, runtimes, caches y worktrees", SourceProposal: "MANTENER",
			Decision: "accept", Kind: "operations", ReleaseTarget: "total_v1", CutoverRequired: true,
			OwnerContext:        "operations_telemetry",
			Dependencies:        []string{"config", "credentials", "recovery_backup", "identity_projects_rbac", "oidc_ad", "codex_e2e", "command_registry", "deploy_notifications", "postgres_s3_multihost"},
			AcceptanceContracts: []string{"AC-V32-OPERATIONS-TELEMETRY"}, Status: "declared", EvidenceRefs: []string{}, Supersedes: []string{},
		},
	}
	for _, entry := range roadmap.CapabilityEntries {
		want, selected := restored[entry.ID]
		if !selected {
			continue
		}
		if !reflect.DeepEqual(entry, want) {
			t.Fatalf("%s no fue restaurada exactamente a HEAD: %#v", entry.ID, entry)
		}
		delete(restored, entry.ID)
	}
	if len(restored) != 0 {
		t.Fatalf("faltan capacidades globales restauradas: %#v", restored)
	}

	prerequisites := roadmapSetOf("AGT-01", "AGT-03", "GOV-21", "ORC-10", "EVD-13")
	for _, entry := range roadmap.CapabilityEntries {
		if _, required := prerequisites[entry.ID]; !required {
			continue
		}
		if entry.Status != "accredited" || len(entry.EvidenceRefs) == 0 {
			t.Fatalf("el prerrequisito %s se reabrió: %#v", entry.ID, entry)
		}
		delete(prerequisites, entry.ID)
	}
	if len(prerequisites) != 0 {
		t.Fatalf("faltan prerrequisitos acreditados: %#v", prerequisites)
	}
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
	const command = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV05ScopeAndExecutableContract|TestAcceptanceV05GoalDAGPhases|TestV05CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/interfaces/mcp ./internal/bootstrap'"
	index := assertRoadmapVerticalContract(t, roadmapVerticalContractSpec{
		version: "V05", owner: "goal_dag_phases", contractID: "AC-V05-GOAL-DAG-PHASES",
		testRef: "acceptance/v05_goal_dag_phases_test.go", fixture: "acceptance/fixtures/v05_goal_dag_phases.json",
		receipt: "product/evidence/v05_goal_dag_phases.json", command: command, assertionCount: 6,
		owned: []string{"GOV-04", "ORC-01", "ORC-02", "ORC-06", "STG-00"}, evidence: []string{
			"acceptance/v05_goal_dag_phases_test.go", "acceptance/fixtures/v05_goal_dag_phases.json",
			"product/evidence/v05_goal_dag_phases.json",
		},
	})

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
		entry := index.entries[id]
		if entry.OwnerContext != owner || roadmapEvidenceContainsVertical(entry.EvidenceRefs, "v05_goal_dag_phases") {
			t.Errorf("moved capability %s = %#v, want owner %s without V05 evidence", id, entry, owner)
		}
	}
	if entry := index.entries["ORC-18"]; entry.Decision != "reject" || entry.ReleaseTarget != "excluded" || entry.CutoverRequired {
		t.Fatalf("ORC-18 must remain rejected and outside cutover: %#v", entry)
	}
	contract := index.contracts["AC-V05-GOAL-DAG-PHASES"]
	if !roadmapCommandHasArgument(contract.Command, "./acceptance") || strings.Contains(contract.Command, "^TestAcceptance$") {
		t.Fatalf("V05 command can omit real acceptance tests: %q", contract.Command)
	}
}

func TestProductRoadmapV06ScopeAndExecutableContract(t *testing.T) {
	const command = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV06ScopeAndExecutableContract|TestAcceptanceV06AtomicStateOutbox|TestV06CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/interfaces/mcp ./internal/bootstrap ./cmd/orquesta'"
	index := assertRoadmapVerticalContract(t, roadmapVerticalContractSpec{
		version: "V06", owner: "atomic_state_outbox", contractID: "AC-V06-ATOMIC-STATE-OUTBOX",
		testRef: "acceptance/v06_atomic_state_outbox_test.go", fixture: "acceptance/fixtures/v06_atomic_state_outbox.json",
		receipt: "product/evidence/v06_atomic_state_outbox.json", command: command, assertionCount: 9, allowDeclared: true,
		owned: []string{"EVD-02", "GOV-05", "GOV-06", "OPS-09", "OPS-10", "OPS-12", "ORC-12", "ORC-13", "ORC-17"},
		evidence: []string{"acceptance/v06_atomic_state_outbox_test.go", "acceptance/fixtures/v06_atomic_state_outbox.json",
			"product/evidence/v06_atomic_state_outbox.json"},
	})

	wantMoved := map[string]string{
		"GOV-17": "command_registry",
		"EVD-01": "test_attestor",
		"OPS-13": "postgres_s3_multihost",
	}
	for id, owner := range wantMoved {
		entry, vertical := index.entries[id], index.verticals[owner]
		if entry.OwnerContext != owner || !reflect.DeepEqual(entry.Dependencies, vertical.DependsOn) ||
			!reflect.DeepEqual(entry.AcceptanceContracts, vertical.AcceptanceContracts) {
			t.Errorf("V06-deferred capability %s lost exact owner %s: %#v", id, owner, entry)
			continue
		}
		if id == "OPS-13" && (entry.Status != "declared" || len(entry.EvidenceRefs) != 0) {
			t.Errorf("still-deferred V06 capability %s = %#v, want exact owner %s", id, entry, owner)
		}
	}
	assertRoadmapV17Lifecycle(t, index, readRoadmapV17Fixture(t))

	contract := index.contracts["AC-V06-ATOMIC-STATE-OUTBOX"]
	if !roadmapCommandHasArgument(contract.Command, "./acceptance") || strings.Contains(contract.Command, "./...") ||
		strings.Contains(contract.Command, "^TestAcceptance$") {
		t.Fatalf("V06 command is broad or can omit real gates: %q", contract.Command)
	}
}

func TestProductRoadmapV07ScopeAndExecutableContract(t *testing.T) {
	const command = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV07ScopeAndExecutableContract|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV07Config|TestV07CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/config ./internal/adapters/config/effectivefile ./internal/adapters/config/toml ./internal/bootstrap ./cmd/orquesta'"
	index := assertRoadmapVerticalContract(t, roadmapVerticalContractSpec{
		version: "V07", owner: "config", contractID: "AC-V07-CONFIG", testRef: "acceptance/v07_config_test.go",
		fixture: "acceptance/fixtures/v07_config.json", receipt: "product/evidence/v07_config.json",
		command: command, assertionCount: 13, allowDeclared: true,
		owned:    []string{"OPS-01", "OPS-02", "OPS-04", "OPS-05", "OPS-06", "OPS-26", "OPS-27", "OPS-28", "OPS-29", "OPS-30"},
		evidence: []string{"acceptance/v07_config_test.go", "acceptance/fixtures/v07_config.json", "product/evidence/v07_config.json"},
	})
	deferred, wantDeferredVertical := index.entries["OPS-07"], index.verticals["web_admin"]
	if deferred.OwnerContext != "web_admin" ||
		!reflect.DeepEqual(deferred.Dependencies, wantDeferredVertical.DependsOn) ||
		!reflect.DeepEqual(deferred.AcceptanceContracts, wantDeferredVertical.AcceptanceContracts) ||
		deferred.Status != "declared" || len(deferred.EvidenceRefs) != 0 {
		t.Fatalf("OPS-07 must remain wholly deferred to V24 public bindings: %#v", deferred)
	}

	contract := index.contracts["AC-V07-CONFIG"]
	if !roadmapCommandHasArgument(contract.Command, "./acceptance") || strings.Contains(contract.Command, "./...") ||
		strings.Contains(contract.Command, "^TestAcceptance$") || strings.Contains(contract.Command, "./internal/interfaces/mcp") {
		t.Fatalf("V07 command is broad, exposes premature public bindings, or can omit real gates: %q", contract.Command)
	}
}

func TestProductRoadmapV08ScopeAndExecutableContract(t *testing.T) {
	const command = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV08ScopeAndExecutableContract|TestV08AcceptanceCommandRunsCodexCredentialIntegration|TestCredentialRefsMatchConfigAndGoalOpaqueRefs|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV08Credentials|TestV08CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/credentials ./internal/adapters/credentials/local ./internal/adapters/agent/codex ./internal/config ./internal/goal ./internal/bootstrap ./cmd/orquesta'"
	index := assertRoadmapVerticalContract(t, roadmapVerticalContractSpec{
		version: "V08", owner: "credentials", contractID: "AC-V08-CREDENTIALS",
		testRef: "acceptance/v08_credentials_test.go", fixture: "acceptance/fixtures/v08_credentials.json",
		receipt: "product/evidence/v08_credentials.json", command: command, assertionCount: 11, allowDeclared: true,
		owned: []string{"EVD-11", "EVD-12", "OPS-03", "OPS-08"}, evidence: []string{
			"acceptance/v08_credentials_test.go", "acceptance/fixtures/v08_credentials.json", "product/evidence/v08_credentials.json",
		},
	})
	attestor, wantAttestorVertical := index.entries["EVD-13"], index.verticals["test_attestor"]
	if attestor.OwnerContext != "test_attestor" ||
		!reflect.DeepEqual(attestor.Dependencies, wantAttestorVertical.DependsOn) ||
		!reflect.DeepEqual(attestor.AcceptanceContracts, wantAttestorVertical.AcceptanceContracts) {
		t.Fatalf("V08-deferred EVD-13 lost exact V17 ownership: %#v", attestor)
	}
	assertRoadmapV17Lifecycle(t, index, readRoadmapV17Fixture(t))

	contract := index.contracts["AC-V08-CREDENTIALS"]
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
	contract := readRoadmapTestIndex(t).contracts["AC-V08-CREDENTIALS"]
	if !roadmapCommandHasArgument(contract.Command, "./internal/adapters/agent/codex") {
		t.Fatalf("V08 acceptance omits the real credential consumer package: %q", contract.Command)
	}
}

func TestProductRoadmapV09ScopeAndExecutableContract(t *testing.T) {
	const command = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV09ScopeAndExecutableContract|TestV09EvidenceBelongsOnlyToRecoveryCapabilities|TestV09AcceptanceCommandRunsRecoveryConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV09RecoveryBackup|TestV09CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/application ./internal/ports ./internal/adapters/state/sqlite ./internal/adapters/artifact/filesystem ./internal/config ./internal/adapters/config/toml ./internal/adapters/config/jsonimport ./cmd/orquesta ./internal/bootstrap'"
	index := assertRoadmapVerticalContract(t, roadmapVerticalContractSpec{
		version: "V09", owner: "recovery_backup", contractID: "AC-V09-RECOVERY-BACKUP",
		testRef: "acceptance/v09_recovery_backup_test.go", fixture: "acceptance/fixtures/v09_recovery_backup.json",
		receipt: "product/evidence/v09_recovery_backup.json", command: command, assertionCount: 12, allowDeclared: true,
		owned: []string{"EVD-15", "OPS-14"}, evidence: []string{
			"acceptance/v09_recovery_backup_test.go", "acceptance/fixtures/v09_recovery_backup.json",
			"product/evidence/v09_recovery_backup.json",
		},
	})

	deferred := index.entries["OPS-15"]
	wantDeferredVertical := index.verticals["operations_telemetry"]
	if deferred.OwnerContext != "operations_telemetry" ||
		!reflect.DeepEqual(deferred.Dependencies, wantDeferredVertical.DependsOn) ||
		!reflect.DeepEqual(deferred.AcceptanceContracts, wantDeferredVertical.AcceptanceContracts) ||
		deferred.Status != "declared" || len(deferred.EvidenceRefs) != 0 {
		t.Fatalf("OPS-15 must remain wholly deferred to V32 operation and rollback: %#v", deferred)
	}

	contract := index.contracts["AC-V09-RECOVERY-BACKUP"]
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
	assertRoadmapAccreditedEvidenceExclusive(t, []string{"EVD-15", "OPS-14"}, []string{
		"acceptance/v09_recovery_backup_test.go", "acceptance/fixtures/v09_recovery_backup.json",
		"product/evidence/v09_recovery_backup.json",
	})
}

func TestV09AcceptanceCommandRunsRecoveryConsumers(t *testing.T) {
	assertRoadmapCommandArguments(t, readRoadmapTestIndex(t).contracts["AC-V09-RECOVERY-BACKUP"],
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
	)
}

func TestProductRoadmapV10ScopeAndExecutableContract(t *testing.T) {
	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV10ScopeAndExecutableContract|TestV10EvidenceBelongsOnlyToIdentityCapabilities|TestV10AcceptanceCommandRunsIdentityConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV10IdentityProjectsRBAC|TestV10CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/identity ./internal/application ./internal/config ./internal/i18n ./internal/adapters/auth/localtoken ./internal/adapters/state/sqlite ./internal/interfaces/mcp ./internal/bootstrap ./cmd/orquesta'"
	index := assertRoadmapVerticalContract(t, roadmapVerticalContractSpec{
		version: "V10", owner: "identity_projects_rbac", contractID: "AC-V10-IDENTITY-PROJECTS-RBAC",
		testRef: "acceptance/v10_identity_projects_rbac_test.go", fixture: "acceptance/fixtures/v10_identity_projects_rbac.json",
		receipt: "product/evidence/v10_identity_projects_rbac.json", command: wantCommand, assertionCount: 12,
		owned: []string{"GOV-19", "GOV-20", "GOV-22"}, evidence: []string{
			"acceptance/v10_identity_projects_rbac_test.go", "acceptance/fixtures/v10_identity_projects_rbac.json",
			"product/evidence/v10_identity_projects_rbac.json",
		},
	})
	fairness := index.entries["ORC-11"]
	wantV15 := index.verticals["budgets_effects"]
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

	contract := index.contracts["AC-V10-IDENTITY-PROJECTS-RBAC"]
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
	index := readRoadmapTestIndex(t)
	fairness, budgets := index.entries["ORC-11"], index.verticals["budgets_effects"]
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
	want := []string{"acceptance/v10_identity_projects_rbac_test.go",
		"acceptance/fixtures/v10_identity_projects_rbac.json", "product/evidence/v10_identity_projects_rbac.json"}
	assertRoadmapAccreditedEvidenceExclusive(t, []string{"GOV-19", "GOV-20", "GOV-22"}, want)
}

func TestV10AcceptanceCommandRunsIdentityConsumers(t *testing.T) {
	assertRoadmapCommandArguments(t, readRoadmapTestIndex(t).contracts["AC-V10-IDENTITY-PROJECTS-RBAC"],
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
	)
}

func TestProductRoadmapV11ScopeAndExecutableContract(t *testing.T) {
	index := readRoadmapTestIndex(t)
	for _, entry := range index.document.CapabilityEntries {
		if entry.OwnerContext == "oidc_ad" {
			t.Fatalf("V11 is a transverse integration vertical and must not invent a capability ID: %#v", entry)
		}
	}
	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV11ScopeAndExecutableContract|TestV11OwnsNoCapabilityIDs|TestV11AcceptanceCommandRunsIdentityConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV11OIDCAD|TestV11CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/identity ./internal/config ./internal/adapters/auth/bearer ./internal/adapters/auth/localtoken ./internal/adapters/auth/oidc ./internal/bootstrap ./cmd/orquesta && ./scripts/smoke_v11_dex_samba_ad.sh'"
	contract := index.contracts["AC-V11-OIDC-AD"]
	if contract.Status != "executable" || contract.TestRef != "acceptance/v11_oidc_ad_test.go" ||
		contract.Fixture != "acceptance/fixtures/v11_oidc_ad.json" ||
		contract.Receipt != "product/evidence/v11_oidc_ad.json" || contract.Command != wantCommand ||
		len(contract.Assertions) != 16 {
		t.Fatalf("invalid V11 executable contract: %#v", contract)
	}
}

func TestV11OwnsNoCapabilityIDs(t *testing.T) {
	roadmap := readRoadmapTestIndex(t).document
	evidence := roadmapSetOf("acceptance/v11_oidc_ad_test.go", "acceptance/fixtures/v11_oidc_ad.json",
		"product/evidence/v11_oidc_ad.json")
	for _, entry := range roadmap.CapabilityEntries {
		if entry.OwnerContext == "oidc_ad" {
			t.Errorf("capability %s is incorrectly owned by transverse V11", entry.ID)
		}
		for _, evidenceRef := range entry.EvidenceRefs {
			if _, ok := evidence[evidenceRef]; ok {
				t.Errorf("capability %s incorrectly claims transverse V11 evidence %q", entry.ID, evidenceRef)
			}
		}
	}
}

func TestV11AcceptanceCommandRunsIdentityConsumers(t *testing.T) {
	assertRoadmapCommandArguments(t, readRoadmapTestIndex(t).contracts["AC-V11-OIDC-AD"],
		"./acceptance", "./internal/identity", "./internal/config",
		"./internal/adapters/auth/bearer", "./internal/adapters/auth/localtoken",
		"./internal/adapters/auth/oidc", "./internal/bootstrap", "./cmd/orquesta",
		"./scripts/smoke_v11_dex_samba_ad.sh",
	)
}

func TestProductRoadmapV12ScopeAndExecutableContract(t *testing.T) {
	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV12ScopeAndExecutableContract|TestV12EvidenceBelongsOnlyToDirectorCapabilities|TestV12AcceptanceCommandRunsDirectorConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV12DirectorLease|TestV12CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/identity ./internal/application ./internal/config ./internal/adapters/state/sqlite ./internal/bootstrap ./cmd/orquesta'"
	assertRoadmapVerticalContract(t, roadmapVerticalContractSpec{
		version: "V12", owner: "director_lease", contractID: "AC-V12-DIRECTOR-LEASE",
		testRef: "acceptance/v12_director_lease_test.go", fixture: "acceptance/fixtures/v12_director_lease.json",
		receipt: "product/evidence/v12_director_lease.json", command: wantCommand, assertionCount: 14,
		owned: []string{"GOV-08", "GOV-09", "GOV-10", "ORC-24"}, evidence: []string{
			"acceptance/v12_director_lease_test.go", "acceptance/fixtures/v12_director_lease.json",
			"product/evidence/v12_director_lease.json",
		},
	})
}

func TestV12EvidenceBelongsOnlyToDirectorCapabilities(t *testing.T) {
	wantEvidence := []string{
		"acceptance/v12_director_lease_test.go",
		"acceptance/fixtures/v12_director_lease.json",
		"product/evidence/v12_director_lease.json",
	}
	assertRoadmapAccreditedEvidenceExclusive(t, []string{"GOV-08", "GOV-09", "GOV-10", "ORC-24"}, wantEvidence)
}

func TestV12AcceptanceCommandRunsDirectorConsumers(t *testing.T) {
	assertRoadmapCommandArguments(t, readRoadmapTestIndex(t).contracts["AC-V12-DIRECTOR-LEASE"],
		"./acceptance", "./internal/goal", "./internal/identity", "./internal/application",
		"./internal/config", "./internal/adapters/state/sqlite", "./internal/bootstrap", "./cmd/orquesta",
	)
}

func TestProductRoadmapV13ScopeAndExecutableContract(t *testing.T) {
	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV13ScopeAndExecutableContract|TestV13EvidenceBelongsOnlyToMailboxCapabilities|TestV13AcceptanceCommandRunsMailboxConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV13Mailbox|TestV13CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/identity ./internal/config ./internal/application ./internal/ports ./internal/adapters/state/sqlite ./internal/bootstrap ./cmd/orquesta'"
	index := assertRoadmapVerticalContract(t, roadmapVerticalContractSpec{
		version: "V13", owner: "mailbox", contractID: "AC-V13-MAILBOX",
		testRef: "acceptance/v13_mailbox_test.go", fixture: "acceptance/fixtures/v13_mailbox.json",
		receipt: "product/evidence/v13_mailbox.json", command: wantCommand, assertions: roadmapV13Assertions(), causal: true,
		owned: []string{"ORC-04", "ORC-05", "ORC-14"}, evidence: []string{
			"acceptance/v13_mailbox_test.go", "acceptance/fixtures/v13_mailbox.json", "product/evidence/v13_mailbox.json",
		},
	})
	contract := index.contracts["AC-V13-MAILBOX"]
	for _, forbidden := range []string{
		"./...", "/codex", "oidc", "ldap", "workspace", "git", "budget", "fairness",
		"/web", "postgres", "s3", "multihost", "pause", "resume", "cancel", "stop",
	} {
		if strings.Contains(strings.ToLower(contract.Command), forbidden) {
			t.Fatalf("V13 command opens broad or deferred surface %q: %q", forbidden, contract.Command)
		}
	}
}

func TestV13EvidenceBelongsOnlyToMailboxCapabilities(t *testing.T) {
	wantEvidence := []string{
		"acceptance/v13_mailbox_test.go",
		"acceptance/fixtures/v13_mailbox.json",
		"product/evidence/v13_mailbox.json",
	}
	assertRoadmapAccreditedEvidenceExclusive(t, []string{"ORC-04", "ORC-05", "ORC-14"}, wantEvidence)
}

func TestV13AcceptanceCommandRunsMailboxConsumers(t *testing.T) {
	assertRoadmapCommandArguments(t, readRoadmapTestIndex(t).contracts["AC-V13-MAILBOX"],
		"./acceptance", "./internal/goal", "./internal/identity", "./internal/application",
		"./internal/config", "./internal/ports", "./internal/adapters/state/sqlite", "./internal/bootstrap", "./cmd/orquesta",
	)
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
	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV14ScopeAndExecutableContract|TestV14EvidenceBelongsOnlyToControlCapabilities|TestV14AcceptanceCommandRunsControlConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV14Controls|TestV14CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/identity ./internal/config ./internal/credentials ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/bootstrap ./cmd/orquesta && go test -mod=vendor -race -count=1 ./internal/application ./internal/adapters/state/sqlite ./internal/adapters/agent/codex ./internal/bootstrap -run \"^(TestConcurrentIdenticalControlCASLoserReturnsExactReplay|TestControlsGoalAndWorkItemCancelCompletionCASBothOrders|TestControlsStopCompletionCASAndUnsupportedMode|TestControlsStopCrashReplayConvergesWithoutDuplicateEffect|TestClaimedRetryRevalidatesPauseBeforeLaunchPreparation|TestClaimedAutomaticReplacementRevalidatesPauseBeforeLaunchPreparation|TestSQLiteControlsRestartAndConcurrentCAS|TestSQLiteForcedStopSupersessionIsAtomicConcurrentAndRestartSafe|TestSQLiteTerminalStopSettlesAfterRestartWithReplacementAgentRouting|TestV14RecoveryAcceptsClaimedTerminalStopThenReclaimsAndSettlesOnce|TestCodexSelectiveStopPreservesSiblingProcessTrees|TestCodexSelectiveStopAdoptsAfterCrashAndRejectsReusedPID|TestCodexLaunchGateCrashNeverOrphansProcess|TestCodexOwnerLockIsExclusiveAndCLOEXEC|TestCodexRestoredDatabaseCannotAdoptSourceProcess|TestBuildBindsAgentToOpenedRepositoryIdentity|TestRealCodexControlsThroughProductionComposition|TestRealCodexCooperativeStopLeavesResidentSchedulerLive)$\"'"
	index := assertRoadmapVerticalContract(t, roadmapVerticalContractSpec{
		version: "V14", owner: "controls", contractID: "AC-V14-CONTROLS",
		testRef: "acceptance/v14_controls_test.go", fixture: "acceptance/fixtures/v14_controls.json",
		receipt: "product/evidence/v14_controls.json", command: wantCommand, assertions: roadmapV14Assertions(), causal: true,
		owned: []string{"GOV-07", "ORC-03", "ORC-16", "STG-15"}, evidence: []string{
			"acceptance/v14_controls_test.go", "acceptance/fixtures/v14_controls.json", "product/evidence/v14_controls.json",
		},
	})
	contract := index.contracts["AC-V14-CONTROLS"]
	for _, forbidden := range []string{
		"./...", "./internal/interfaces/mcp", "oidc", "ldap", "workspace", "git", "budget",
		"fairness", "effects", "/web", "postgres", "s3", "multihost", "claude", "gemini",
		"ollama", "hermes",
	} {
		if strings.Contains(strings.ToLower(contract.Command), forbidden) {
			t.Fatalf("V14 command opens broad or deferred surface %q: %q", forbidden, contract.Command)
		}
	}
	if next := index.contracts["AC-V15-BUDGETS-EFFECTS"]; next.Status != "executable" ||
		next.Receipt != "product/evidence/v15_budgets_effects.json" {
		t.Fatalf("V14 successor V15 must advance only through its executable contract: %#v", next)
	}
	assertRoadmapV20Lifecycle(t, index, readRoadmapV20Fixture(t))
	if deferred := index.contracts["AC-V31-POSTGRES-S3-MULTIHOST"]; deferred.Status != "planned" || deferred.Receipt != "" {
		t.Fatalf("V14 prematurely opens deferred contract AC-V31-POSTGRES-S3-MULTIHOST: %#v", deferred)
	}
}

func TestV14EvidenceBelongsOnlyToControlCapabilities(t *testing.T) {
	roadmap := readRoadmapTestIndex(t).document
	wantEvidence := []string{
		"acceptance/v14_controls_test.go",
		"acceptance/fixtures/v14_controls.json",
		"product/evidence/v14_controls.json",
	}
	v14Evidence := roadmapSetOf(append(wantEvidence, "product/evidence/v14_controls.output.txt")...)
	assertRoadmapEvidenceExclusive(t, roadmap.CapabilityEntries,
		roadmapSetOf("GOV-07", "STG-15", "ORC-03", "ORC-16"), v14Evidence,
		func(t *testing.T, entry roadmapEntry) {
			if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence) {
				t.Errorf("owned V14 capability %s lacks exact accreditation: status=%q evidence=%v",
					entry.ID, entry.Status, entry.EvidenceRefs)
			}
		})
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
				if _, ok := v14Evidence[evidenceRef]; ok {
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
	contract := readRoadmapTestIndex(t).contracts["AC-V14-CONTROLS"]
	assertRoadmapCommandArguments(t, contract,
		"./acceptance", "./internal/goal", "./internal/identity", "./internal/config",
		"./internal/credentials", "./internal/application", "./internal/ports",
		"./internal/adapters/agent/fake", "./internal/adapters/agent/codex",
		"./internal/adapters/state/sqlite", "./internal/bootstrap", "./cmd/orquesta",
	)
	if !strings.Contains(contract.Command, "TestV14CandidateSubjectsCoverCommittedDelta") {
		t.Errorf("V14 acceptance omits candidate delta gate: %q", contract.Command)
	}
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
	const wantCommand = "sh -c 'go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV15ScopeAndExecutableContract|TestV15EvidenceBelongsOnlyToBudgetsEffectsCapabilities|TestV15AcceptanceCommandRunsBudgetEffectConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestTraceabilityRebuildHistoricalBugIDs|TestTraceabilityRebuildHistoricalBugReviewBindings|TestTraceabilityRebuildSchemaValidatesCanonicalLedgers|TestHistoricalBugCapabilityCoverageNeverInfersLegacyClosure|TestAcceptanceV15BudgetsEffects|TestV15CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/governance ./internal/identity ./internal/config ./internal/credentials ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/bootstrap ./cmd/orquesta && timeout --kill-after=10s 150s go test -mod=vendor -race -count=1 -timeout=120s ./internal/application ./internal/adapters/state/sqlite ./internal/adapters/agent/fake ./internal/bootstrap -run \"^(TestBudgetContractUsesOneCanonicalEnvelopeAcrossLayers|TestConcurrentBudgetReservationsNeverExceedEnvelope|TestTemporaryQuotaParksActionWithoutTerminalFailure|TestHierarchicalFairnessBoundsProjectAndGoalStarvation|TestEffectRequiresExactLiveApprovalBeforeAdapterInvocation|TestEffectCrashAfterApplyBeforeReceiptReconcilesOnce|TestPendingStopBackoffPreservesFirstUrgencyThenYieldsAndCaps|TestRepositoryOpenAppliesPrivateModesMigrationsAndPragmas|TestFastSemanticRepositoryRestartPersistsCommittedData|TestSQLiteBudgetsEffectsRestartRaceAndReplay|TestV15RecoveryRejectsBudgetEffectCausalTampering|TestV15BackupRestorePreservesBudgetsAndEffects|TestRealCodexBudgetsAndEffectsThroughProductionComposition)$\"'"
	index := assertRoadmapVerticalContract(t, roadmapVerticalContractSpec{
		version: "V15", owner: "budgets_effects", contractID: "AC-V15-BUDGETS-EFFECTS",
		testRef: "acceptance/v15_budgets_effects_test.go", fixture: "acceptance/fixtures/v15_budgets_effects.json",
		receipt: "product/evidence/v15_budgets_effects.json", command: wantCommand, assertions: roadmapV15Assertions(), causal: true,
		owned: []string{"EVD-03", "EVD-14", "GOV-15", "ORC-08", "ORC-09", "ORC-10", "ORC-11", "STG-09"},
		evidence: []string{"acceptance/v15_budgets_effects_test.go", "acceptance/fixtures/v15_budgets_effects.json",
			"product/evidence/v15_budgets_effects.json"},
	})
	next := index.contracts["AC-V16-WORKSPACE-GIT"]
	if next.Status != "executable" || next.Receipt != "product/evidence/v16_workspace_git.json" {
		t.Fatalf("V15 successor V16 must advance only through its executable contract: %#v", next)
	}
	assertRoadmapV20Lifecycle(t, index, readRoadmapV20Fixture(t))
	if deferred := index.contracts["AC-V31-POSTGRES-S3-MULTIHOST"]; deferred.Status != "planned" || deferred.Receipt != "" {
		t.Fatalf("V15 contract preparation prematurely opens deferred contract AC-V31-POSTGRES-S3-MULTIHOST: %#v", deferred)
	}
}

func TestV15AcceptanceCommandRunsBudgetEffectConsumers(t *testing.T) {
	assertRoadmapCommandContains(t, readRoadmapTestIndex(t).contracts["AC-V15-BUDGETS-EFFECTS"], nil,
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
	)
}

func TestV15EvidenceBelongsOnlyToBudgetsEffectsCapabilities(t *testing.T) {
	roadmap := readRoadmapTestIndex(t).document
	want := []string{
		"acceptance/v15_budgets_effects_test.go",
		"acceptance/fixtures/v15_budgets_effects.json",
		"product/evidence/v15_budgets_effects.json",
	}
	v15Evidence := roadmapSetOf(append(want, "product/evidence/v15_budgets_effects.output.txt")...)
	assertRoadmapEvidenceExclusive(t, roadmap.CapabilityEntries,
		roadmapSetOf("GOV-15", "STG-09", "ORC-08", "ORC-09", "ORC-10", "ORC-11", "EVD-03", "EVD-14"),
		v15Evidence, func(t *testing.T, entry roadmapEntry) {
			if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, want) {
				t.Errorf("owned V15 capability %s lacks exact accreditation: status=%q evidence=%v",
					entry.ID, entry.Status, entry.EvidenceRefs)
			}
		})
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

func TestProductRoadmapV16ScopeAndExecutableContract(t *testing.T) {
	index := assertRoadmapVerticalContract(t, roadmapVerticalContractSpec{
		version: "V16", owner: "workspace_git", contractID: "AC-V16-WORKSPACE-GIT",
		testRef: "acceptance/v16_workspace_git_test.go", fixture: "acceptance/fixtures/v16_workspace_git.json",
		receipt: "product/evidence/v16_workspace_git.json", command: roadmapV16ValidationCommand(),
		assertions: roadmapV16Assertions(), causal: true, owned: []string{"EXT-10", "STG-02", "STG-10"},
		evidence: []string{"acceptance/v16_workspace_git_test.go", "acceptance/fixtures/v16_workspace_git.json",
			"product/evidence/v16_workspace_git.json"},
	})
	assertRoadmapV17Lifecycle(t, index, readRoadmapV17Fixture(t))
	if forge := index.entries["EXT-11"]; forge.Status != "declared" || forge.OwnerContext != "domain_plugins" ||
		len(forge.EvidenceRefs) != 0 {
		t.Fatalf("V16 must leave remote forge capability deferred: %#v", forge)
	}
}

func TestV16AcceptanceCommandRunsWorkspaceGitConsumers(t *testing.T) {
	contract := readRoadmapTestIndex(t).contracts["AC-V16-WORKSPACE-GIT"]
	if contract.Command == "" {
		t.Fatal("AC-V16-WORKSPACE-GIT missing")
	}
	assertRoadmapCommandContains(t, contract, []string{"./..."},
		"git diff --check 3820df2ae89f1a217de1b14d5b88abf8e86c898b HEAD --",
		"TestAcceptanceV16WorkspaceGit", "TestV16CandidateSubjectsCoverCommittedDelta",
		"TestAcceptanceV02AuthorityRules", "TestAcceptanceV03CanonicalLedgers",
		"TestAcceptanceV06AtomicStateOutbox", "TestAcceptanceV09RecoveryBackup",
		"TestAcceptanceV10IdentityProjectsRBAC",
		"TestProductRoadmapV16ScopeAndExecutableContract", "TestV16EvidenceBelongsOnlyToWorkspaceGitCapabilities",
		"TestV16AcceptanceCommandRunsWorkspaceGitConsumers", "./internal/goal", "./internal/governance",
		"./internal/identity", "./internal/config", "./internal/credentials", "./internal/application",
		"./internal/ports", "./internal/adapters/agent/fake", "./internal/adapters/agent/codex",
		"./internal/adapters/state/sqlite", "./internal/adapters/workspace/gitlocal", "./internal/bootstrap",
		"./cmd/orquesta", "timeout --kill-after=10s 180s", "-timeout=150s", "-race",
		"TestWorkspaceConcurrentPrepareCommitIntegrateRace", "TestWorkspaceEffectsReplayEveryCrashFrontierExactlyOnce",
		"TestRecoveryV16RejectsWorkspaceCausalTampering", "TestRealGitSQLiteWorkspaceLifecycleEndToEnd",
		"TestRecoveryV16RejectsMissingIntegrationFacts",
		"TestIntegrateChangeRejectsMalformedTargetBeforeAuthorizationOrAdmission", "TestValidateGitOIDRejectsMalformedValues",
		"TestGitWorkspaceRejectsRepositoryOverlappingPrivateRoot",
		"TestIntegrationReplayRejectsSameKeyWithDifferentPayload",
		"GOFLAGS=-mod=vendor go vet",
	)
}

func TestV16EvidenceBelongsOnlyToWorkspaceGitCapabilities(t *testing.T) {
	wantEvidence := []string{
		"acceptance/v16_workspace_git_test.go",
		"acceptance/fixtures/v16_workspace_git.json",
		"product/evidence/v16_workspace_git.json",
	}
	assertRoadmapAccreditedEvidenceExclusive(t, []string{"STG-02", "STG-10", "EXT-10"}, wantEvidence,
		"product/evidence/v16_workspace_git.output.txt")
}

func roadmapV16ValidationCommand() string {
	return "sh -c 'git diff --check 3820df2ae89f1a217de1b14d5b88abf8e86c898b HEAD -- && go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV16ScopeAndExecutableContract|TestV16EvidenceBelongsOnlyToWorkspaceGitCapabilities|TestV16AcceptanceCommandRunsWorkspaceGitConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestTraceabilityRebuildHistoricalBugIDs|TestTraceabilityRebuildHistoricalBugReviewBindings|TestTraceabilityRebuildSchemaValidatesCanonicalLedgers|TestHistoricalBugCapabilityCoverageNeverInfersLegacyClosure|TestAcceptanceV02AuthorityRules|TestAcceptanceV03CanonicalLedgers|TestAcceptanceV06AtomicStateOutbox|TestAcceptanceV09RecoveryBackup|TestAcceptanceV10IdentityProjectsRBAC|TestAcceptanceV16WorkspaceGit|TestV16CandidateSubjectsCoverCommittedDelta)$\" && go test -mod=vendor -count=1 ./internal/goal ./internal/governance ./internal/identity ./internal/config ./internal/credentials ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/adapters/workspace/gitlocal ./internal/bootstrap ./cmd/orquesta && timeout --kill-after=10s 180s go test -mod=vendor -race -count=1 -timeout=150s ./internal/application ./internal/ports ./internal/adapters/state/sqlite ./internal/adapters/workspace/gitlocal ./internal/adapters/agent/codex ./internal/bootstrap -run \"^(TestWorkspacePrepareIsIdempotentAndUniquePerExecution|TestReplacementExecutionGetsDistinctWorkspace|TestLaunchUsesExactOpaqueWorkspaceBinding|TestCommitBindsBaseTreeDiffWriteSetAndExecution|TestOutOfWriteSetChangeLeavesGitUnmodified|TestReworkRequiresExplicitParentChangeRef|TestConflictAndStaleIntegrationLeaveTargetUnchanged|TestConcurrentIntegrationCASPreservesLoserPending|TestWorkspaceEffectsReplayEveryCrashFrontierExactlyOnce|TestPendingChangesAreRBACScopedAndSurviveRestart|TestGitWorkspaceRejectsUnsafeFilesystemAndGitControls|TestGitWorkspaceRejectsRepositoryOverlappingPrivateRoot|TestIntegrationReplayRejectsSameKeyWithDifferentPayload|TestWorkspaceEvidenceLeaksNoPrivateAdapterData|TestWorkspaceArchitectureKeepsOneWriterStateOutboxScheduler|TestRealGitSQLiteWorkspaceLifecycleEndToEnd|TestWorkspaceConcurrentPrepareCommitIntegrateRace|TestSQLiteWorkspaceGitRestartRaceAndReplay|TestRecoveryV16RejectsWorkspaceCausalTampering|TestRecoveryV16RejectsMissingIntegrationFacts|TestIntegrateChangeRejectsMalformedTargetBeforeAuthorizationOrAdmission|TestValidateGitOIDRejectsMalformedValues)$\" && GOFLAGS=-mod=vendor go vet ./internal/goal ./internal/governance ./internal/identity ./internal/config ./internal/credentials ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/adapters/workspace/gitlocal ./internal/bootstrap ./cmd/orquesta'"
}

func roadmapV16Assertions() []string {
	return []string{
		"STG-02 STG-10 and EXT-10 are the exact accepted V16 ownership; remote forge push pull request and EXT-11 remain deferred to V28",
		"one private isolated workspace with opaque identity target and exact base object exists per project Goal WorkItem and Execution and a replacement Execution never reuses it",
		"a WorkItem without a write set stays on the non-workspace path while a write-scoped launch receives only its exact opaque ExecutionWorkspaceRef",
		"workspace preparation commit and integration reuse the V06 outbox claim lease fence scheduler and StateRepository plus the V15 intent approval attempt receipt ledger without a second lifecycle store queue daemon or writer",
		"WorkspaceBinding ChangeSet MergeObservation and IntegrationReceipt are immutable causal facts bound to exact actor project repository Goal WorkItem Execution attempt generations action intent attempt fence and receipt",
		"Codex and every interchangeable agent run only in the workspace resolved for the exact Execution and work outside the canonical write set produces zero Git mutation",
		"commit binds exact base parent head tree diff changed paths write-set digest execution and idempotency key while explicit rework names its immutable parent ChangeSet",
		"commit replay and restart converge only on the deterministic object for the exact parent tree causal digest identity timestamp and message; an arbitrary child conflicts and no receipt exposes filesystem paths commands URLs credentials environment provider details or Git private metadata",
		"integration first records a nonmutating merge observation and updates target plus idempotency marker only through one expected-object Git ref transaction",
		"conflict stale base malformed object stale fence and concurrent CAS loser preserve target and ChangeSet as pending work rather than discarding or force-updating it",
		"pending committed conflicted and stale work is durably queryable only through application authorization scoped by principal actor project repository Goal and execution after restart",
		"every external crash frontier preserves one effect idempotency key and stable semantic payload while retry AttemptRef and fence may advance as separately persisted envelopes; recovery rejects cross-linked workspace change integration or receipt facts",
		"the Git CLI adapter uses argv-only machine-readable commands explicit object IDs disabled ambient Git controls and private filesystem roots while domain and application receive only opaque refs OIDs and digests",
		"traversal unsafe ancestors links special files foreign ownership group or world writable roots and metadata foreign Git common directories hooks filters pagers credential helpers signing programs and inherited Git environment fail before protected refs mutate",
		"workspace.local.root is defined once in the canonical registry validated disjoint from all state secret artifact runtime and source roots and consumed through typed configuration",
		"V16 preserves the V02 single Goal writer V05 DAG V06 atomic state and outbox V07 configuration V08 credentials V09 recovery V10 RBAC V12 Director V13 mailbox V14 controls and V15 budgets and effects",
		"local Git integration is an explicit application use case and receipt but never closes a Goal authorizes review publication or proves remote effect; attestation review council forge and multihost remain later verticals",
	}
}

type roadmapV17Fixture struct {
	ProductDeltaSealedGitCommitOID string                    `json:"product_delta_sealed_git_commit_oid"`
	SealStatus                     string                    `json:"seal_status"`
	Command                        string                    `json:"command"`
	ExecutionArgv                  []string                  `json:"execution_argv"`
	OutputPath                     string                    `json:"output_path"`
	ReceiptPath                    string                    `json:"receipt_path"`
	OwnedCapabilityIDs             []string                  `json:"owned_capability_ids"`
	DependencyVerticals            []string                  `json:"dependency_verticals"`
	RequiredBehaviorTests          []roadmapV17BehaviorTest  `json:"required_behavior_tests"`
	FocalSuite                     roadmapV17ValidationSuite `json:"focal_suite"`
	RaceSuite                      roadmapV17ValidationSuite `json:"race_suite"`
	E2ESuite                       roadmapV17ValidationSuite `json:"e2e_suite"`
	VetPackages                    []string                  `json:"vet_packages"`
	RoadmapAssertions              []string                  `json:"roadmap_assertions"`
}

type roadmapV17BehaviorTest struct {
	Directory string `json:"directory"`
	Name      string `json:"name"`
	Gate      string `json:"gate"`
}

type roadmapV17ValidationSuite struct {
	Packages []string `json:"packages"`
	Tests    []string `json:"tests"`
}

func TestProductRoadmapV17ScopeAndExecutableContract(t *testing.T) {
	index := readRoadmapTestIndex(t)
	fixture := readRoadmapV17Fixture(t)
	assertRoadmapV17Lifecycle(t, index, fixture)
}

func TestHistoricalRoadmapScopesAcceptOnlyExactV17Lifecycle(t *testing.T) {
	assertRoadmapV17Lifecycle(t, readRoadmapTestIndex(t), readRoadmapV17Fixture(t))
}

func assertRoadmapV17Lifecycle(t *testing.T, index roadmapTestIndex, fixture roadmapV17Fixture) {
	t.Helper()
	var owned []string
	for _, entry := range index.document.CapabilityEntries {
		if entry.OwnerContext == "test_attestor" && entry.Decision == "accept" {
			owned = append(owned, entry.ID)
		}
	}
	sort.Strings(owned)
	if !reflect.DeepEqual(owned, fixture.OwnedCapabilityIDs) {
		t.Fatalf("V17 accepted ownership = %v, want exact %v", owned, fixture.OwnedCapabilityIDs)
	}
	wantEvidence := []string{
		"acceptance/v17_test_attestor_test.go",
		"acceptance/fixtures/v17_test_attestor.json",
		"product/evidence/v17_test_attestor.json",
	}
	_, receiptErr := os.Stat(fixture.ReceiptPath)
	receiptExists := receiptErr == nil
	if receiptErr != nil && !os.IsNotExist(receiptErr) {
		t.Fatal(receiptErr)
	}
	for _, id := range fixture.OwnedCapabilityIDs {
		entry := index.entries[id]
		if !reflect.DeepEqual(entry.Dependencies, fixture.DependencyVerticals) {
			t.Errorf("V17 capability %s dependencies=%v, want %v", id, entry.Dependencies, fixture.DependencyVerticals)
		}
		if receiptExists {
			if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence) {
				t.Errorf("V17 capability %s lacks exact post-E accreditation: %#v", id, entry)
			}
		} else if entry.Status != "declared" || len(entry.EvidenceRefs) != 0 {
			t.Errorf("V17 capability %s is not exactly deferred before E: %#v", id, entry)
		}
	}
	contract := index.contracts["AC-V17-TEST-ATTESTOR"]
	if receiptExists {
		if contract.Status != "executable" || contract.TestRef != "acceptance/v17_test_attestor_test.go" ||
			contract.Command != fixture.Command || contract.Fixture != "acceptance/fixtures/v17_test_attestor.json" ||
			contract.Receipt != fixture.ReceiptPath || !reflect.DeepEqual(contract.Assertions, fixture.RoadmapAssertions) {
			t.Fatalf("invalid V17 post-E executable contract: %#v", contract)
		}
	} else {
		if contract.Status != "planned" || !strings.HasPrefix(contract.TestRef, "planned:") ||
			!strings.HasPrefix(contract.Command, "planned:") || !strings.HasPrefix(contract.Fixture, "planned:") ||
			contract.Receipt != "" {
			t.Fatalf("invalid V17 pre-E planned contract: %#v", contract)
		}
	}
	assertRoadmapV18Lifecycle(t, index, readRoadmapV18Fixture(t))
}

func TestV17AcceptanceCommandRunsTestAttestorConsumers(t *testing.T) {
	index := readRoadmapTestIndex(t)
	fixture := readRoadmapV17Fixture(t)
	contract := index.contracts["AC-V17-TEST-ATTESTOR"]
	command := fixture.Command
	if contract.Status == "executable" && contract.Command != command {
		t.Fatalf("V17 executable roadmap command differs from fixture argv: roadmap=%q fixture=%q", contract.Command, command)
	}
	assertRoadmapCommandContains(t, roadmapAcceptanceContract{ID: contract.ID, Command: command},
		[]string{"./...", "./internal/..."},
		"git diff --check eb272b6645928d800619709c9afd272440b0dabf HEAD --",
		"TestProductRoadmapV17ScopeAndExecutableContract",
		"TestV17EvidenceBelongsOnlyToTestAttestorCapabilities",
		"TestV17AcceptanceCommandRunsTestAttestorConsumers",
		"TestAcceptanceV17TestAttestor", "TestAcceptanceV06AtomicStateOutbox",
		"TestAcceptanceV09RecoveryBackup", "TestAcceptanceV15BudgetsEffects",
		"TestAcceptanceV16WorkspaceGit", "./internal/goal", "./internal/application",
		"./internal/ports", "./internal/adapters/artifact/filesystem",
		"./internal/adapters/workspace/gitlocal", "./internal/adapters/attestor/bubblewrap",
		"./internal/adapters/state/sqlite", "./internal/bootstrap", "./cmd/orquesta",
		"timeout --kill-after=10s", "-race", "GOFLAGS=-mod=vendor go vet",
	)
	for _, behavior := range fixture.RequiredBehaviorTests {
		if !strings.Contains(command, "./"+behavior.Directory) {
			t.Errorf("V17 acceptance command does not execute behavior package %q for %s", behavior.Directory, behavior.Name)
		}
	}
}

func TestV17EvidenceBelongsOnlyToTestAttestorCapabilities(t *testing.T) {
	roadmap := readRoadmapTestIndex(t).document
	fixture := readRoadmapV17Fixture(t)
	owned := roadmapSetOf(fixture.OwnedCapabilityIDs...)
	wantEvidence := []string{
		"acceptance/v17_test_attestor_test.go",
		"acceptance/fixtures/v17_test_attestor.json",
		"product/evidence/v17_test_attestor.json",
	}
	evidence := roadmapSetOf(
		"acceptance/v17_test_attestor_test.go",
		"acceptance/fixtures/v17_test_attestor.json",
		"product/evidence/v17_test_attestor.json",
		"product/evidence/v17_test_attestor.output.txt",
	)
	assertRoadmapEvidenceExclusive(t, roadmap.CapabilityEntries, owned, evidence,
		func(t *testing.T, entry roadmapEntry) {
			if entry.Status == "accredited" && !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence) {
				t.Errorf("owned V17 capability %s has noncanonical accreditation: status=%q evidence=%v",
					entry.ID, entry.Status, entry.EvidenceRefs)
			}
			if entry.Status != "accredited" && len(entry.EvidenceRefs) != 0 {
				t.Errorf("owned V17 capability %s claims evidence before accreditation: status=%q evidence=%v",
					entry.ID, entry.Status, entry.EvidenceRefs)
			}
		})
}

func readRoadmapV17Fixture(t *testing.T) roadmapV17Fixture {
	t.Helper()
	content, err := os.ReadFile("acceptance/fixtures/v17_test_attestor.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture roadmapV17Fixture
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.OwnedCapabilityIDs) != 4 || len(fixture.RequiredBehaviorTests) < 30 ||
		len(fixture.FocalSuite.Packages) == 0 || len(fixture.RaceSuite.Tests) == 0 ||
		len(fixture.E2ESuite.Tests) == 0 || len(fixture.VetPackages) == 0 ||
		fixture.Command == "" || len(fixture.ExecutionArgv) != 3 ||
		fixture.OutputPath == "" || fixture.ReceiptPath == "" || len(fixture.RoadmapAssertions) != 12 {
		t.Fatalf("invalid V17 roadmap fixture: %+v", fixture)
	}
	return fixture
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
	if len(decisions) != 14 {
		t.Fatalf("operator decision count = %d, want 14", len(decisions))
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
		"priority_vertical":   "agent_runtime_elastic",
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

func assertRoadmapImplementationDecisions(t *testing.T, decisions []roadmapImplementationDecision) {
	t.Helper()
	if len(decisions) != 1 {
		t.Fatalf("implementation decision count = %d, want 1", len(decisions))
	}
	decision := decisions[0]
	if decision.ID != "agent_microvm_network" || decision.Status != "planned_not_applied" ||
		decision.Transport != "vsock_only" ||
		decision.Authentication != "single_use_credential_store_proof_and_attestation" ||
		decision.TestAttestorScope != "unchanged_no_network_no_vsock" {
		t.Fatalf("invalid agent microVM implementation decision: %#v", decision)
	}
	if !reflect.DeepEqual(decision.CapabilityRefs, []string{"AGT-01", "AGT-03", "EVD-13", "ORC-15"}) ||
		!reflect.DeepEqual(decision.AllowedServices, []string{"orquesta_broker", "controlled_egress_proxy"}) ||
		!reflect.DeepEqual(decision.ForbiddenConnectivity,
			[]string{"guest_ip_network", "tap", "bridge", "nat", "inbound", "east_west", "direct_internet"}) {
		t.Fatalf("agent microVM implementation scope drifted: %#v", decision)
	}
	if len(decision.TestRefs) != 6 {
		t.Fatalf("agent microVM implementation tests = %v, want 6", decision.TestRefs)
	}
	for _, ref := range decision.TestRefs {
		requireRepositoryFile(t, ".", ref)
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

type roadmapTestIndex struct {
	document  roadmapDocument
	verticals map[string]roadmapVertical
	contracts map[string]roadmapAcceptanceContract
	entries   map[string]roadmapEntry
}

type roadmapVerticalContractSpec struct {
	version, owner, contractID, testRef, fixture, receipt, command string
	owned, evidence                                                []string
	assertions                                                     []string
	assertionCount                                                 int
	causal, allowDeclared                                          bool
}

func readRoadmapTestIndex(t *testing.T) roadmapTestIndex {
	t.Helper()
	var document roadmapDocument
	decodeRoadmapStrictJSON(t, "product/roadmap.json", &document)
	index := roadmapTestIndex{document: document, verticals: make(map[string]roadmapVertical, len(document.Verticals)),
		contracts: make(map[string]roadmapAcceptanceContract, len(document.AcceptanceContracts)),
		entries:   make(map[string]roadmapEntry, len(document.CapabilityEntries))}
	for _, value := range document.Verticals {
		index.verticals[value.ID] = value
	}
	for _, value := range document.AcceptanceContracts {
		index.contracts[value.ID] = value
	}
	for _, value := range document.CapabilityEntries {
		index.entries[value.ID] = value
	}
	return index
}

func assertRoadmapVerticalContract(t *testing.T, spec roadmapVerticalContractSpec) roadmapTestIndex {
	t.Helper()
	index := readRoadmapTestIndex(t)
	var owned []string
	for _, entry := range index.document.CapabilityEntries {
		if entry.OwnerContext == spec.owner && entry.Decision == "accept" {
			owned = append(owned, entry.ID)
		}
	}
	sort.Strings(owned)
	if !reflect.DeepEqual(owned, spec.owned) {
		t.Fatalf("%s accepted ownership = %v, want exact %v", spec.version, owned, spec.owned)
	}
	vertical := index.verticals[spec.owner]
	for _, id := range spec.owned {
		entry := index.entries[id]
		valid := entry.Status == "accredited" && reflect.DeepEqual(entry.EvidenceRefs, spec.evidence)
		if spec.allowDeclared && entry.Status == "declared" && len(entry.EvidenceRefs) == 0 {
			valid = true
		}
		if spec.causal {
			valid = valid && reflect.DeepEqual(entry.Dependencies, vertical.DependsOn) &&
				reflect.DeepEqual(entry.AcceptanceContracts, vertical.AcceptanceContracts)
		}
		if !valid {
			t.Errorf("%s capability %s lacks exact accreditation or causality: %#v", spec.version, id, entry)
		}
	}
	contract := index.contracts[spec.contractID]
	valid := contract.Status == "executable" && contract.TestRef == spec.testRef &&
		contract.Fixture == spec.fixture && contract.Receipt == spec.receipt && contract.Command == spec.command
	if spec.assertions != nil {
		valid = valid && reflect.DeepEqual(contract.Assertions, spec.assertions)
	} else {
		valid = valid && len(contract.Assertions) == spec.assertionCount
	}
	if !valid {
		t.Fatalf("invalid %s executable contract: %#v", spec.version, contract)
	}
	return index
}

func assertRoadmapCommandContains(t *testing.T, contract roadmapAcceptanceContract, forbidden []string, required ...string) {
	t.Helper()
	for _, value := range forbidden {
		if strings.Contains(contract.Command, value) {
			t.Fatalf("%s command contains forbidden %q: %q", contract.ID, value, contract.Command)
		}
	}
	for _, value := range required {
		if !strings.Contains(contract.Command, value) {
			t.Errorf("%s command does not execute %q", contract.ID, value)
		}
	}
}

func assertRoadmapCommandArguments(t *testing.T, contract roadmapAcceptanceContract, required ...string) {
	t.Helper()
	if contract.Command == "" {
		t.Fatalf("%s missing", contract.ID)
	}
	for _, value := range required {
		if !roadmapCommandHasArgument(contract.Command, value) {
			t.Errorf("%s command omits package %q: %q", contract.ID, value, contract.Command)
		}
	}
}

func assertRoadmapEvidenceExclusive(
	t *testing.T, entries []roadmapEntry, owned map[string]struct{}, evidence map[string]struct{},
	assertOwned func(*testing.T, roadmapEntry),
) {
	t.Helper()
	for _, entry := range entries {
		if _, ok := owned[entry.ID]; ok {
			assertOwned(t, entry)
			continue
		}
		for _, ref := range entry.EvidenceRefs {
			if _, leaked := evidence[ref]; leaked {
				t.Errorf("unowned capability %s claims evidence %q", entry.ID, ref)
			}
		}
	}
}

func assertRoadmapAccreditedEvidenceExclusive(
	t *testing.T, owned []string, canonical []string, extraEvidence ...string,
) {
	t.Helper()
	evidence := append(append([]string(nil), canonical...), extraEvidence...)
	assertRoadmapEvidenceExclusive(t, readRoadmapTestIndex(t).document.CapabilityEntries,
		roadmapSetOf(owned...), roadmapSetOf(evidence...), func(t *testing.T, entry roadmapEntry) {
			if entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, canonical) {
				t.Errorf("owned capability %s lacks exact accreditation: status=%q evidence=%v",
					entry.ID, entry.Status, entry.EvidenceRefs)
			}
		})
}

func roadmapTwoDigits(value int) string {
	return string([]byte{'0' + byte(value/10), '0' + byte(value%10)})
}
