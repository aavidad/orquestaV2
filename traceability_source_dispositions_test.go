package orquesta_test

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type tracePendingSourceLedger struct {
	DocumentKind  string               `json:"document_kind"`
	SchemaVersion int                  `json:"schema_version"`
	Sources       []tracePendingSource `json:"sources"`
}

type tracePendingSource struct {
	ID                string   `json:"id"`
	Kind              string   `json:"kind"`
	Status            string   `json:"status"`
	ExactPaths        []string `json:"exact_paths"`
	PathPatterns      []string `json:"path_patterns"`
	ExpectedFileCount int      `json:"expected_file_count"`
	ManifestSHA256    string   `json:"manifest_sha256"`
	NextGate          string   `json:"next_gate"`
}

type traceDispositionReasonLedger struct {
	DocumentKind       string   `json:"document_kind"`
	SchemaVersion      int      `json:"schema_version"`
	DecisionVocabulary []string `json:"decision_vocabulary"`
	EvidenceProtocol   struct {
		Algorithm        string `json:"algorithm"`
		Subject          string `json:"subject"`
		VerificationTest string `json:"verification_test"`
	} `json:"evidence_protocol"`
	Reasons map[string]traceDispositionReason `json:"reasons"`
}

type traceDispositionReason struct {
	Decision      string `json:"decision"`
	EvidenceState string `json:"evidence_state"`
	Reason        string `json:"reason"`
}

type traceSourceDisposition struct {
	SchemaVersion      int      `json:"schema_version"`
	Kind               string   `json:"kind"`
	AdditionalRoles    []string `json:"additional_roles,omitempty"`
	SourceRef          string   `json:"source_ref"`
	SourceSHA256       string   `json:"source_sha256"`
	Decision           string   `json:"decision"`
	CapabilityIDs      []string `json:"capability_ids"`
	ReasonCode         string   `json:"reason_code"`
	EvidenceState      string   `json:"evidence_state"`
	LessonID           string   `json:"lesson_id,omitempty"`
	InvariantTestRef   string   `json:"invariant_test_ref,omitempty"`
	LessonState        string   `json:"lesson_state,omitempty"`
	ScopeRef           string   `json:"scope_ref,omitempty"`
	TrustRef           string   `json:"trust_ref,omitempty"`
	ActivationTestRef  string   `json:"activation_test_ref,omitempty"`
	ReviewBasis        string   `json:"review_basis,omitempty"`
	ReviewNote         string   `json:"review_note,omitempty"`
	TransformationNote string   `json:"transformation_note,omitempty"`
	DelegatedLedgerRef string   `json:"delegated_ledger_ref,omitempty"`
}

func TestTraceabilityRebuildPendingSourceInventory(t *testing.T) {
	var ledger tracePendingSourceLedger
	traceDecodeStrict(t, "product/traceability/pending_sources.json", &ledger)
	if ledger.DocumentKind != "pending_source_inventory" || ledger.SchemaVersion != 1 || len(ledger.Sources) != 4 {
		t.Fatalf("invalid pending source ledger header: %#v", ledger)
	}
	wantKinds := map[string]string{
		"historical_task_sources":     "task",
		"historical_bug_sources":      "bug",
		"historical_skill_sources":    "skill",
		"historical_rulepack_sources": "rulepack",
	}
	seen := make(map[string]struct{})
	for _, source := range ledger.Sources {
		wantKind, ok := wantKinds[source.ID]
		if !ok || source.Kind != wantKind || strings.TrimSpace(source.NextGate) == "" {
			t.Fatalf("invalid pending source entry: %#v", source)
		}
		if _, duplicate := seen[source.ID]; duplicate {
			t.Fatalf("duplicate pending source id %q", source.ID)
		}
		seen[source.ID] = struct{}{}
		paths := traceExpandSourcePaths(t, source)
		gotDigest := traceFileManifestDigest(t, paths)
		if len(paths) != source.ExpectedFileCount || gotDigest != source.ManifestSHA256 {
			t.Errorf("pending source %q drift: files=%d digest=%s, want files=%d digest=%s",
				source.ID, len(paths), gotDigest, source.ExpectedFileCount, source.ManifestSHA256)
		}
		if source.Kind == "rulepack" {
			if source.Status != "absent_pending_contract" || len(paths) != 0 {
				t.Errorf("rulepack source must remain explicitly absent until TLS-13 contract exists")
			}
		} else if len(paths) == 0 {
			t.Errorf("source %q must remain non-empty", source.ID)
		} else {
			wantStatus := map[string]string{
				"task":  "entries_semantically_disposed_characterization_pending",
				"bug":   "entries_disposed_lesson_tests_pending",
				"skill": "entries_disposed_governance_pending",
			}[source.Kind]
			if source.Status != wantStatus {
				t.Errorf("source %q status=%q, want %q", source.ID, source.Status, wantStatus)
			}
		}
	}
	t.Log("pending source ranges frozen; task characterization, bug lesson tests and skill governance remain explicit")
}

func TestTraceabilityRebuildSourceDispositions(t *testing.T) {
	var pending tracePendingSourceLedger
	traceDecodeStrict(t, "product/traceability/pending_sources.json", &pending)
	expected := make(map[string]string)
	for _, source := range pending.Sources {
		for _, path := range traceExpandSourcePaths(t, source) {
			if previous, duplicate := expected[path]; duplicate {
				t.Fatalf("source %q appears in both %q and %q inventories", path, previous, source.Kind)
			}
			expected[path] = source.Kind
		}
	}

	var reasons traceDispositionReasonLedger
	traceDecodeStrict(t, "product/traceability/disposition_reasons.json", &reasons)
	wantDecisions := []string{"accepted", "rejected", "conditional", "historical"}
	if reasons.DocumentKind != "disposition_reasons" || reasons.SchemaVersion != 1 ||
		!reflect.DeepEqual(reasons.DecisionVocabulary, wantDecisions) ||
		reasons.EvidenceProtocol.Algorithm != "sha256" ||
		reasons.EvidenceProtocol.Subject != "exact_file_bytes_at_source_ref" ||
		reasons.EvidenceProtocol.VerificationTest != "TestTraceabilityRebuildSourceDispositions" {
		t.Fatalf("invalid disposition reason ledger header: %#v", reasons)
	}
	if len(reasons.Reasons) != 10 {
		t.Fatalf("reason count=%d, want 10", len(reasons.Reasons))
	}
	decisionSet := make(map[string]struct{}, len(wantDecisions))
	for _, decision := range wantDecisions {
		decisionSet[decision] = struct{}{}
	}
	for code, reason := range reasons.Reasons {
		if code == "" || strings.TrimSpace(reason.Reason) == "" || strings.TrimSpace(reason.EvidenceState) == "" {
			t.Fatalf("incomplete disposition reason %q: %#v", code, reason)
		}
		if _, ok := decisionSet[reason.Decision]; !ok {
			t.Fatalf("reason %q has invalid decision %q", code, reason.Decision)
		}
	}

	acceptedCapabilities := traceAcceptedCapabilities(t)
	taskEntryCapabilities := make(map[string]map[string]struct{})
	for _, taskEntry := range v2ReadJSONL[v2TaskEntry](t, v2TaskEntriesPath) {
		if taskEntryCapabilities[taskEntry.SourceRef] == nil {
			taskEntryCapabilities[taskEntry.SourceRef] = make(map[string]struct{})
		}
		taskEntryCapabilities[taskEntry.SourceRef][taskEntry.CapabilityID] = struct{}{}
	}
	entries := traceReadDispositionJSONL(t, "product/traceability/source_dispositions.jsonl")
	seen := make(map[string]struct{}, len(entries))
	lessonIDs := make(map[string]struct{})
	skillRefs := make(map[string]struct{})
	kindCounts := map[string]int{}
	decisionCounts := map[string]int{}
	previousKey := ""
	for index, entry := range entries {
		key := entry.Kind + "\x00" + entry.SourceRef
		if index > 0 && key <= previousKey {
			t.Fatalf("disposition rows not strictly sorted at %q after %q", key, previousKey)
		}
		previousKey = key
		if entry.SchemaVersion != 1 || filepath.Clean(entry.SourceRef) != entry.SourceRef || strings.ContainsAny(entry.SourceRef, "*?[") {
			t.Fatalf("invalid source disposition identity: %#v", entry)
		}
		if !sort.StringsAreSorted(entry.AdditionalRoles) {
			t.Fatalf("source %q additional roles are not sorted: %v", entry.SourceRef, entry.AdditionalRoles)
		}
		for roleIndex, role := range entry.AdditionalRoles {
			if role != "task" && role != "bug" && role != "skill" {
				t.Fatalf("source %q has unknown additional role %q", entry.SourceRef, role)
			}
			if role == entry.Kind || (roleIndex > 0 && role == entry.AdditionalRoles[roleIndex-1]) {
				t.Fatalf("source %q repeats primary/additional role %q", entry.SourceRef, role)
			}
		}
		wantKind, inventoried := expected[entry.SourceRef]
		if !inventoried || wantKind != entry.Kind {
			t.Fatalf("disposition source %q kind=%q not in exact inventory kind=%q", entry.SourceRef, entry.Kind, wantKind)
		}
		if _, duplicate := seen[entry.SourceRef]; duplicate {
			t.Fatalf("duplicate source disposition %q", entry.SourceRef)
		}
		seen[entry.SourceRef] = struct{}{}
		gotHash := traceFileSHA256(t, entry.SourceRef)
		if gotHash != entry.SourceSHA256 {
			t.Errorf("source %q digest=%s, want %s", entry.SourceRef, gotHash, entry.SourceSHA256)
		}
		reason, knownReason := reasons.Reasons[entry.ReasonCode]
		if !knownReason || reason.Decision != entry.Decision || reason.EvidenceState != entry.EvidenceState {
			t.Errorf("source %q has inconsistent reason/decision/evidence: %#v reason=%#v", entry.SourceRef, entry, reason)
		}
		if len(entry.CapabilityIDs) == 0 || !sort.StringsAreSorted(entry.CapabilityIDs) {
			t.Errorf("source %q capability IDs empty or unsorted: %v", entry.SourceRef, entry.CapabilityIDs)
		}
		for capabilityIndex, capabilityID := range entry.CapabilityIDs {
			if capabilityIndex > 0 && capabilityID == entry.CapabilityIDs[capabilityIndex-1] {
				t.Errorf("source %q repeats capability %q", entry.SourceRef, capabilityID)
			}
			_, accepted := acceptedCapabilities[capabilityID]
			_, exactTaskEntryDecision := taskEntryCapabilities[entry.SourceRef][capabilityID]
			if !accepted && !(entry.Kind == "task" && exactTaskEntryDecision) {
				t.Errorf("source %q maps to non-accepted capability %q", entry.SourceRef, capabilityID)
			}
		}
		switch entry.Kind {
		case "bug":
			digestHex := strings.TrimPrefix(entry.SourceSHA256, "sha256:")
			wantLessonID := "LESSON-" + digestHex[:20]
			if entry.LessonID != wantLessonID || entry.InvariantTestRef != "planned:lesson-tests/"+digestHex[:20] ||
				entry.LessonState != "pending_invariant_test" || entry.ScopeRef != "" || entry.TrustRef != "" || entry.ActivationTestRef != "" {
				t.Errorf("bug source %q lacks individual pending lesson refs: %#v", entry.SourceRef, entry)
			}
			if _, duplicate := lessonIDs[entry.LessonID]; duplicate {
				t.Errorf("duplicate lesson id %q", entry.LessonID)
			}
			lessonIDs[entry.LessonID] = struct{}{}
		case "skill":
			slug := strings.TrimSuffix(strings.TrimPrefix(entry.SourceRef, "skills/"), "/SKILL.md")
			digestHex := strings.TrimPrefix(entry.SourceSHA256, "sha256:")
			wantScope := "pending:skill-scope/" + slug
			wantTrust := "pending:skill-trust/" + slug + "@sha256:" + digestHex[:20]
			wantActivation := "planned:skill-activation/" + slug
			if entry.ScopeRef != wantScope || entry.TrustRef != wantTrust || entry.ActivationTestRef != wantActivation ||
				entry.LessonID != "" || entry.InvariantTestRef != "" || entry.LessonState != "" {
				t.Errorf("skill source %q lacks individual scope/trust/activation refs: %#v", entry.SourceRef, entry)
			}
			for _, ref := range []string{entry.ScopeRef, entry.TrustRef, entry.ActivationTestRef} {
				if _, duplicate := skillRefs[ref]; duplicate {
					t.Errorf("duplicate skill governance ref %q", ref)
				}
				skillRefs[ref] = struct{}{}
			}
		case "task":
			if entry.LessonID != "" || entry.InvariantTestRef != "" || entry.LessonState != "" ||
				entry.ScopeRef != "" || entry.TrustRef != "" || entry.ActivationTestRef != "" {
				t.Errorf("task source %q carries bug/skill-only refs", entry.SourceRef)
			}
		}
		kindCounts[entry.Kind]++
		decisionCounts[entry.Decision]++
	}
	missing := make([]string, 0)
	for path := range expected {
		if _, exists := seen[path]; !exists {
			missing = append(missing, path)
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 || len(entries) != len(expected) {
		t.Fatalf("source disposition coverage incomplete: entries=%d expected=%d missing=%v", len(entries), len(expected), missing)
	}
	wantKinds := map[string]int{"task": 164, "bug": 152, "skill": 19}
	wantDecisionCounts := map[string]int{"accepted": 77, "historical": 239, "conditional": 19}
	if !reflect.DeepEqual(kindCounts, wantKinds) || !reflect.DeepEqual(decisionCounts, wantDecisionCounts) {
		t.Fatalf("source disposition counts kinds=%v decisions=%v, want kinds=%v decisions=%v", kindCounts, decisionCounts, wantKinds, wantDecisionCounts)
	}
	t.Logf("source_dispositions: entries=%d tasks=164 bugs=152 skills=19 coverage=100%% hashes=verified", len(entries))
}

func traceSourceHasRole(source traceSourceDisposition, role string) bool {
	if source.Kind == role {
		return true
	}
	for _, additional := range source.AdditionalRoles {
		if additional == role {
			return true
		}
	}
	return false
}

func traceExpandSourcePaths(t *testing.T, source tracePendingSource) []string {
	t.Helper()
	unique := make(map[string]struct{})
	for _, exactPath := range source.ExactPaths {
		if _, err := os.Stat(exactPath); err != nil {
			t.Fatalf("pending source %q exact path %q: %v", source.ID, exactPath, err)
		}
		unique[filepath.ToSlash(exactPath)] = struct{}{}
	}
	for _, pattern := range source.PathPatterns {
		matches, err := filepath.Glob(filepath.FromSlash(pattern))
		if err != nil {
			t.Fatalf("pending source %q invalid pattern %q: %v", source.ID, pattern, err)
		}
		if len(matches) == 0 {
			t.Fatalf("pending source %q pattern %q matches no files", source.ID, pattern)
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || !info.Mode().IsRegular() {
				t.Fatalf("pending source %q path %q is not a regular file", source.ID, match)
			}
			unique[filepath.ToSlash(match)] = struct{}{}
		}
	}
	paths := make([]string, 0, len(unique))
	for path := range unique {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func traceFileManifestDigest(t *testing.T, paths []string) string {
	t.Helper()
	lines := make([]string, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(content)
		lines = append(lines, path+"|sha256:"+hex.EncodeToString(sum[:]))
	}
	return traceStringsDigest(lines)
}

func traceFileSHA256(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func traceReadDispositionJSONL(t *testing.T, path string) []traceSourceDisposition {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var entries []traceSourceDisposition
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			t.Fatalf("%s:%d empty line", path, lineNumber)
		}
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		var entry traceSourceDisposition
		if err := decoder.Decode(&entry); err != nil {
			t.Fatalf("decode %s:%d: %v", path, lineNumber, err)
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			t.Fatalf("decode %s:%d trailing content: %v", path, lineNumber, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return entries
}
