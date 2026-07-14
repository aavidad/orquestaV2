package acceptance_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
)

const v05FixturePath = "acceptance/fixtures/v05_goal_dag_phases.json"
const v05TrustedBaseGitCommitOID = "1a9ce60df7374fda0499379d04a97a5c1563ed7a"

type v05Fixture struct {
	SchemaVersion                    int            `json:"schema_version"`
	ReceiptSchemaVersion             int            `json:"receipt_schema_version"`
	ContractID                       string         `json:"contract_id"`
	TrustedBaseGitCommitOID          string         `json:"trusted_base_git_commit_oid"`
	ProductDeltaBaseGitCommitOID     string         `json:"product_delta_base_git_commit_oid"`
	ProductDeltaSealedGitCommitOID   string         `json:"product_delta_sealed_git_commit_oid"`
	Command                          string         `json:"command"`
	ExecutionArgv                    []string       `json:"execution_argv"`
	OutputPath                       string         `json:"output_path"`
	ReceiptPath                      string         `json:"receipt_path"`
	CandidateSubjects                []string       `json:"candidate_subjects"`
	OwnedCapabilityIDs               []string       `json:"owned_capability_ids"`
	RejectedCapabilityIDs            []string       `json:"rejected_capability_ids"`
	PhaseRequiredSemantics           []string       `json:"phase_required_semantics"`
	PhaseForbiddenLifecycleSemantics []string       `json:"phase_forbidden_lifecycle_semantics"`
	WorkRequirementSemantics         []string       `json:"work_requirement_semantics"`
	ReadyCohort                      v05ReadyCohort `json:"ready_cohort"`
	Assertions                       []string       `json:"assertions"`
}

type v05ReadyCohort struct {
	Items             []v05WorkDefinition `json:"items"`
	ExpectedReadyRefs []string            `json:"expected_ready_refs"`
}

type v05WorkDefinition struct {
	Ref            string   `json:"ref"`
	DependencyRefs []string `json:"dependency_refs"`
	WriteSet       []string `json:"write_set"`
}

func TestAcceptanceV05GoalDAGPhases(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v05Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v05FixturePath)))
	v05AssertFixtureHeader(t, repositoryRoot, fixture)

	t.Run("phase_and_execution_requirements_are_structural_metadata", func(t *testing.T) {
		v05AssertPhaseShape(t, fixture)
		v05AssertWorkItemShape(t, fixture)
	})

	t.Run("phase_and_work_metadata_round_trip_exactly", func(t *testing.T) {
		v05AssertMetadataRoundTrip(t)
	})

	t.Run("contractual_lineage_keeps_goal_open_until_all_work_is_terminal", func(t *testing.T) {
		v05AssertContractualLineageDoesNotCloseGoalEarly(t)
	})

	t.Run("unsatisfied_dependency_never_starts", func(t *testing.T) {
		v05AssertUnsatisfiedDependencyCannotStart(t)
	})

	t.Run("ready_cohort_is_deterministic_maximal_and_conflict_free", func(t *testing.T) {
		v05AssertReadyCohort(t, fixture.ReadyCohort)
	})

	t.Run("restore_rejects_unsatisfied_active_dependency", func(t *testing.T) {
		v05AssertRestoreRejectsUnsatisfiedDependency(t)
	})

	t.Run("restore_rejects_overlapping_running_items", func(t *testing.T) {
		v05AssertRestoreRejectsRunningConflict(t)
	})

	t.Run("plan_generations_are_monotonic", func(t *testing.T) {
		v05AssertPlansCannotDeleteWork(t)
		v05AssertRunningGoalCanReplanMonotonically(t)
	})
}

func TestAcceptanceV05GoalDAGPhasesReceipt(t *testing.T) {
	evidenceAssertReceiptV3(t, evidenceRepositoryRoot(t), evidenceReceiptV3Expectation{
		Contract: "AC-V05-GOAL-DAG-PHASES", FixturePath: v05FixturePath,
		ReceiptPath:       "product/evidence/v05_goal_dag_phases.json",
		ExecutedNotBefore: "2026-07-14T00:00:00Z", TrustedBaseGitCommitOID: v05TrustedBaseGitCommitOID,
	})
}

func TestV05CandidateSubjectsCoverCommittedDelta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v05Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v05FixturePath)))
	if err := evidenceValidateSealedCommit(
		repositoryRoot, fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID,
	); err != nil {
		t.Fatal(err)
	}
	output, err := evidenceGit(
		repositoryRoot, "diff", "--name-only",
		fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID, "--",
	)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.TrimSpace(string(output))
	var changed []string
	if text != "" {
		changed = strings.Split(text, "\n")
	}
	sort.Strings(changed)
	if !reflect.DeepEqual(changed, fixture.CandidateSubjects) {
		t.Fatalf("V05 candidate subjects differ from sealed product delta:\nchanged=%v\nfixture=%v", changed, fixture.CandidateSubjects)
	}
}

func v05AssertFixtureHeader(t *testing.T, repositoryRoot string, fixture v05Fixture) {
	t.Helper()
	wantCommand := "sh -c '" + v05ValidationShellBody() + "'"
	wantArgv := []string{"sh", "-c", v05ValidationShellBody()}
	wantCapabilities := []string{
		"GOV-04",
		"ORC-01", "ORC-02", "ORC-06",
		"STG-00",
	}
	sort.Strings(wantCapabilities)
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V05-GOAL-DAG-PHASES" || fixture.TrustedBaseGitCommitOID != v05TrustedBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v05TrustedBaseGitCommitOID ||
		fixture.ProductDeltaSealedGitCommitOID != "a7d5774086a517674d2a56f9707f2ecfd4d73487" ||
		fixture.Command != wantCommand || !reflect.DeepEqual(fixture.ExecutionArgv, wantArgv) ||
		fixture.OutputPath != "product/evidence/v05_goal_dag_phases.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v05_goal_dag_phases.json" ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, wantCapabilities) ||
		!reflect.DeepEqual(fixture.RejectedCapabilityIDs, []string{"ORC-18"}) || len(fixture.Assertions) != 6 {
		t.Fatalf("invalid V05 fixture header: %+v", fixture)
	}
	if !sort.StringsAreSorted(fixture.CandidateSubjects) || v05HasDuplicate(fixture.CandidateSubjects) {
		t.Fatalf("candidate subjects must be unique and sorted: %v", fixture.CandidateSubjects)
	}
	for _, relative := range fixture.CandidateSubjects {
		if _, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(relative))); err != nil {
			t.Errorf("candidate subject %q is not readable: %v", relative, err)
		}
	}
}

func v05ValidationShellBody() string {
	return "go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapV05ScopeAndExecutableContract|TestAcceptanceV05GoalDAGPhases|TestV05CandidateSubjectsCoverCommittedDelta)$\"" +
		" && go test -mod=vendor -count=1 ./internal/goal ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/interfaces/mcp ./internal/bootstrap"
}

func v05AssertPhaseShape(t *testing.T, fixture v05Fixture) {
	t.Helper()
	phaseType := reflect.TypeOf(goal.PhaseInstance{})
	tokens := v05DirectTypeTokens(phaseType)
	for _, semantic := range fixture.PhaseRequiredSemantics {
		if !v05ContainsSemantic(tokens, semantic) {
			t.Errorf("PhaseInstance lacks required immutable metadata %q; fields/types=%v", semantic, tokens)
		}
	}
	for _, semantic := range fixture.PhaseForbiddenLifecycleSemantics {
		if v05ContainsSemantic(tokens, semantic) || v05TypeHasMethodSemantic(phaseType, semantic) {
			t.Errorf("PhaseInstance owns forbidden lifecycle semantic %q", semantic)
		}
	}
}

func v05AssertWorkItemShape(t *testing.T, fixture v05Fixture) {
	t.Helper()
	workItemType := reflect.TypeOf(goal.WorkItem{})
	tokens := v05RecursiveTypeTokens(workItemType, 3, make(map[reflect.Type]bool))
	for _, semantic := range fixture.WorkRequirementSemantics {
		if !v05ContainsSemantic(tokens, semantic) {
			t.Errorf("WorkItem execution requirements lack %q metadata; shape=%v", semantic, tokens)
		}
	}
	if !v05HasTypedParentRef(workItemType, 3, make(map[reflect.Type]bool)) {
		t.Errorf("WorkItem lacks an explicit contractual parent typed with WorkItemRef")
	}
}

func v05DirectTypeTokens(value reflect.Type) []string {
	value = v05UnwrapType(value)
	tokens := []string{strings.ToLower(value.Name())}
	for index := 0; index < value.NumField(); index++ {
		field := value.Field(index)
		tokens = append(tokens, strings.ToLower(field.Name), strings.ToLower(field.Type.String()))
	}
	return tokens
}

func v05RecursiveTypeTokens(value reflect.Type, depth int, seen map[reflect.Type]bool) []string {
	value = v05UnwrapType(value)
	if depth < 0 || value.Kind() != reflect.Struct || seen[value] {
		return nil
	}
	seen[value] = true
	tokens := []string{strings.ToLower(value.Name())}
	for index := 0; index < value.NumField(); index++ {
		field := value.Field(index)
		tokens = append(tokens, strings.ToLower(field.Name), strings.ToLower(field.Type.String()))
		nested := v05UnwrapType(field.Type)
		if nested.Kind() == reflect.Struct && nested.PkgPath() == value.PkgPath() {
			tokens = append(tokens, v05RecursiveTypeTokens(nested, depth-1, seen)...)
		}
	}
	return tokens
}

func v05HasTypedParentRef(value reflect.Type, depth int, seen map[reflect.Type]bool) bool {
	value = v05UnwrapType(value)
	if depth < 0 || value.Kind() != reflect.Struct || seen[value] {
		return false
	}
	seen[value] = true
	for index := 0; index < value.NumField(); index++ {
		field := value.Field(index)
		if strings.Contains(strings.ToLower(field.Name), "parent") && strings.Contains(field.Type.String(), "WorkItemRef") {
			return true
		}
		nested := v05UnwrapType(field.Type)
		if nested.Kind() == reflect.Struct && nested.PkgPath() == value.PkgPath() && v05HasTypedParentRef(nested, depth-1, seen) {
			return true
		}
	}
	return false
}

func v05UnwrapType(value reflect.Type) reflect.Type {
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		value = value.Elem()
	}
	return value
}

func v05ContainsSemantic(tokens []string, semantic string) bool {
	aliases := []string{semantic}
	if semantic == "identity_or_ref" {
		aliases = []string{"identity", "phaseref"}
	}
	if semantic == "criteria" {
		aliases = []string{"criteria", "criterion"}
	}
	if semantic == "capability" {
		aliases = []string{"capability", "capabilities"}
	}
	for _, token := range tokens {
		for _, alias := range aliases {
			if strings.Contains(token, alias) {
				return true
			}
		}
	}
	return false
}

func v05TypeHasMethodSemantic(value reflect.Type, semantic string) bool {
	for _, candidate := range []reflect.Type{value, reflect.PointerTo(value)} {
		for index := 0; index < candidate.NumMethod(); index++ {
			if strings.Contains(strings.ToLower(candidate.Method(index).Name), semantic) {
				return true
			}
		}
	}
	return false
}

func v05AssertMetadataRoundTrip(t *testing.T) {
	t.Helper()
	fixture := v05NewDomainFixture(t, "metadata-round-trip")
	phaseRef := v05MustRef(t, "phase-instance:v05-metadata", goal.NewPhaseRef)
	phaseKey := v05MustRef(t, "phase:v05-metadata", goal.NewPhaseKey)
	templateRef := v05MustRef(t, "phase-template:v05-metadata", goal.NewPhaseTemplateRef)
	inputRefs := []goal.InputRef{
		v05MustRef(t, "input:v05-source", goal.NewInputRef),
		v05MustRef(t, "input:v05-context", goal.NewInputRef),
	}
	criterionRefs := []goal.CriterionRef{
		v05MustRef(t, "criterion:v05-complete", goal.NewCriterionRef),
		v05MustRef(t, "criterion:v05-reviewed", goal.NewCriterionRef),
	}
	phase, err := goal.NewPhaseInstanceWithMetadata(goal.PhaseInstanceInput{
		Ref: phaseRef, Key: phaseKey, TemplateRef: templateRef,
		InputRefs: inputRefs, CriterionRefs: criterionRefs,
	})
	if err != nil {
		t.Fatalf("NewPhaseInstanceWithMetadata: %v", err)
	}

	skillRefs := []goal.SkillRef{
		v05MustRef(t, "skill:v05-research", goal.NewSkillRef),
		v05MustRef(t, "skill:v05-review", goal.NewSkillRef),
	}
	toolRefs := []goal.ToolRef{
		v05MustRef(t, "tool:v05-workspace", goal.NewToolRef),
		v05MustRef(t, "tool:v05-tests", goal.NewToolRef),
	}
	capabilityRefs := []goal.CapabilityRef{
		v05MustRef(t, "capability:v05-dag", goal.NewCapabilityRef),
		v05MustRef(t, "capability:v05-review", goal.NewCapabilityRef),
	}
	itemRef := v05MustRef(t, "work-item:v05-metadata", goal.NewWorkItemRef)
	item := fixture.itemWithMetadata(t, v05ItemMetadata{
		Ref: itemRef, Phase: phaseKey,
		SkillRefs: skillRefs, ToolRefs: toolRefs, CapabilityRefs: capabilityRefs,
	})
	applied := fixture.apply(t, 1, phase, []goal.WorkItem{item})
	snapshot := applied.Snapshot()
	if len(snapshot.Phases) != 1 || !reflect.DeepEqual(snapshot.Phases[0], goal.PhaseInstanceSnapshot{
		Ref: phaseRef.String(), Key: phaseKey.String(), TemplateRef: templateRef.String(),
		InputRefs: v05RefStrings(inputRefs), CriterionRefs: v05RefStrings(criterionRefs),
	}) {
		t.Errorf("phase snapshot did not preserve exact metadata: %+v", snapshot.Phases)
	}
	if len(snapshot.WorkItems) != 1 ||
		!reflect.DeepEqual(snapshot.WorkItems[0].SkillRefs, v05RefStrings(skillRefs)) ||
		!reflect.DeepEqual(snapshot.WorkItems[0].ToolRefs, v05RefStrings(toolRefs)) ||
		!reflect.DeepEqual(snapshot.WorkItems[0].CapabilityRefs, v05RefStrings(capabilityRefs)) {
		t.Errorf("work requirement snapshot did not preserve exact refs: %+v", snapshot.WorkItems)
	}

	restored, err := goal.RestoreGoal(snapshot)
	if err != nil {
		t.Fatalf("RestoreGoal(metadata): %v", err)
	}
	if !reflect.DeepEqual(restored.Snapshot(), snapshot) {
		t.Errorf("metadata snapshot changed after restore:\n got: %+v\nwant: %+v", restored.Snapshot(), snapshot)
	}
	restoredPhases := restored.Phases()
	if len(restoredPhases) != 1 || restoredPhases[0].Ref() != phaseRef ||
		restoredPhases[0].TemplateRef() != templateRef ||
		!reflect.DeepEqual(restoredPhases[0].InputRefs(), inputRefs) ||
		!reflect.DeepEqual(restoredPhases[0].CriterionRefs(), criterionRefs) {
		t.Errorf("restored PhaseInstance metadata differs: %+v", restoredPhases)
	}
	restoredItem, exists := restored.WorkItem(itemRef)
	if !exists || !reflect.DeepEqual(restoredItem.SkillRefs(), skillRefs) ||
		!reflect.DeepEqual(restoredItem.ToolRefs(), toolRefs) ||
		!reflect.DeepEqual(restoredItem.CapabilityRefs(), capabilityRefs) {
		t.Errorf("restored WorkItem requirements differ: %+v exists=%v", restoredItem, exists)
	}
}

func v05AssertContractualLineageDoesNotCloseGoalEarly(t *testing.T) {
	t.Helper()
	fixture := v05NewDomainFixture(t, "parent-child")
	phase := v05MustPhase(t, "phase:v05-parent-child")
	parentRef := v05MustRef(t, "work-item:v05-parent", goal.NewWorkItemRef)
	childRef := v05MustRef(t, "work-item:v05-child", goal.NewWorkItemRef)
	parent := fixture.item(t, parentRef, phase.Key(), nil, []goal.WriteScope{v05MustScope(t, "internal/parent")})
	child := fixture.itemWithMetadata(t, v05ItemMetadata{
		Ref: childRef, Phase: phase.Key(), Parent: parentRef,
		WriteSet: []goal.WriteScope{v05MustScope(t, "internal/child")},
	})
	running := fixture.applyAndStart(t, phase, []goal.WorkItem{parent, child})
	children := running.ChildWorkItems(parentRef)
	if len(children) != 1 || children[0].Ref() != childRef {
		t.Fatalf("ChildWorkItems(%q) = %v, want exact child %q", parentRef, v05WorkItemRefs(children), childRef)
	}
	if got, ok := children[0].Parent(); !ok || got != parentRef {
		t.Fatalf("child Parent() = %q/%v, want %q/true", got, ok, parentRef)
	}

	running = v05StartItem(t, running, parentRef, "execution:v05-parent", fixture.startedAt.Add(time.Minute))
	running = v05StartItem(t, running, childRef, "execution:v05-child", fixture.startedAt.Add(2*time.Minute))
	parent, _ = running.WorkItem(parentRef)
	running, err := running.SucceedWorkItem(
		running.Revision(), parent.Revision(), parentRef,
		[]goal.ArtifactRef{v05MustRef(t, "artifact:v05-parent", goal.NewArtifactRef)},
		[]goal.AttestationRef{v05MustRef(t, "attestation:v05-parent", goal.NewAttestationRef)},
		fixture.startedAt.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatalf("SucceedWorkItem(parent): %v", err)
	}
	if _, err := running.Close(running.Revision(), goal.GoalOutcomeSucceeded, fixture.startedAt.Add(4*time.Minute)); err == nil {
		t.Error("Goal closed while a contractual child remained nonterminal")
	}

	child, _ = running.WorkItem(childRef)
	running, err = running.SucceedWorkItem(
		running.Revision(), child.Revision(), childRef,
		[]goal.ArtifactRef{v05MustRef(t, "artifact:v05-child", goal.NewArtifactRef)},
		[]goal.AttestationRef{v05MustRef(t, "attestation:v05-child", goal.NewAttestationRef)},
		fixture.startedAt.Add(4*time.Minute),
	)
	if err != nil {
		t.Fatalf("SucceedWorkItem(child): %v", err)
	}
	parent, _ = running.WorkItem(parentRef)
	if parent.State() != goal.WorkItemStateSucceeded {
		t.Errorf("parent state changed while child completed = %q, want succeeded", parent.State())
	}
}

func v05AssertUnsatisfiedDependencyCannotStart(t *testing.T) {
	t.Helper()
	fixture := v05NewDomainFixture(t, "dependency")
	phase := v05MustPhase(t, "phase:v05-dependency")
	rootRef := v05MustRef(t, "work-item:v05-dependency-root", goal.NewWorkItemRef)
	childRef := v05MustRef(t, "work-item:v05-dependency-child", goal.NewWorkItemRef)
	root := fixture.item(t, rootRef, phase.Key(), nil, nil)
	child := fixture.item(t, childRef, phase.Key(), []goal.WorkItemRef{rootRef}, nil)
	running := fixture.applyAndStart(t, phase, []goal.WorkItem{root, child})
	if got := v05ReadyRefs(running); !reflect.DeepEqual(got, []string{rootRef.String()}) {
		t.Errorf("ready refs with unsatisfied child = %v, want only root", got)
	}
	child, _ = running.WorkItem(childRef)
	_, err := running.StartWorkItem(
		running.Revision(), child.Revision(), childRef,
		v05MustRef(t, "execution:v05-too-early", goal.NewExecutionRef), fixture.startedAt.Add(time.Minute),
	)
	if err == nil {
		t.Error("StartWorkItem accepted an unsatisfied dependency")
	}
}

func v05AssertReadyCohort(t *testing.T, contract v05ReadyCohort) {
	t.Helper()
	fixture := v05NewDomainFixture(t, "ready")
	phase := v05MustPhase(t, "phase:v05-ready")
	items := make([]goal.WorkItem, 0, len(contract.Items))
	for _, definition := range contract.Items {
		dependencies := make([]goal.WorkItemRef, 0, len(definition.DependencyRefs))
		for _, value := range definition.DependencyRefs {
			dependencies = append(dependencies, v05MustRef(t, value, goal.NewWorkItemRef))
		}
		writeSet := make([]goal.WriteScope, 0, len(definition.WriteSet))
		for _, value := range definition.WriteSet {
			writeSet = append(writeSet, v05MustScope(t, value))
		}
		items = append(items, fixture.item(
			t, v05MustRef(t, definition.Ref, goal.NewWorkItemRef), phase.Key(), dependencies, writeSet,
		))
	}
	running := fixture.applyAndStart(t, phase, items)
	got := v05ReadyRefs(running)
	if !reflect.DeepEqual(got, contract.ExpectedReadyRefs) {
		t.Errorf("ReadyWorkItems refs = %v, want deterministic cohort %v", got, contract.ExpectedReadyRefs)
	}
	if replay := v05ReadyRefs(running); !reflect.DeepEqual(replay, got) {
		t.Errorf("ReadyWorkItems is non-deterministic: first=%v replay=%v", got, replay)
	}
	v05AssertCohortConflictFree(t, got, contract.Items)
	v05AssertCohortMaximal(t, got, contract.Items)
}

func v05AssertCohortConflictFree(t *testing.T, refs []string, definitions []v05WorkDefinition) {
	t.Helper()
	byRef := v05DefinitionsByRef(definitions)
	for left := 0; left < len(refs); left++ {
		for right := left + 1; right < len(refs); right++ {
			if v05WriteSetsOverlap(byRef[refs[left]].WriteSet, byRef[refs[right]].WriteSet) {
				t.Errorf("ready cohort contains overlapping refs %q and %q", refs[left], refs[right])
			}
		}
	}
}

func v05AssertCohortMaximal(t *testing.T, refs []string, definitions []v05WorkDefinition) {
	t.Helper()
	byRef := v05DefinitionsByRef(definitions)
	selected := make(map[string]bool, len(refs))
	for _, ref := range refs {
		selected[ref] = true
	}
	for _, candidate := range definitions {
		if selected[candidate.Ref] {
			continue
		}
		conflicts := false
		for _, selectedRef := range refs {
			conflicts = conflicts || v05WriteSetsOverlap(candidate.WriteSet, byRef[selectedRef].WriteSet)
		}
		if !conflicts {
			t.Errorf("ready cohort is not maximal; compatible ref %q was omitted", candidate.Ref)
		}
	}
}

func v05DefinitionsByRef(definitions []v05WorkDefinition) map[string]v05WorkDefinition {
	result := make(map[string]v05WorkDefinition, len(definitions))
	for _, definition := range definitions {
		result[definition.Ref] = definition
	}
	return result
}

func v05WriteSetsOverlap(left, right []string) bool {
	for _, leftScope := range left {
		for _, rightScope := range right {
			if leftScope == rightScope || strings.HasPrefix(leftScope, rightScope+"/") || strings.HasPrefix(rightScope, leftScope+"/") {
				return true
			}
		}
	}
	return false
}

func v05AssertRestoreRejectsUnsatisfiedDependency(t *testing.T) {
	t.Helper()
	fixture := v05NewDomainFixture(t, "restore-dependency")
	phase := v05MustPhase(t, "phase:v05-restore-dependency")
	rootRef := v05MustRef(t, "work-item:v05-restore-root", goal.NewWorkItemRef)
	childRef := v05MustRef(t, "work-item:v05-restore-child", goal.NewWorkItemRef)
	running := fixture.applyAndStart(t, phase, []goal.WorkItem{
		fixture.item(t, rootRef, phase.Key(), nil, nil),
		fixture.item(t, childRef, phase.Key(), []goal.WorkItemRef{rootRef}, nil),
	})
	snapshot := running.Snapshot()
	child := v05SnapshotItem(t, &snapshot, childRef.String())
	child.State = goal.WorkItemStateRunning
	child.Revision = 2
	child.StartedAt = fixture.startedAt.Add(time.Minute)
	child.ExecutionRef = "execution:v05-restored-child"
	snapshot.Revision++
	if _, err := goal.RestoreGoal(snapshot); err == nil {
		t.Error("RestoreGoal accepted a running child whose dependency is pending")
	}
}

func v05AssertRestoreRejectsRunningConflict(t *testing.T) {
	t.Helper()
	fixture := v05NewDomainFixture(t, "restore-conflict")
	phase := v05MustPhase(t, "phase:v05-restore-conflict")
	leftRef := v05MustRef(t, "work-item:v05-conflict-left", goal.NewWorkItemRef)
	rightRef := v05MustRef(t, "work-item:v05-conflict-right", goal.NewWorkItemRef)
	running := fixture.applyAndStart(t, phase, []goal.WorkItem{
		fixture.item(t, leftRef, phase.Key(), nil, []goal.WriteScope{v05MustScope(t, "internal/shared")}),
		fixture.item(t, rightRef, phase.Key(), nil, []goal.WriteScope{v05MustScope(t, "internal/shared/file.go")}),
	})
	snapshot := running.Snapshot()
	for index, ref := range []goal.WorkItemRef{leftRef, rightRef} {
		item := v05SnapshotItem(t, &snapshot, ref.String())
		item.State = goal.WorkItemStateRunning
		item.Revision = 2
		item.StartedAt = fixture.startedAt.Add(time.Minute)
		item.ExecutionRef = "execution:v05-conflict-" + string(rune('a'+index))
	}
	snapshot.Revision += 2
	if _, err := goal.RestoreGoal(snapshot); err == nil {
		t.Error("RestoreGoal accepted two running WorkItems with overlapping write-sets")
	}
}

func v05AssertPlansCannotDeleteWork(t *testing.T) {
	t.Helper()
	fixture := v05NewDomainFixture(t, "monotonic-pending")
	phase := v05MustPhase(t, "phase:v05-monotonic-pending")
	originalRef := v05MustRef(t, "work-item:v05-original", goal.NewWorkItemRef)
	replacementRef := v05MustRef(t, "work-item:v05-replacement", goal.NewWorkItemRef)
	applied := fixture.apply(t, 1, phase, []goal.WorkItem{
		fixture.item(t, originalRef, phase.Key(), nil, nil),
	})
	replacementPlan := v05MustPlan(t, goal.PlanInput{
		Generation: 2,
		Phases:     []goal.PhaseInstance{phase},
		WorkItems: []goal.WorkItem{
			fixture.item(t, replacementRef, phase.Key(), nil, nil),
		},
	})
	updated, err := applied.ApplyPlan(applied.Revision(), replacementPlan)
	if err == nil {
		if _, preserved := updated.WorkItem(originalRef); !preserved {
			t.Error("ApplyPlan accepted generation N+1 after deleting an existing WorkItem")
		}
	} else if _, preserved := applied.WorkItem(originalRef); !preserved {
		t.Error("rejected replacement mutated the prior Goal snapshot")
	}
}

func v05AssertRunningGoalCanReplanMonotonically(t *testing.T) {
	t.Helper()
	fixture := v05NewDomainFixture(t, "monotonic-running")
	phase := v05MustPhase(t, "phase:v05-monotonic-running")
	parentRef := v05MustRef(t, "work-item:v05-running-parent", goal.NewWorkItemRef)
	childRef := v05MustRef(t, "work-item:v05-running-child", goal.NewWorkItemRef)
	parent := fixture.item(t, parentRef, phase.Key(), nil, nil)
	running := fixture.start(t, fixture.apply(t, 1, phase, []goal.WorkItem{parent}))
	running = v05StartItem(t, running, parentRef, "execution:v05-running-parent", fixture.startedAt.Add(time.Minute))
	child := fixture.itemWithMetadata(t, v05ItemMetadata{
		Ref: childRef, Phase: phase.Key(), Parent: parentRef,
	})
	replan := v05MustPlan(t, goal.PlanInput{
		Generation: 2,
		Phases:     running.Phases(),
		WorkItems:  append(running.WorkItems(), child),
	})
	updated, err := running.ApplyPlan(running.Revision(), replan)
	if err != nil {
		t.Errorf("running Goal cannot accept a monotonic split/replan generation: %v", err)
		return
	}
	if updated.PlanGeneration() != 2 || updated.WorkItemCount() != 2 {
		t.Errorf("running replan generation/items = %d/%d, want 2/2", updated.PlanGeneration(), updated.WorkItemCount())
	}
	for _, ref := range []goal.WorkItemRef{parentRef, childRef} {
		if _, exists := updated.WorkItem(ref); !exists {
			t.Errorf("running replan lost WorkItem %q", ref)
		}
	}
	updatedParent, _ := updated.WorkItem(parentRef)
	if updatedParent.State() != goal.WorkItemStateRunning {
		t.Errorf("running replan changed parent lifecycle to %q", updatedParent.State())
	}
	children := updated.ChildWorkItems(parentRef)
	if len(children) != 1 || children[0].Ref() != childRef {
		t.Errorf("running replan children = %v, want %q", v05WorkItemRefs(children), childRef)
	}
}

type v05DomainFixture struct {
	aggregate goal.Goal
	actor     goal.ActorRef
	project   goal.ProjectRef
	itemAt    time.Time
	startedAt time.Time
}

func v05NewDomainFixture(t *testing.T, label string) v05DomainFixture {
	t.Helper()
	base := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
	actor := v05MustRef(t, "actor:v05-"+label, goal.NewActorRef)
	project := v05MustRef(t, "project:v05-"+label, goal.NewProjectRef)
	manifest, err := goal.NewIntentManifest(goal.IntentManifestInput{
		Ref: v05MustRef(t, "intent:v05-"+label, goal.NewIntentRef), Actor: actor, Project: project,
		Statement: "validate V05 " + label, SubmittedAt: base,
	})
	if err != nil {
		t.Fatalf("NewIntentManifest: %v", err)
	}
	spec, err := goal.NewInitialAppSpec(goal.AppSpecInput{
		Ref: v05MustRef(t, "app-spec:v05-"+label, goal.NewAppSpecRef), Intent: manifest,
		Objective: "validate V05 " + label, Reason: "operator.v05_contract",
		ConfirmedBy: actor, ConfirmedAt: base.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("NewInitialAppSpec: %v", err)
	}
	aggregate, err := goal.NewGoal(
		v05MustRef(t, "goal:v05-"+label, goal.NewGoalRef), spec, base.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatalf("NewGoal: %v", err)
	}
	return v05DomainFixture{
		aggregate: aggregate, actor: actor, project: project,
		itemAt: base.Add(3 * time.Minute), startedAt: base.Add(4 * time.Minute),
	}
}

func (fixture v05DomainFixture) item(
	t *testing.T,
	ref goal.WorkItemRef,
	phase goal.PhaseKey,
	dependencies []goal.WorkItemRef,
	writeSet []goal.WriteScope,
) goal.WorkItem {
	t.Helper()
	return fixture.itemWithMetadata(t, v05ItemMetadata{
		Ref: ref, Phase: phase, Dependencies: dependencies, WriteSet: writeSet,
	})
}

type v05ItemMetadata struct {
	Ref            goal.WorkItemRef
	Phase          goal.PhaseKey
	Parent         goal.WorkItemRef
	Dependencies   []goal.WorkItemRef
	WriteSet       []goal.WriteScope
	SkillRefs      []goal.SkillRef
	ToolRefs       []goal.ToolRef
	CapabilityRefs []goal.CapabilityRef
}

func (fixture v05DomainFixture) itemWithMetadata(t *testing.T, metadata v05ItemMetadata) goal.WorkItem {
	t.Helper()
	item, err := goal.NewWorkItem(goal.NewWorkItemInput{
		Ref: metadata.Ref, Goal: fixture.aggregate.Ref(), Actor: fixture.actor, Project: fixture.project,
		Objective: "execute " + metadata.Ref.String(), CreatedAt: fixture.itemAt,
		Phase: metadata.Phase, Role: goal.DefaultRoleKey(), Parent: metadata.Parent,
		Dependencies: metadata.Dependencies, WriteSet: metadata.WriteSet,
		SkillRefs: metadata.SkillRefs, ToolRefs: metadata.ToolRefs, CapabilityRefs: metadata.CapabilityRefs,
		OutputContract: goal.EvidenceBundleOutputContract(),
	})
	if err != nil {
		t.Fatalf("NewWorkItem(%q): %v", metadata.Ref, err)
	}
	return item
}

func (fixture v05DomainFixture) apply(
	t *testing.T,
	generation goal.PlanGeneration,
	phase goal.PhaseInstance,
	items []goal.WorkItem,
) goal.Goal {
	t.Helper()
	applied, err := fixture.aggregate.ApplyPlan(fixture.aggregate.Revision(), v05MustPlan(t, goal.PlanInput{
		Generation: generation, Phases: []goal.PhaseInstance{phase}, WorkItems: items,
	}))
	if err != nil {
		t.Fatalf("ApplyPlan: %v", err)
	}
	return applied
}

func (fixture v05DomainFixture) start(t *testing.T, aggregate goal.Goal) goal.Goal {
	t.Helper()
	running, err := aggregate.Start(aggregate.Revision(), fixture.startedAt)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	return running
}

func (fixture v05DomainFixture) applyAndStart(t *testing.T, phase goal.PhaseInstance, items []goal.WorkItem) goal.Goal {
	t.Helper()
	return fixture.start(t, fixture.apply(t, 1, phase, items))
}

func v05SnapshotItem(t *testing.T, snapshot *goal.GoalSnapshot, ref string) *goal.WorkItemSnapshot {
	t.Helper()
	for index := range snapshot.WorkItems {
		if snapshot.WorkItems[index].Ref == ref {
			return &snapshot.WorkItems[index]
		}
	}
	t.Fatalf("snapshot lacks WorkItem %q", ref)
	return nil
}

func v05ReadyRefs(aggregate goal.Goal) []string {
	ready := aggregate.ReadyWorkItems()
	refs := make([]string, len(ready))
	for index, item := range ready {
		refs[index] = item.Ref().String()
	}
	return refs
}

func v05StartItem(t *testing.T, aggregate goal.Goal, ref goal.WorkItemRef, execution string, at time.Time) goal.Goal {
	t.Helper()
	item, exists := aggregate.WorkItem(ref)
	if !exists {
		t.Fatalf("Goal lacks WorkItem %q", ref)
	}
	updated, err := aggregate.StartWorkItem(
		aggregate.Revision(), item.Revision(), ref,
		v05MustRef(t, execution, goal.NewExecutionRef), at,
	)
	if err != nil {
		t.Fatalf("StartWorkItem(%q): %v", ref, err)
	}
	return updated
}

func v05WorkItemRefs(items []goal.WorkItem) []string {
	refs := make([]string, len(items))
	for index, item := range items {
		refs[index] = item.Ref().String()
	}
	return refs
}

func v05RefStrings[T interface{ String() string }](refs []T) []string {
	values := make([]string, len(refs))
	for index, ref := range refs {
		values[index] = ref.String()
	}
	return values
}

func v05MustPhase(t *testing.T, value string) goal.PhaseInstance {
	t.Helper()
	key, err := goal.NewPhaseKey(value)
	if err != nil {
		t.Fatalf("NewPhaseKey(%q): %v", value, err)
	}
	phase, err := goal.NewPhaseInstance(key)
	if err != nil {
		t.Fatalf("NewPhaseInstance(%q): %v", value, err)
	}
	return phase
}

func v05MustPlan(t *testing.T, input goal.PlanInput) goal.Plan {
	t.Helper()
	plan, err := goal.NewPlan(input)
	if err != nil {
		t.Fatalf("NewPlan: %v", err)
	}
	return plan
}

func v05MustScope(t *testing.T, value string) goal.WriteScope {
	t.Helper()
	scope, err := goal.NewWriteScope(value)
	if err != nil {
		t.Fatalf("NewWriteScope(%q): %v", value, err)
	}
	return scope
}

func v05MustRef[T any](t *testing.T, value string, constructor func(string) (T, error)) T {
	t.Helper()
	ref, err := constructor(value)
	if err != nil {
		t.Fatalf("new ref %q: %v", value, err)
	}
	return ref
}

func v05HasDuplicate(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, duplicate := seen[value]; duplicate {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}
