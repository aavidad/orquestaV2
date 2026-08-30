package acceptance_test

import (
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

func TestV26TLS11PluginOperationsAreCuratedRevisionFencedIdempotentAndAudited(t *testing.T) {
	catalog, context, project, actor, versions := v26PluginOperationCatalog(t, "1", "1", "2")
	base := tooling.NewPluginOperationState()
	install := v26PluginOperationRequest(catalog, context, project, actor, versions[0], tooling.PluginOperationInstall, 0, 0)
	installed, receipt, changed, err := base.Apply(catalog, install, v26PluginOperationCommit(install))
	if err != nil || !changed || installed == base || receipt.Operation != tooling.PluginOperationInstall ||
		receipt.RequestRef != install.RequestRef || receipt.RequestDigest == "" ||
		receipt.ActorRef != actor.String() || receipt.ProjectRef != project.String() ||
		receipt.ReasonCode != install.ReasonCode || receipt.CurationReviewRef != "review:v26:tls11" ||
		receipt.CurationReviewDigest != "sha256:"+strings.Repeat("e", 64) ||
		receipt.After.Status != tooling.PluginDispositionEnabled || receipt.After.Revision != 1 ||
		receipt.After.CuratedCatalogDigest != catalog.Digest() ||
		!receipt.RequestedAt.Equal(install.RequestedAt) || !receipt.CommittedAt.Equal(v26PluginOperationCommit(install)) {
		t.Fatalf("install changed=%v receipt=%+v error=%v", changed, receipt, err)
	}

	upgrade := v26PluginOperationRequest(catalog, context, project, actor, versions[1], tooling.PluginOperationUpgrade, 1, time.Minute)
	upgraded, _, changed, err := installed.Apply(catalog, upgrade, v26PluginOperationCommit(upgrade))
	if err != nil || !changed {
		t.Fatalf("upgrade changed=%v error=%v", changed, err)
	}
	rotated, _, _, _, rotatedVersions := v26PluginOperationCatalog(t, "2", "1")
	disable := v26PluginOperationRequest(rotated, context, project, actor, versions[1], tooling.PluginOperationDisable, 2, 2*time.Minute)
	disabled, _, changed, err := upgraded.Apply(rotated, disable, v26PluginOperationCommit(disable))
	if err != nil || !changed {
		t.Fatalf("disable changed=%v error=%v", changed, err)
	}
	remove := v26PluginOperationRequest(rotated, context, project, actor, versions[1], tooling.PluginOperationRemove, 3, 3*time.Minute)
	removed, removeReceipt, changed, err := disabled.Apply(rotated, remove, v26PluginOperationCommit(remove))
	disposition, found := removed.Lookup(project, "forge.review")
	if err != nil || !changed || !found || removeReceipt.After.Status != tooling.PluginDispositionRemoved ||
		disposition.Status != tooling.PluginDispositionRemoved || disposition.Revision != 4 {
		t.Fatalf("remove changed=%v receipt=%+v disposition=%+v error=%v", changed, removeReceipt, disposition, err)
	}

	replayed, replayReceipt, changed, err := removed.Apply(nil, install, time.Time{})
	if err != nil || changed || replayed.Digest() != removed.Digest() || replayReceipt != receipt {
		t.Fatalf("historical replay changed=%v receipt=%+v error=%v", changed, replayReceipt, err)
	}
	conflict := install
	conflict.ActorRef, _ = goal.NewActorRef("actor:v26-other")
	if _, _, _, err := removed.Apply(nil, conflict, time.Time{}); tooling.ErrorCode(err) != tooling.ErrorPluginOperationIdempotencyConflict {
		t.Fatalf("divergent replay error=%v", err)
	}
	stale := v26PluginOperationRequest(rotated, context, project, actor, versions[1], tooling.PluginOperationDisable, 1, 4*time.Minute)
	stale.RequestRef, stale.IdempotencyKey = "request:v26:stale", "idempotency:v26:stale"
	v26PluginOperationRejected(t, removed, rotated, stale, tooling.ErrorPluginOperationRevisionConflict)
	downgrade := v26PluginOperationRequest(rotated, context, project, actor, rotatedVersions[0], tooling.PluginOperationInstall, 4, 5*time.Minute)
	downgrade.RequestRef, downgrade.IdempotencyKey = "request:v26:downgrade", "idempotency:v26:downgrade"
	v26PluginOperationRejected(t, removed, rotated, downgrade, tooling.ErrorPluginOperationTransitionDenied)
	mutatedCatalog, _, _, _, mutatedVersions := v26PluginOperationCatalogWithV2Fill(t, "3", "4", "2")
	mutated := v26PluginOperationRequest(mutatedCatalog, context, project, actor, mutatedVersions[1], tooling.PluginOperationInstall, 4, 6*time.Minute)
	mutated.RequestRef, mutated.IdempotencyKey = "request:v26:mutated", "idempotency:v26:mutated"
	v26PluginOperationRejected(t, removed, mutatedCatalog, mutated, tooling.ErrorPluginOperationTransitionDenied)

	reinstallCatalog, _, _, _, reinstallVersions := v26PluginOperationCatalog(t, "3", "2")
	reinstall := v26PluginOperationRequest(reinstallCatalog, context, project, actor, reinstallVersions[1], tooling.PluginOperationInstall, 4, 7*time.Minute)
	reinstalled, reinstallReceipt, changed, err := removed.Apply(reinstallCatalog, reinstall, v26PluginOperationCommit(reinstall))
	if err != nil || !changed || reinstallReceipt.Before.Status != tooling.PluginDispositionRemoved ||
		reinstallReceipt.After.Status != tooling.PluginDispositionEnabled || reinstallReceipt.After.Revision != 5 ||
		reinstalled.Digest() == removed.Digest() {
		t.Fatalf("reinstall changed=%v receipt=%+v error=%v", changed, reinstallReceipt, err)
	}
}

func v26PluginOperationCatalog(t *testing.T, catalogVersion string, selected ...string) (
	*tooling.CuratedCatalog, tooling.CuratedContext, goal.ProjectRef, goal.ActorRef, []tooling.PluginRegistration,
) {
	return v26PluginOperationCatalogWithV2Fill(t, catalogVersion, "2", selected...)
}

func v26PluginOperationCatalogWithV2Fill(t *testing.T, catalogVersion, v2Fill string, selected ...string) (
	*tooling.CuratedCatalog, tooling.CuratedContext, goal.ProjectRef, goal.ActorRef, []tooling.PluginRegistration,
) {
	t.Helper()
	tools, err := tooling.NewRegistry()
	v26PluginNoError(t, err)
	skills, err := tooling.NewSkillRegistry(tools)
	v26PluginNoError(t, err)
	releases, err := tooling.NewSkillReleaseCatalog(skills, nil, nil, nil)
	v26PluginNoError(t, err)
	plugins, err := tooling.NewPluginCatalog(tools, releases,
		v26PluginOperationSpec("1", "1"), v26PluginOperationSpec("2", v2Fill))
	v26PluginNoError(t, err)
	versions := make([]tooling.PluginRegistration, 2)
	for index, version := range []string{"1", "2"} {
		versions[index], _ = plugins.Lookup("forge.review", version)
	}
	project, _ := goal.NewProjectRef("project:v26-tls11")
	actor, _ := goal.NewActorRef("actor:v26-operator")
	selections := make([]tooling.CuratedPluginSelection, 0, len(selected))
	for _, version := range selected {
		registration, _ := plugins.Lookup("forge.review", version)
		selections = append(selections, tooling.CuratedPluginSelection{
			ID: registration.Spec.ID, Version: version, PluginDigest: registration.Digest,
		})
	}
	catalog, err := tooling.NewCuratedCatalog(tools, releases, plugins, tooling.CuratedCatalogSpec{
		ID: "v26.operations", Version: catalogVersion, DescriptionKey: "curation.v26.operations.description",
		ReviewRef: "review:v26:tls11", ReviewDigest: "sha256:" + strings.Repeat("e", 64),
		Capabilities: []string{tooling.PluginOperationsCapability},
		Scopes:       []tooling.SkillScope{{Kind: tooling.SkillScopeProject, ProjectRef: project.String()}},
		Plugins:      selections,
	})
	v26PluginNoError(t, err)
	scope, _ := tooling.NewSkillScopeContext("orquesta", project, identity.RoleOperator, goal.GoalRef{})
	capability, _ := goal.NewCapabilityRef(tooling.PluginOperationsCapability)
	context, err := tooling.NewCuratedContext(capability, scope)
	v26PluginNoError(t, err)
	return catalog, context, project, actor, versions
}

func v26PluginOperationSpec(version, fill string) tooling.PluginSpec {
	digest := "sha256:" + strings.Repeat(fill, 64)
	return tooling.PluginSpec{
		ID: "forge.review", Version: version, DescriptionKey: "plugin.forge.review.description",
		Connectors: []tooling.PluginConnectorSpec{{
			ID: "forge.events", Version: version, ContractRef: "artifact:" + digest,
			ContractDigest: digest, Permissions: []identity.Permission{identity.PermissionArtifactsRead},
		}},
		Permissions: []identity.Permission{identity.PermissionArtifactsRead},
	}
}

func v26PluginOperationRequest(catalog *tooling.CuratedCatalog, context tooling.CuratedContext,
	project goal.ProjectRef, actor goal.ActorRef, plugin tooling.PluginRegistration,
	operation tooling.PluginOperation, expected uint64, offset time.Duration,
) tooling.PluginOperationRequest {
	return tooling.PluginOperationRequest{
		RequestRef:     "request:v26:" + string(operation) + ":" + plugin.Spec.Version,
		IdempotencyKey: "idempotency:v26:" + string(operation) + ":" + plugin.Spec.Version,
		ActorRef:       actor, ProjectRef: project, Context: context, CuratedCatalogDigest: catalog.Digest(),
		Operation: operation, PluginID: plugin.Spec.ID, Version: plugin.Spec.Version,
		PluginDigest: plugin.Digest, ExpectedRevision: expected,
		ReasonCode:  "plugin." + string(operation) + ".requested",
		RequestedAt: time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC).Add(offset),
	}
}

func v26PluginOperationCommit(request tooling.PluginOperationRequest) time.Time {
	return request.RequestedAt.Add(30 * time.Second)
}

func v26PluginOperationRejected(t *testing.T, state *tooling.PluginOperationState,
	catalog *tooling.CuratedCatalog, request tooling.PluginOperationRequest, code string,
) {
	t.Helper()
	next, receipt, changed, err := state.Apply(catalog, request, v26PluginOperationCommit(request))
	if changed || receipt.Ref != "" || next.Digest() != state.Digest() || tooling.ErrorCode(err) != code {
		t.Fatalf("changed=%v receipt=%+v error=%v", changed, receipt, err)
	}
}

func v26PluginNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
