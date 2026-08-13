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

const behaviorBatch15Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_15.jsonl"

type behaviorBatch15FunctionLink struct {
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

type behaviorBatch15Assessment struct {
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
		State                string                        `json:"state"`
		Rationale            string                        `json:"rationale"`
		SnapshotCensusSHA256 string                        `json:"snapshot_census_sha256"`
		Links                []behaviorBatch15FunctionLink `json:"links"`
	} `json:"function_mapping"`
}

func TestBehaviorCharacterizationAgentBatch15(t *testing.T) {
	fixture := readBehaviorBatch15File(t, behaviorBatch15Fixture)
	ledger := readBehaviorBatch15File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch15File(t, "product/roadmap.json")
	if err := validateBehaviorBatch15(fixture, ledger, roadmap, behaviorBatch15PreviousFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch15RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch15File(t, behaviorBatch15Fixture)
	ledger := readBehaviorBatch15File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch15File(t, "product/roadmap.json")
	previous := behaviorBatch15PreviousFixtures(t)
	mutations := []struct{ name, old, replacement string }{
		{"autoridad canónica", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`},
		{"cambio canónico", `"canonical_state_change":false`, `"canonical_state_change":true`},
		{"crea trabajo", `"creates_work_item":false`, `"creates_work_item":true`},
		{"cierra capacidad", `"closes_capability":false`, `"closes_capability":true`},
		{"afirma acreditación", `"claims_accreditation":false`, `"claims_accreditation":true`},
		{"revisión aceptada", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`},
		{"capacidad incorrecta", `"capability_id":"UI-13"`, `"capability_id":"UI-12"`},
		{"referencia reutilizada", `"TASKENTRY-d280135cfa433911099f03d5"`, `"TASKENTRY-76c5cb177737ae2b21014d61"`},
		{"fuentes solapadas independientes", `no son evidencia independiente`, `son evidencia independiente`},
		{"campo desconocido", `{"schema_version":1`, `{"unknown":true,"schema_version":1`},
		{"valor posterior", `}` + "\n", `} {}` + "\n"},
		{"representación no canónica", `{"schema_version":1`, ` {"schema_version":1`},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(fixture, []byte(mutation.old), []byte(mutation.replacement), 1)
			if bytes.Equal(changed, fixture) {
				t.Fatal("la mutación no cambió el fixture")
			}
			if err := validateBehaviorBatch15(changed, ledger, roadmap, previous); err == nil {
				t.Fatal("la mutación autoritativa, reutilizada o ambigua fue aceptada")
			}
		})
	}
	t.Run("LF final ausente", func(t *testing.T) {
		if err := validateBehaviorBatch15(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous); err == nil {
			t.Fatal("el fixture sin LF final fue aceptado")
		}
	})
	t.Run("roadmap acredita UI-13 sin evidencia", func(t *testing.T) {
		changed := mutateBehaviorBatch15RoadmapStatus(t, roadmap, "UI-13", "accredited")
		if err := validateBehaviorBatch15(fixture, ledger, changed, previous); err == nil {
			t.Fatal("el estado roadmap mutado fue aceptado")
		}
	})
	t.Run("roadmap degrada EVD-12", func(t *testing.T) {
		changed := mutateBehaviorBatch15RoadmapStatus(t, roadmap, "EVD-12", "declared")
		if err := validateBehaviorBatch15(fixture, ledger, changed, previous); err == nil {
			t.Fatal("el estado roadmap acreditado mutado fue aceptado")
		}
	})
}

func TestBehaviorCharacterizationAgentBatch15QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch15File(t, behaviorBatch15Fixture))
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

func TestBehaviorCharacterizationAgentBatch15AssessmentsReviewedAndExact(t *testing.T) {
	fixtureRecords, err := decodeBehaviorBatchRecords(readBehaviorBatch15File(t, behaviorBatch15Fixture))
	if err != nil {
		t.Fatal(err)
	}
	behaviorSources := make(map[string]map[string]struct{}, len(fixtureRecords))
	for _, record := range fixtureRecords {
		refs := make(map[string]struct{}, len(record.Evidence))
		for _, evidence := range record.Evidence {
			refs[evidence.SourceRef] = struct{}{}
		}
		behaviorSources[record.Ref] = refs
	}
	rows, err := decodeBehaviorBatch15Assessments(readBehaviorBatch15AssessmentFile(t))
	if err != nil {
		t.Fatal(err)
	}
	wants := map[string][]string{
		"BEHAVIOR-AGENT-BATCH-15-HONEST-STATUS":     {"ClassifyRunLivenessV0", "mcpQueueGlobalStatusNeedsActionV0", "NewWebAutoprogrammingStatusViewModelV0"},
		"BEHAVIOR-AGENT-BATCH-15-EGRESS-REDACTION":  {"SanitizeMaterializedContextBundleV0", "BuildAgentStartPacketV0", "CodexUsageAccountingReportRedactedV0"},
		"BEHAVIOR-AGENT-BATCH-15-ATTRIBUTED-BUDGET": {"DecideAutoprogrammingIdleSelfImprovementBudgetV0", "ApplyDirectorAgentUsageStatsV0", "BuildCodexStackAgentUsageMetricsV0"},
		"BEHAVIOR-AGENT-BATCH-15-ADVISORY-RAILS":    {"ValuesContainOperationalSensitiveDetailForFieldV0", "normalizeLooseEnumTokenV0", "domainWorkRejectedSubmissionRecoverableV0"},
		"BEHAVIOR-AGENT-BATCH-15-TYPED-HTTP":        {"ServeHTTP", "NewHTTPHandlerV0", "DecodeStrictBoundedJSONV0"},
	}
	if len(rows) != len(wants) {
		t.Fatalf("assessments=%d", len(rows))
	}
	pairs := make(map[string]struct{})
	for _, row := range rows {
		names, exists := wants[row.CharacterizationRef]
		if !exists || row.SchemaVersion != 1 || row.ReviewState != "independent_bootstrap_counterreview_completed_advisory" || row.Authority != "advisory_not_capability_state" || row.FunctionMapping.State != "linked_exact" || row.FunctionMapping.SnapshotCensusSHA256 != "sha256:5e8f30dc3c6640c57f595e3757cc6e07ba4be6eb36a6bcdd094152a2967fddfc" || len(row.FunctionMapping.Links) != 3 {
			t.Fatalf("assessment incompleto o no contrarrevisado: %s", row.CharacterizationRef)
		}
		for index, link := range row.FunctionMapping.Links {
			if link.Name != names[index] || link.SchemaVersion != 1 || link.OccurrenceRef == "" || link.DeclarationRef == "" || link.VariantRef == "" || link.GitBlobOID == "" || link.Signature == "" || link.BodySHA == "" || link.StartLine < 1 || link.EndLine < link.StartLine || link.ReviewState != "bootstrap_symbol_mapping_reviewed_advisory" || len(strings.TrimSpace(link.SemanticRationale)) < 30 || len(link.EvidenceRefs) == 0 {
				t.Fatalf("enlace incompleto en %s: %#v", row.CharacterizationRef, link)
			}
			if link.Relation != "primary_implementation" && link.Relation != "supporting_mechanism" && link.Relation != "negative_example" {
				t.Fatalf("relación inválida: %s", link.Relation)
			}
			for _, evidenceRef := range link.EvidenceRefs {
				if _, allowed := behaviorSources[row.CharacterizationRef][evidenceRef]; !allowed {
					t.Fatalf("evidence_ref ajena en %s: %s", row.CharacterizationRef, evidenceRef)
				}
			}
			pair := row.CharacterizationRef + "\x00" + link.OccurrenceRef
			if _, duplicate := pairs[pair]; duplicate {
				t.Fatalf("enlace duplicado: %s", pair)
			}
			pairs[pair] = struct{}{}
		}
		delete(wants, row.CharacterizationRef)
	}
	if len(wants) != 0 {
		t.Fatalf("assessments ausentes: %v", wants)
	}
}

func behaviorBatch15PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	result := make([][]byte, 0, 14)
	for index := 1; index <= 14; index++ {
		path := fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", index)
		result = append(result, readBehaviorBatch15File(t, path))
	}
	return result
}

func readBehaviorBatch15File(t *testing.T, path string) []byte {
	t.Helper()
	if path == behaviorBatch15Fixture {
		if override := strings.TrimSpace(os.Getenv("ORQUESTA_BATCH15_FIXTURE")); override != "" {
			path = override
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func readBehaviorBatch15AssessmentFile(t *testing.T) []byte {
	t.Helper()
	path := strings.TrimSpace(os.Getenv("ORQUESTA_BATCH15_ASSESSMENTS"))
	if path == "" {
		path = "product/knowledge/legacy_reuse_assessments_v1.jsonl"
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func decodeBehaviorBatch15Assessments(raw []byte) ([]behaviorBatch15Assessment, error) {
	if len(raw) == 0 || !bytes.HasSuffix(raw, []byte("\n")) || bytes.Contains(raw, []byte("\r")) {
		return nil, errors.New("assessment JSONL requiere LF final y prohíbe CR")
	}
	lines := bytes.Split(bytes.TrimSuffix(raw, []byte("\n")), []byte("\n"))
	rows := make([]behaviorBatch15Assessment, 0, len(lines))
	for index, line := range lines {
		var identity struct {
			CharacterizationRef string `json:"characterization_ref"`
		}
		if err := json.Unmarshal(line, &identity); err != nil {
			return nil, fmt.Errorf("assessment %d: %w", index+1, err)
		}
		if !strings.HasPrefix(identity.CharacterizationRef, "BEHAVIOR-AGENT-BATCH-15-") {
			continue
		}
		var row behaviorBatch15Assessment
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

func mutateBehaviorBatch15RoadmapStatus(t *testing.T, raw []byte, capabilityID, status string) []byte {
	t.Helper()
	var roadmap map[string]any
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		t.Fatal(err)
	}
	entries, ok := roadmap["capability_entries"].([]any)
	if !ok {
		t.Fatal("capability_entries ausente")
	}
	for _, value := range entries {
		entry, entryOK := value.(map[string]any)
		if !entryOK || entry["id"] != capabilityID {
			continue
		}
		entry["status"] = status
		changed, err := json.Marshal(roadmap)
		if err != nil {
			t.Fatal(err)
		}
		return changed
	}
	t.Fatalf("%s ausente", capabilityID)
	return nil
}
