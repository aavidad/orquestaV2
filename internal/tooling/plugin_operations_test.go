package tooling

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestPluginOperationsApplyExactCuratedLifecycleWithRevisionFenceAndAudit(t *testing.T) {
	catalog, context, project, actor, plugins := pluginOperationsFixture(t)
	base := NewPluginOperationState()
	install := pluginOperationRequest(catalog, context, project, actor, plugins[0], PluginOperationInstall, 0, 0)
	installed, receipt, changed, err := base.Apply(catalog, install, pluginOperationCommit(install))
	disposition, found := installed.Lookup(project, "forge.review")
	if err != nil || !changed || installed == base || !found || disposition.Status != PluginDispositionEnabled ||
		disposition.Version != "1" || disposition.PluginDigest != plugins[0].Digest || disposition.Revision != 1 ||
		disposition.CuratedCatalogID != "operations.plugins" || disposition.CuratedCatalogVersion != "1" ||
		disposition.CuratedCatalogDigest != catalog.Digest() || base.Digest() == installed.Digest() {
		t.Fatalf("install changed=%v disposition=%+v error=%v", changed, disposition, err)
	}
	if receipt.RequestRef != install.RequestRef || receipt.RequestDigest == "" ||
		receipt.ActorRef != actor.String() || receipt.ProjectRef != project.String() ||
		receipt.ReasonCode != install.ReasonCode || receipt.CurationReviewRef != "review:plugin-operations:1" ||
		receipt.CurationReviewDigest != pluginOperationRepeatedDigest("e") ||
		!receipt.RequestedAt.Equal(install.RequestedAt) || !receipt.CommittedAt.Equal(pluginOperationCommit(install)) ||
		!strings.HasPrefix(receipt.Ref, "receipt:plugin-operation:") || !strings.HasPrefix(receipt.Digest, "sha256:") {
		t.Fatalf("receipt=%+v", receipt)
	}
	for name, edit := range map[string]func(*PluginOperationReceipt){
		"review": func(value *PluginOperationReceipt) { value.CurationReviewDigest = pluginOperationRepeatedDigest("f") },
		"catalog": func(value *PluginOperationReceipt) {
			value.After.CuratedCatalogDigest = pluginOperationRepeatedDigest("a")
		},
		"actor":  func(value *PluginOperationReceipt) { value.ActorRef = "actor:other" },
		"reason": func(value *PluginOperationReceipt) { value.ReasonCode = "plugin.install.retry" },
		"time":   func(value *PluginOperationReceipt) { value.RequestedAt = value.RequestedAt.Add(time.Nanosecond) },
	} {
		t.Run("receipt binds "+name, func(t *testing.T) {
			altered := receipt
			edit(&altered)
			if pluginOperationReceiptDigest(altered) == receipt.Digest {
				t.Fatalf("receipt digest omitted %s", name)
			}
		})
	}

	upgrade := pluginOperationRequest(catalog, context, project, actor, plugins[1], PluginOperationUpgrade, 1, time.Minute)
	upgraded, _, changed, err := installed.Apply(catalog, upgrade, pluginOperationCommit(upgrade))
	if err != nil || !changed {
		t.Fatalf("upgrade changed=%v error=%v", changed, err)
	}
	rotated := pluginOperationCatalogVersion(t, catalog, catalog.plugins, "2", []CuratedPluginSelection{{
		ID: plugins[0].Spec.ID, Version: plugins[0].Spec.Version, PluginDigest: plugins[0].Digest,
	}})
	disable := pluginOperationRequest(rotated, context, project, actor, plugins[1], PluginOperationDisable, 2, 2*time.Minute)
	disabled, _, changed, err := upgraded.Apply(rotated, disable, pluginOperationCommit(disable))
	if err != nil || !changed {
		t.Fatalf("disable changed=%v error=%v", changed, err)
	}
	remove := pluginOperationRequest(rotated, context, project, actor, plugins[1], PluginOperationRemove, 3, 3*time.Minute)
	removed, removeReceipt, changed, err := disabled.Apply(rotated, remove, pluginOperationCommit(remove))
	if err != nil || !changed || removeReceipt.Before.Status != PluginDispositionDisabled ||
		removeReceipt.After.Status != PluginDispositionRemoved || removeReceipt.After.Revision != 4 {
		t.Fatalf("remove changed=%v receipt=%+v error=%v", changed, removeReceipt, err)
	}

	mutatedPlugins, err := NewPluginCatalog(catalog.tools, catalog.skills,
		pluginOperationSpec("2", pluginOperationRepeatedDigest("4")))
	pluginOperationNoError(t, err)
	mutatedV2, _ := mutatedPlugins.Lookup("forge.review", "2")
	mutatedCatalog := pluginOperationCatalogVersion(t, catalog, mutatedPlugins, "3", []CuratedPluginSelection{{
		ID: mutatedV2.Spec.ID, Version: mutatedV2.Spec.Version, PluginDigest: mutatedV2.Digest,
	}})
	mutated := pluginOperationRequest(mutatedCatalog, context, project, actor, mutatedV2, PluginOperationInstall, 4, 4*time.Minute)
	mutated.RequestRef, mutated.IdempotencyKey = "request:plugin:mutated", "idempotency:plugin:mutated"
	pluginOperationRejected(t, removed, mutatedCatalog, mutated, pluginOperationCommit(mutated), ErrorPluginOperationTransitionDenied)

	reinstallCatalog := pluginOperationCatalogVersion(t, catalog, catalog.plugins, "3", []CuratedPluginSelection{{
		ID: plugins[1].Spec.ID, Version: plugins[1].Spec.Version, PluginDigest: plugins[1].Digest,
	}})
	reinstall := pluginOperationRequest(reinstallCatalog, context, project, actor, plugins[1], PluginOperationInstall, 4, 5*time.Minute)
	reinstall.RequestRef, reinstall.IdempotencyKey = "request:plugin:reinstall", "idempotency:plugin:reinstall"
	reinstalled, reinstallReceipt, changed, err := removed.Apply(reinstallCatalog, reinstall, pluginOperationCommit(reinstall))
	if err != nil || !changed || reinstallReceipt.Before.Status != PluginDispositionRemoved ||
		reinstallReceipt.After.Status != PluginDispositionEnabled || reinstallReceipt.After.Revision != 5 ||
		reinstalled.Digest() == removed.Digest() {
		t.Fatalf("reinstall changed=%v receipt=%+v error=%v", changed, reinstallReceipt, err)
	}
	replayed, replayReceipt, changed, err := reinstalled.Apply(nil, install, time.Time{})
	if err != nil || changed || replayed.Digest() != reinstalled.Digest() || replayReceipt != receipt {
		t.Fatalf("historical replay changed=%v receipt=%+v error=%v", changed, replayReceipt, err)
	}
	conflict := install
	conflict.ReasonCode = "plugin.install.changed"
	if _, _, _, err := reinstalled.Apply(nil, conflict, time.Time{}); ErrorCode(err) != ErrorPluginOperationIdempotencyConflict {
		t.Fatalf("divergent replay error=%v", err)
	}
	conflict = install
	conflict.RequestedAt = conflict.RequestedAt.Add(time.Nanosecond)
	if _, _, _, err := reinstalled.Apply(nil, conflict, time.Time{}); ErrorCode(err) != ErrorPluginOperationIdempotencyConflict {
		t.Fatalf("divergent replay time error=%v", err)
	}
}

func TestPluginOperationsConcurrentProposalsAndDefaultDeny(t *testing.T) {
	catalog, context, project, actor, plugins := pluginOperationsFixture(t)
	base := NewPluginOperationState()
	requests := []PluginOperationRequest{
		pluginOperationRequest(catalog, context, project, actor, plugins[0], PluginOperationInstall, 0, 0),
		pluginOperationRequest(catalog, context, project, actor, plugins[1], PluginOperationInstall, 0, time.Minute),
	}
	type result struct {
		state   *PluginOperationState
		receipt PluginOperationReceipt
		changed bool
		err     error
	}
	start, results := make(chan struct{}), make(chan result, len(requests))
	for _, candidate := range requests {
		go func(request PluginOperationRequest) {
			<-start
			next, receipt, changed, err := base.Apply(catalog, request, pluginOperationCommit(request))
			results <- result{next, receipt, changed, err}
		}(candidate)
	}
	close(start)
	left, right := <-results, <-results
	if left.err != nil || right.err != nil || !left.changed || !right.changed ||
		left.receipt.After.Revision != 1 || right.receipt.After.Revision != 1 ||
		left.state.Digest() == right.state.Digest() || base.Digest() == left.state.Digest() {
		t.Fatalf("left=%+v right=%+v", left, right)
	}
	installed, _, _, err := base.Apply(catalog, requests[0], pluginOperationCommit(requests[0]))
	pluginOperationNoError(t, err)
	pluginOperationRejected(t, installed, catalog, requests[1], pluginOperationCommit(requests[1]), ErrorPluginOperationRevisionConflict)
	tests := []struct {
		name   string
		state  *PluginOperationState
		edit   func(*PluginOperationRequest)
		commit func(PluginOperationRequest) time.Time
		code   string
	}{
		{"catalog digest", base, func(value *PluginOperationRequest) { value.CuratedCatalogDigest = pluginOperationRepeatedDigest("0") }, nil, ErrorPluginOperationCatalogDenied},
		{"zero actor", base, func(value *PluginOperationRequest) { value.ActorRef = goal.ActorRef{} }, nil, ErrorPluginOperationInvalid},
		{"wrong capability", base, func(value *PluginOperationRequest) {
			capability, _ := goal.NewCapabilityRef("TLS-12")
			value.Context, _ = NewCuratedContext(capability, value.Context.scope)
		}, nil, ErrorPluginOperationCatalogDenied},
		{"stale revision", installed, func(value *PluginOperationRequest) { value.ExpectedRevision = 0 }, nil, ErrorPluginOperationRevisionConflict},
		{"remove enabled", installed, func(value *PluginOperationRequest) {
			value.Operation, value.ExpectedRevision, value.ReasonCode = PluginOperationRemove, 1, "plugin.remove.requested"
		}, nil, ErrorPluginOperationTransitionDenied},
		{"upgrade in place", installed, func(value *PluginOperationRequest) {
			value.Operation, value.ExpectedRevision, value.ReasonCode = PluginOperationUpgrade, 1, "plugin.upgrade.requested"
		}, nil, ErrorPluginOperationTransitionDenied},
		{"uncurated version", installed, func(value *PluginOperationRequest) {
			value.Operation, value.ExpectedRevision, value.Version = PluginOperationUpgrade, 1, "3"
			value.PluginDigest, value.ReasonCode = plugins[2].Digest, "plugin.upgrade.requested"
		}, nil, ErrorPluginOperationCatalogDenied},
		{"commit before state", installed, func(value *PluginOperationRequest) {
			value.Operation, value.ExpectedRevision, value.ReasonCode = PluginOperationDisable, 1, "plugin.disable.requested"
			value.RequestedAt = value.RequestedAt.Add(-2 * time.Minute)
		}, nil, ErrorPluginOperationTransitionDenied},
		{"commit before request", installed, func(value *PluginOperationRequest) {},
			func(value PluginOperationRequest) time.Time { return value.RequestedAt.Add(-time.Nanosecond) }, ErrorPluginOperationInvalid},
		{"zero commit", installed, func(value *PluginOperationRequest) {},
			func(PluginOperationRequest) time.Time { return time.Time{} }, ErrorPluginOperationInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := requests[0]
			request.RequestRef, request.IdempotencyKey = "request:negative:"+test.name, "idempotency:negative:"+test.name
			test.edit(&request)
			committedAt := pluginOperationCommit(request)
			if test.commit != nil {
				committedAt = test.commit(request)
			}
			pluginOperationRejected(t, test.state, catalog, request, committedAt, test.code)
		})
	}

	projectBeta, _ := goal.NewProjectRef("project:beta")
	betaScope, _ := NewSkillScopeContext("orquesta", projectBeta, identity.RoleOperator, goal.GoalRef{})
	capability, _ := goal.NewCapabilityRef(PluginOperationsCapability)
	betaContext, _ := NewCuratedContext(capability, betaScope)
	crossScope := pluginOperationRequest(catalog, betaContext, projectBeta, actor, plugins[0], PluginOperationInstall, 0, 0)
	pluginOperationRejected(t, base, catalog, crossScope, pluginOperationCommit(crossScope), ErrorPluginOperationCatalogDenied)
}

func TestPluginOperationsRejectInvalidFactsAndPhysicalClaims(t *testing.T) {
	catalog, context, project, actor, plugins := pluginOperationsFixture(t)
	valid := pluginOperationRequest(catalog, context, project, actor, plugins[0], PluginOperationInstall, 0, 0)
	invalid := string([]byte{0xff})
	for name, edit := range map[string]func(*PluginOperationRequest){
		"request": func(value *PluginOperationRequest) { value.RequestRef = "request:" + invalid },
		"context": func(value *PluginOperationRequest) { value.Context.scope.productID = invalid },
		"reason":  func(value *PluginOperationRequest) { value.ReasonCode = "plugin." + invalid },
	} {
		t.Run(name, func(t *testing.T) {
			request := valid
			edit(&request)
			if pluginOperationRequestDigest(request) != "" {
				t.Fatal("invalid UTF-8 request was digested")
			}
			pluginOperationRejected(t, NewPluginOperationState(), catalog, request, pluginOperationCommit(request), ErrorPluginOperationInvalid)
		})
	}
	_, receipt, _, err := NewPluginOperationState().Apply(catalog, valid, pluginOperationCommit(valid))
	pluginOperationNoError(t, err)
	receipt.CurationReviewRef = invalid
	if pluginOperationReceiptDigest(receipt) != "" {
		t.Fatal("invalid UTF-8 receipt was digested")
	}
	encoded, err := json.Marshal(receipt)
	pluginOperationNoError(t, err)
	for _, prohibited := range []string{"download", "filesystem", "path", "endpoint", "signature", "effect_applied", "authorization_allowed"} {
		if strings.Contains(string(encoded), prohibited) {
			t.Fatalf("receipt claims unsupported fact %q", prohibited)
		}
	}
	for _, field := range []string{"Downloaded", "InstalledBytes", "FilesystemPath", "EffectApplied", "Authorized"} {
		if _, exists := reflect.TypeOf(PluginOperationReceipt{}).FieldByName(field); exists {
			t.Fatalf("receipt exposes unsupported fact %q", field)
		}
	}
}

func pluginOperationsFixture(t *testing.T) (*CuratedCatalog, CuratedContext, goal.ProjectRef, goal.ActorRef, []PluginRegistration) {
	t.Helper()
	tools, err := NewRegistry()
	pluginOperationNoError(t, err)
	skills, err := NewSkillRegistry(tools)
	pluginOperationNoError(t, err)
	releases, err := NewSkillReleaseCatalog(skills, nil, nil, nil)
	pluginOperationNoError(t, err)
	plugins, err := NewPluginCatalog(tools, releases,
		pluginOperationSpec("1", pluginOperationRepeatedDigest("1")),
		pluginOperationSpec("2", pluginOperationRepeatedDigest("2")),
		pluginOperationSpec("3", pluginOperationRepeatedDigest("3")))
	pluginOperationNoError(t, err)
	registrations := make([]PluginRegistration, 3)
	for index, version := range []string{"1", "2", "3"} {
		registrations[index], _ = plugins.Lookup("forge.review", version)
	}
	project, _ := goal.NewProjectRef("project:alpha")
	actor, _ := goal.NewActorRef("actor:operator")
	catalog, err := NewCuratedCatalog(tools, releases, plugins, CuratedCatalogSpec{
		ID: "operations.plugins", Version: "1", DescriptionKey: "curation.operations.plugins.description",
		ReviewRef: "review:plugin-operations:1", ReviewDigest: pluginOperationRepeatedDigest("e"),
		Capabilities: []string{PluginOperationsCapability},
		Scopes:       []SkillScope{{Kind: SkillScopeProject, ProjectRef: project.String()}},
		Plugins: []CuratedPluginSelection{
			{ID: "forge.review", Version: "1", PluginDigest: registrations[0].Digest},
			{ID: "forge.review", Version: "2", PluginDigest: registrations[1].Digest},
		},
	})
	pluginOperationNoError(t, err)
	scope, _ := NewSkillScopeContext("orquesta", project, identity.RoleOperator, goal.GoalRef{})
	capability, _ := goal.NewCapabilityRef(PluginOperationsCapability)
	context, err := NewCuratedContext(capability, scope)
	pluginOperationNoError(t, err)
	return catalog, context, project, actor, registrations
}

func pluginOperationSpec(version, contractDigest string) PluginSpec {
	return PluginSpec{
		ID: "forge.review", Version: version, DescriptionKey: "plugin.forge.review.description",
		Connectors: []PluginConnectorSpec{{
			ID: "forge.events", Version: version, ContractRef: "artifact:" + contractDigest,
			ContractDigest: contractDigest, Permissions: []identity.Permission{identity.PermissionArtifactsRead},
		}},
		Permissions: []identity.Permission{identity.PermissionArtifactsRead},
	}
}

func pluginOperationRequest(catalog *CuratedCatalog, context CuratedContext, project goal.ProjectRef,
	actor goal.ActorRef, plugin PluginRegistration, operation PluginOperation, expected uint64,
	offset time.Duration,
) PluginOperationRequest {
	return PluginOperationRequest{
		RequestRef:     "request:plugin:" + string(operation) + ":" + plugin.Spec.Version,
		IdempotencyKey: "idempotency:plugin:" + string(operation) + ":" + plugin.Spec.Version,
		ActorRef:       actor, ProjectRef: project, Context: context, CuratedCatalogDigest: catalog.Digest(),
		Operation: operation, PluginID: plugin.Spec.ID, Version: plugin.Spec.Version,
		PluginDigest: plugin.Digest, ExpectedRevision: expected,
		ReasonCode:  "plugin." + string(operation) + ".requested",
		RequestedAt: time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC).Add(offset),
	}
}

func pluginOperationCatalogVersion(t *testing.T, base *CuratedCatalog, plugins *PluginCatalog,
	version string, selections []CuratedPluginSelection,
) *CuratedCatalog {
	t.Helper()
	spec := base.Registration().Spec
	spec.Version, spec.Plugins = version, append([]CuratedPluginSelection(nil), selections...)
	catalog, err := NewCuratedCatalog(base.tools, base.skills, plugins, spec)
	pluginOperationNoError(t, err)
	return catalog
}

func pluginOperationRejected(t *testing.T, state *PluginOperationState, catalog *CuratedCatalog,
	request PluginOperationRequest, committedAt time.Time, code string,
) {
	t.Helper()
	next, receipt, changed, err := state.Apply(catalog, request, committedAt)
	if changed || receipt.Ref != "" || next.Digest() != state.Digest() || ErrorCode(err) != code {
		t.Fatalf("changed=%v receipt=%+v error=%v code=%q", changed, receipt, err, ErrorCode(err))
	}
}

func pluginOperationRepeatedDigest(fill string) string { return "sha256:" + strings.Repeat(fill, 64) }
func pluginOperationCommit(request PluginOperationRequest) time.Time {
	return request.RequestedAt.Add(30 * time.Second)
}
func pluginOperationNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
