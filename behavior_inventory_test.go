package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type behaviorInventoryPolicy struct {
	DocumentKind    string              `json:"document_kind"`
	SchemaVersion   int                 `json:"schema_version"`
	Authority       string              `json:"authority"`
	CoverageState   string              `json:"coverage_state"`
	RequiredGapRefs []string            `json:"required_gap_refs"`
	AnchorKinds     []string            `json:"anchor_kinds"`
	StateAxes       map[string][]string `json:"state_axes"`
	Invariants      []string            `json:"invariants"`
}

type behaviorInventoryEntry struct {
	SchemaVersion   int                       `json:"schema_version"`
	BehaviorRef     string                    `json:"behavior_ref"`
	Title           string                    `json:"title"`
	CapabilityIDs   []string                  `json:"capability_ids"`
	OwnerResolution string                    `json:"owner_resolution"`
	SourceAnchors   []behaviorInventoryAnchor `json:"source_anchors"`
	V2Anchors       []behaviorInventoryAnchor `json:"v2_anchors"`
	States          struct {
		Census           string `json:"census"`
		Characterization string `json:"characterization"`
		Implementation   string `json:"implementation"`
		Equivalence      string `json:"equivalence"`
		Accreditation    string `json:"accreditation"`
		Retirement       string `json:"retirement"`
	} `json:"states"`
	EquivalenceProofRefs      []string                 `json:"equivalence_proof_refs"`
	AccreditationEvidenceRefs []string                 `json:"accreditation_evidence_refs"`
	GapTask                   behaviorInventoryGapTask `json:"gap_task"`
}

type behaviorInventoryAnchor struct {
	Side          string `json:"side"`
	Kind          string `json:"kind"`
	Path          string `json:"path"`
	Selector      string `json:"selector"`
	ContentSHA256 string `json:"content_sha256"`
	Role          string `json:"role"`
}

type behaviorInventoryGapTask struct {
	GapRef       string   `json:"gap_ref"`
	Status       string   `json:"status"`
	Obligation   string   `json:"obligation"`
	TestRef      string   `json:"test_ref"`
	EvidenceRefs []string `json:"evidence_refs"`
}

func TestBehaviorInventoryInitialContractKeepsConfirmedGapsPending(t *testing.T) {
	var policy behaviorInventoryPolicy
	decodeRoadmapStrictJSON(t, "product/traceability/behavior_inventory_policy.json", &policy)
	wantGaps := []string{
		"GAP-AGENT-RUNTIME-ELASTIC",
		"GAP-AGENT-RUNTIME-FIRECRACKER-PHYSICAL",
		"GAP-AGENT-SESSION-LIVE-INSTRUCTION",
		"GAP-QUOTA-OBSERVATION-SELECTION",
	}
	if policy.DocumentKind != "behavior_inventory_policy" || policy.SchemaVersion != 1 ||
		policy.Authority != "initial_contract_without_completion_or_accreditation_claim" ||
		policy.CoverageState != "initial_confirmed_gaps_only" ||
		!reflect.DeepEqual(policy.RequiredGapRefs, wantGaps) ||
		!reflect.DeepEqual(policy.AnchorKinds, []string{"function", "test", "doc", "config", "migration", "surface"}) {
		t.Fatalf("política de inventario inválida: %+v", policy)
	}

	entries := readBehaviorInventory(t)
	if len(entries) != 4 {
		t.Fatalf("brechas inventariadas=%d, se esperaban 4", len(entries))
	}
	accepted := traceAcceptedCapabilities(t)
	legacyPaths := map[string]bool{}
	for _, entry := range entries {
		for _, anchor := range entry.SourceAnchors {
			legacyPaths[anchor.Path] = true
		}
	}
	legacy := traceLoadGitIndexSnapshot(t, ".", "modulos", func(path string) bool {
		return legacyPaths[path]
	})
	gotGaps := make([]string, 0, len(entries))
	var firecracker behaviorInventoryEntry
	var liveInstruction behaviorInventoryEntry
	for _, entry := range entries {
		if err := validateBehaviorInventoryEntry(entry); err != nil {
			t.Fatalf("%s: %v", entry.BehaviorRef, err)
		}
		if entry.OwnerResolution != "pending_canonical_roadmap_confirmation" {
			t.Fatalf("%s atribuye autoridad canónica prematura", entry.BehaviorRef)
		}
		for _, capabilityID := range entry.CapabilityIDs {
			if _, exists := accepted[capabilityID]; !exists {
				t.Fatalf("%s enlaza una capacidad no aceptada: %s", entry.BehaviorRef, capabilityID)
			}
		}
		for _, anchor := range append(append([]behaviorInventoryAnchor(nil), entry.SourceAnchors...), entry.V2Anchors...) {
			var content []byte
			if anchor.Side == "legacy" {
				content = legacy.Contents[anchor.Path]
				if len(content) == 0 {
					t.Fatalf("%s no encuentra ancla legacy exacta %s", entry.BehaviorRef, anchor.Path)
				}
			} else {
				var err error
				content, err = os.ReadFile(anchor.Path)
				if err != nil {
					t.Fatalf("%s no puede leer ancla V2 %s: %v", entry.BehaviorRef, anchor.Path, err)
				}
			}
			if got := traceBytesSHA256(content); got != anchor.ContentSHA256 {
				t.Fatalf("%s ancla %s cambió: %s, se esperaba %s", entry.BehaviorRef, anchor.Path, got, anchor.ContentSHA256)
			}
			if !bytes.Contains(content, []byte(anchor.Selector)) {
				t.Fatalf("%s ancla %s no contiene selector exacto %q", entry.BehaviorRef, anchor.Path, anchor.Selector)
			}
		}
		if entry.BehaviorRef == "BEHAVIOR-AGENT-RUNTIME-FIRECRACKER-PHYSICAL" {
			firecracker = entry
		}
		if entry.BehaviorRef == "BEHAVIOR-AGENT-SESSION-LIVE-INSTRUCTION" {
			liveInstruction = entry
		}
		if (entry.BehaviorRef == "BEHAVIOR-AGENT-RUNTIME-ELASTIC" ||
			entry.BehaviorRef == "BEHAVIOR-QUOTA-OBSERVATION-SELECTION") &&
			len(entry.SourceAnchors) == 0 {
			t.Fatalf("%s carece de anclas legacy", entry.BehaviorRef)
		}
		gotGaps = append(gotGaps, entry.GapTask.GapRef)
	}
	sort.Strings(gotGaps)
	if !reflect.DeepEqual(gotGaps, wantGaps) {
		t.Fatalf("brechas=%v, se esperaban %v", gotGaps, wantGaps)
	}

	if firecracker.BehaviorRef != "BEHAVIOR-AGENT-RUNTIME-FIRECRACKER-PHYSICAL" ||
		len(firecracker.SourceAnchors) != 0 ||
		firecracker.States.Implementation != "planned" {
		t.Fatalf("la ausencia física Firecracker se presentó como legado o implementación: %+v", firecracker)
	}
	hasDistinctAttestorBoundary := false
	for _, anchor := range firecracker.V2Anchors {
		hasDistinctAttestorBoundary = hasDistinctAttestorBoundary ||
			(anchor.Path == "internal/e2e/firecrackerattestor/supervisor.go" && anchor.Role == "distinct_boundary")
	}
	if !hasDistinctAttestorBoundary {
		t.Fatal("la brecha Firecracker no conserva TestAttestor como frontera distinta")
	}
	if !reflect.DeepEqual(liveInstruction.CapabilityIDs, []string{"ORC-15", "ORC-22", "ORC-29"}) ||
		liveInstruction.States.Implementation != "absent" {
		t.Fatalf("la conversación viva no está inventariada como brecha ausente: %+v", liveInstruction)
	}
	hasEphemeral, hasSinglePrompt, hasMailboxBoundary := false, false, false
	for _, anchor := range liveInstruction.V2Anchors {
		hasEphemeral = hasEphemeral || anchor.Selector == `"--ephemeral"`
		hasSinglePrompt = hasSinglePrompt || anchor.Selector == "command.Stdin = stringsReaderAndClear(envelope.Prompt)"
		hasMailboxBoundary = hasMailboxBoundary ||
			(anchor.Path == "internal/application/mailbox_state.go" && anchor.Role == "distinct_boundary")
	}
	if !hasEphemeral || !hasSinglePrompt || !hasMailboxBoundary {
		t.Fatalf("faltan límites exactos de la ejecución Codex viva: %+v", liveInstruction.V2Anchors)
	}
}

func TestBehaviorInventoryElasticConfigAnchorMatchesCurrentRegistry(t *testing.T) {
	requireBehaviorInventoryAnchor(t, "BEHAVIOR-AGENT-RUNTIME-ELASTIC", "config/registry.json")
}

func TestBehaviorInventoryLiveInstructionAnchorMatchesCurrentPorts(t *testing.T) {
	requireBehaviorInventoryAnchor(t, "BEHAVIOR-AGENT-SESSION-LIVE-INSTRUCTION", "internal/application/ports.go")
}

func requireBehaviorInventoryAnchor(t *testing.T, referencia, ruta string) {
	t.Helper()
	for _, entrada := range readBehaviorInventory(t) {
		if entrada.BehaviorRef != referencia {
			continue
		}
		for _, ancla := range entrada.V2Anchors {
			if ancla.Path == ruta {
				contenido, err := os.ReadFile(ruta)
				if err != nil {
					t.Fatal(err)
				}
				if actual := traceBytesSHA256(contenido); actual != ancla.ContentSHA256 {
					t.Fatalf("ancla %s=%s, se esperaba %s", ruta, actual, ancla.ContentSHA256)
				}
				return
			}
		}
	}
	t.Fatalf("no existe el ancla %s de %s", ruta, referencia)
}

func TestBehaviorInventoryCensusNeverImpliesEquivalence(t *testing.T) {
	entry := readBehaviorInventory(t)[0]
	entry.States.Census = "reviewed"
	entry.States.Equivalence = "equivalent"
	entry.EquivalenceProofRefs = nil
	if err := validateBehaviorInventoryEntry(entry); err == nil {
		t.Fatal("el censo revisado implicó equivalencia sin prueba")
	}
}

func TestBehaviorInventoryVerifiedGapRequiresExecutableTestAndEvidence(t *testing.T) {
	entry := readBehaviorInventory(t)[0]
	entry.GapTask.Status = "verified"
	entry.GapTask.TestRef = "planned:behavior-tests/false-green"
	entry.GapTask.EvidenceRefs = nil
	if err := validateBehaviorInventoryEntry(entry); err == nil {
		t.Fatal("una brecha verificada carece de prueba ejecutable y evidencia")
	}
}

func validateBehaviorInventoryEntry(entry behaviorInventoryEntry) error {
	if entry.SchemaVersion != 1 || entry.BehaviorRef == "" || len(entry.CapabilityIDs) == 0 ||
		len(entry.V2Anchors) == 0 || entry.GapTask.GapRef == "" {
		return fmt.Errorf("registro incompleto")
	}
	if entry.States.Equivalence == "equivalent" && len(entry.EquivalenceProofRefs) == 0 {
		return fmt.Errorf("equivalencia sin prueba")
	}
	if entry.States.Accreditation == "accredited" && len(entry.AccreditationEvidenceRefs) == 0 {
		return fmt.Errorf("acreditación sin evidencia")
	}
	if entry.GapTask.Status == "verified" &&
		(strings.HasPrefix(entry.GapTask.TestRef, "planned:") || len(entry.GapTask.EvidenceRefs) == 0) {
		return fmt.Errorf("brecha verificada sin prueba ejecutable y evidencia")
	}
	if entry.GapTask.Status == "pending" &&
		(entry.States.Equivalence != "unassessed" ||
			entry.States.Accreditation != "ineligible" ||
			len(entry.EquivalenceProofRefs) != 0 ||
			len(entry.AccreditationEvidenceRefs) != 0 ||
			len(entry.GapTask.EvidenceRefs) != 0 ||
			!strings.HasPrefix(entry.GapTask.TestRef, "planned:")) {
		return fmt.Errorf("brecha pendiente atribuye equivalencia, acreditación o evidencia")
	}
	return nil
}

func readBehaviorInventory(t *testing.T) []behaviorInventoryEntry {
	t.Helper()
	file, err := os.Open("product/traceability/behavior_inventory.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var entries []behaviorInventoryEntry
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		decoder := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
		decoder.DisallowUnknownFields()
		var entry behaviorInventoryEntry
		if err := decoder.Decode(&entry); err != nil {
			t.Fatalf("línea %d: %v", line, err)
		}
		if decoder.Decode(new(any)) == nil {
			t.Fatalf("línea %d contiene JSON adicional", line)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return entries
}
