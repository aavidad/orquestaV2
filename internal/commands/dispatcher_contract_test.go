package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestCanonicalSchemasAreRecursiveClosedWorldAndRejectNull(t *testing.T) {
	for _, definition := range CanonicalDefinitions() {
		if err := ValidateSchemaContract(definition.InputSchema); err != nil {
			t.Fatalf("%s input: %v", definition.ID, err)
		}
		if err := ValidateSchemaContract(definition.OutputSchema); err != nil {
			t.Fatalf("%s output: %v", definition.ID, err)
		}
	}
	dispatcher, _, audit := testDispatcher(t)
	invocation := Invocation{
		CommandID: "orquesta.system.status", CommandVersion: "1", RequestRef: "request:null",
		ProjectRef: "project:test", Principal: testPrincipal(t), Payload: json.RawMessage("null"),
	}
	if result := dispatcher.Dispatch(context.Background(), invocation); result.Failure == nil || result.Failure.Code != CodeInvalidRequest {
		t.Fatalf("root null=%+v", result)
	}
	nested := map[string]any{
		"statement": "build", "confirm": true,
		"plan": map[string]any{
			"phases":     []any{map[string]any{"ref": "phase-instance:p", "key": "phase:p", "template_ref": "phase-template:p", "unknown": true}},
			"work_items": []any{planWorkItem()},
		},
	}
	if result := invoke(t, dispatcher, "orquesta.goals.create", "request:nested", nested, false); result.Failure == nil || result.Failure.Code != CodeInvalidRequest {
		t.Fatalf("nested unknown=%+v", result)
	}
	if audit.admits != 0 {
		t.Fatalf("invalid payload admitted=%d", audit.admits)
	}
	output := json.RawMessage(`{"goal":{"goal_ref":"goal:g","project_ref":"project:p","state":"running","revision":1,"plan_generation":1,"spec_hash":"sha256:x","work_item_count":1,"unknown":true},"created":true}`)
	if _, err := validatePayload(CanonicalDefinitions()[0].OutputSchema, output); err == nil {
		t.Fatal("nested unknown output accepted")
	}
}

func TestAPILimitsApplyToRequestBytesAndAllListQueries(t *testing.T) {
	api, audit := newFakeApplication(), newMemoryAudit()
	dispatcher, err := newDispatcher(
		api, audit, APILimits{
			MaxRequestBytes: 1024, MaxListLimit: 2, IntakeMaxQuestionRounds: 6,
		},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	large := invoke(t, dispatcher, "orquesta.goals.get", "request:large", map[string]any{"goal_ref": "goal:" + strings.Repeat("x", 2000)}, false)
	if large.Failure == nil || large.Failure.Code != CodeInvalidRequest {
		t.Fatalf("large=%+v", large)
	}
	cases := []struct {
		id        string
		payload   map[string]any
		execution bool
	}{
		{"orquesta.goals.list", map[string]any{"limit": 3}, false},
		{"orquesta.changes.list", map[string]any{"limit": 3}, false},
		{"orquesta.mailbox.list", map[string]any{"goal_ref": "goal:g", "recipient_work_item_ref": "work-item:w", "limit": 3}, true},
	}
	for index, test := range cases {
		result := invoke(t, dispatcher, test.id, "request:list:"+string(rune('a'+index)), test.payload, test.execution)
		if result.Failure == nil || result.Failure.Code != CodeInvalidRequest {
			t.Errorf("%s=%+v", test.id, result)
		}
		test.payload["limit"] = 2
		result = invoke(t, dispatcher, test.id, "request:list:ok:"+string(rune('a'+index)), test.payload, test.execution)
		if result.Failure != nil {
			t.Errorf("%s max=%+v", test.id, result)
		}
	}
}

func TestDirectorPlanInitialAndReplanFencesAreAllOrNothing(t *testing.T) {
	dispatcher, api, _ := testDispatcher(t)
	base := map[string]any{
		"goal_ref": "goal:g", "expected_goal_revision": 1, "expected_plan_generation": 1,
		"lease_token": "token:t", "lease_fence": 1, "reason": "extend",
		"plan": map[string]any{"phases": []any{}, "work_items": []any{planWorkItem()}},
	}
	if result := invoke(t, dispatcher, "orquesta.director.plan.propose", "request:initial", base, false); result.Failure != nil {
		t.Fatalf("initial=%+v", result)
	}
	partial := cloneMap(base)
	partial["cause"] = "execution_failed"
	if result := invoke(t, dispatcher, "orquesta.director.plan.propose", "request:partial", partial, false); result.Failure == nil || result.Failure.Code != CodeInvalidRequest {
		t.Fatalf("partial=%+v", result)
	}
	full := cloneMap(base)
	full["cause"], full["source_work_item_ref"] = "execution_failed", "work-item:w"
	full["expected_work_item_revision"], full["source_execution_ref"], full["source_execution_attempt"] = 1, "execution:e", 1
	if result := invoke(t, dispatcher, "orquesta.director.plan.propose", "request:replan", full, false); result.Failure != nil {
		t.Fatalf("replan=%+v", result)
	}
	if len(api.proposals) != 2 || api.proposals[0].Cause != "" || api.proposals[1].Cause != goal.ReplanCauseExecutionFailed {
		t.Fatalf("proposals=%+v", api.proposals)
	}
}

func TestPublicPlansCarryExactOptionalEgressPolicyRefIntoApplication(t *testing.T) {
	dispatcher, api, _ := testDispatcher(t)
	const policyRef = "egress-policy:codex-controlled-proxy"
	item := planWorkItem()
	item["egress_policy_ref"] = policyRef
	plan := map[string]any{
		"phases": []any{map[string]any{
			"ref": "phase-instance:egress", "key": "phase:main", "template_ref": "phase-template:egress",
		}},
		"work_items": []any{item},
	}
	created := invoke(t, dispatcher, "orquesta.goals.create", "request:egress-create", map[string]any{
		"statement": "build through controlled egress", "confirm": true, "plan": plan,
	}, false)
	if created.Failure != nil {
		t.Fatalf("create=%+v", created)
	}
	if len(api.submissions) != 1 || api.submissions[0].Plan == nil ||
		len(api.submissions[0].Plan.WorkItems) != 1 ||
		api.submissions[0].Plan.WorkItems[0].EgressPolicyRef != policyRef {
		t.Fatalf("submission egress=%+v", api.submissions)
	}

	proposed := invoke(t, dispatcher, "orquesta.director.plan.propose", "request:egress-propose", map[string]any{
		"goal_ref": "goal:g", "expected_goal_revision": 1, "expected_plan_generation": 1,
		"lease_token": "token:t", "lease_fence": 1, "reason": "extend controlled work", "plan": plan,
	}, false)
	if proposed.Failure != nil {
		t.Fatalf("propose=%+v", proposed)
	}
	if len(api.proposals) != 1 || len(api.proposals[0].Plan.WorkItems) != 1 ||
		api.proposals[0].Plan.WorkItems[0].EgressPolicyRef != policyRef {
		t.Fatalf("proposal egress=%+v", api.proposals)
	}
}

func TestCanonicalRegistryHas35CommandsAndPropagatesCouncilContracts(t *testing.T) {
	dispatcher, api, _ := testDispatcher(t)
	if got := len(dispatcher.Definitions()); got != 35 {
		t.Fatalf("definitions=%d", got)
	}
	byID := compiledDefinitionsByID()
	for _, id := range []string{"orquesta.council.round.open", "orquesta.council.skip"} {
		if _, found := byID[id]; !found {
			t.Fatalf("missing %s", id)
		}
	}
	create := map[string]any{
		"statement": "build with Council", "confirm": true,
		"plan": map[string]any{
			"phases": []any{map[string]any{"ref": "phase-instance:main", "key": "phase:main", "template_ref": "phase-template:main"}},
			"work_items": []any{map[string]any{
				"key": "work:writer", "objective": "write", "phase": "phase:main", "role": "role:worker",
				"dependencies": []any{}, "write_set": []any{"internal/commands"}, "output_contract": "evidence_bundle",
				"council_policy": "required",
			}},
		},
	}
	if result := invoke(t, dispatcher, "orquesta.goals.create", "request:council-create", create, false); result.Failure != nil {
		t.Fatalf("create=%+v", result)
	}
	if len(api.submissions) != 1 || api.submissions[0].Plan.WorkItems[0].CouncilPolicy != council.PolicyRequired {
		t.Fatalf("submission=%+v", api.submissions)
	}
	proposal := map[string]any{
		"goal_ref": "goal:g", "expected_goal_revision": 1, "expected_plan_generation": 1,
		"lease_token": "token:t", "lease_fence": 1, "cause": "governance_decision",
		"source_work_item_ref": "work-item:w", "expected_work_item_revision": 1,
		"source_execution_ref": "execution:e", "source_execution_attempt": 1,
		"council_subject_digest":  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"council_decision_ref":    "council-decision:one",
		"council_decision_digest": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"reason":                  "Council rejected", "plan": map[string]any{
			"phases": []any{}, "work_items": []any{map[string]any{
				"key": "work:successor", "objective": "rework", "phase": "phase:main", "role": "role:worker",
				"dependencies": []any{}, "write_set": []any{"internal/commands"}, "output_contract": "evidence_bundle",
				"council_policy": "auto",
			}},
		},
	}
	if result := invoke(t, dispatcher, "orquesta.director.plan.propose", "request:council-replan", proposal, false); result.Failure != nil {
		t.Fatalf("proposal=%+v", result)
	}
	got := api.proposals[len(api.proposals)-1]
	if got.Cause != goal.ReplanCauseGovernanceDecision ||
		got.CouncilSubjectDigest != application.CouncilSubjectDigest(proposal["council_subject_digest"].(string)) ||
		got.CouncilDecisionRef != proposal["council_decision_ref"].(string) ||
		got.CouncilDecisionDigest != application.CouncilSubjectDigest(proposal["council_decision_digest"].(string)) ||
		got.Plan.WorkItems[0].CouncilPolicy != council.PolicyAuto {
		t.Fatalf("proposal not Council-bound: %+v", got)
	}
}

func TestCouncilErrorsAreTypedAndNormalized(t *testing.T) {
	tests := []struct {
		err  error
		code string
	}{
		{fmt.Errorf("wrapped: %w", council.ErrInvalidPolicy), CodeInvalidRequest},
		{fmt.Errorf("wrapped: %w", council.ErrSubjectMismatch), CodeConflict},
		{fmt.Errorf("wrapped: %w", application.ErrPlanParentUnknown), CodeInvalidRequest},
		{fmt.Errorf("wrapped: %w", application.ErrPlanDependencyUnknown), CodeInvalidRequest},
		{fmt.Errorf("wrapped: %w", application.ErrWorkItemBudgetDemandInvalid), CodeInvalidRequest},
		{errors.New("council.open_request_invalid"), CodeInvalidRequest},
		{errors.New("council.resolution_required"), CodeConflict},
	}
	for _, test := range tests {
		if got := applicationFailure(test.err); got == nil || got.Code != test.code {
			t.Errorf("error=%v failure=%+v", test.err, got)
		}
	}
}

func TestTypedFailuresProduceStableCodesAndTerminalStates(t *testing.T) {
	dispatcher, api, audit := testDispatcher(t)
	tests := []struct {
		name, code, status string
		err                error
	}{
		{"invalid", CodeInvalidRequest, "rejected", &goal.DomainError{Code: goal.ErrorInvalidArgument}},
		{"forbidden", CodeForbidden, "rejected", fmt.Errorf("wrapped: %w", application.ErrForbidden)},
		{"critical_separation", CodeForbidden, "rejected", fmt.Errorf("wrapped: %w", application.ErrEffectCriticalSeparationRequired)},
		{"critical_authority", CodeForbidden, "rejected", fmt.Errorf("wrapped: %w", application.ErrEffectCriticalProjectAuthorityRequired)},
		{"forbidden_text_only", CodeInternal, "failed", fmt.Errorf("wrapped: %s", "application.forbidden")},
		{"conflict", CodeConflict, "rejected", &application.StateError{Code: application.StateConflict}},
		{"unavailable", CodeUnavailable, "failed", context.DeadlineExceeded},
	}
	for _, test := range tests {
		api.statusErr = test.err
		result := invoke(t, dispatcher, "orquesta.system.status", "request:error:"+test.name, map[string]any{}, false)
		if result.Failure == nil || result.Failure.Code != test.code {
			t.Fatalf("%s=%+v", test.name, result)
		}
		entry := audit.entries[result.AuditRef]
		terminal := entry.terminals[result.AuditRef+":outcome"]
		if terminal.Status != test.status || terminal.ErrorCode != test.code || !validTerminal(terminal, true) {
			t.Fatalf("%s terminal=%+v", test.name, terminal)
		}
	}
}

func TestConcurrentReplayAppendsOneExactTerminalWithoutRawData(t *testing.T) {
	dispatcher, api, audit := testDispatcher(t)
	const workers = 16
	results := make([]Result, workers)
	var wait sync.WaitGroup
	wait.Add(workers)
	for index := range results {
		go func(index int) {
			defer wait.Done()
			results[index] = invoke(t, dispatcher, "orquesta.goals.create", "request:concurrent", map[string]any{"statement": "build", "confirm": true}, false)
		}(index)
	}
	wait.Wait()
	for _, result := range results {
		if result.Failure != nil || result.AuditRef != results[0].AuditRef {
			t.Fatalf("result=%+v first=%+v", result, results[0])
		}
	}
	entry := audit.entries[results[0].AuditRef]
	if len(entry.terminals) != 1 || entry.record.AdmittedAt.IsZero() || api.calls["Submit"] != workers {
		t.Fatalf("terminals=%d calls=%d", len(entry.terminals), api.calls["Submit"])
	}
	terminal := entry.terminals[results[0].AuditRef+":outcome"]
	if terminal.Status != "completed" || terminal.ErrorCode != "" || !validTerminal(terminal, true) {
		t.Fatalf("terminal=%+v", terminal)
	}
	encoded, _ := json.Marshal(terminal)
	if strings.Contains(string(encoded), "\"data\"") || strings.Contains(string(encoded), "build") {
		t.Fatalf("audit retained raw output/input: %s", encoded)
	}
}

type existingCompletionAudit struct {
	*memoryAudit
	mutate bool
}

func (audit existingCompletionAudit) Complete(ctx context.Context, request AuditCompletionRequest) (AuditCompletion, error) {
	completion, err := audit.memoryAudit.Complete(ctx, request)
	completion.Created = false
	if audit.mutate {
		completion.Terminal.OutputDigest = "changed"
	}
	return completion, err
}

func TestCompletionCreatedFalseRequiresExactConcurrentTerminal(t *testing.T) {
	for _, test := range []struct {
		name, wantCode string
		mutate         bool
	}{{name: "exact"}, {name: "changed", mutate: true, wantCode: CodeConflict}} {
		audit := existingCompletionAudit{memoryAudit: newMemoryAudit(), mutate: test.mutate}
		dispatcher, err := newDispatcher(newFakeApplication(), audit, APILimits{
			MaxRequestBytes: 1024, MaxListLimit: 10, IntakeMaxQuestionRounds: 6,
		})
		if err != nil {
			t.Fatal(err)
		}
		result := invoke(t, dispatcher, "orquesta.system.status", "request:created:"+test.name, map[string]any{}, false)
		if failureCode(result.Failure) != test.wantCode {
			t.Errorf("%s=%+v", test.name, result)
		}
	}
}

func TestAuditTerminalStateInvariantRejectsImpossibleCombinations(t *testing.T) {
	base := CommandAuditTerminal{
		OutcomeRef: "command-audit:test:outcome", Status: "completed",
		OutputDigest: "digest:output",
	}
	if !validTerminal(base, false) {
		t.Fatal("valid completion rejected")
	}
	invalid := []CommandAuditTerminal{
		func() CommandAuditTerminal { value := base; value.Status = "succeeded"; return value }(),
		func() CommandAuditTerminal { value := base; value.ErrorCode = CodeConflict; return value }(),
		func() CommandAuditTerminal { value := base; value.Status = "rejected"; return value }(),
		func() CommandAuditTerminal {
			value := base
			value.Status = "rejected"
			value.ErrorCode = CodeInternal
			return value
		}(),
		func() CommandAuditTerminal {
			value := base
			value.Status = "failed"
			value.ErrorCode = CodeForbidden
			return value
		}(),
	}
	for _, terminal := range invalid {
		if validTerminal(terminal, false) {
			t.Errorf("invalid terminal accepted: %+v", terminal)
		}
	}
}

func TestOutputDigestCoversCanonicalFullResultEnvelope(t *testing.T) {
	base := Result{
		CommandID: "orquesta.system.status", CommandVersion: "1",
		RequestRef: "request:digest", AuditRef: "command-audit:digest",
		Data: json.RawMessage(`{"b":2,"a":1}`),
	}
	want := terminalFor(base, "command-audit:digest:outcome").OutputDigest
	canonicalOrder := base
	canonicalOrder.Data = json.RawMessage(`{"a":1,"b":2}`)
	if got := terminalFor(canonicalOrder, "command-audit:digest:outcome").OutputDigest; got != want {
		t.Fatalf("equivalent canonical envelopes differ: %q != %q", got, want)
	}
	mutations := map[string]func(*Result){
		"command": func(value *Result) { value.CommandID = "orquesta.goals.list" },
		"version": func(value *Result) { value.CommandVersion = "2" },
		"request": func(value *Result) { value.RequestRef = "request:changed" },
		"audit":   func(value *Result) { value.AuditRef = "command-audit:changed" },
		"data":    func(value *Result) { value.Data = json.RawMessage(`{"a":1,"b":3}`) },
		"failure": func(value *Result) { value.Data = nil; value.Failure = failure(CodeConflict) },
	}
	for name, mutate := range mutations {
		changed := base
		mutate(&changed)
		if got := terminalFor(changed, "command-audit:digest:outcome").OutputDigest; got == want || got == "" {
			t.Errorf("%s mutation digest=%q", name, got)
		}
	}
}

func TestAuditIdentityScopesRequestRefAndConflictsOnChangedFacts(t *testing.T) {
	dispatcher, api, audit := testDispatcher(t)
	basePayload, _ := json.Marshal(map[string]any{"statement": "build", "confirm": true})
	base := Invocation{
		CommandID: "orquesta.goals.create", CommandVersion: "1", RequestRef: "request:identity",
		ProjectRef: "project:test", Principal: testPrincipal(t), Payload: basePayload,
	}
	if result := dispatcher.Dispatch(context.Background(), base); result.Failure != nil {
		t.Fatalf("base=%+v", result)
	}
	changedInput := base
	changedInput.Payload, _ = json.Marshal(map[string]any{"statement": "changed", "confirm": true})
	if result := dispatcher.Dispatch(context.Background(), changedInput); result.Failure == nil || result.Failure.Code != CodeConflict || result.AuditRef != "" {
		t.Fatalf("changed input=%+v", result)
	}

	differentScope := []Invocation{base, base, base}
	differentScope[0].ProjectRef = "project:other"
	otherRef, _ := identity.NewPrincipalRef("principal:other")
	otherActor, _ := goal.NewActorRef("actor:other")
	differentScope[1].Principal, _ = identity.NewPrincipal(otherRef, otherActor, identity.PrincipalKindHuman, "local_token")
	differentScope[2].CommandID = "orquesta.system.status"
	differentScope[2].Payload = json.RawMessage(`{}`)
	for index, invocation := range differentScope {
		result := dispatcher.Dispatch(context.Background(), invocation)
		if result.Failure != nil || result.AuditRef == "" {
			t.Errorf("case %d=%+v", index, result)
		}
	}
	if len(audit.entries) != 4 || api.calls["Submit"] != 3 || api.calls["Status"] != 1 {
		t.Fatalf("entries=%d calls=%v", len(audit.entries), api.calls)
	}
}

func TestAuditIdentityConflictsOnChangedExecutionOrSchema(t *testing.T) {
	dispatcher, api, _ := testDispatcher(t)
	payload, _ := json.Marshal(map[string]any{"goal_ref": "goal:g", "message_ref": "mailbox-message:m", "recipient_work_item_ref": "work-item:w"})
	base := Invocation{
		CommandID: "orquesta.mailbox.claim", CommandVersion: "1", RequestRef: "request:execution-facts",
		ProjectRef: "project:test", Principal: testPrincipal(t), Payload: payload,
		ClaimedExecutionRef: "execution:first",
	}
	if result := dispatcher.Dispatch(context.Background(), base); result.Failure != nil {
		t.Fatalf("base=%+v", result)
	}
	changedExecution := base
	changedExecution.ClaimedExecutionRef = "execution:second"
	if result := dispatcher.Dispatch(context.Background(), changedExecution); result.Failure == nil || result.Failure.Code != CodeConflict || result.AuditRef != "" {
		t.Fatalf("changed execution=%+v", result)
	}

	status := Invocation{CommandID: "orquesta.system.status", CommandVersion: "1", RequestRef: "request:schema-facts", ProjectRef: "project:test", Principal: testPrincipal(t), Payload: json.RawMessage(`{}`)}
	if result := dispatcher.Dispatch(context.Background(), status); result.Failure != nil {
		t.Fatalf("status=%+v", result)
	}
	definition := dispatcher.byID[status.CommandID]
	definition.OutputSchema = append(append(json.RawMessage(nil), definition.OutputSchema...), ' ')
	dispatcher.byID[status.CommandID] = definition
	if result := dispatcher.Dispatch(context.Background(), status); result.Failure == nil || result.Failure.Code != CodeConflict || result.AuditRef != "" {
		t.Fatalf("changed schema=%+v", result)
	}
	if api.calls["ClaimMailbox"] != 1 || api.calls["Status"] != 1 {
		t.Fatalf("calls=%v", api.calls)
	}
}

func planWorkItem() map[string]any {
	return map[string]any{
		"key": "work:next", "objective": "next", "phase": "phase:main", "role": "role:worker",
		"dependencies": []any{}, "write_set": []any{}, "output_contract": "evidence_bundle",
	}
}

func cloneMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func TestAllCanonicalHandlersInvokeExistingApplicationUseCases(t *testing.T) {
	dispatcher, api, _ := testDispatcher(t)
	cases := canonicalPayloads()
	for index, definition := range dispatcher.Definitions() {
		result := invoke(t, dispatcher, definition.ID, "request:all:"+string(rune('a'+index)), cases[definition.ID], definition.ExecutionBound)
		if result.Failure != nil {
			t.Errorf("%s: %+v", definition.ID, result.Failure)
		}
	}
	for handler := range expectedHandlerPermissions {
		if api.calls[handler] != 1 {
			t.Errorf("%s calls=%d", handler, api.calls[handler])
		}
	}
}

func canonicalPayloads() map[string]any {
	return map[string]any{
		"orquesta.goals.create": map[string]any{"statement": "build", "confirm": true},
		"orquesta.goals.amend":  map[string]any{"source_goal_ref": "goal:g", "expected_source_revision": 1, "expected_source_spec_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "statement": "amend", "reason": "fix", "confirm": true},
		"orquesta.goals.get":    map[string]any{"goal_ref": "goal:g"}, "orquesta.goals.list": map[string]any{"limit": 1},
		"orquesta.intakes.create":            map[string]any{"intake_ref": "intake:test", "max_question_rounds": 2},
		"orquesta.intakes.get":               map[string]any{"intake_ref": "intake:test"},
		"orquesta.intakes.apply":             canonicalIntakeApplyPayload(),
		"orquesta.intakes.wizard.gaps.apply": canonicalWizardGapsPayload(),
		"orquesta.intakes.recommendations.accept": map[string]any{
			"intake_ref": "intake:test", "expected_revision": 2,
			"origin": "chat", "question_round": 1,
		},
		"orquesta.intakes.context.get": map[string]any{
			"intake_ref": "intake:test", "expected_revision": 2,
			"origin": "form", "kind": "help", "question_refs": []any{},
		},
		"orquesta.intakes.dossier.prepare":        canonicalIntakeDossierPreparePayload(),
		"orquesta.intakes.wizard.dossier.prepare": canonicalWizardDossierPreparePayload(),
		"orquesta.intakes.dossier.get":            map[string]any{"dossier_ref": "intake-dossier:test"},
		"orquesta.intakes.dossier.confirm":        map[string]any{"dossier_ref": "intake-dossier:test", "confirm": true},
		"orquesta.artifacts.read":                 map[string]any{"goal_ref": "goal:g", "artifact_ref": "artifact:a"}, "orquesta.system.status": map[string]any{},
		"orquesta.projects.memberships.grant":  map[string]any{"target_principal_ref": "principal:target", "target_actor_ref": "actor:target", "target_kind": "human", "target_method": "local_token", "role": "contributor", "expected_revision": 0},
		"orquesta.projects.memberships.revoke": map[string]any{"target_principal_ref": "principal:target", "expected_revision": 1},
		"orquesta.director.claim":              map[string]any{"goal_ref": "goal:g"}, "orquesta.director.renew": map[string]any{"goal_ref": "goal:g", "token": "token:t", "fence": 1},
		"orquesta.director.plan.propose": map[string]any{"goal_ref": "goal:g", "expected_goal_revision": 1, "expected_plan_generation": 1, "lease_token": "token:t", "lease_fence": 1, "cause": "execution_failed", "source_work_item_ref": "work-item:w", "expected_work_item_revision": 1, "source_execution_ref": "execution:e", "source_execution_attempt": 1, "reason": "retry", "plan": map[string]any{"phases": []any{}, "work_items": []any{map[string]any{"key": "work:next", "objective": "retry", "phase": "phase:main", "role": "role:worker", "dependencies": []any{}, "write_set": []any{}, "output_contract": "evidence_bundle"}}}},
		"orquesta.goals.control":         map[string]any{"operation": "pause", "target": "goal", "goal_ref": "goal:g", "expected_goal_revision": 1, "expected_plan_generation": 1, "expected_app_spec_generation": 1, "expected_spec_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "reason": "pause"},
		"orquesta.effects.decide":        map[string]any{"goal_ref": "goal:g", "intent_ref": "intent:i", "expected_intent_digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "decision": "denied", "reason": "risk"},
		"orquesta.changes.list":          map[string]any{"limit": 1}, "orquesta.changes.integrate": map[string]any{"goal_ref": "goal:g", "change_ref": "change:c", "expected_target_oid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		"orquesta.mailbox.admit":          map[string]any{"goal_ref": "goal:g", "expected_plan_generation": 1, "kind": "child_delivery", "parent_work_item_ref": "work-item:p", "child_work_item_ref": "work-item:c", "recipient_principal_ref": "principal:parent", "recipient_execution_ref": "execution:parent", "summary": "done", "artifact_refs": []string{}},
		"orquesta.mailbox.claim":          map[string]any{"goal_ref": "goal:g", "message_ref": "mailbox-message:m", "recipient_work_item_ref": "work-item:w"},
		"orquesta.mailbox.mark_delivered": map[string]any{"goal_ref": "goal:g", "message_ref": "mailbox-message:m", "recipient_work_item_ref": "work-item:w", "claim_token": "claim:t", "fence": 1},
		"orquesta.mailbox.consume":        map[string]any{"goal_ref": "goal:g", "message_ref": "mailbox-message:m", "recipient_work_item_ref": "work-item:w", "claim_token": "claim:t", "fence": 1},
		"orquesta.mailbox.get":            map[string]any{"goal_ref": "goal:g", "message_ref": "mailbox-message:m", "recipient_work_item_ref": "work-item:w"},
		"orquesta.mailbox.list":           map[string]any{"goal_ref": "goal:g", "recipient_work_item_ref": "work-item:w", "limit": 1},
		"orquesta.mailbox.acknowledge":    map[string]any{"goal_ref": "goal:g", "message_ref": "mailbox-message:m", "recipient_work_item_ref": "work-item:w", "claim_token": "claim:t", "fence": 1, "expected_goal_revision": 1, "expected_plan_generation": 1, "effect_or_rework_ref": "effect:x"},
		"orquesta.mailbox.block":          map[string]any{"goal_ref": "goal:g", "message_ref": "mailbox-message:m", "recipient_work_item_ref": "work-item:w", "claim_token": "claim:t", "fence": 1, "expected_goal_revision": 1, "expected_plan_generation": 1, "effect_or_rework_ref": "rework:x"},
		"orquesta.council.round.open":     map[string]any{"goal_ref": "goal:g", "change_ref": "change:c", "expected_goal_revision": 1, "expected_item_revision": 1, "lease_token": "director-lease:t", "lease_fence": 1},
		"orquesta.council.skip":           map[string]any{"goal_ref": "goal:g", "change_ref": "change:c", "expected_goal_revision": 1, "expected_item_revision": 1, "reason": "operator decision"},
	}
}
