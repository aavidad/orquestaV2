package orquestagoal

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
)

// GoalWorkspaceBindingV0 is provider-neutral: Codex, Claude and Gemini must
// execute the same goal inside the same physically isolated project worktree.
type GoalWorkspaceBindingV0 struct {
	IntentManifestRef               string   `json:"intent_manifest_ref,omitempty"`
	IntentManifestSHA256            string   `json:"intent_manifest_sha256,omitempty"`
	WorkspaceAuthoritySchemaVersion string   `json:"workspace_authority_schema_version,omitempty"`
	WorkspaceRef                    string   `json:"workspace_ref"`
	ProviderRef                     string   `json:"provider_ref,omitempty"`
	RuntimeGenerationRef            string   `json:"runtime_generation_ref,omitempty"`
	ProjectWorkDir                  string   `json:"project_work_dir"`
	EvidenceRefs                    []string `json:"evidence_refs,omitempty"`
}

// GoalExecutionAuthorityV0 is the immutable causal identity shared by launch,
// durable process state, observation and stop. Provider adapters may project
// it into their public DTOs, but must not manufacture a missing field after a
// process effect has happened.
type GoalExecutionAuthorityV0 struct {
	GoalRef                         string `json:"goal_ref"`
	IntentManifestRef               string `json:"intent_manifest_ref,omitempty"`
	IntentManifestSHA256            string `json:"intent_manifest_sha256,omitempty"`
	WorkspaceAuthoritySchemaVersion string `json:"workspace_authority_schema_version,omitempty"`
	WorkspaceRef                    string `json:"workspace_ref,omitempty"`
	ProviderRef                     string `json:"provider_ref,omitempty"`
	RuntimeGenerationRef            string `json:"runtime_generation_ref,omitempty"`
}

func NormalizeGoalExecutionAuthorityV0(authority GoalExecutionAuthorityV0) GoalExecutionAuthorityV0 {
	authority.GoalRef = strings.TrimSpace(authority.GoalRef)
	authority.IntentManifestRef = strings.TrimSpace(authority.IntentManifestRef)
	authority.IntentManifestSHA256 = strings.TrimSpace(authority.IntentManifestSHA256)
	authority.WorkspaceAuthoritySchemaVersion = strings.TrimSpace(authority.WorkspaceAuthoritySchemaVersion)
	authority.WorkspaceRef = strings.TrimSpace(authority.WorkspaceRef)
	authority.ProviderRef = strings.TrimSpace(authority.ProviderRef)
	authority.RuntimeGenerationRef = strings.TrimSpace(authority.RuntimeGenerationRef)
	return authority
}

func GoalExecutionAuthorityForProviderV0(spec GoalWorkSpecV0, providerRef string) GoalExecutionAuthorityV0 {
	spec = NormalizeGoalWorkSpecV0(spec)
	providerRef = strings.TrimSpace(providerRef)
	authority := GoalExecutionAuthorityV0{
		GoalRef:                         spec.GoalRef,
		IntentManifestRef:               spec.IntentManifestRef,
		IntentManifestSHA256:            spec.IntentManifestSHA256,
		WorkspaceAuthoritySchemaVersion: GoalWorkspaceAuthoritySchemaV0,
		WorkspaceRef:                    GoalWorkspaceRefForGoalV0(spec.GoalRef),
		ProviderRef:                     providerRef,
	}
	authority.RuntimeGenerationRef = GoalRuntimeGenerationRefV0(authority)
	return NormalizeGoalExecutionAuthorityV0(authority)
}

// GoalRuntimeGenerationRefV0 is stable for a single Goal/provider/manifest
// authority. A rework Goal gets a different GoalRef and therefore a different
// generation without relying on process-local counters.
func GoalRuntimeGenerationRefV0(authority GoalExecutionAuthorityV0) string {
	authority = NormalizeGoalExecutionAuthorityV0(authority)
	if authority.GoalRef == "" || authority.ProviderRef == "" || authority.WorkspaceRef == "" {
		return ""
	}
	payload := strings.Join([]string{
		authority.GoalRef,
		authority.IntentManifestRef,
		authority.IntentManifestSHA256,
		authority.WorkspaceAuthoritySchemaVersion,
		authority.WorkspaceRef,
		authority.ProviderRef,
	}, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("runtime-generation-ref-%x", sum[:])
}

func GoalExecutionAuthorityFromReceiptV0(receipt GoalLaunchReceiptV0) GoalExecutionAuthorityV0 {
	receipt = NormalizeGoalLaunchReceiptV0(receipt)
	return NormalizeGoalExecutionAuthorityV0(GoalExecutionAuthorityV0{
		GoalRef: receipt.GoalRef, IntentManifestRef: receipt.IntentManifestRef,
		IntentManifestSHA256:            receipt.IntentManifestSHA256,
		WorkspaceAuthoritySchemaVersion: receipt.WorkspaceAuthoritySchemaVersion,
		WorkspaceRef:                    receipt.WorkspaceRef, ProviderRef: receipt.ProviderRef,
		RuntimeGenerationRef: receipt.RuntimeGenerationRef,
	})
}

func GoalExecutionAuthorityFromObservationV0(request GoalObservationRequestV0) GoalExecutionAuthorityV0 {
	request = NormalizeGoalObservationRequestV0(request)
	return NormalizeGoalExecutionAuthorityV0(GoalExecutionAuthorityV0{
		GoalRef: request.GoalRef, IntentManifestRef: request.IntentManifestRef,
		IntentManifestSHA256:            request.IntentManifestSHA256,
		WorkspaceAuthoritySchemaVersion: request.WorkspaceAuthoritySchemaVersion,
		WorkspaceRef:                    request.WorkspaceRef, ProviderRef: request.ProviderRef,
		RuntimeGenerationRef: request.RuntimeGenerationRef,
	})
}

func GoalObservationRequestForAuthorityV0(authority GoalExecutionAuthorityV0, externalGoalRef string) GoalObservationRequestV0 {
	authority = NormalizeGoalExecutionAuthorityV0(authority)
	return NormalizeGoalObservationRequestV0(GoalObservationRequestV0{
		GoalRef: authority.GoalRef, ExternalGoalRef: externalGoalRef,
		IntentManifestRef:               authority.IntentManifestRef,
		IntentManifestSHA256:            authority.IntentManifestSHA256,
		WorkspaceAuthoritySchemaVersion: authority.WorkspaceAuthoritySchemaVersion,
		WorkspaceRef:                    authority.WorkspaceRef, ProviderRef: authority.ProviderRef,
		RuntimeGenerationRef: authority.RuntimeGenerationRef,
	})
}

func ApplyGoalExecutionAuthorityToReceiptV0(receipt GoalLaunchReceiptV0, authority GoalExecutionAuthorityV0) GoalLaunchReceiptV0 {
	authority = NormalizeGoalExecutionAuthorityV0(authority)
	receipt.GoalRef = authority.GoalRef
	receipt.IntentManifestRef = authority.IntentManifestRef
	receipt.IntentManifestSHA256 = authority.IntentManifestSHA256
	receipt.WorkspaceAuthoritySchemaVersion = authority.WorkspaceAuthoritySchemaVersion
	receipt.WorkspaceRef = authority.WorkspaceRef
	receipt.ProviderRef = authority.ProviderRef
	receipt.RuntimeGenerationRef = authority.RuntimeGenerationRef
	return NormalizeGoalLaunchReceiptV0(receipt)
}

func GoalExecutionAuthorityIssuesV0(authority GoalExecutionAuthorityV0, allowLegacy bool) []GoalWorkIssueV0 {
	authority = NormalizeGoalExecutionAuthorityV0(authority)
	var issues []GoalWorkIssueV0
	validateRequiredGoalRefV0(&issues, "goal_ref", authority.GoalRef)
	validateGoalIntentManifestIdentityV0(&issues, authority.IntentManifestRef, authority.IntentManifestSHA256)
	issues = append(issues, ValidateGoalWorkspaceAuthorityV0(
		authority.GoalRef, authority.WorkspaceAuthoritySchemaVersion,
		authority.WorkspaceRef, authority.ProviderRef,
		authority.RuntimeGenerationRef, allowLegacy,
	)...)
	return issues
}

func GoalExecutionAuthorityMatchesV0(expected, actual GoalExecutionAuthorityV0) bool {
	return NormalizeGoalExecutionAuthorityV0(expected) == NormalizeGoalExecutionAuthorityV0(actual)
}

// GoalWorkspaceRefForGoalV0 derives the stable opaque workspace identity. It
// deliberately excludes the provider so one Goal keeps one workspace across
// provider adapters.
func GoalWorkspaceRefForGoalV0(goalRef string) string {
	goalRef = strings.TrimSpace(goalRef)
	if goalRef == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(goalRef))
	return fmt.Sprintf("workspace-ref-%x", sum[:])
}

func NormalizeGoalWorkspaceBindingV0(binding GoalWorkspaceBindingV0) GoalWorkspaceBindingV0 {
	binding.IntentManifestRef = strings.TrimSpace(binding.IntentManifestRef)
	binding.IntentManifestSHA256 = strings.TrimSpace(binding.IntentManifestSHA256)
	binding.WorkspaceAuthoritySchemaVersion = strings.TrimSpace(binding.WorkspaceAuthoritySchemaVersion)
	binding.WorkspaceRef = strings.TrimSpace(binding.WorkspaceRef)
	binding.ProviderRef = strings.TrimSpace(binding.ProviderRef)
	binding.RuntimeGenerationRef = strings.TrimSpace(binding.RuntimeGenerationRef)
	binding.ProjectWorkDir = strings.TrimSpace(binding.ProjectWorkDir)
	for i := range binding.EvidenceRefs {
		binding.EvidenceRefs[i] = strings.TrimSpace(binding.EvidenceRefs[i])
	}
	return binding
}

// ValidateGoalWorkspaceAuthorityV0 validates only opaque authority. Physical
// path policy belongs to the workspace adapter.
func ValidateGoalWorkspaceAuthorityV0(
	goalRef string,
	authoritySchemaVersion string,
	workspaceRef string,
	providerRef string,
	runtimeGenerationRef string,
	allowLegacy bool,
) []GoalWorkIssueV0 {
	goalRef = strings.TrimSpace(goalRef)
	authoritySchemaVersion = strings.TrimSpace(authoritySchemaVersion)
	workspaceRef = strings.TrimSpace(workspaceRef)
	providerRef = strings.TrimSpace(providerRef)
	runtimeGenerationRef = strings.TrimSpace(runtimeGenerationRef)
	var issues []GoalWorkIssueV0
	if authoritySchemaVersion == "" && allowLegacy {
		// Pre-057 persisted receipts carried either no authority or only the
		// runtime generation. Keep that state readable without promoting it to
		// the new workspace/provider/generation contract.
		issues = validateGoalWorkspaceLegacyAuthorityV0(workspaceRef, providerRef, runtimeGenerationRef)
		return issues
	}
	if authoritySchemaVersion != GoalWorkspaceAuthoritySchemaV0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWorkspaceBindingInvalidV0, Field: "workspace_authority_schema_version"})
	}
	issues = append(issues, validateGoalWorkspaceAuthorityShapeV0(workspaceRef, providerRef, runtimeGenerationRef, false)...)
	if workspaceRef != "" && goalRef != "" && workspaceRef != GoalWorkspaceRefForGoalV0(goalRef) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWorkspaceRefMismatchV0, Field: "workspace_ref"})
	}
	return issues
}

func validateGoalWorkspaceLegacyAuthorityV0(workspaceRef, providerRef, runtimeGenerationRef string) []GoalWorkIssueV0 {
	issues := validateGoalWorkspaceAuthorityRefsV0(workspaceRef, providerRef, runtimeGenerationRef)
	if workspaceRef != "" || providerRef != "" {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWorkspaceBindingInvalidV0, Field: "workspace_authority_schema_version"})
	}
	return issues
}

func validateGoalWorkspaceAuthorityShapeV0(
	workspaceRef string,
	providerRef string,
	runtimeGenerationRef string,
	allowLegacy bool,
) []GoalWorkIssueV0 {
	issues := validateGoalWorkspaceAuthorityRefsV0(workspaceRef, providerRef, runtimeGenerationRef)
	present := 0
	for _, value := range []string{workspaceRef, providerRef, runtimeGenerationRef} {
		if value != "" {
			present++
		}
	}
	if present == 0 && allowLegacy {
		return issues
	}
	if present != 3 {
		if workspaceRef == "" {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWorkspaceBindingInvalidV0, Field: "workspace_ref"})
		}
		if providerRef == "" {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWorkspaceBindingInvalidV0, Field: "provider_ref"})
		}
		if runtimeGenerationRef == "" {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWorkspaceBindingInvalidV0, Field: "runtime_generation_ref"})
		}
	}
	return issues
}

func validateGoalWorkspaceAuthorityRefsV0(workspaceRef, providerRef, runtimeGenerationRef string) []GoalWorkIssueV0 {
	var issues []GoalWorkIssueV0
	validateGoalRefsV0(&issues, "workspace_ref", workspaceRef)
	validateGoalRefsV0(&issues, "provider_ref", providerRef)
	validateGoalRefsV0(&issues, "runtime_generation_ref", runtimeGenerationRef)
	for _, ref := range []struct{ field, value string }{
		{field: "workspace_ref", value: workspaceRef},
		{field: "provider_ref", value: providerRef},
		{field: "runtime_generation_ref", value: runtimeGenerationRef},
	} {
		if len([]byte(ref.value)) > GoalWorkspaceAuthorityRefMaxBytesV0 {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalSpecLimitExceededV0, Field: ref.field})
		}
	}
	return issues
}

func ValidateGoalWorkspaceBindingV0(
	binding GoalWorkspaceBindingV0,
	requireRuntimeGeneration bool,
) []GoalWorkIssueV0 {
	binding = NormalizeGoalWorkspaceBindingV0(binding)
	var issues []GoalWorkIssueV0
	validateGoalIntentManifestIdentityV0(&issues, binding.IntentManifestRef, binding.IntentManifestSHA256)
	if !requireRuntimeGeneration && binding.ProviderRef == "" && binding.RuntimeGenerationRef == "" {
		// Prepare may project the Goal-owned workspace before a provider runtime
		// exists. Persisted launch/observation authority still requires the triple.
		issues = validateGoalWorkspaceAuthorityRefsV0(binding.WorkspaceRef, "", "")
	} else {
		issues = validateGoalWorkspaceAuthorityShapeV0(
			binding.WorkspaceRef,
			binding.ProviderRef,
			binding.RuntimeGenerationRef,
			false,
		)
	}
	if binding.WorkspaceRef == "" {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWorkspaceBindingInvalidV0, Field: "workspace_ref"})
	}
	if binding.ProjectWorkDir == "" {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWorkspaceBindingInvalidV0, Field: "project_work_dir"})
	}
	return issues
}

// GoalWorkspaceAuthorityIssuesV0 compares an adapter projection against the
// immutable authority carried by Goal. Empty expected fields preserve legacy
// readers; adapters publishing the new contract must carry the complete triple.
func GoalWorkspaceAuthorityIssuesV0(
	expected GoalObservationRequestV0,
	actual GoalWorkspaceBindingV0,
) []GoalWorkIssueV0 {
	expected = NormalizeGoalObservationRequestV0(expected)
	actual = NormalizeGoalWorkspaceBindingV0(actual)
	issues := ValidateGoalObservationRequestV0(expected)
	if len(issues) != 0 {
		return issues
	}
	if expected.WorkspaceAuthoritySchemaVersion == "" {
		return nil
	}
	if expected.IntentManifestRef != actual.IntentManifestRef ||
		expected.IntentManifestSHA256 != actual.IntentManifestSHA256 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWorkspaceBindingInvalidV0, Field: "intent_manifest_ref"})
	}
	if expected.WorkspaceAuthoritySchemaVersion != actual.WorkspaceAuthoritySchemaVersion {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWorkspaceBindingInvalidV0, Field: "workspace_authority_schema_version"})
	}
	if expected.WorkspaceRef != "" && actual.WorkspaceRef != expected.WorkspaceRef {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWorkspaceRefMismatchV0, Field: "workspace_ref"})
	}
	if expected.ProviderRef != "" && actual.ProviderRef != expected.ProviderRef {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWorkspaceProviderMismatchV0, Field: "provider_ref"})
	}
	if expected.RuntimeGenerationRef != "" && actual.RuntimeGenerationRef != expected.RuntimeGenerationRef {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWorkspaceRuntimeGenerationMismatchV0, Field: "runtime_generation_ref"})
	}
	return issues
}

// GoalWorkspaceRouterPortV0 prepares and later resolves the durable workspace
// identity without exposing provider-specific packets to process backends.
type GoalWorkspaceRouterPortV0 interface {
	PrepareGoalWorkspaceV0(context.Context, GoalWorkSpecV0) (GoalWorkspaceBindingV0, error)
	ResolveGoalWorkspaceV0(context.Context, GoalObservationRequestV0) (GoalWorkspaceBindingV0, error)
}
