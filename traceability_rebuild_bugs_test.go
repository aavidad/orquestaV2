package orquesta_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type traceRebuildBug struct {
	SchemaVersion int      `json:"schema_version"`
	BugID         string   `json:"bug_id"`
	CapabilityIDs []string `json:"capability_ids"`
	Cause         string   `json:"cause"`
	Invariant     string   `json:"invariant"`
	LessonTestRef string   `json:"lesson_test_ref"`
	Status        string   `json:"status"`
	EvidenceRef   string   `json:"evidence_ref"`
}

func TestTraceabilityRebuildBugLessons(t *testing.T) {
	bugs := traceReadRebuildBugs(t, "product/traceability/rebuild_bugs.jsonl")
	accepted := traceAcceptedCapabilities(t)
	requiredClosedLessons := map[string]string{
		"BUG-REBUILD-20260714-017": "traceability_rebuild_bugs_test.go#TestTraceabilityRebuildMarkdownReviewEvidenceIsBoundRowByRow",
		"BUG-REBUILD-20260714-018": "product_roadmap_test.go#TestProductRoadmapAccreditationDoesNotExceedEvidence",
		"BUG-REBUILD-20260714-019": "product_roadmap_test.go#TestProductRoadmapPlannedContractsAreNonRunnable",
		"BUG-REBUILD-20260714-020": "acceptance/v01_source_integration_test.go#TestAcceptanceV01SourceIntegrationReceipt",
		"BUG-REBUILD-20260714-021": "product_roadmap_test.go#TestProductRoadmapExecutableCommandsRunDeclaredTestPackage",
		"BUG-REBUILD-20260714-023": "internal/application/amend_test.go#TestV04ProviderSpecHashMismatchCreatesNoEvidenceOrClosure",
		"BUG-REBUILD-20260714-024": "internal/application/amend_test.go#TestV04SubmitRejectsCreatedSnapshotSubstitution",
		"BUG-REBUILD-20260714-025": "internal/ports/agent_contract_test.go#TestAgentContractClassifiesReceiptSpecHashBeforeCausalMismatch",
		"BUG-REBUILD-20260714-026": "internal/adapters/state/sqlite/app_specs_test.go#TestRepositoryPersistsAppSpecConfirmedByReviewerDifferentFromIntentActor",
		"BUG-REBUILD-20260714-027": "internal/adapters/state/sqlite/app_specs_test.go#TestRepositoryDomainInvalidV2GoalRollsBackAppSpecMigration",
		"BUG-REBUILD-20260714-028": "internal/application/amend_test.go#TestV04InitialAppSpecReasonMatchesAcceptanceContract",
		"BUG-REBUILD-20260714-031": "internal/bootstrap/mcp_e2e_test.go#TestRealMCPAPIClosesDurableGoalThroughSQLiteAndArtifactStore",
	}
	baselineIDs := []string{
		"BUG-REBUILD-20260714-001",
		"BUG-REBUILD-20260714-002",
		"BUG-REBUILD-20260714-003",
		"BUG-REBUILD-20260714-004",
		"BUG-REBUILD-20260714-005",
		"BUG-REBUILD-20260714-006",
	}
	if len(bugs) < len(baselineIDs) {
		t.Fatalf("rebuild bug count=%d, want at least baseline %d", len(bugs), len(baselineIDs))
	}
	statuses := map[string]int{}
	lessonRefs := make(map[string]struct{})
	previousBugID := ""
	for index, bug := range bugs {
		if index < len(baselineIDs) && bug.BugID != baselineIDs[index] {
			t.Fatalf("rebuild bug baseline changed at %d: got %q want %q", index, bug.BugID, baselineIDs[index])
		}
		if bug.SchemaVersion != 1 || bug.BugID <= previousBugID ||
			strings.TrimSpace(bug.Cause) == "" || strings.TrimSpace(bug.Invariant) == "" ||
			len(bug.CapabilityIDs) == 0 || !sort.StringsAreSorted(bug.CapabilityIDs) {
			t.Fatalf("incomplete or unordered rebuild bug: %#v", bug)
		}
		previousBugID = bug.BugID
		for capabilityIndex, capabilityID := range bug.CapabilityIDs {
			if capabilityIndex > 0 && capabilityID == bug.CapabilityIDs[capabilityIndex-1] {
				t.Errorf("bug %q repeats capability %q", bug.BugID, capabilityID)
			}
			if _, ok := accepted[capabilityID]; !ok {
				t.Errorf("bug %q maps non-accepted capability %q", bug.BugID, capabilityID)
			}
		}
		if _, duplicate := lessonRefs[bug.LessonTestRef]; duplicate {
			t.Errorf("duplicate rebuild lesson ref %q", bug.LessonTestRef)
		}
		lessonRefs[bug.LessonTestRef] = struct{}{}
		switch bug.Status {
		case "closed":
			if strings.HasPrefix(bug.LessonTestRef, "planned:") || !strings.HasPrefix(bug.EvidenceRef, "test:go test ") {
				t.Errorf("closed bug %q lacks executed concrete test evidence", bug.BugID)
			}
			traceRequireGoTestRef(t, bug.LessonTestRef, true)
		case "open":
			if strings.HasPrefix(bug.LessonTestRef, "planned:") {
				if bug.EvidenceRef != "pending:test_not_present" {
					t.Errorf("open bug %q with planned test needs pending evidence", bug.BugID)
				}
				traceRequireGoTestRef(t, strings.TrimPrefix(bug.LessonTestRef, "planned:"), false)
			} else {
				if !strings.HasPrefix(bug.EvidenceRef, "failed:") && !strings.HasPrefix(bug.EvidenceRef, "blocked:") {
					t.Errorf("open bug %q with concrete test needs failed/blocked evidence", bug.BugID)
				}
				traceRequireGoTestRef(t, bug.LessonTestRef, true)
			}
		default:
			t.Errorf("bug %q invalid status %q", bug.BugID, bug.Status)
		}
		if index < len(baselineIDs) && bug.Status != "closed" {
			t.Errorf("baseline bug %q regressed to status %q", bug.BugID, bug.Status)
		}
		if wantLessonRef, required := requiredClosedLessons[bug.BugID]; required {
			if bug.Status != "closed" || bug.LessonTestRef != wantLessonRef {
				t.Errorf("required structural bug %q status/lesson=%q/%q, want closed/%q", bug.BugID, bug.Status, bug.LessonTestRef, wantLessonRef)
			}
			delete(requiredClosedLessons, bug.BugID)
		}
		statuses[bug.Status]++
	}
	if len(requiredClosedLessons) != 0 {
		missing := make([]string, 0, len(requiredClosedLessons))
		for bugID := range requiredClosedLessons {
			missing = append(missing, bugID)
		}
		sort.Strings(missing)
		t.Fatalf("missing required closed structural bugs: %v", missing)
	}
	t.Logf("rebuild_bugs: total=%d baseline_closed=%d closed=%d open=%d", len(bugs), len(baselineIDs), statuses["closed"], statuses["open"])
}

func TestTraceabilityRebuildMarkdownReviewEvidenceIsBoundRowByRow(t *testing.T) {
	fixtures := traceReadMarkdownReviewFixtures(t)
	consumedReviews := make(map[traceMarkdownReviewEvidenceKey]struct{}, len(fixtures.Evidence))
	consumedAdjudications := make(map[string]struct{}, len(fixtures.Adjudications))
	for _, entry := range traceReadMarkdownSourceRoles(t) {
		traceValidateMarkdownReviews(t, entry, fixtures, consumedReviews, consumedAdjudications)
	}
	traceRequireAllMarkdownReviewsConsumed(t, fixtures, consumedReviews, consumedAdjudications)
}

func TestTraceabilityRebuildSafeVerificationCommandsExcludeLegacyWildcard(t *testing.T) {
	type commandEntry struct {
		ID      string `json:"id"`
		Command string `json:"command"`
	}
	var roadmap struct {
		AcceptanceContracts []commandEntry `json:"acceptance_contracts"`
	}
	content, err := os.ReadFile("product/roadmap.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &roadmap); err != nil {
		t.Fatal(err)
	}
	if len(roadmap.AcceptanceContracts) == 0 {
		t.Fatal("roadmap has no canonical acceptance commands")
	}
	for _, contract := range roadmap.AcceptanceContracts {
		traceRejectLegacyWildcard(t, "product/roadmap.json#"+contract.ID, contract.Command)
	}

	fixtures, err := filepath.Glob("acceptance/fixtures/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("acceptance fixtures are missing")
	}
	for _, fixture := range fixtures {
		content, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatal(err)
		}
		var envelope struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(content, &envelope); err != nil {
			t.Fatalf("decode %s: %v", fixture, err)
		}
		if strings.TrimSpace(envelope.Command) != "" {
			traceRejectLegacyWildcard(t, fixture, envelope.Command)
		}
	}
}

func traceRejectLegacyWildcard(t *testing.T, source, command string) {
	t.Helper()
	for _, field := range strings.Fields(command) {
		if field == "./..." {
			t.Errorf("%s crosses the frozen legacy boundary with bare ./...: %q", source, command)
		}
	}
}

func traceReadRebuildBugs(t *testing.T, path string) []traceRebuildBug {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var entries []traceRebuildBug
	traceScanStrictJSONL(t, file, path, func(decoder *json.Decoder, lineNumber int) {
		var entry traceRebuildBug
		if err := decoder.Decode(&entry); err != nil {
			t.Fatalf("decode %s:%d: %v", path, lineNumber, err)
		}
		entries = append(entries, entry)
	})
	return entries
}
