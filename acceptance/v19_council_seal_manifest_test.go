package acceptance_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"orquesta/internal/council"
)

type v19SealManifest struct {
	SchemaVersion int                  `json:"schema_version"`
	ManifestKind  string               `json:"manifest_kind"`
	ContractID    string               `json:"contract_id"`
	SealState     string               `json:"seal_state"`
	ProductSource v19SealProductSource `json:"product_source"`
	Binary        struct {
		SourceGitCommitOID string   `json:"source_git_commit_oid"`
		Target             string   `json:"target"`
		BuildArgv          []string `json:"build_argv"`
		SHA256             string   `json:"sha256"`
		SizeBytes          int64    `json:"size_bytes"`
		GoVersion          string   `json:"go_version"`
	} `json:"binary"`
	EffectiveConfig struct {
		Mode             string `json:"mode"`
		SchemaVersion    int    `json:"schema_version"`
		RegistryRevision string `json:"registry_revision"`
		RegistrySHA256   string `json:"registry_sha256"`
		SnapshotSHA256   string `json:"snapshot_sha256"`
		EffectiveSHA256  string `json:"effective_json_sha256"`
		Redaction        string `json:"redaction"`
	} `json:"effective_config"`
	DependencyV18 struct {
		ReceiptPath              string `json:"receipt_path"`
		ReceiptSHA256            string `json:"receipt_sha256"`
		Contract                 string `json:"contract"`
		Result                   string `json:"result"`
		SealedSourceGitCommitOID string `json:"sealed_source_git_commit_oid"`
		CandidateSHA256          string `json:"candidate_sha256"`
	} `json:"dependency_v18"`
	CouncilSubjects []v19SealedCouncilSubject `json:"council_subjects"`
	Budget          v19SealBudget             `json:"budget"`
}

type v19SealProductSource struct {
	BaseGitCommitOID         string `json:"base_git_commit_oid"`
	GitCommitOID             string `json:"git_commit_oid"`
	GitTreeOID               string `json:"git_tree_oid"`
	CandidateSubjectsCount   int    `json:"candidate_subjects_count"`
	CandidateDigestAlgorithm string `json:"candidate_digest_algorithm"`
	CandidateSubjectsSHA     string `json:"candidate_subjects_sha256"`
}

type v19SealedCouncilSubject struct {
	Policy              string `json:"policy"`
	ProjectRef          string `json:"project_ref"`
	ReviewSubjectDigest string `json:"review_subject_digest"`
	ReviewGateDigest    string `json:"review_gate_digest"`
	GoalRef             string `json:"goal_ref"`
	WorkItemRef         string `json:"work_item_ref"`
	ChangeSetRef        string `json:"change_set_ref"`
	SpecHash            string `json:"spec_hash"`
	PlanGeneration      uint64 `json:"plan_generation"`
	WorkItemGeneration  uint64 `json:"work_item_generation"`
	AppSpecGeneration   uint64 `json:"app_spec_generation"`
	SubjectDigest       string `json:"subject_digest"`
}

type v19SealLOC struct {
	Domain            int `json:"domain"`
	Application       int `json:"application"`
	SQLiteRecovery    int `json:"sqlite_recovery"`
	Bootstrap         int `json:"bootstrap"`
	TransportAdapters int `json:"transport_adapters"`
	Total             int `json:"total"`
}

type v19SealLargeFile struct {
	Path string `json:"path"`
	LOC  int    `json:"loc"`
	Kind string `json:"kind"`
}

type v19SealBudget struct {
	BaseGitCommitOID    string             `json:"base_git_commit_oid"`
	ProductGitCommitOID string             `json:"product_git_commit_oid"`
	Classifier          string             `json:"classifier"`
	ProductionNet       v19SealLOC         `json:"production_net_loc"`
	TestSupportNet      v19SealLOC         `json:"test_support_net_loc"`
	Ceilings            v19SealLOC         `json:"production_ceilings"`
	FileLOCMax          int                `json:"file_loc_max"`
	Quality             string             `json:"quality_accreditation"`
	PostV22Debt         string             `json:"post_v22_debt"`
	FilesOver350        []v19SealLargeFile `json:"files_over_350"`
}

func v19SealBuildArgv() []string {
	return []string{"env", "CGO_ENABLED=0", "go", "build", "-mod=vendor", "-trimpath", "-buildvcs=false", "-ldflags=-buildid=", "-o", "$TMPDIR/orquesta-v19", "./cmd/orquesta"}
}

func loadV19SealManifest(t *testing.T, repositoryRoot string, fixture v19Fixture) v19SealManifest {
	t.Helper()
	manifest := evidenceDecodeStrictJSON[v19SealManifest](t, filepath.Join(repositoryRoot, v19SealManifestPath))
	if err := validateV19SealManifest(manifest, fixture); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func validateV19SealManifest(manifest v19SealManifest, fixture v19Fixture) error {
	if manifest.SchemaVersion != 1 || manifest.ManifestKind != "orquesta.v19_council.seal_manifest.v1" ||
		manifest.ContractID != "AC-V19-COUNCIL" || manifest.SealState != "sealed_unexecuted" {
		return fmt.Errorf("v19 seal identity invalid")
	}
	product := manifest.ProductSource
	if fixture.ProductDeltaSealedGitCommitOID == "" || product.BaseGitCommitOID != fixture.ProductDeltaBaseGitCommitOID ||
		product.GitCommitOID != fixture.ProductDeltaSealedGitCommitOID ||
		evidenceValidateGitOID(product.GitCommitOID) != nil || evidenceValidateGitOID(product.GitTreeOID) != nil ||
		product.CandidateSubjectsCount != len(fixture.CandidateSubjects) ||
		product.CandidateDigestAlgorithm != evidenceCandidateDigestAlgorithmV3 || !v19SealSHA(product.CandidateSubjectsSHA) {
		return fmt.Errorf("v19 seal product source invalid")
	}
	binary := manifest.Binary
	if binary.SourceGitCommitOID != product.GitCommitOID || binary.Target != "./cmd/orquesta" ||
		!reflect.DeepEqual(binary.BuildArgv, v19SealBuildArgv()) || !v19SealSHA(binary.SHA256) ||
		binary.SizeBytes <= 0 || binary.GoVersion == "" || strings.ContainsAny(binary.GoVersion, "\r\n") {
		return fmt.Errorf("v19 seal binary invalid")
	}
	effective := manifest.EffectiveConfig
	if effective.Mode != "canonical_registry_defaults" || effective.SchemaVersion != 2 ||
		effective.RegistryRevision == "" || !v19SealSHA(effective.RegistrySHA256) ||
		!v19SealSHA(effective.SnapshotSHA256) || !v19SealSHA(effective.EffectiveSHA256) ||
		effective.Redaction != "Snapshot.EffectiveJSON_sensitive_values_redacted" {
		return fmt.Errorf("v19 seal effective config invalid")
	}
	dependency := manifest.DependencyV18
	if dependency.ReceiptPath != "product/evidence/v18_independent_reviews.json" ||
		dependency.Contract != "AC-V18-INDEPENDENT-REVIEWS" || dependency.Result != "PASS" ||
		dependency.SealedSourceGitCommitOID != "32ee17e407006d9e0aeb46557b1e160769dd4848" ||
		!v19SealSHA(dependency.ReceiptSHA256) || !v19SealSHA(dependency.CandidateSHA256) {
		return fmt.Errorf("v19 seal V18 dependency invalid")
	}
	if err := validateV19SealedCouncilSubjects(manifest.CouncilSubjects); err != nil {
		return err
	}
	return validateV19SealBudget(manifest.Budget, product)
}

func validateV19SealedCouncilSubjects(subjects []v19SealedCouncilSubject) error {
	want := map[council.Policy]bool{council.PolicyAuto: false, council.PolicyRequired: false, council.PolicySkipByOperator: false}
	if len(subjects) != len(want) {
		return fmt.Errorf("v19 seal Council subject count invalid")
	}
	for _, sealed := range subjects {
		policy := council.Policy(sealed.Policy)
		if _, found := want[policy]; !found || want[policy] {
			return fmt.Errorf("v19 seal Council policy invalid")
		}
		subject, err := council.NewSubject(council.Subject{
			ProjectRef: sealed.ProjectRef, ReviewSubjectDigest: sealed.ReviewSubjectDigest,
			ReviewGateDigest: sealed.ReviewGateDigest, Policy: policy, GoalRef: sealed.GoalRef,
			WorkItemRef: sealed.WorkItemRef, ChangeSetRef: sealed.ChangeSetRef, SpecHash: sealed.SpecHash,
			PlanGeneration: sealed.PlanGeneration, WorkItemGeneration: sealed.WorkItemGeneration,
			AppSpecGeneration: sealed.AppSpecGeneration,
		})
		if err != nil || subject.Digest() != sealed.SubjectDigest {
			return fmt.Errorf("v19 seal Council subject invalid for %s", policy)
		}
		want[policy] = true
	}
	return nil
}

func validateV19SealBudget(budget v19SealBudget, product v19SealProductSource) error {
	production, tests, ceilings := budget.ProductionNet, budget.TestSupportNet, budget.Ceilings
	if budget.BaseGitCommitOID != product.BaseGitCommitOID || budget.ProductGitCommitOID != product.GitCommitOID ||
		budget.Classifier != "v19_candidate_layered_net_v2" ||
		production != (v19SealLOC{Domain: 387, Application: 1214, SQLiteRecovery: 1835, Bootstrap: 0, TransportAdapters: 5, Total: 3441}) ||
		tests != (v19SealLOC{Domain: 376, Application: 1223, SQLiteRecovery: 1421, Bootstrap: 668, TransportAdapters: 43, Total: 3731}) ||
		ceilings != (v19SealLOC{Domain: 400, Application: 1300, SQLiteRecovery: 1850, Bootstrap: 400, TransportAdapters: 50, Total: 3650}) ||
		budget.FileLOCMax != 1500 || budget.Quality != "debt_recorded_not_simplicity_green" ||
		strings.TrimSpace(budget.PostV22Debt) == "" || len(budget.FilesOver350) != 53 {
		return fmt.Errorf("v19 seal budget invalid")
	}
	paths := make([]string, len(budget.FilesOver350))
	for index, file := range budget.FilesOver350 {
		if file.Path == "" || file.LOC <= 350 || (file.Kind != "production" && file.Kind != "test" && file.Kind != "docs_data") {
			return fmt.Errorf("v19 seal large file invalid")
		}
		paths[index] = file.Path
	}
	if !sort.StringsAreSorted(paths) {
		return fmt.Errorf("v19 seal large files not sorted")
	}
	for index := 1; index < len(paths); index++ {
		if paths[index] == paths[index-1] {
			return fmt.Errorf("v19 seal duplicate large file")
		}
	}
	return nil
}

func v19SealSHA(value string) bool {
	raw := strings.TrimPrefix(value, "sha256:")
	if len(value) != 71 || len(raw) != 64 || strings.ToLower(raw) != raw {
		return false
	}
	_, err := hex.DecodeString(raw)
	return err == nil
}

func TestV19SealManifestStrictSchemaRejectsUnknownAndTamperedIdentity(t *testing.T) {
	fixture := loadV19Fixture(t)
	fixture.ProductDeltaSealedGitCommitOID = strings.Repeat("a", 40)
	manifest := validV19SealManifestFixture(t, fixture)
	if err := validateV19SealManifest(manifest, fixture); err != nil {
		t.Fatal(err)
	}
	manifest.Binary.SHA256 = "sha256:bad"
	if validateV19SealManifest(manifest, fixture) == nil {
		t.Fatal("tampered V19 binary digest accepted")
	}
	if v19ConfigIdentityMatches([]string{"2", "test", "sha256:" + strings.Repeat("0", 64), manifest.EffectiveConfig.SnapshotSHA256, manifest.EffectiveConfig.EffectiveSHA256}, manifest) {
		t.Fatal("tampered V19 sealed registry identity accepted")
	}
	content, err := json.Marshal(validV19SealManifestFixture(t, fixture))
	if err != nil {
		t.Fatal(err)
	}
	baseContent := append([]byte(nil), content...)
	content = append(content[:len(content)-1], []byte(`,"unexpected":true}`)...)
	path := filepath.Join(t.TempDir(), "seal.json")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := evidenceDecodeStrictJSONFile[v19SealManifest](path); err == nil {
		t.Fatal("unknown V19 seal field accepted")
	}
	nested := strings.Replace(string(baseContent), `"domain":387`, `"domain":387,"unknown":1`, 1)
	if nested == string(baseContent) {
		t.Fatal("V19 nested strict-schema fixture did not mutate")
	}
	if err := os.WriteFile(path, []byte(nested), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := evidenceDecodeStrictJSONFile[v19SealManifest](path); err == nil {
		t.Fatal("unknown nested V19 seal field accepted")
	}
}

func validV19SealManifestFixture(t *testing.T, fixture v19Fixture) v19SealManifest {
	t.Helper()
	var result v19SealManifest
	result.SchemaVersion, result.ManifestKind, result.ContractID, result.SealState = 1, "orquesta.v19_council.seal_manifest.v1", "AC-V19-COUNCIL", "sealed_unexecuted"
	result.ProductSource.BaseGitCommitOID = fixture.ProductDeltaBaseGitCommitOID
	result.ProductSource.GitCommitOID, result.ProductSource.GitTreeOID = fixture.ProductDeltaSealedGitCommitOID, strings.Repeat("b", 40)
	result.ProductSource.CandidateDigestAlgorithm = evidenceCandidateDigestAlgorithmV3
	result.ProductSource.CandidateSubjectsCount, result.ProductSource.CandidateSubjectsSHA = len(fixture.CandidateSubjects), "sha256:"+strings.Repeat("a", 64)
	result.Binary.SourceGitCommitOID, result.Binary.Target = result.ProductSource.GitCommitOID, "./cmd/orquesta"
	result.Binary.BuildArgv, result.Binary.SHA256, result.Binary.SizeBytes, result.Binary.GoVersion = v19SealBuildArgv(), "sha256:"+strings.Repeat("b", 64), 1, "go version test"
	result.EffectiveConfig.Mode, result.EffectiveConfig.SchemaVersion = "canonical_registry_defaults", 2
	result.EffectiveConfig.RegistryRevision, result.EffectiveConfig.Redaction = "test", "Snapshot.EffectiveJSON_sensitive_values_redacted"
	result.EffectiveConfig.RegistrySHA256, result.EffectiveConfig.SnapshotSHA256, result.EffectiveConfig.EffectiveSHA256 = "sha256:"+strings.Repeat("c", 64), "sha256:"+strings.Repeat("d", 64), "sha256:"+strings.Repeat("e", 64)
	result.DependencyV18.ReceiptPath, result.DependencyV18.ReceiptSHA256 = "product/evidence/v18_independent_reviews.json", "sha256:"+strings.Repeat("f", 64)
	result.DependencyV18.Contract, result.DependencyV18.Result = "AC-V18-INDEPENDENT-REVIEWS", "PASS"
	result.DependencyV18.SealedSourceGitCommitOID, result.DependencyV18.CandidateSHA256 = "32ee17e407006d9e0aeb46557b1e160769dd4848", "sha256:"+strings.Repeat("1", 64)
	for _, policy := range []council.Policy{council.PolicyAuto, council.PolicyRequired, council.PolicySkipByOperator} {
		sealed := v19SealedCouncilSubject{Policy: string(policy), ProjectRef: "project:seal", ReviewSubjectDigest: "sha256:" + strings.Repeat("2", 64), ReviewGateDigest: "sha256:" + strings.Repeat("3", 64), GoalRef: "goal:seal", WorkItemRef: "work-item:seal", ChangeSetRef: "change-set:seal", SpecHash: strings.Repeat("4", 64), PlanGeneration: 1, WorkItemGeneration: 2, AppSpecGeneration: 3}
		subject, err := council.NewSubject(council.Subject{ProjectRef: sealed.ProjectRef, ReviewSubjectDigest: sealed.ReviewSubjectDigest, ReviewGateDigest: sealed.ReviewGateDigest, Policy: policy, GoalRef: sealed.GoalRef, WorkItemRef: sealed.WorkItemRef, ChangeSetRef: sealed.ChangeSetRef, SpecHash: sealed.SpecHash, PlanGeneration: 1, WorkItemGeneration: 2, AppSpecGeneration: 3})
		if err != nil {
			t.Fatal(err)
		}
		sealed.SubjectDigest = subject.Digest()
		result.CouncilSubjects = append(result.CouncilSubjects, sealed)
	}
	result.Budget = v19SealBudget{BaseGitCommitOID: fixture.ProductDeltaBaseGitCommitOID, ProductGitCommitOID: fixture.ProductDeltaSealedGitCommitOID, Classifier: "v19_candidate_layered_net_v2", ProductionNet: v19SealLOC{Domain: 387, Application: 1214, SQLiteRecovery: 1835, Bootstrap: 0, TransportAdapters: 5, Total: 3441}, TestSupportNet: v19SealLOC{Domain: 376, Application: 1223, SQLiteRecovery: 1421, Bootstrap: 668, TransportAdapters: 43, Total: 3731}, Ceilings: v19SealLOC{Domain: 400, Application: 1300, SQLiteRecovery: 1850, Bootstrap: 400, TransportAdapters: 50, Total: 3650}, FileLOCMax: 1500, Quality: "debt_recorded_not_simplicity_green", PostV22Debt: "split after V22"}
	for index := 0; index < 53; index++ {
		result.Budget.FilesOver350 = append(result.Budget.FilesOver350, v19SealLargeFile{Path: fmt.Sprintf("path/%02d", index), LOC: 351, Kind: "test"})
	}
	return result
}
