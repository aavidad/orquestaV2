// Estos contratos validan de extremo a extremo los enlaces conducta-función del catálogo legacy.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type reuseAssessment struct {
	SchemaVersion        int                   `json:"schema_version"`
	CharacterizationRef  string                `json:"characterization_ref"`
	HistoricalGreenState string                `json:"historical_green_state"`
	GreenRationale       string                `json:"green_rationale"`
	ReuseKind            string                `json:"reuse_kind"`
	ExactV2Fit           string                `json:"exact_v2_fit"`
	AgentRecommendation  string                `json:"agent_recommendation"`
	ReviewState          string                `json:"review_state"`
	Authority            string                `json:"authority"`
	FunctionMapping      assessmentFunctionMap `json:"function_mapping"`
}

type assessmentFunctionMap struct {
	State                string                   `json:"state"`
	Rationale            string                   `json:"rationale"`
	SnapshotCensusSHA256 string                   `json:"snapshot_census_sha256"`
	Links                []assessmentFunctionLink `json:"links"`
}

type reuseAssessmentManifest struct {
	DocumentKind                  string `json:"document_kind"`
	SchemaVersion                 int    `json:"schema_version"`
	JSONLSHA256                   string `json:"jsonl_sha256"`
	JSONLBytes                    int64  `json:"jsonl_bytes"`
	AssessmentCount               int    `json:"assessment_count"`
	UniqueCharacterizationCount   int    `json:"unique_characterization_ref_count"`
	FunctionBehaviorLinkCount     int    `json:"function_behavior_link_count"`
	UniqueFunctionOccurrenceCount int    `json:"unique_function_occurrence_count"`
	SnapshotCensusSHA256          string `json:"snapshot_census_sha256"`
	Authority                     string `json:"authority"`
	CanonicalStateChange          bool   `json:"canonical_state_change"`
	ClaimsAccreditation           bool   `json:"claims_accreditation"`
}

type assessmentFunctionLink struct {
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

type historicalTestObservation struct {
	SchemaVersion           int      `json:"schema_version"`
	ObservationRef          string   `json:"observation_ref"`
	RepositoryRole          string   `json:"repository_role"`
	RepositoryCommit        string   `json:"repository_commit"`
	RepositoryTrackedStatus string   `json:"repository_tracked_status"`
	ModulePath              string   `json:"module_path"`
	ModuleTreeOID           string   `json:"module_tree_oid"`
	Command                 string   `json:"command"`
	Result                  string   `json:"result"`
	VerificationKind        string   `json:"verification_kind"`
	TestRefs                []string `json:"test_refs"`
	BehaviorRefs            []string `json:"behavior_refs"`
	Limitations             []string `json:"limitations"`
	Authority               string   `json:"authority"`
	ClaimsAccreditation     bool     `json:"claims_accreditation"`
}

func (link assessmentFunctionLink) symbolRecord() indexSymbolRecord {
	return indexSymbolRecord{
		SchemaVersion: link.SchemaVersion, OccurrenceRef: link.OccurrenceRef,
		DeclarationRef: link.DeclarationRef, VariantRef: link.VariantRef,
		SourceRoot: link.SourceRoot, SourcePath: link.SourcePath,
		GitBlobOID: link.GitBlobOID, BlobDigest: link.BlobDigest,
		PackagePath: link.PackagePath, PackageName: link.PackageName,
		SymbolKind: link.SymbolKind, Name: link.Name, Receiver: link.Receiver,
		Signature: link.Signature, SourceSHA: link.SourceSHA, ASTSHA: link.ASTSHA,
		BodySHA: link.BodySHA, StartLine: link.StartLine, EndLine: link.EndLine,
	}
}

func readReuseAssessments(path string) ([]reuseAssessment, error) {
	rawFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(rawFile) == 0 || !bytes.HasSuffix(rawFile, []byte{'\n'}) || bytes.Contains(rawFile, []byte{'\r'}) {
		return nil, errors.New("JSONL requiere LF final y prohíbe CR")
	}

	var rows []reuseAssessment
	for index, raw := range bytes.Split(rawFile[:len(rawFile)-1], []byte{'\n'}) {
		lineNumber := index + 1
		if len(raw) == 0 {
			return nil, fmt.Errorf("línea JSONL vacía: %d", lineNumber)
		}
		var row reuseAssessment
		if err := decodeUniqueStrictJSON(raw, &row); err != nil {
			return nil, fmt.Errorf("assessment inválido en línea %d: %w", lineNumber, err)
		}
		canonical, err := json.Marshal(row)
		if err != nil || !bytes.Equal(canonical, raw) {
			return nil, fmt.Errorf("JSON no canónico en línea %d", lineNumber)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func validateAssessmentFunctionLinks(repository string, rows []reuseAssessment, contract legacyGoCensusContract) error {
	behaviorSources, err := loadBehaviorSourceRefs(repository)
	if err != nil {
		return err
	}
	observations, err := loadHistoricalTestObservations(repository, contract)
	if err != nil {
		return err
	}
	behaviorRefs := make(map[string]struct{}, len(rows))
	pairs := make(map[string]struct{})
	for _, row := range rows {
		if row.SchemaVersion != 1 || row.CharacterizationRef == "" {
			return fmt.Errorf("assessment sin identidad válida: %q", row.CharacterizationRef)
		}
		if _, duplicate := behaviorRefs[row.CharacterizationRef]; duplicate {
			return fmt.Errorf("assessment duplicado: %s", row.CharacterizationRef)
		}
		behaviorRefs[row.CharacterizationRef] = struct{}{}
		if _, exists := behaviorSources[row.CharacterizationRef]; !exists {
			return fmt.Errorf("assessment huérfano: %s", row.CharacterizationRef)
		}
		mapping := row.FunctionMapping
		if mapping.SnapshotCensusSHA256 != contract.Baseline.CensusSHA256 {
			return fmt.Errorf("snapshot de censo inesperado en %s", row.CharacterizationRef)
		}
		switch mapping.State {
		case "linked_exact":
			if len(mapping.Links) == 0 {
				return fmt.Errorf("linked_exact vacío en %s", row.CharacterizationRef)
			}
		case "pending", "reviewed_no_direct_function", "blocked_historical_snapshot":
			if len(mapping.Links) != 0 {
				return fmt.Errorf("estado %s contiene enlaces en %s", mapping.State, row.CharacterizationRef)
			}
		default:
			return fmt.Errorf("estado de mapping desconocido: %q", mapping.State)
		}
		for _, link := range mapping.Links {
			pair := row.CharacterizationRef + "\x00" + link.OccurrenceRef
			if _, duplicate := pairs[pair]; duplicate {
				return fmt.Errorf("enlace conducta/aparición duplicado: %s", row.CharacterizationRef)
			}
			pairs[pair] = struct{}{}
			if err := validateAssessmentFunctionLink(repository, link); err != nil {
				return fmt.Errorf("%s: %w", row.CharacterizationRef, err)
			}
			if err := validateFunctionLinkEvidence(
				repository, row.CharacterizationRef, link,
				behaviorSources[row.CharacterizationRef], observations, contract,
			); err != nil {
				return fmt.Errorf("%s: %w", row.CharacterizationRef, err)
			}
		}
	}
	for behaviorRef := range behaviorSources {
		if _, exists := behaviorRefs[behaviorRef]; !exists {
			return fmt.Errorf("conducta sin assessment: %s", behaviorRef)
		}
	}
	return nil
}

func loadBehaviorSourceRefs(repository string) (map[string]map[string]struct{}, error) {
	ledgerPath := filepath.Join(repository, "product", "traceability", "task_entries.jsonl")
	ledgerRaw, err := os.ReadFile(ledgerPath)
	if err != nil {
		return nil, err
	}
	ledgerRefs := make(map[string]struct{})
	for _, raw := range bytes.Split(bytes.TrimSuffix(ledgerRaw, []byte{'\n'}), []byte{'\n'}) {
		var row struct {
			EntryRef string `json:"entry_ref"`
		}
		if err := decodeUniqueJSON(raw, &row); err != nil || row.EntryRef == "" {
			return nil, fmt.Errorf("ledger task entry inválido: %w", err)
		}
		if _, duplicate := ledgerRefs[row.EntryRef]; duplicate {
			return nil, fmt.Errorf("task entry duplicada: %s", row.EntryRef)
		}
		ledgerRefs[row.EntryRef] = struct{}{}
	}
	paths, err := filepath.Glob(filepath.Join(repository, "product", "traceability", "fixtures", "behavior_characterization_agent_batch_*.jsonl"))
	if err != nil {
		return nil, err
	}
	result := make(map[string]map[string]struct{})
	for _, path := range paths {
		rawFile, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		for _, raw := range bytes.Split(bytes.TrimSuffix(rawFile, []byte{'\n'}), []byte{'\n'}) {
			var row struct {
				CharacterizationRef string   `json:"characterization_ref"`
				TaskEntryRefs       []string `json:"task_entry_refs"`
				Evidence            []struct {
					SourceRef string `json:"source_ref"`
				} `json:"evidence"`
			}
			if err := decodeUniqueJSON(raw, &row); err != nil {
				return nil, fmt.Errorf("fixture %s: %w", path, err)
			}
			if _, duplicate := result[row.CharacterizationRef]; duplicate {
				return nil, fmt.Errorf("behavior duplicado: %s", row.CharacterizationRef)
			}
			refs := make(map[string]struct{})
			for _, evidence := range row.Evidence {
				refs[evidence.SourceRef] = struct{}{}
			}
			for _, taskEntryRef := range row.TaskEntryRefs {
				if _, exists := ledgerRefs[taskEntryRef]; !exists {
					return nil, fmt.Errorf("task entry %s de %s no existe en ledger",
						taskEntryRef, row.CharacterizationRef)
				}
				refs[taskEntryRef] = struct{}{}
			}
			result[row.CharacterizationRef] = refs
		}
	}
	return result, nil
}

func loadHistoricalTestObservations(repository string, contract legacyGoCensusContract) (map[string]historicalTestObservation, error) {
	path := filepath.Join(repository, "product", "knowledge", "legacy_test_observations_v1.jsonl")
	rawFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	result := make(map[string]historicalTestObservation)
	for _, raw := range bytes.Split(bytes.TrimSuffix(rawFile, []byte{'\n'}), []byte{'\n'}) {
		var observation historicalTestObservation
		if err := decodeUniqueStrictJSON(raw, &observation); err != nil {
			return nil, err
		}
		if _, duplicate := result[observation.ObservationRef]; duplicate {
			return nil, fmt.Errorf("observación duplicada: %s", observation.ObservationRef)
		}
		if err := validateHistoricalTestObservation(repository, observation, contract); err != nil {
			return nil, fmt.Errorf("%s: %w", observation.ObservationRef, err)
		}
		result[observation.ObservationRef] = observation
	}
	return result, nil
}

func validateHistoricalTestObservation(repository string, observation historicalTestObservation, contract legacyGoCensusContract) error {
	if observation.SchemaVersion != 1 || observation.RepositoryCommit == "" ||
		observation.ModulePath == "" || observation.ModuleTreeOID == "" ||
		observation.Result != "passed" || observation.ClaimsAccreditation ||
		len(observation.TestRefs) == 0 || len(observation.BehaviorRefs) == 0 {
		return errors.New("observación histórica incompleta")
	}
	tree, err := gitBytes(repository, nil, "rev-parse", observation.RepositoryCommit+":"+observation.ModulePath)
	if err != nil || strings.TrimSpace(string(tree)) != observation.ModuleTreeOID {
		return fmt.Errorf("commit:tree no coincide para %s", observation.ModulePath)
	}
	for _, testRef := range observation.TestRefs {
		path, name, ok := strings.Cut(testRef, "#")
		if !ok || !strings.HasPrefix(path, observation.ModulePath+"/") ||
			!validLegacyTestPath(path, contract) {
			return fmt.Errorf("test_ref histórico inválido: %s", testRef)
		}
		content, err := gitBytes(repository, nil, "show", observation.RepositoryCommit+":"+path)
		if err != nil {
			return err
		}
		if err := requireFunctionInGoBlob(content, path, name); err != nil {
			return err
		}
	}
	return nil
}

func validateFunctionLinkEvidence(
	repository, behaviorRef string,
	link assessmentFunctionLink,
	behaviorSources map[string]struct{},
	observations map[string]historicalTestObservation,
	contract legacyGoCensusContract,
) error {
	for _, evidenceRef := range link.EvidenceRefs {
		if observation, ok := observations[evidenceRef]; ok {
			if !containsString(observation.BehaviorRefs, behaviorRef) {
				return fmt.Errorf("observación %s no pertenece a %s", evidenceRef, behaviorRef)
			}
			continue
		}
		if path, name, found := strings.Cut(evidenceRef, "#"); found {
			if !validLegacyTestPath(path, contract) {
				return fmt.Errorf("evidence test_ref fuera de policy: %s", evidenceRef)
			}
			object, err := readIndexBlob(repository, path)
			if err != nil {
				return err
			}
			if err := requireFunctionInGoBlob(object.content, path, name); err != nil {
				return err
			}
			continue
		}
		if _, ok := behaviorSources[evidenceRef]; !ok {
			return fmt.Errorf("evidencia %s no pertenece a la conducta", evidenceRef)
		}
		if strings.HasPrefix(evidenceRef, "TASKENTRY-") {
			continue
		}
		if _, err := readIndexBlob(repository, evidenceRef); err != nil {
			return err
		}
	}
	return nil
}

func validLegacyTestPath(path string, contract legacyGoCensusContract) bool {
	if path == "" || filepath.IsAbs(path) || filepath.Clean(path) != path ||
		!strings.HasSuffix(path, contract.SourcePolicy.ExcludeTestSuffix) ||
		legacyGoPathExcluded(path, contract.SourcePolicy.ExactExclusions) {
		return false
	}
	for _, rule := range contract.ModuleRules {
		for _, modulePath := range rule.Paths {
			if strings.HasPrefix(path, modulePath+"/") {
				return true
			}
		}
	}
	return false
}

func requireFunctionInGoBlob(content []byte, path, name string) error {
	parsed, err := parseBlob(content, path)
	if err != nil || parsed.failure != nil {
		return fmt.Errorf("test Go no parseable: %s", path)
	}
	for _, record := range parsed.records {
		if record.Name == name {
			return nil
		}
	}
	return fmt.Errorf("función de evidencia ausente: %s#%s", path, name)
}

func decodeUniqueJSON(raw []byte, destination any) error {
	unique := json.NewDecoder(bytes.NewReader(raw))
	if err := readUniqueJSONValue(unique); err != nil {
		return err
	}
	return json.Unmarshal(raw, destination)
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func validateAssessmentFunctionLink(repository string, link assessmentFunctionLink) error {
	if link.Relation != "primary_implementation" &&
		link.Relation != "supporting_mechanism" && link.Relation != "negative_example" {
		return fmt.Errorf("relación desconocida: %q", link.Relation)
	}
	if len(strings.TrimSpace(link.SemanticRationale)) < 30 || len(link.EvidenceRefs) == 0 ||
		link.ReviewState != "bootstrap_symbol_mapping_reviewed_advisory" {
		return fmt.Errorf("revisión semántica incompleta para %s", link.OccurrenceRef)
	}
	actual, err := lookupIndexSymbols(repository, link.SourcePath, link.Name)
	if err != nil {
		return err
	}
	wanted := link.symbolRecord()
	for _, candidate := range actual {
		if candidate.OccurrenceRef == link.OccurrenceRef {
			if !reflect.DeepEqual(candidate, wanted) {
				return fmt.Errorf("metadata manipulada para %s", link.OccurrenceRef)
			}
			return nil
		}
	}
	for _, candidate := range actual {
		if candidate.SymbolKind == link.SymbolKind && candidate.Receiver == link.Receiver &&
			candidate.Name == link.Name {
			if candidate.Signature != link.Signature {
				return fmt.Errorf("signature_drift para %s", link.OccurrenceRef)
			}
			if candidate.VariantRef != link.VariantRef {
				return fmt.Errorf("implementation_drift para %s", link.OccurrenceRef)
			}
		}
	}
	return fmt.Errorf("aparición huérfana: %s", link.OccurrenceRef)
}

func TestAssessmentFunctionLinksMatchCurrentGitIndex(t *testing.T) {
	repository := filepath.Join("..", "..")
	contract, err := loadLegacyGoCensusContract(repository)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := readReuseAssessments(filepath.Join(repository, "product", "knowledge", "legacy_reuse_assessments_v1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	manifestRaw, err := os.ReadFile(filepath.Join(repository, "product", "knowledge",
		"legacy_reuse_assessments_v1.manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest reuseAssessmentManifest
	if err := decodeUniqueStrictJSON(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.DocumentKind != "legacy_reuse_assessments_manifest" ||
		manifest.SchemaVersion != 1 || manifest.Authority != "derived_advisory_read_model" ||
		manifest.CanonicalStateChange || manifest.ClaimsAccreditation ||
		manifest.SnapshotCensusSHA256 != contract.Baseline.CensusSHA256 ||
		manifest.AssessmentCount != len(rows) ||
		manifest.UniqueCharacterizationCount != len(rows) || len(rows) < 53 {
		t.Fatalf("manifiesto/assessment incoherente: rows=%d manifest=%#v", len(rows), manifest)
	}
	if err := validateAssessmentFunctionLinks(repository, rows, contract); err != nil {
		t.Fatal(err)
	}
	linked, reviewedWithoutDirectFunction, links := 0, 0, 0
	occurrences := make(map[string]struct{})
	for _, row := range rows {
		if row.FunctionMapping.State == "linked_exact" {
			linked++
			links += len(row.FunctionMapping.Links)
			for _, link := range row.FunctionMapping.Links {
				occurrences[link.OccurrenceRef] = struct{}{}
			}
		} else if row.FunctionMapping.State == "reviewed_no_direct_function" {
			reviewedWithoutDirectFunction++
		}
	}
	if linked+reviewedWithoutDirectFunction != manifest.AssessmentCount ||
		linked < 53 || links != manifest.FunctionBehaviorLinkCount ||
		len(occurrences) != manifest.UniqueFunctionOccurrenceCount {
		t.Fatalf("cobertura inesperada: linked=%d no_direct=%d links=%d occurrences=%d manifest=%#v",
			linked, reviewedWithoutDirectFunction, links, len(occurrences), manifest)
	}
}

func TestAssessmentFunctionLinkRejectsDriftAndIncompleteReview(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	path := "modulos/orquesta-sample/worker.go"
	writeTestFile(t, repository, path, "package sample\nfunc Stable(value int) int { return value }\n")
	writeLookupCensusContract(t, repository, "modulos/orquesta-sample")
	gitTest(t, repository, "add", ".")
	records, err := lookupIndexSymbols(repository, path, "Stable")
	if err != nil || len(records) != 1 {
		t.Fatalf("preparar símbolo: records=%#v err=%v", records, err)
	}
	base := assessmentFunctionLink{
		SchemaVersion: records[0].SchemaVersion, OccurrenceRef: records[0].OccurrenceRef,
		DeclarationRef: records[0].DeclarationRef, VariantRef: records[0].VariantRef,
		SourceRoot: records[0].SourceRoot, SourcePath: records[0].SourcePath,
		GitBlobOID: records[0].GitBlobOID, BlobDigest: records[0].BlobDigest,
		PackagePath: records[0].PackagePath, PackageName: records[0].PackageName,
		SymbolKind: records[0].SymbolKind, Name: records[0].Name,
		Receiver: records[0].Receiver, Signature: records[0].Signature,
		SourceSHA: records[0].SourceSHA, ASTSHA: records[0].ASTSHA,
		BodySHA: records[0].BodySHA, StartLine: records[0].StartLine, EndLine: records[0].EndLine,
		Relation:          "primary_implementation",
		SemanticRationale: "Implementación primaria revisada para una prueba negativa aislada.",
		EvidenceRefs:      []string{"sample_test.go#TestStable"},
		ReviewState:       "bootstrap_symbol_mapping_reviewed_advisory",
	}
	if err := validateAssessmentFunctionLink(repository, base); err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name   string
		mutate func(*assessmentFunctionLink)
	}{
		{"occurrence", func(item *assessmentFunctionLink) { item.OccurrenceRef = "sha256:manipulado" }},
		{"signature", func(item *assessmentFunctionLink) { item.Signature = "func Stable()" }},
		{"variant", func(item *assessmentFunctionLink) { item.VariantRef = "go-function-variant:sha256:manipulado" }},
		{"path", func(item *assessmentFunctionLink) { item.SourcePath = "modulos/orquesta-sample/missing.go" }},
		{"review", func(item *assessmentFunctionLink) { item.ReviewState = "pending" }},
	}
	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			changed := base
			test.mutate(&changed)
			if err := validateAssessmentFunctionLink(repository, changed); err == nil {
				t.Fatal("se aceptó metadata manipulada")
			}
		})
	}
}

func TestReadReuseAssessmentsRejectsDuplicateNoncanonicalAndMissingLF(t *testing.T) {
	valid, err := json.Marshal(reuseAssessment{SchemaVersion: 1, CharacterizationRef: "BEHAVIOR-TEST"})
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string][]byte{
		"duplicate": bytes.Replace(valid, []byte(`"schema_version":1`),
			[]byte(`"schema_version":2,"schema_version":1`), 1),
		"noncanonical": append([]byte(" "), append(valid, '\n')...),
		"missing_lf":   valid,
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "assessments.jsonl")
			if name == "duplicate" {
				content = append(content, '\n')
			}
			if err := os.WriteFile(path, content, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := readReuseAssessments(path); err == nil {
				t.Fatal("assessment no canónico aceptado")
			}
		})
	}
}

func TestFunctionLinkEvidenceRejectsForeignObservationAndMissingTest(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	writeLookupCensusContract(t, repository, "modulos/orquesta-sample")
	writeTestFile(t, repository, "modulos/orquesta-sample/worker_test.go",
		"package sample\nfunc TestStable() {}\n")
	gitTest(t, repository, "add", ".")
	contract, err := loadLegacyGoCensusContract(repository)
	if err != nil {
		t.Fatal(err)
	}
	link := assessmentFunctionLink{
		EvidenceRefs: []string{"OBS-1"},
	}
	observations := map[string]historicalTestObservation{
		"OBS-1": {BehaviorRefs: []string{"BEHAVIOR-OTHER"}},
	}
	if err := validateFunctionLinkEvidence(
		repository, "BEHAVIOR-WANTED", link, nil, observations, contract,
	); err == nil || !strings.Contains(err.Error(), "no pertenece") {
		t.Fatalf("observación ajena aceptada: %v", err)
	}
	link.EvidenceRefs = []string{"modulos/orquesta-sample/worker_test.go#TestMissing"}
	if err := validateFunctionLinkEvidence(
		repository, "BEHAVIOR-WANTED", link, nil, nil, contract,
	); err == nil || !strings.Contains(err.Error(), "ausente") {
		t.Fatalf("test ausente aceptado: %v", err)
	}
}

func TestHistoricalObservationRejectsTreeDrift(t *testing.T) {
	repository := filepath.Join("..", "..")
	contract, err := loadLegacyGoCensusContract(repository)
	if err != nil {
		t.Fatal(err)
	}
	observations, err := loadHistoricalTestObservations(repository, contract)
	if err != nil {
		t.Fatal(err)
	}
	for _, observation := range observations {
		observation.ModuleTreeOID = "0000000000000000000000000000000000000000"
		if err := validateHistoricalTestObservation(repository, observation, contract); err == nil ||
			!strings.Contains(err.Error(), "commit:tree") {
			t.Fatalf("tree drift aceptado: %v", err)
		}
		return
	}
	t.Fatal("faltan observaciones para la mutación")
}
