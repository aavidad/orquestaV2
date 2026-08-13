// Este contrato conserva una propuesta trazable; no crea trabajo, cierre, admisión ni estado canónico.
package orquesta_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

const behaviorBatch19Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_19.jsonl"

type behaviorBatch19Link struct {
	SchemaVersion     int      `json:"schema_version"`
	OccurrenceRef     string   `json:"occurrence_ref"`
	DeclarationRef    string   `json:"declaration_ref"`
	VariantRef        string   `json:"variant_ref"`
	SourceRoot        string   `json:"source_root"`
	SourcePath        string   `json:"source_path"`
	GitBlobOID        string   `json:"git_blob_oid"`
	BlobDigest        string   `json:"blob_digest"`
	PackagePath       string   `json:"package_path"`
	PackageName       string   `json:"package_name"`
	SymbolKind        string   `json:"symbol_kind"`
	Name              string   `json:"name"`
	Receiver          string   `json:"receiver,omitempty"`
	Signature         string   `json:"signature"`
	SourceSHA         string   `json:"source_sha256"`
	ASTSHA            string   `json:"ast_sha256"`
	BodySHA           string   `json:"body_sha256,omitempty"`
	StartLine         int      `json:"start_line"`
	EndLine           int      `json:"end_line"`
	Relation          string   `json:"relation"`
	SemanticRationale string   `json:"semantic_rationale"`
	EvidenceRefs      []string `json:"evidence_refs"`
	ReviewState       string   `json:"review_state"`
}

type behaviorBatch19Assessment struct {
	SchemaVersion        int    `json:"schema_version"`
	CharacterizationRef  string `json:"characterization_ref"`
	HistoricalGreenState string `json:"historical_green_state"`
	GreenRationale       string `json:"green_rationale"`
	ReuseKind            string `json:"reuse_kind"`
	ExactV2Fit           string `json:"exact_v2_fit"`
	AgentRecommendation  string `json:"agent_recommendation"`
	ReviewState          string `json:"review_state"`
	Authority            string `json:"authority"`
	FunctionMapping      struct {
		State                string                `json:"state"`
		Rationale            string                `json:"rationale"`
		SnapshotCensusSHA256 string                `json:"snapshot_census_sha256"`
		Links                []behaviorBatch19Link `json:"links"`
	} `json:"function_mapping"`
}

func TestBehaviorCharacterizationAgentBatch19(t *testing.T) {
	fixture := readBehaviorBatch19File(t, behaviorBatch19Fixture)
	ledger := readBehaviorBatch19File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch19File(t, "product/roadmap.json")
	if err := validateBehaviorBatch19(fixture, ledger, roadmap, behaviorBatch19PreviousFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch19RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch19File(t, behaviorBatch19Fixture)
	ledger := readBehaviorBatch19File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch19File(t, "product/roadmap.json")
	previous := behaviorBatch19PreviousFixtures(t)
	mutations := []struct{ name, old, replacement string }{
		{"autoridad", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`},
		{"estado", `"canonical_state_change":false`, `"canonical_state_change":true`},
		{"trabajo", `"creates_work_item":false`, `"creates_work_item":true`},
		{"cierre", `"closes_capability":false`, `"closes_capability":true`},
		{"acreditación", `"claims_accreditation":false`, `"claims_accreditation":true`},
		{"revisión", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`},
		{"capability", `"capability_id":"OPS-19"`, `"capability_id":"OPS-18"`},
		{"ref usada", `"TASKENTRY-c4a3e317aa9e91719acc1565"`, `"TASKENTRY-76c5cb177737ae2b21014d61"`},
		{"independencia falsa", `no son evidencia independiente`, `son evidencia independiente`},
		{"campo desconocido", `{"schema_version":1`, `{"unknown":true,"schema_version":1`},
		{"valor posterior", `}` + "\n", `} {}` + "\n"},
		{"no canónico", `{"schema_version":1`, ` {"schema_version":1`},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(fixture, []byte(mutation.old), []byte(mutation.replacement), 1)
			if bytes.Equal(changed, fixture) {
				t.Fatal("mutación inerte")
			}
			if err := validateBehaviorBatch19(changed, ledger, roadmap, previous); err == nil {
				t.Fatal("mutación aceptada")
			}
		})
	}
	t.Run("LF ausente", func(t *testing.T) {
		if err := validateBehaviorBatch19(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous); err == nil {
			t.Fatal("fixture sin LF aceptado")
		}
	})
	t.Run("roadmap declarado mutado", func(t *testing.T) {
		if err := validateBehaviorBatch19(fixture, ledger, mutateBehaviorBatch19RoadmapStatus(t, roadmap, "OPS-19", "accredited"), previous); err == nil {
			t.Fatal("roadmap mutado aceptado")
		}
	})
}

func TestBehaviorCharacterizationAgentBatch19ClaimsQualified(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch19File(t, behaviorBatch19Fixture))
	if err != nil {
		t.Fatal(err)
	}
	qualifiers := []string{"declar", "document", "registr", "inform", "históric", "no consta", "código legacy"}
	for _, record := range records {
		for _, claims := range [][]string{record.Worked, record.Failed, record.Attempts} {
			for _, claim := range claims {
				normalized := strings.ToLower(claim)
				qualified := false
				for _, qualifier := range qualifiers {
					qualified = qualified || strings.Contains(normalized, qualifier)
				}
				if !qualified {
					t.Errorf("afirmación histórica sin calificar: %q", claim)
				}
			}
		}
	}
}

func TestBehaviorCharacterizationAgentBatch19Assessments(t *testing.T) {
	fixture, err := decodeBehaviorBatchRecords(readBehaviorBatch19File(t, behaviorBatch19Fixture))
	if err != nil {
		t.Fatal(err)
	}
	sources := make(map[string]map[string]struct{})
	for _, record := range fixture {
		sources[record.Ref] = make(map[string]struct{})
		for _, evidence := range record.Evidence {
			sources[record.Ref][evidence.SourceRef] = struct{}{}
		}
	}
	rows, err := decodeBehaviorBatch19Assessments(readBehaviorBatch19AssessmentFile(t))
	if err != nil {
		t.Fatal(err)
	}
	wants := map[string][]string{
		"BEHAVIOR-AGENT-BATCH-19-STRUCTURED-TELEMETRY":   {"ValidateOrquestaEventV0", "AcceptPublishOrquestaEventRequestV0", "ValidateOperationalStatusQueryV0"},
		"BEHAVIOR-AGENT-BATCH-19-DISJOINT-WORKSETS":      {"DetectWorksetConflictsV0", "EvaluateParallelGroupsV0", "EvaluateConcurrencyGateV0"},
		"BEHAVIOR-AGENT-BATCH-19-IMMUTABLE-APPSPEC":      {"ValidateAppSpecRequestV0", "SolicitarNuevaAppV0", "assembleAppSpecV0"},
		"BEHAVIOR-AGENT-BATCH-19-EVIDENCED-COUNCIL":      {"BuildDecisionCouncilPlanV0", "BuildDecisionCouncilOperationalRoundsV0", "EvaluateDecisionCouncilVotesV0"},
		"BEHAVIOR-AGENT-BATCH-19-RESTART-RECONCILIATION": {"NewOperationalDirectorPlanStateV0", "reconcileDurableCapacityDecisionsForRunV0", "reconcileClaimedLaunchOutboxForProcessRegistryV0"},
	}
	if len(rows) != 5 {
		t.Fatalf("assessments=%d", len(rows))
	}
	for _, row := range rows {
		names, exists := wants[row.CharacterizationRef]
		if !exists || row.SchemaVersion != 1 || row.ReviewState != "independent_bootstrap_counterreview_completed_advisory" || row.Authority != "advisory_not_capability_state" || row.FunctionMapping.State != "linked_exact" || row.FunctionMapping.SnapshotCensusSHA256 != "sha256:5e8f30dc3c6640c57f595e3757cc6e07ba4be6eb36a6bcdd094152a2967fddfc" || len(row.FunctionMapping.Links) != 3 {
			t.Fatalf("assessment inválido: %s", row.CharacterizationRef)
		}
		for index, link := range row.FunctionMapping.Links {
			if link.Name != names[index] || link.SchemaVersion != 1 || link.OccurrenceRef == "" || link.DeclarationRef == "" || link.VariantRef == "" || link.GitBlobOID == "" || link.BodySHA == "" || link.EndLine < link.StartLine || link.ReviewState != "bootstrap_symbol_mapping_reviewed_advisory" || len(strings.TrimSpace(link.SemanticRationale)) < 30 || len(link.EvidenceRefs) == 0 {
				t.Fatalf("link inválido en %s", row.CharacterizationRef)
			}
			for _, ref := range link.EvidenceRefs {
				if _, allowed := sources[row.CharacterizationRef][ref]; !allowed {
					t.Fatalf("evidencia ajena: %s", ref)
				}
			}
		}
		delete(wants, row.CharacterizationRef)
	}
	if len(wants) != 0 {
		t.Fatalf("assessments ausentes: %v", wants)
	}
}

func behaviorBatch19PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	result := make([][]byte, 0, 18)
	for index := 1; index <= 18; index++ {
		path := fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", index)
		content, err := os.ReadFile(path)
		if err == nil {
			result = append(result, content)
			continue
		}
		if index <= 16 || !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		if override := strings.TrimSpace(os.Getenv(fmt.Sprintf("ORQUESTA_BATCH19_PREVIOUS_%02d", index))); override != "" {
			result = append(result, readBehaviorBatch19File(t, override))
		}
	}
	return result
}

func readBehaviorBatch19File(t *testing.T, path string) []byte {
	t.Helper()
	if path == behaviorBatch19Fixture {
		if override := strings.TrimSpace(os.Getenv("ORQUESTA_BATCH19_FIXTURE")); override != "" {
			path = override
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func readBehaviorBatch19AssessmentFile(t *testing.T) []byte {
	t.Helper()
	path := strings.TrimSpace(os.Getenv("ORQUESTA_BATCH19_ASSESSMENTS"))
	if path == "" {
		path = "product/knowledge/legacy_reuse_assessments_v1.jsonl"
	}
	return readBehaviorBatch19File(t, path)
}

func decodeBehaviorBatch19Assessments(raw []byte) ([]behaviorBatch19Assessment, error) {
	if len(raw) == 0 || !bytes.HasSuffix(raw, []byte("\n")) || bytes.Contains(raw, []byte("\r")) {
		return nil, errors.New("assessment JSONL inválido")
	}
	var rows []behaviorBatch19Assessment
	for index, line := range bytes.Split(bytes.TrimSuffix(raw, []byte("\n")), []byte("\n")) {
		var identity struct {
			CharacterizationRef string `json:"characterization_ref"`
		}
		if err := json.Unmarshal(line, &identity); err != nil {
			return nil, fmt.Errorf("assessment %d: %w", index+1, err)
		}
		if !strings.HasPrefix(identity.CharacterizationRef, "BEHAVIOR-AGENT-BATCH-19-") {
			continue
		}
		var row behaviorBatch19Assessment
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&row); err != nil {
			return nil, fmt.Errorf("assessment %d: %w", index+1, err)
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("assessment %d con valor posterior", index+1)
		}
		canonical, err := json.Marshal(row)
		if err != nil || !bytes.Equal(canonical, line) {
			return nil, fmt.Errorf("assessment %d no canónico", index+1)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func mutateBehaviorBatch19RoadmapStatus(t *testing.T, raw []byte, capabilityID, status string) []byte {
	t.Helper()
	var roadmap map[string]any
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		t.Fatal(err)
	}
	for _, value := range roadmap["capability_entries"].([]any) {
		entry := value.(map[string]any)
		if entry["id"] == capabilityID {
			entry["status"] = status
			changed, err := json.Marshal(roadmap)
			if err != nil {
				t.Fatal(err)
			}
			return changed
		}
	}
	t.Fatalf("%s ausente", capabilityID)
	return nil
}
