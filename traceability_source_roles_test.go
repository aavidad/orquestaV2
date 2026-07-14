package orquesta_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const traceMarkdownSourceRolesPath = "product/traceability/markdown_source_roles.jsonl"

var traceMarkdownReviewBatches = map[string]struct {
	Path          string
	FixtureCount  int
	EvidenceCount int
}{
	"v03_candidate_sources_review_s0@sha256:323a1793826dff16d371ddff0a878026cf113b2af7dd3c67a5b5ee777aa8044b":           {Path: "product/traceability/fixtures/markdown_source_review_s0.jsonl", FixtureCount: 41, EvidenceCount: 41},
	"v03_candidate_sources_review_s1@sha256:77f1e6972a21e6730bd6febb004d1bbe3495ae08c6b7b26acf283ec2f1971556":           {Path: "product/traceability/fixtures/markdown_source_review_s1.jsonl", FixtureCount: 38, EvidenceCount: 38},
	"v03_candidate_sources_review_s2@sha256:1a10145bc01bb339c37aa3efd479e6d1caef1db1eed3e29e5118a7d2ea0d78c9":           {Path: "product/traceability/fixtures/markdown_source_review_s2.jsonl", FixtureCount: 37, EvidenceCount: 36},
	"v03_tasklike_source_review@sha256:11d6801f24602e64adf7ddc98def8302d2660621f0397583544fb7fd0fe864c5":                {Path: "product/traceability/fixtures/markdown_tasklike_source_review.jsonl", FixtureCount: 21, EvidenceCount: 21},
	"v03_markdown_source_role_followup_reviews@sha256:aa6c6a5cc79f360abdebb260802034a97c3ca3e1f845d3ec6a3afe208fc619d6": {Path: "product/traceability/fixtures/markdown_source_role_followup_reviews.jsonl", FixtureCount: 2, EvidenceCount: 2},
	"v03_broad_source_review_s0@sha256:fabe108cac513a312226d6335f189312c0ba8d2e644b1bf0cb4e88e545eb4844":                {Path: "product/traceability/fixtures/markdown_broad_source_review_s0.jsonl", FixtureCount: 21, EvidenceCount: 21},
	"v03_broad_source_review_s1@sha256:c9ff30ed5f8cad4bbcf26e31154bf96a497663a3710649da37c3bd3e0dcc605d":                {Path: "product/traceability/fixtures/markdown_broad_source_review_s1.jsonl", FixtureCount: 20, EvidenceCount: 20},
	"v03_broad_source_review_s2@sha256:4e5218535b2093aa8f9dca648aaab60b87f7ede08108c3fb5f12f49fbc053cc4":                {Path: "product/traceability/fixtures/markdown_broad_source_review_s2.jsonl", FixtureCount: 20, EvidenceCount: 20},
	"v03_source_counterreview@sha256:2cbe467f301987f2e891331dd90991416a7242ddf9c50a22efeff42f31b1a2db":                  {Path: "product/traceability/fixtures/markdown_source_counterreview.jsonl", FixtureCount: 28, EvidenceCount: 21},
}

var traceMarkdownFixedReviewAdjudications = map[string]traceMarkdownReviewAdjudication{
	"modulos/orquesta-core-concurrency/docs/promocion_core_workflow.md": {
		Classification: "task_delegated",
		Ref:            "operator-adjudication:2026-07-14:core-concurrency-promotion",
		Reason:         "La promoción de concurrencia no abre un hueco nuevo; las unidades ejecutables quedan delegadas en owners canónicos ya inventariados.",
	},
	"modulos/orquesta-core-leases/docs/promocion_core_workflow.md": {
		Classification: "task_delegated",
		Ref:            "review-precedence:scoped-candidate-review-over-filename-review",
		Reason:         "La revisión acotada identifica owners delegados concretos y prevalece sobre la señal mecánica del nombre promoción.",
	},
	"modulos/orquesta-core-workflow/docs/roadmap_cierre_nucleo.md": {
		Classification: "task_source",
		Ref:            "operator-adjudication:2026-07-14:core-workflow-roadmap",
		Reason:         "Existe un hueco ejecutable exacto en las líneas 256-257; el resto contextual no elimina esa unidad propia.",
	},
}

const traceMarkdownDeployReviewOutsideCensus = "modulos/orquesta-deploy/README.md"

type traceMarkdownSourceRole struct {
	SchemaVersion       int                           `json:"schema_version"`
	SourceRef           string                        `json:"source_ref"`
	SourceSHA256        string                        `json:"source_sha256"`
	Origin              string                        `json:"origin"`
	Classification      string                        `json:"classification"`
	ClassificationBasis string                        `json:"classification_basis"`
	Roles               []string                      `json:"roles"`
	ReviewEvidence      []traceMarkdownReviewEvidence `json:"review_evidence,omitempty"`
	AdjudicationRef     string                        `json:"adjudication_ref,omitempty"`
	AdjudicationReason  string                        `json:"adjudication_reason,omitempty"`
}

type traceMarkdownReviewEvidence struct {
	ReviewRef      string `json:"review_ref"`
	Classification string `json:"classification"`
	Reason         string `json:"reason"`
}

type traceMarkdownPrimaryReviewFixture struct {
	SourceRef        string            `json:"source_ref"`
	Classification   string            `json:"classification"`
	Reason           string            `json:"reason"`
	CapabilityIDs    []string          `json:"capability_ids,omitempty"`
	ActionableBlocks []json.RawMessage `json:"actionable_blocks,omitempty"`
	DelegatedRefs    []string          `json:"delegated_refs,omitempty"`
	ScannerV2        string            `json:"scanner_v2,omitempty"`
}

type traceMarkdownCounterreviewFixture struct {
	ReviewKind                string            `json:"review_kind"`
	SourceRef                 string            `json:"source_ref"`
	Verdict                   string            `json:"verdict"`
	CorrectedClassification   string            `json:"corrected_classification,omitempty"`
	CorrectedCapabilityIDs    []string          `json:"corrected_capability_ids,omitempty"`
	CorrectedActionableBlocks []json.RawMessage `json:"corrected_actionable_blocks,omitempty"`
	Reason                    string            `json:"reason"`
}

type traceMarkdownReviewEvidenceKey struct {
	ReviewRef string
	SourceRef string
}

type traceMarkdownReviewAdjudication struct {
	Classification string
	Ref            string
	Reason         string
}

type traceMarkdownReviewFixtureSet struct {
	Evidence      map[traceMarkdownReviewEvidenceKey]traceMarkdownReviewEvidence
	Adjudications map[string]traceMarkdownReviewAdjudication
}

func TestTraceabilityRebuildLegacyMarkdownSourceRoles(t *testing.T) {
	reviewFixtures := traceReadMarkdownReviewFixtures(t)
	paths := traceMarkdownFilesystemScope(t)
	legacyPaths, rebuildAuthorityPaths := traceMarkdownPartitionFilesystemScope(paths)
	entries := traceReadMarkdownSourceRoles(t)
	if len(entries) != len(legacyPaths) {
		t.Fatalf("legacy markdown census rows=%d, legacy filesystem files=%d, rebuild authority files=%d", len(entries), len(legacyPaths), len(rebuildAuthorityPaths))
	}

	dispositions := traceReadDispositionJSONL(t, "product/traceability/source_dispositions.jsonl")
	dispositionBySource := make(map[string]traceSourceDisposition, len(dispositions))
	for _, disposition := range dispositions {
		if _, duplicate := dispositionBySource[disposition.SourceRef]; duplicate {
			t.Fatalf("duplicate source disposition %q", disposition.SourceRef)
		}
		dispositionBySource[disposition.SourceRef] = disposition
	}

	allowedClassifications := map[string]struct{}{
		"bug_evidence": {}, "decision": {}, "historical_decision_evidence": {},
		"manual_mixed": {}, "non_task_template": {}, "rebuild_authority": {},
		"reference": {}, "skill_contract": {}, "supporting_contract": {},
		"supporting_test": {}, "task_delegated": {}, "task_source": {},
	}
	consumedReviews := make(map[traceMarkdownReviewEvidenceKey]struct{}, len(reviewFixtures.Evidence))
	consumedAdjudications := make(map[string]struct{}, len(reviewFixtures.Adjudications))
	seen := make(map[string]struct{}, len(entries))
	previous := ""
	classificationCounts := make(map[string]int)
	roleCounts := make(map[string]int)
	var missingTaskSignals []string
	var missingCanonicalTaskSources []string

	for index, entry := range entries {
		if entry.SchemaVersion != 1 || filepath.Clean(entry.SourceRef) != entry.SourceRef || entry.SourceRef == "" {
			t.Fatalf("invalid markdown census identity: %#v", entry)
		}
		if index > 0 && entry.SourceRef <= previous {
			t.Fatalf("markdown census is not strictly sorted at %q after %q", entry.SourceRef, previous)
		}
		previous = entry.SourceRef
		if entry.SourceRef != legacyPaths[index] {
			t.Fatalf("legacy markdown census/filesystem mismatch at %d: row=%q filesystem=%q", index, entry.SourceRef, legacyPaths[index])
		}
		if _, duplicate := seen[entry.SourceRef]; duplicate {
			t.Fatalf("duplicate markdown census source %q", entry.SourceRef)
		}
		seen[entry.SourceRef] = struct{}{}

		content, err := os.ReadFile(entry.SourceRef)
		if err != nil {
			t.Fatal(err)
		}
		if got := traceMarkdownSHA256(content); got != entry.SourceSHA256 {
			t.Fatalf("markdown source digest drift %s: got %s want %s", entry.SourceRef, got, entry.SourceSHA256)
		}
		wantOrigin := traceMarkdownOrigin(entry.SourceRef)
		if entry.Origin != wantOrigin {
			t.Fatalf("markdown source %q origin=%q, want %q", entry.SourceRef, entry.Origin, wantOrigin)
		}
		if _, ok := allowedClassifications[entry.Classification]; !ok {
			t.Fatalf("markdown source %q has unknown classification %q", entry.SourceRef, entry.Classification)
		}

		disposition, hasDisposition := dispositionBySource[entry.SourceRef]
		traceValidateMarkdownReviews(t, entry, reviewFixtures, consumedReviews, consumedAdjudications)
		wantBasis := traceMarkdownClassificationBasis(entry, hasDisposition, content)
		if entry.ClassificationBasis != wantBasis {
			t.Fatalf("markdown source %q basis=%q, want %q", entry.SourceRef, entry.ClassificationBasis, wantBasis)
		}
		if len(entry.ReviewEvidence) == 0 {
			wantClassification := traceMarkdownMechanicalClassification(entry.SourceRef, disposition, hasDisposition, content)
			if entry.Classification != wantClassification {
				t.Fatalf("markdown source %q classification=%q, want mechanical %q", entry.SourceRef, entry.Classification, wantClassification)
			}
		}

		wantRoles := traceMarkdownExpectedRoles(entry, disposition, hasDisposition, content)
		if !reflect.DeepEqual(entry.Roles, wantRoles) {
			t.Fatalf("markdown source %q roles=%v, want %v", entry.SourceRef, entry.Roles, wantRoles)
		}
		dispositionHasTaskRole := hasDisposition && (disposition.Kind == "task" || traceStringSliceContains(disposition.AdditionalRoles, "task"))
		if traceStringSliceContains(entry.Roles, "task_source") && !dispositionHasTaskRole {
			missingCanonicalTaskSources = append(missingCanonicalTaskSources, entry.SourceRef)
		}
		if bytes.Contains(content, []byte("BUG-ORQ")) && !traceStringSliceContains(entry.Roles, "bug_evidence") {
			t.Fatalf("markdown source %q contains BUG-ORQ without bug_evidence role", entry.SourceRef)
		}

		hasTaskRole := traceStringSliceContains(entry.Roles, "task_source") || traceStringSliceContains(entry.Roles, "task_delegated")
		hasManualReview := len(entry.ReviewEvidence) != 0
		hasExplicitDisposition := hasDisposition
		if traceMarkdownTasklikeFilename(entry.SourceRef) && !hasTaskRole && !hasManualReview && !hasExplicitDisposition {
			missingTaskSignals = append(missingTaskSignals, entry.SourceRef+":task-like-filename")
		}
		source := v2TaskSource{SourceRef: entry.SourceRef, SourceSHA256: entry.SourceSHA256}
		hasTaskCandidate := false
		for range v2ScanTaskCandidates(source, content) {
			hasTaskCandidate = true
		}
		if hasTaskCandidate && !hasTaskRole && !hasManualReview && !hasExplicitDisposition {
			missingTaskSignals = append(missingTaskSignals, entry.SourceRef+":task-candidate")
		}

		classificationCounts[entry.Classification]++
		for _, role := range entry.Roles {
			roleCounts[role]++
		}
	}

	traceRequireAllMarkdownReviewsConsumed(t, reviewFixtures, consumedReviews, consumedAdjudications)
	for sourceRef := range dispositionBySource {
		if _, ok := seen[sourceRef]; !ok {
			t.Fatalf("source disposition %q is outside or absent from markdown census", sourceRef)
		}
	}
	if len(missingTaskSignals) != 0 {
		t.Fatalf("markdown task signals lack task role or explicit review (%d): %v", len(missingTaskSignals), missingTaskSignals)
	}
	if len(missingCanonicalTaskSources) != 0 {
		t.Fatalf("reviewed markdown task sources lack canonical source disposition (%d): %v", len(missingCanonicalTaskSources), missingCanonicalTaskSources)
	}

	t.Logf("markdown_source_roles: legacy_files=%d rebuild_authority_live=%d filesystem_total=%d ledger_sha256=%s classifications=%v roles=%v", len(entries), len(rebuildAuthorityPaths), len(paths), traceFileSHA256(t, traceMarkdownSourceRolesPath), classificationCounts, roleCounts)
}

func traceReadMarkdownReviewFixtures(t *testing.T) traceMarkdownReviewFixtureSet {
	t.Helper()
	fixtures := traceMarkdownReviewFixtureSet{
		Evidence:      make(map[traceMarkdownReviewEvidenceKey]traceMarkdownReviewEvidence),
		Adjudications: make(map[string]traceMarkdownReviewAdjudication),
	}
	broadReviews := make(map[string]traceMarkdownReviewEvidence)
	counterReviewRef := ""
	counterReviewPath := ""
	counterReviewFixtureCount := 0
	counterReviewEvidenceCount := 0

	for reviewRef, batch := range traceMarkdownReviewBatches {
		parts := strings.Split(reviewRef, "@")
		if len(parts) != 2 || traceFileSHA256(t, batch.Path) != parts[1] {
			t.Fatalf("markdown review fixture %q does not bind review ref %q", batch.Path, reviewRef)
		}
		if strings.HasPrefix(reviewRef, "v03_source_counterreview@") {
			if counterReviewRef != "" {
				t.Fatalf("multiple markdown source counterreview batches: %q and %q", counterReviewRef, reviewRef)
			}
			counterReviewRef = reviewRef
			counterReviewPath = batch.Path
			counterReviewFixtureCount = batch.FixtureCount
			counterReviewEvidenceCount = batch.EvidenceCount
			continue
		}
		file, err := os.Open(batch.Path)
		if err != nil {
			t.Fatal(err)
		}
		rows := 0
		evidenceRows := 0
		traceScanStrictJSONL(t, file, batch.Path, func(decoder *json.Decoder, lineNumber int) {
			var fixture traceMarkdownPrimaryReviewFixture
			if err := decoder.Decode(&fixture); err != nil {
				t.Fatalf("decode %s:%d: %v", batch.Path, lineNumber, err)
			}
			rows++
			if fixture.SourceRef == "" || fixture.Classification == "" || len(strings.TrimSpace(fixture.Reason)) < 20 {
				t.Fatalf("invalid markdown review fixture %s:%d: %#v", batch.Path, lineNumber, fixture)
			}
			if fixture.SourceRef == traceMarkdownDeployReviewOutsideCensus {
				if batch.Path != "product/traceability/fixtures/markdown_source_review_s2.jsonl" {
					t.Fatalf("unexpected out-of-census markdown review %s:%d: %q", batch.Path, lineNumber, fixture.SourceRef)
				}
				return
			}
			key := traceMarkdownReviewEvidenceKey{ReviewRef: reviewRef, SourceRef: fixture.SourceRef}
			if _, duplicate := fixtures.Evidence[key]; duplicate {
				t.Fatalf("duplicate markdown review fixture evidence: %#v", key)
			}
			evidence := traceMarkdownReviewEvidence{
				ReviewRef: reviewRef, Classification: fixture.Classification, Reason: fixture.Reason,
			}
			fixtures.Evidence[key] = evidence
			evidenceRows++
			if strings.HasPrefix(reviewRef, "v03_broad_source_review_") {
				if _, duplicate := broadReviews[fixture.SourceRef]; duplicate {
					t.Fatalf("duplicate primary broad review for markdown source %q", fixture.SourceRef)
				}
				broadReviews[fixture.SourceRef] = evidence
			}
		})
		_ = file.Close()
		if rows != batch.FixtureCount {
			t.Fatalf("markdown review fixture %q rows=%d, want %d", batch.Path, rows, batch.FixtureCount)
		}
		if evidenceRows != batch.EvidenceCount {
			t.Fatalf("markdown review fixture %q source evidence rows=%d, want %d", batch.Path, evidenceRows, batch.EvidenceCount)
		}
	}
	if counterReviewRef == "" {
		t.Fatal("markdown source counterreview batch is absent")
	}
	traceReadMarkdownSourceCounterreviews(t, counterReviewRef, counterReviewPath, counterReviewFixtureCount, counterReviewEvidenceCount, broadReviews, &fixtures)
	for sourceRef, adjudication := range traceMarkdownFixedReviewAdjudications {
		if _, duplicate := fixtures.Adjudications[sourceRef]; duplicate {
			t.Fatalf("duplicate markdown review adjudication for %q", sourceRef)
		}
		fixtures.Adjudications[sourceRef] = adjudication
	}
	if len(fixtures.Adjudications) != 5 {
		t.Fatalf("markdown review adjudications=%d, want 5", len(fixtures.Adjudications))
	}
	return fixtures
}

func traceReadMarkdownSourceCounterreviews(
	t *testing.T,
	reviewRef string,
	path string,
	fixtureCount int,
	evidenceCount int,
	broadReviews map[string]traceMarkdownReviewEvidence,
	fixtures *traceMarkdownReviewFixtureSet,
) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows := 0
	sourceReviewRows := 0
	blockSupplementRows := 0
	traceScanStrictJSONL(t, file, path, func(decoder *json.Decoder, lineNumber int) {
		var fixture traceMarkdownCounterreviewFixture
		if err := decoder.Decode(&fixture); err != nil {
			t.Fatalf("decode %s:%d: %v", path, lineNumber, err)
		}
		rows++
		if fixture.SourceRef == "" || len(strings.TrimSpace(fixture.Reason)) < 20 {
			t.Fatalf("invalid markdown counterreview fixture %s:%d: %#v", path, lineNumber, fixture)
		}
		if fixture.ReviewKind == "block_supplement" {
			blockSupplementRows++
			return
		}
		if fixture.ReviewKind != "source_review" && fixture.ReviewKind != "sdk_conflict" {
			t.Fatalf("unknown markdown counterreview kind %s:%d: %q", path, lineNumber, fixture.ReviewKind)
		}
		primary, ok := broadReviews[fixture.SourceRef]
		if !ok {
			t.Fatalf("markdown source counterreview %s:%d lacks primary broad review for %q", path, lineNumber, fixture.SourceRef)
		}
		classification := ""
		switch fixture.Verdict {
		case "confirm":
			if fixture.CorrectedClassification != "" {
				t.Fatalf("confirmed markdown counterreview %s:%d carries corrected classification %q", path, lineNumber, fixture.CorrectedClassification)
			}
			classification = primary.Classification
		case "correct":
			if fixture.CorrectedClassification == "" {
				t.Fatalf("corrected markdown counterreview %s:%d lacks corrected classification", path, lineNumber)
			}
			classification = fixture.CorrectedClassification
		default:
			t.Fatalf("unknown markdown counterreview verdict %s:%d: %q", path, lineNumber, fixture.Verdict)
		}
		key := traceMarkdownReviewEvidenceKey{ReviewRef: reviewRef, SourceRef: fixture.SourceRef}
		if _, duplicate := fixtures.Evidence[key]; duplicate {
			t.Fatalf("duplicate markdown counterreview evidence: %#v", key)
		}
		fixtures.Evidence[key] = traceMarkdownReviewEvidence{
			ReviewRef: reviewRef, Classification: classification, Reason: fixture.Reason,
		}
		sourceReviewRows++
		if classification != primary.Classification {
			fixtures.Adjudications[fixture.SourceRef] = traceMarkdownReviewAdjudication{
				Classification: classification,
				Ref:            reviewRef + "#" + fixture.SourceRef,
				Reason:         fixture.Reason,
			}
		}
	})
	if rows != fixtureCount {
		t.Fatalf("markdown counterreview fixture %q rows=%d, want %d", path, rows, fixtureCount)
	}
	if sourceReviewRows != evidenceCount || blockSupplementRows != fixtureCount-evidenceCount {
		t.Fatalf("markdown counterreview fixture %q source_reviews=%d block_supplements=%d, want %d/%d", path, sourceReviewRows, blockSupplementRows, evidenceCount, fixtureCount-evidenceCount)
	}
}

func traceMarkdownPartitionFilesystemScope(paths []string) (legacy, rebuildAuthority []string) {
	for _, path := range paths {
		if strings.HasPrefix(path, "docs/reconstruccion/") {
			rebuildAuthority = append(rebuildAuthority, path)
			continue
		}
		legacy = append(legacy, path)
	}
	return legacy, rebuildAuthority
}

func traceMarkdownFilesystemScope(t *testing.T) []string {
	t.Helper()
	set := make(map[string]struct{})
	err := filepath.WalkDir("docs", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".md") {
			set[filepath.ToSlash(path)] = struct{}{}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	modules, err := filepath.Glob("modulos/orquesta-*")
	if err != nil {
		t.Fatal(err)
	}
	for _, module := range modules {
		matches, globErr := filepath.Glob(filepath.Join(module, "docs", "*.md"))
		if globErr != nil {
			t.Fatal(globErr)
		}
		for _, path := range matches {
			set[filepath.ToSlash(path)] = struct{}{}
		}
	}
	skills, err := filepath.Glob("skills/*/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range skills {
		set[filepath.ToSlash(path)] = struct{}{}
	}
	paths := make([]string, 0, len(set))
	for path := range set {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func traceReadMarkdownSourceRoles(t *testing.T) []traceMarkdownSourceRole {
	t.Helper()
	file, err := os.Open(traceMarkdownSourceRolesPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	entries := make([]traceMarkdownSourceRole, 0)
	traceScanStrictJSONL(t, file, traceMarkdownSourceRolesPath, func(decoder *json.Decoder, lineNumber int) {
		var entry traceMarkdownSourceRole
		if err := decoder.Decode(&entry); err != nil {
			t.Fatalf("decode %s:%d: %v", traceMarkdownSourceRolesPath, lineNumber, err)
		}
		entries = append(entries, entry)
	})
	return entries
}

func traceValidateMarkdownReviews(
	t *testing.T,
	entry traceMarkdownSourceRole,
	fixtures traceMarkdownReviewFixtureSet,
	consumedReviews map[traceMarkdownReviewEvidenceKey]struct{},
	consumedAdjudications map[string]struct{},
) {
	t.Helper()
	previous := ""
	classifications := make(map[string]struct{})
	for index, review := range entry.ReviewEvidence {
		key := traceMarkdownReviewEvidenceKey{ReviewRef: review.ReviewRef, SourceRef: entry.SourceRef}
		expected, ok := fixtures.Evidence[key]
		if !ok {
			t.Fatalf("markdown source %q has review evidence without an exact fixture row: %#v", entry.SourceRef, review)
		}
		if review != expected {
			t.Fatalf("markdown source %q review evidence=%#v, want exact fixture evidence %#v", entry.SourceRef, review, expected)
		}
		if _, duplicate := consumedReviews[key]; duplicate {
			t.Fatalf("markdown review fixture evidence consumed more than once: %#v", key)
		}
		if index > 0 && review.ReviewRef <= previous {
			t.Fatalf("markdown source %q reviews are not strictly sorted", entry.SourceRef)
		}
		previous = review.ReviewRef
		consumedReviews[key] = struct{}{}
		classifications[review.Classification] = struct{}{}
	}
	if len(classifications) <= 1 {
		if entry.AdjudicationRef != "" || entry.AdjudicationReason != "" {
			t.Fatalf("markdown source %q has unnecessary adjudication", entry.SourceRef)
		}
		if len(classifications) == 1 {
			for classification := range classifications {
				if entry.Classification != classification {
					t.Fatalf("markdown source %q classification=%q, reviewed=%q", entry.SourceRef, entry.Classification, classification)
				}
			}
		}
		return
	}
	want, known := fixtures.Adjudications[entry.SourceRef]
	if !known || entry.Classification != want.Classification || entry.AdjudicationRef != want.Ref || entry.AdjudicationReason != want.Reason {
		t.Fatalf("markdown source %q has unresolved review conflict: classifications=%v entry=%#v", entry.SourceRef, classifications, entry)
	}
	if _, duplicate := consumedAdjudications[entry.SourceRef]; duplicate {
		t.Fatalf("markdown review adjudication consumed more than once for %q", entry.SourceRef)
	}
	consumedAdjudications[entry.SourceRef] = struct{}{}
}

func traceRequireAllMarkdownReviewsConsumed(
	t *testing.T,
	fixtures traceMarkdownReviewFixtureSet,
	consumedReviews map[traceMarkdownReviewEvidenceKey]struct{},
	consumedAdjudications map[string]struct{},
) {
	t.Helper()
	missingReviews := make([]string, 0)
	for key := range fixtures.Evidence {
		if _, consumed := consumedReviews[key]; !consumed {
			missingReviews = append(missingReviews, key.ReviewRef+"#"+key.SourceRef)
		}
	}
	sort.Strings(missingReviews)
	if len(missingReviews) != 0 {
		t.Fatalf("markdown review fixture rows not consumed (%d): %v", len(missingReviews), missingReviews)
	}
	missingAdjudications := make([]string, 0)
	for sourceRef := range fixtures.Adjudications {
		if _, consumed := consumedAdjudications[sourceRef]; !consumed {
			missingAdjudications = append(missingAdjudications, sourceRef)
		}
	}
	sort.Strings(missingAdjudications)
	if len(missingAdjudications) != 0 {
		t.Fatalf("markdown review adjudications not consumed (%d): %v", len(missingAdjudications), missingAdjudications)
	}
}

func traceMarkdownClassificationBasis(entry traceMarkdownSourceRole, hasDisposition bool, content []byte) string {
	if len(entry.ReviewEvidence) != 0 {
		return "manual_review"
	}
	if entry.Origin == "rebuild_authority" {
		return "rebuild_authority"
	}
	if hasDisposition {
		return "source_disposition"
	}
	if bytes.Contains(content, []byte("BUG-ORQ")) {
		return "literal_bug_signal"
	}
	return "mechanical_path_policy"
}

func traceMarkdownMechanicalClassification(path string, disposition traceSourceDisposition, hasDisposition bool, content []byte) string {
	if strings.HasPrefix(path, "docs/reconstruccion/") {
		return "rebuild_authority"
	}
	if hasDisposition {
		switch disposition.Kind {
		case "task":
			return "task_source"
		case "bug":
			return "bug_evidence"
		case "skill":
			return "skill_contract"
		}
	}
	if bytes.Contains(content, []byte("BUG-ORQ")) {
		return "bug_evidence"
	}
	if strings.HasPrefix(path, "skills/") {
		return "skill_contract"
	}
	base := strings.ToLower(filepath.Base(path))
	switch {
	case strings.HasPrefix(base, "contratos"):
		return "supporting_contract"
	case strings.HasPrefix(base, "pruebas"):
		return "supporting_test"
	case strings.HasPrefix(base, "decisiones"):
		return "decision"
	default:
		return "reference"
	}
}

func traceMarkdownExpectedRoles(entry traceMarkdownSourceRole, disposition traceSourceDisposition, hasDisposition bool, content []byte) []string {
	set := map[string]struct{}{"documentation": {}}
	switch entry.Classification {
	case "bug_evidence":
		set["bug_evidence"] = struct{}{}
	case "decision", "historical_decision_evidence":
		set["decision"] = struct{}{}
	case "manual_mixed":
		set["reference"] = struct{}{}
		set["task_source"] = struct{}{}
	case "non_task_template":
		set["template"] = struct{}{}
	case "rebuild_authority":
		set["rebuild_authority"] = struct{}{}
	case "reference":
		set["reference"] = struct{}{}
	case "skill_contract":
		set["skill_contract"] = struct{}{}
	case "supporting_contract":
		set["contract"] = struct{}{}
	case "supporting_test":
		set["test"] = struct{}{}
	case "task_delegated":
		set["task_delegated"] = struct{}{}
	case "task_source":
		set["task_source"] = struct{}{}
	}
	if entry.Origin == "rebuild_authority" {
		set["rebuild_authority"] = struct{}{}
	}
	if strings.HasPrefix(entry.SourceRef, "skills/") {
		set["skill_contract"] = struct{}{}
	}
	if bytes.Contains(content, []byte("BUG-ORQ")) {
		set["bug_evidence"] = struct{}{}
	}
	if hasDisposition {
		for _, role := range append([]string{disposition.Kind}, disposition.AdditionalRoles...) {
			switch role {
			case "task":
				set["task_source"] = struct{}{}
			case "bug":
				set["bug_evidence"] = struct{}{}
			case "skill":
				set["skill_contract"] = struct{}{}
			}
		}
	}
	roles := make([]string, 0, len(set))
	for role := range set {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

func traceMarkdownOrigin(path string) string {
	switch {
	case strings.HasPrefix(path, "docs/reconstruccion/"):
		return "rebuild_authority"
	case strings.HasPrefix(path, "docs/"):
		return "legacy_root_document"
	case strings.HasPrefix(path, "modulos/"):
		return "legacy_module_document"
	case strings.HasPrefix(path, "skills/"):
		return "legacy_skill_document"
	default:
		return ""
	}
}

func traceMarkdownTasklikeFilename(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.HasPrefix(base, "plan_") || strings.HasPrefix(base, "roadmap_") ||
		strings.HasPrefix(base, "borrador_rebase_") || strings.HasPrefix(base, "promocion_") ||
		strings.HasPrefix(base, "pendientes_")
}

func traceMarkdownSHA256(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
