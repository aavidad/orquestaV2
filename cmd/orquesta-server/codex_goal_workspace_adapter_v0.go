package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexappserver "orquesta/modulos/orquesta-runtime-codex-appserver"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
	orquestasecurefile "orquesta/modulos/orquesta-secure-file"
)

const (
	codexGoalWorkspaceAdapterIndexSchemaV0 = "codex_goal_workspace_adapter_index.v0"
	codexGoalWorkspaceAdapterIndexSchemaV1 = "codex_goal_workspace_adapter_index.v1"
	codexGoalWorkspaceAdapterIndexSchemaV2 = "codex_goal_workspace_adapter_index.v2"
)

var errCodexGoalWorkspaceAdapterUnavailableV0 = errors.New("codex_goal_workspace_unavailable")
var errCodexGoalWorkspaceAdapterConflictV0 = errors.New("codex_goal_workspace_conflict")
var errCodexGoalWorkspaceRefMismatchV0 = errors.New("codex_goal_workspace_ref_mismatch")
var errCodexGoalWorkspaceProviderMismatchV0 = errors.New("codex_goal_workspace_provider_mismatch")
var errCodexGoalWorkspaceGenerationMismatchV0 = errors.New("codex_goal_workspace_generation_mismatch")
var errCodexGoalWorkspaceExternalGoalMismatchV0 = errors.New("codex_goal_workspace_external_goal_mismatch")

type codexGoalWorkspaceReworkSourceNotStoppedErrorV0 struct{}

func (*codexGoalWorkspaceReworkSourceNotStoppedErrorV0) Error() string {
	return "requires_056_selective_stop_receipt"
}

var errCodexGoalWorkspaceReworkSourceNotStoppedV0 = &codexGoalWorkspaceReworkSourceNotStoppedErrorV0{}

// codexGoalWorkspaceAdapterV0 persists the start request identity because goal
// observation only carries GoalRef and must survive process restarts.
type codexGoalWorkspaceAdapterV0 struct {
	SourceWorkDir       string
	WorkspaceRoot       string
	ProjectRefFallback  string
	WorktreeRefFallback string
	IntentManifestStore orquestaautoprogramming.AutoprogrammingIntentManifestStorePortV0
}

type codexGoalWorkspaceAdapterIndexV0 struct {
	SchemaVersion string                              `json:"schema_version"`
	GoalRef       string                              `json:"goal_ref"`
	Identity      codexGoalWorkspaceRequestIdentityV0 `json:"identity"`
}

type codexGoalWorkspaceRequestIdentityV0 struct {
	RunRef               string `json:"run_ref"`
	GoalRef              string `json:"goal_ref"`
	ProjectRef           string `json:"project_ref"`
	WorktreeRef          string `json:"worktree_ref"`
	IntentManifestRef    string `json:"intent_manifest_ref,omitempty"`
	IntentManifestSHA256 string `json:"intent_manifest_sha256,omitempty"`
	SourceGoalRef        string `json:"source_goal_ref,omitempty"`
	WriteSetSHA256       string `json:"write_set_sha256"`
	ReworkPolicySHA256   string `json:"rework_policy_sha256"`
	WorkspaceRef         string `json:"workspace_ref,omitempty"`
	ProviderRef          string `json:"provider_ref,omitempty"`
	RuntimeGenerationRef string `json:"runtime_generation_ref,omitempty"`
	ExternalGoalRef      string `json:"external_goal_ref,omitempty"`
}

var _ orquestaruntimecodexappserver.GoalWorkspaceRouterPortV0 = codexGoalWorkspaceAdapterV0{}
var _ orquestaruntimecodexappserver.GoalWorkspaceExecutionBinderPortV0 = codexGoalWorkspaceAdapterV0{}
var _ orquestagoal.GoalWorkspaceRouterPortV0 = codexGoalWorkspaceAdapterV0{}
var _ serverGoalWorkspaceBindingLookupV0 = codexGoalWorkspaceAdapterV0{}

func (adapter codexGoalWorkspaceAdapterV0) PrepareGoalWorkspaceV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalWorkspaceBindingV0, error) {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	if len(orquestagoal.ValidateGoalWorkSpecV0(spec)) != 0 {
		return orquestagoal.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	// This is a routing projection, deliberately not a Codex prompt packet:
	// Claude and Gemini must not inherit Codex prompt-size policy.
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		SchemaVersion:        orquestaruntimecodexgoal.CodexGoalStartPacketSchemaV0,
		GoalRef:              spec.GoalRef,
		WorkspaceRef:         orquestagoal.GoalWorkspaceRefForGoalV0(spec.GoalRef),
		RequestRef:           spec.RequestRef,
		IntentManifestRef:    spec.IntentManifestRef,
		IntentManifestSHA256: spec.IntentManifestSHA256,
		ProjectRef:           spec.ProjectRef,
		ContextRefs:          append([]orquestagoal.GoalContextRefV0(nil), spec.ContextRefs...),
		WriteSet:             append([]orquestagoal.GoalWriteScopeV0(nil), spec.WriteSet...),
		EvidenceRefs:         append([]string(nil), spec.EvidenceRefs...),
		ReworkPolicy:         spec.ReworkPolicy,
	}
	binding, err := adapter.PrepareCodexGoalWorkspaceV0(ctx, packet)
	return goalWorkspaceBindingFromCodexV0(
		binding,
		packet.IntentManifestRef,
		packet.IntentManifestSHA256,
		"",
		packet.WorkspaceRef,
		"",
		"",
	), err
}

func (adapter codexGoalWorkspaceAdapterV0) ResolveGoalWorkspaceV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkspaceBindingV0, error) {
	binding, identity, err := adapter.resolveCodexGoalWorkspaceProjectionV0(ctx, orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:                   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:                         request.GoalRef,
		ExternalGoalRef:                 request.ExternalGoalRef,
		IntentManifestRef:               request.IntentManifestRef,
		IntentManifestSHA256:            request.IntentManifestSHA256,
		WorkspaceAuthoritySchemaVersion: request.WorkspaceAuthoritySchemaVersion,
		WorkspaceRef:                    request.WorkspaceRef,
		ProviderRef:                     request.ProviderRef,
		RuntimeGenerationRef:            request.RuntimeGenerationRef,
	})
	return goalWorkspaceBindingFromCodexV0(
		binding,
		identity.IntentManifestRef,
		identity.IntentManifestSHA256,
		request.WorkspaceAuthoritySchemaVersion,
		identity.WorkspaceRef,
		identity.ProviderRef,
		identity.RuntimeGenerationRef,
	), err
}

func (adapter codexGoalWorkspaceAdapterV0) HasGoalWorkspaceBindingV0(
	ctx context.Context,
	goalRef string,
) (bool, error) {
	return adapter.HasCodexGoalWorkspaceBindingV0(ctx, goalRef)
}

func goalWorkspaceBindingFromCodexV0(
	binding orquestaruntimecodexappserver.GoalWorkspaceBindingV0,
	intentManifestRef string,
	intentManifestSHA256 string,
	workspaceAuthoritySchemaVersion string,
	workspaceRef string,
	providerRef string,
	runtimeGenerationRef string,
) orquestagoal.GoalWorkspaceBindingV0 {
	return orquestagoal.GoalWorkspaceBindingV0{
		IntentManifestRef:               strings.TrimSpace(intentManifestRef),
		IntentManifestSHA256:            strings.TrimSpace(intentManifestSHA256),
		WorkspaceAuthoritySchemaVersion: strings.TrimSpace(workspaceAuthoritySchemaVersion),
		WorkspaceRef:                    strings.TrimSpace(workspaceRef),
		ProviderRef:                     strings.TrimSpace(providerRef),
		RuntimeGenerationRef:            strings.TrimSpace(runtimeGenerationRef),
		ProjectWorkDir:                  binding.ProjectWorkDir,
		EvidenceRefs:                    append([]string(nil), binding.EvidenceRefs...),
	}
}

type serverCodexGoalWorkspaceBindingLookupV0 interface {
	HasCodexGoalWorkspaceBindingV0(context.Context, string) (bool, error)
	ResolveCodexGoalWorkspaceV0(context.Context, orquestaruntimecodexgoal.CodexGoalObservationRequestV0) (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, error)
}

type serverGoalWorkspaceBindingLookupV0 interface {
	HasGoalWorkspaceBindingV0(context.Context, string) (bool, error)
	ResolveGoalWorkspaceV0(context.Context, orquestagoal.GoalObservationRequestV0) (orquestagoal.GoalWorkspaceBindingV0, error)
}

func (adapter codexGoalWorkspaceAdapterV0) HasCodexGoalWorkspaceBindingV0(
	ctx context.Context,
	goalRef string,
) (bool, error) {
	if ctx == nil {
		return false, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	var err error
	adapter, err = adapter.normalizedV0()
	if err != nil {
		return false, err
	}
	goalRef = strings.TrimSpace(goalRef)
	if goalRef == "" {
		return false, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	_, found, err := adapter.loadGoalIndexV0(goalRef)
	return found, err
}

func (adapter codexGoalWorkspaceAdapterV0) PrepareCodexGoalWorkspaceV0(
	ctx context.Context,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, error) {
	if ctx == nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	if err := ctx.Err(); err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
	}
	var err error
	adapter, err = adapter.normalizedV0()
	if err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
	}
	request := adapter.requestForStartV0(packet)
	if strings.TrimSpace(request.GoalRef) == "" {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	sourceGoalRef, err := codexGoalWorkspaceReworkSourceV0(packet)
	if err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
	}
	identity, err := codexGoalWorkspaceRequestIdentityFromPacketV0(request, packet, sourceGoalRef)
	if err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
	}
	return adapter.withGoalIndexLockV0(ctx, request.GoalRef, func() (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, error) {
		if indexed, found, err := adapter.loadGoalIndexV0(request.GoalRef); err != nil {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
		} else if found && !codexGoalWorkspaceIndexedIdentityCompatibleV0(indexed, identity) {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterConflictV0
		} else if found {
			workspace, issues := (orquestaruntimeworktree.GitGoalWorkspaceProvisionerV0{}).ResolveGoalWorkspaceV0(ctx, adapter.requestFromIdentityV0(indexed.Identity))
			if len(issues) > 0 || adapter.verifyIntentManifestV0(ctx, indexed.Identity, workspace.ProjectWorkDir) != nil {
				return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterConflictV0
			}
			// A prepare replay has no runtime generation. Preserve pre-057
			// indices verbatim; only a complete observation authority may migrate
			// a v1 index to v2.
			return codexGoalWorkspaceBindingFromWorkspaceV0(workspace), nil
		}
		if sourceGoalRef != "" {
			source, found, loadErr := adapter.loadGoalIndexV0(sourceGoalRef)
			if loadErr != nil || !found || !codexGoalWorkspaceValidReworkLineageV0(source.Identity, identity) {
				return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterConflictV0
			}
			sourceResolved, sourceIssues := (orquestaruntimeworktree.GitGoalWorkspaceProvisionerV0{}).ResolveGoalWorkspaceV0(ctx, adapter.requestFromIdentityV0(source.Identity))
			if len(sourceIssues) > 0 || adapter.verifyIntentManifestV0(ctx, source.Identity, sourceResolved.ProjectWorkDir) != nil {
				return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterConflictV0
			}
			if packet.ReworkPolicy.PreserveArtifacts {
				// 057 has no authoritative selective-stop/terminal receipt. Copying a
				// live workspace would race the source agent and sharing it would allow
				// two writers. 056 must provide that authority before transfer is safe.
				return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceReworkSourceNotStoppedV0
			}
		}
		workspace, issues := (orquestaruntimeworktree.GitGoalWorkspaceProvisionerV0{}).PrepareGoalWorkspaceV0(ctx, request)
		if len(issues) > 0 {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
		}
		if err := adapter.materializeIntentManifestV0(ctx, packet, workspace.ProjectWorkDir); err != nil {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
		}
		if err := adapter.writeGoalIndexV0(request.GoalRef, identity); err != nil {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
		}
		return codexGoalWorkspaceBindingFromWorkspaceV0(workspace), nil
	})
}

func (adapter codexGoalWorkspaceAdapterV0) verifyIntentManifestV0(
	ctx context.Context,
	identity codexGoalWorkspaceRequestIdentityV0,
	workspace string,
) error {
	if identity.IntentManifestRef == "" && identity.IntentManifestSHA256 == "" {
		return nil
	}
	if adapter.IntentManifestStore == nil || identity.IntentManifestRef == "" || identity.IntentManifestSHA256 == "" {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	manifest, err := adapter.IntentManifestStore.LoadAutoprogrammingIntentManifestV0(ctx, identity.RunRef)
	if err != nil || len(orquestaautoprogramming.ValidateAutoprogrammingIntentManifestV0(manifest)) != 0 || manifest.ManifestRef != identity.IntentManifestRef || manifest.RequestSHA256 != identity.IntentManifestSHA256 {
		return errCodexGoalWorkspaceAdapterConflictV0
	}
	root := filepath.Join(workspace, ".orquesta-runtime", "intent-manifests")
	dir, err := openPrivateIntentManifestDirV0(root, false)
	if err != nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	defer dir.Close()
	raw, err := readIntentManifestAtV0(dir, manifest.ManifestRef+".json")
	if err != nil || !bytesEqualIntentManifestV0(raw, manifest.RequestJSON) {
		return errCodexGoalWorkspaceAdapterConflictV0
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != manifest.RequestSHA256 {
		return errCodexGoalWorkspaceAdapterConflictV0
	}
	return nil
}

func (adapter codexGoalWorkspaceAdapterV0) materializeIntentManifestV0(ctx context.Context, packet orquestaruntimecodexgoal.CodexGoalStartPacketV0, workspace string) error {
	if strings.TrimSpace(packet.IntentManifestRef) == "" && strings.TrimSpace(packet.IntentManifestSHA256) == "" {
		return nil
	}
	if adapter.IntentManifestStore == nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	manifest, err := adapter.IntentManifestStore.LoadAutoprogrammingIntentManifestV0(ctx, packet.RequestRef)
	if err != nil || len(orquestaautoprogramming.ValidateAutoprogrammingIntentManifestV0(manifest)) != 0 || manifest.ManifestRef != strings.TrimSpace(packet.IntentManifestRef) || manifest.RequestSHA256 != strings.TrimSpace(packet.IntentManifestSHA256) {
		return errCodexGoalWorkspaceAdapterConflictV0
	}
	sum := sha256.Sum256(manifest.RequestJSON)
	if hex.EncodeToString(sum[:]) != manifest.RequestSHA256 {
		return errCodexGoalWorkspaceAdapterConflictV0
	}
	root := filepath.Join(workspace, ".orquesta-runtime", "intent-manifests")
	dir, err := openPrivateIntentManifestDirV0(root, true)
	if err != nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	defer dir.Close()
	name := manifest.ManifestRef + ".json"
	if existing, readErr := readIntentManifestAtV0(dir, name); readErr == nil {
		if !bytesEqualIntentManifestV0(existing, manifest.RequestJSON) {
			return errCodexGoalWorkspaceAdapterConflictV0
		}
		return nil
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return errCodexGoalWorkspaceAdapterConflictV0
	}
	raw, _, err := orquestasecurefile.CreateFileIfAbsentAtV0(dir, name, manifest.RequestJSON, orquestasecurefile.FileOptionsV0{
		MaxBytes:  maxAutoprogrammingIntentManifestBytesV0,
		ExactMode: 0o400,
	})
	if err != nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	verify := sha256.Sum256(raw)
	if hex.EncodeToString(verify[:]) != manifest.RequestSHA256 || !bytesEqualIntentManifestV0(raw, manifest.RequestJSON) {
		return errCodexGoalWorkspaceAdapterConflictV0
	}
	return nil
}

func (adapter codexGoalWorkspaceAdapterV0) ResolveCodexGoalWorkspaceV0(
	ctx context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
) (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, error) {
	binding, _, err := adapter.resolveCodexGoalWorkspaceProjectionV0(ctx, request)
	return binding, err
}

func (adapter codexGoalWorkspaceAdapterV0) BindCodexGoalExecutionV0(
	ctx context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
) error {
	_, _, err := adapter.resolveCodexGoalWorkspaceProjectionV0(ctx, request)
	return err
}

func (adapter codexGoalWorkspaceAdapterV0) resolveCodexGoalWorkspaceProjectionV0(
	ctx context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
) (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, codexGoalWorkspaceRequestIdentityV0, error) {
	if ctx == nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, codexGoalWorkspaceRequestIdentityV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	if err := ctx.Err(); err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, codexGoalWorkspaceRequestIdentityV0{}, err
	}
	var err error
	adapter, err = adapter.normalizedV0()
	if err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, codexGoalWorkspaceRequestIdentityV0{}, err
	}
	goalRef := strings.TrimSpace(request.GoalRef)
	if goalRef == "" {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, codexGoalWorkspaceRequestIdentityV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	if err := codexGoalWorkspaceObservationAuthorityErrorV0(request); err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, codexGoalWorkspaceRequestIdentityV0{}, err
	}
	var projection codexGoalWorkspaceRequestIdentityV0
	binding, err := adapter.withGoalIndexLockV0(ctx, goalRef, func() (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, error) {
		indexed, found, err := adapter.loadGoalIndexV0(goalRef)
		if err != nil {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
		}
		if !found {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
		}
		bound, changed, err := codexGoalWorkspaceBindAuthorityV0(indexed, request)
		if err != nil {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
		}
		workspace, issues := (orquestaruntimeworktree.GitGoalWorkspaceProvisionerV0{}).ResolveGoalWorkspaceV0(ctx, adapter.requestFromIdentityV0(indexed.Identity))
		if len(issues) > 0 {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
		}
		if err := adapter.verifyIntentManifestV0(ctx, indexed.Identity, workspace.ProjectWorkDir); err != nil {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
		}
		if changed {
			if err := adapter.writeGoalIndexV0(goalRef, bound.Identity); err != nil {
				return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
			}
		}
		projection = bound.Identity
		return codexGoalWorkspaceBindingFromWorkspaceV0(workspace), nil
	})
	if err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, codexGoalWorkspaceRequestIdentityV0{}, err
	}
	return binding, projection, nil
}

func (adapter codexGoalWorkspaceAdapterV0) normalizedV0() (codexGoalWorkspaceAdapterV0, error) {
	adapter.SourceWorkDir = strings.TrimSpace(adapter.SourceWorkDir)
	adapter.WorkspaceRoot = strings.TrimSpace(adapter.WorkspaceRoot)
	if adapter.SourceWorkDir == "" || adapter.WorkspaceRoot == "" {
		return codexGoalWorkspaceAdapterV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	source, sourceErr := filepath.Abs(adapter.SourceWorkDir)
	root, rootErr := filepath.Abs(adapter.WorkspaceRoot)
	if sourceErr != nil || rootErr != nil {
		return codexGoalWorkspaceAdapterV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	adapter.SourceWorkDir = filepath.Clean(source)
	adapter.WorkspaceRoot = filepath.Clean(root)
	adapter.ProjectRefFallback = strings.TrimSpace(adapter.ProjectRefFallback)
	adapter.WorktreeRefFallback = strings.TrimSpace(adapter.WorktreeRefFallback)
	return adapter, nil
}

func (adapter codexGoalWorkspaceAdapterV0) requestForStartV0(packet orquestaruntimecodexgoal.CodexGoalStartPacketV0) orquestaruntimeworktree.GoalWorkspaceRequestV0 {
	goalRef := strings.TrimSpace(packet.GoalRef)
	return orquestaruntimeworktree.GoalWorkspaceRequestV0{
		RunRef:        firstNonEmptyCodexGoalWorkspaceAdapterV0(packet.RequestRef, goalRef),
		GoalRef:       goalRef,
		ProjectRef:    firstNonEmptyCodexGoalWorkspaceAdapterV0(packet.ProjectRef, adapter.ProjectRefFallback),
		WorktreeRef:   firstNonEmptyCodexGoalWorkspaceAdapterV0(codexGoalWorkspacePacketContextRefV0(packet, "worktree"), adapter.WorktreeRefFallback),
		SourceWorkDir: strings.TrimSpace(adapter.SourceWorkDir),
		WorkspaceRoot: strings.TrimSpace(adapter.WorkspaceRoot),
	}
}

func codexGoalWorkspacePacketContextRefV0(
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
	kind string,
) string {
	kind = strings.TrimSpace(kind)
	for _, ref := range packet.ContextRefs {
		if strings.TrimSpace(ref.Kind) == kind {
			return strings.TrimSpace(ref.Ref)
		}
	}
	return ""
}

func codexGoalWorkspaceReworkSourceV0(packet orquestaruntimecodexgoal.CodexGoalStartPacketV0) (string, error) {
	kinds := []string{"source_goal"}
	if packet.ReworkPolicy.PreserveArtifacts {
		// The standard lifecycle uses kind=goal for a causal predecessor. Limit
		// that broader alias to artifact-preserving rework so ordinary goal
		// dependencies are not mistaken for a rework lineage.
		kinds = append(kinds, "goal")
	}
	for _, kind := range kinds {
		var candidates []string
		seenKind := false
		for _, ref := range packet.ContextRefs {
			if strings.TrimSpace(ref.Kind) != kind {
				continue
			}
			seenKind = true
			value := strings.TrimSpace(ref.Ref)
			if value == "" || !ref.Required {
				return "", errCodexGoalWorkspaceAdapterConflictV0
			}
			if codexGoalWorkspaceCausalSuccessorRefV0(value, packet.GoalRef) {
				candidates = append(candidates, value)
			}
		}
		if len(candidates) > 0 {
			// Resident rework preserves its complete source_goal history for
			// budgeting. The immediate predecessor is the longest causal ref.
			sort.Slice(candidates, func(i, j int) bool {
				if len(candidates[i]) != len(candidates[j]) {
					return len(candidates[i]) > len(candidates[j])
				}
				return candidates[i] < candidates[j]
			})
			if len(candidates) > 1 && len(candidates[0]) == len(candidates[1]) && candidates[0] != candidates[1] {
				return "", errCodexGoalWorkspaceAdapterConflictV0
			}
			return candidates[0], nil
		}
		if seenKind {
			return "", errCodexGoalWorkspaceAdapterConflictV0
		}
	}
	// PreserveArtifacts is also valid for an initial goal. It becomes a
	// snapshot only when an explicit causal predecessor exists.
	return "", nil
}

func codexGoalWorkspaceValidReworkLineageV0(source, successor codexGoalWorkspaceRequestIdentityV0) bool {
	if source.GoalRef == "" || successor.SourceGoalRef != source.GoalRef || !codexGoalWorkspaceCausalSuccessorRefV0(source.GoalRef, successor.GoalRef) || source.ProjectRef != successor.ProjectRef || source.WorktreeRef != successor.WorktreeRef {
		return false
	}
	sourceManifest := source.IntentManifestRef != "" || source.IntentManifestSHA256 != ""
	successorManifest := successor.IntentManifestRef != "" || successor.IntentManifestSHA256 != ""
	if sourceManifest || successorManifest {
		return source.IntentManifestRef != "" && source.IntentManifestSHA256 != "" && source.IntentManifestRef == successor.IntentManifestRef && source.IntentManifestSHA256 == successor.IntentManifestSHA256 && source.RunRef == successor.RunRef
	}
	// Resident legacy rework derives a new request_ref; the immutable-manifest
	// path above deliberately forbids that substitution.
	return successor.RunRef == source.RunRef || strings.HasPrefix(successor.RunRef, source.RunRef+"-rework-")
}

func codexGoalWorkspaceCausalSuccessorRefV0(sourceGoalRef, successorGoalRef string) bool {
	sourceGoalRef = strings.TrimSpace(sourceGoalRef)
	successorGoalRef = strings.TrimSpace(successorGoalRef)
	if sourceGoalRef == "" || successorGoalRef == "" {
		return false
	}
	base, sourceIndex, sourceNumeric := codexGoalWorkspaceNumericReworkRefV0(sourceGoalRef)
	successorBase, successorIndex, successorNumeric := codexGoalWorkspaceNumericReworkRefV0(successorGoalRef)
	if successorNumeric {
		if sourceNumeric {
			return base == successorBase && successorIndex == sourceIndex+1
		}
		return sourceGoalRef == successorBase && successorIndex == 1
	}
	// Resident repair uses nested deterministic non-numeric suffixes. The
	// caller chooses the longest matching required source as the immediate one.
	return strings.HasPrefix(successorGoalRef, sourceGoalRef+"-rework-")
}

func codexGoalWorkspaceNumericReworkRefV0(goalRef string) (string, int, bool) {
	marker := strings.LastIndex(goalRef, "-rework-")
	if marker <= 0 {
		return goalRef, 0, false
	}
	index, err := strconv.Atoi(goalRef[marker+len("-rework-"):])
	if err != nil || index < 1 {
		return goalRef, 0, false
	}
	return goalRef[:marker], index, true
}

func codexGoalWorkspaceHashJSONV0(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", errCodexGoalWorkspaceAdapterUnavailableV0
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func codexGoalWorkspaceWriteSetIdentityV0(writeSet []orquestagoal.GoalWriteScopeV0) ([]orquestagoal.GoalWriteScopeV0, error) {
	canonical := make([]orquestagoal.GoalWriteScopeV0, 0, len(writeSet))
	for _, scope := range writeSet {
		path := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(scope.Path))))
		if path == "" || path == "." || filepath.IsAbs(filepath.FromSlash(path)) || path == ".." || strings.HasPrefix(path, "../") {
			return nil, errCodexGoalWorkspaceAdapterConflictV0
		}
		canonical = append(canonical, orquestagoal.GoalWriteScopeV0{Path: path, Purpose: strings.TrimSpace(scope.Purpose)})
	}
	sort.Slice(canonical, func(i, j int) bool {
		if canonical[i].Path != canonical[j].Path {
			return canonical[i].Path < canonical[j].Path
		}
		return canonical[i].Purpose < canonical[j].Purpose
	})
	return canonical, nil
}

func (adapter codexGoalWorkspaceAdapterV0) requestFromIdentityV0(identity codexGoalWorkspaceRequestIdentityV0) orquestaruntimeworktree.GoalWorkspaceRequestV0 {
	return orquestaruntimeworktree.GoalWorkspaceRequestV0{
		RunRef:        identity.RunRef,
		GoalRef:       identity.GoalRef,
		ProjectRef:    identity.ProjectRef,
		WorktreeRef:   identity.WorktreeRef,
		SourceWorkDir: adapter.SourceWorkDir,
		WorkspaceRoot: adapter.WorkspaceRoot,
	}
}

func (adapter codexGoalWorkspaceAdapterV0) withGoalIndexLockV0(
	ctx context.Context,
	goalRef string,
	fn func() (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, error),
) (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, error) {
	if ctx == nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	dir, err := openCodexGoalWorkspaceIndexDirV1(filepath.Dir(adapter.goalIndexLockPathV0(goalRef)), true)
	if err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	defer dir.Close()
	lock, err := openCodexGoalWorkspaceIndexLockV1(dir, filepath.Base(adapter.goalIndexLockPathV0(goalRef)))
	if err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	defer lock.Close()
	if err := lockCodexGoalWorkspaceIndexContextV1(ctx, lock); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, ctxErr
		}
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	return fn()
}

func (adapter codexGoalWorkspaceAdapterV0) loadGoalIndexV0(goalRef string) (codexGoalWorkspaceAdapterIndexV0, bool, error) {
	path := adapter.goalIndexPathV0(goalRef)
	if _, err := os.Lstat(filepath.Dir(path)); errors.Is(err, os.ErrNotExist) {
		return codexGoalWorkspaceAdapterIndexV0{}, false, nil
	} else if err != nil {
		return codexGoalWorkspaceAdapterIndexV0{}, false, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	dir, err := openCodexGoalWorkspaceIndexDirV1(filepath.Dir(path), false)
	if err != nil {
		return codexGoalWorkspaceAdapterIndexV0{}, false, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	defer dir.Close()
	raw, err := readCodexGoalWorkspaceIndexV1(dir, filepath.Base(path))
	if errors.Is(err, os.ErrNotExist) {
		return codexGoalWorkspaceAdapterIndexV0{}, false, nil
	}
	if err != nil {
		return codexGoalWorkspaceAdapterIndexV0{}, false, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	index, err := decodeCodexGoalWorkspaceIndexV1(raw)
	if err != nil || !validateCodexGoalWorkspaceIndexV1(index, goalRef) {
		return codexGoalWorkspaceAdapterIndexV0{}, false, errCodexGoalWorkspaceAdapterConflictV0
	}
	return index, true, nil
}

func (adapter codexGoalWorkspaceAdapterV0) writeGoalIndexV0(goalRef string, identity codexGoalWorkspaceRequestIdentityV0) error {
	path := adapter.goalIndexPathV0(goalRef)
	dir, err := openCodexGoalWorkspaceIndexDirV1(filepath.Dir(path), true)
	if err != nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	defer dir.Close()
	index := codexGoalWorkspaceAdapterIndexV0{
		SchemaVersion: codexGoalWorkspaceAdapterIndexSchemaV2,
		GoalRef:       strings.TrimSpace(goalRef),
		Identity:      identity,
	}
	if !validateCodexGoalWorkspaceIndexV1(index, goalRef) {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	data, err := json.Marshal(index)
	if err != nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	if err := writeCodexGoalWorkspaceIndexV1(dir, filepath.Base(path), append(data, '\n')); err != nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	return nil
}

func codexGoalWorkspaceIndexedIdentityCompatibleV0(index codexGoalWorkspaceAdapterIndexV0, desired codexGoalWorkspaceRequestIdentityV0) bool {
	if index.SchemaVersion == codexGoalWorkspaceAdapterIndexSchemaV2 {
		indexedBase := index.Identity
		desiredBase := desired
		indexedBase.WorkspaceRef, indexedBase.ProviderRef, indexedBase.RuntimeGenerationRef, indexedBase.ExternalGoalRef = "", "", "", ""
		desiredBase.WorkspaceRef, desiredBase.ProviderRef, desiredBase.RuntimeGenerationRef, desiredBase.ExternalGoalRef = "", "", "", ""
		return indexedBase == desiredBase &&
			index.Identity.WorkspaceRef == desired.WorkspaceRef &&
			(desired.ProviderRef == "" || index.Identity.ProviderRef == "" || index.Identity.ProviderRef == desired.ProviderRef)
	}
	if index.SchemaVersion == codexGoalWorkspaceAdapterIndexSchemaV1 {
		legacyDesired := desired
		legacyDesired.WorkspaceRef, legacyDesired.ProviderRef, legacyDesired.RuntimeGenerationRef, legacyDesired.ExternalGoalRef = "", "", "", ""
		return index.Identity == legacyDesired
	}
	if index.SchemaVersion != codexGoalWorkspaceAdapterIndexSchemaV0 ||
		index.Identity.IntentManifestRef != "" ||
		index.Identity.IntentManifestSHA256 != "" ||
		index.Identity.SourceGoalRef != "" ||
		index.Identity.WriteSetSHA256 != "" ||
		index.Identity.ReworkPolicySHA256 != "" ||
		desired.IntentManifestRef != "" ||
		desired.IntentManifestSHA256 != "" ||
		desired.SourceGoalRef != "" {
		return false
	}
	return index.Identity.RunRef == desired.RunRef &&
		index.Identity.GoalRef == desired.GoalRef &&
		index.Identity.ProjectRef == desired.ProjectRef &&
		index.Identity.WorktreeRef == desired.WorktreeRef
}

func (adapter codexGoalWorkspaceAdapterV0) goalIndexPathV0(goalRef string) string {
	return filepath.Join(strings.TrimSpace(adapter.WorkspaceRoot), "codex-goal-workspace-index-v0", codexGoalWorkspaceAdapterIDV0(goalRef)+".json")
}

func (adapter codexGoalWorkspaceAdapterV0) goalIndexLockPathV0(goalRef string) string {
	return filepath.Join(strings.TrimSpace(adapter.WorkspaceRoot), "codex-goal-workspace-index-v0", codexGoalWorkspaceAdapterIDV0(goalRef)+".lock")
}

func codexGoalWorkspaceAdapterIDV0(goalRef string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(goalRef)))
	return hex.EncodeToString(digest[:16])
}

func codexGoalWorkspaceRootForSourceV0(sourceWorkDir string) string {
	sourceWorkDir = strings.TrimSpace(sourceWorkDir)
	abs, err := filepath.Abs(sourceWorkDir)
	if sourceWorkDir == "" || err != nil {
		return ""
	}
	abs = filepath.Clean(abs)
	digest := sha256.Sum256([]byte(abs))
	return filepath.Join(filepath.Dir(abs), ".orquesta-goal-workspaces-"+hex.EncodeToString(digest[:8]))
}

func codexGoalWorkspaceRequestIdentityFromRequestV0(request orquestaruntimeworktree.GoalWorkspaceRequestV0) codexGoalWorkspaceRequestIdentityV0 {
	return codexGoalWorkspaceRequestIdentityV0{
		RunRef:      strings.TrimSpace(request.RunRef),
		GoalRef:     strings.TrimSpace(request.GoalRef),
		ProjectRef:  strings.TrimSpace(request.ProjectRef),
		WorktreeRef: strings.TrimSpace(request.WorktreeRef),
	}
}

func codexGoalWorkspaceRequestIdentityFromPacketV0(
	request orquestaruntimeworktree.GoalWorkspaceRequestV0,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
	sourceGoalRef string,
) (codexGoalWorkspaceRequestIdentityV0, error) {
	identity := codexGoalWorkspaceRequestIdentityFromRequestV0(request)
	identity.IntentManifestRef = strings.TrimSpace(packet.IntentManifestRef)
	identity.IntentManifestSHA256 = strings.TrimSpace(packet.IntentManifestSHA256)
	identity.SourceGoalRef = strings.TrimSpace(sourceGoalRef)
	canonicalWorkspaceRef := orquestagoal.GoalWorkspaceRefForGoalV0(request.GoalRef)
	identity.WorkspaceRef = strings.TrimSpace(packet.WorkspaceRef)
	if identity.WorkspaceRef == "" {
		identity.WorkspaceRef = canonicalWorkspaceRef
	} else if identity.WorkspaceRef != canonicalWorkspaceRef {
		return codexGoalWorkspaceRequestIdentityV0{}, errCodexGoalWorkspaceRefMismatchV0
	}
	providerRef := strings.TrimSpace(packet.ProviderRef)
	if providerRef != "" && providerRef != orquestaruntimecodexgoal.CodexGoalProviderRefV0 {
		return codexGoalWorkspaceRequestIdentityV0{}, errCodexGoalWorkspaceProviderMismatchV0
	}
	// Prepare owns only the stable workspace identity. Provider and runtime
	// generation become immutable together after a successful launch.
	identity.ProviderRef = ""
	writeSet, err := codexGoalWorkspaceWriteSetIdentityV0(packet.WriteSet)
	if err != nil {
		return codexGoalWorkspaceRequestIdentityV0{}, err
	}
	identity.WriteSetSHA256, err = codexGoalWorkspaceHashJSONV0(writeSet)
	if err != nil {
		return codexGoalWorkspaceRequestIdentityV0{}, err
	}
	identity.ReworkPolicySHA256, err = codexGoalWorkspaceHashJSONV0(packet.ReworkPolicy)
	if err != nil {
		return codexGoalWorkspaceRequestIdentityV0{}, err
	}
	return identity, nil
}

// codexGoalWorkspaceBindAuthorityV0 only materializes authority supplied by
// Goal. The index can detect replay conflicts, but never replaces request
// workspace/provider/generation with locally chosen values.
func codexGoalWorkspaceBindAuthorityV0(
	index codexGoalWorkspaceAdapterIndexV0,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
) (codexGoalWorkspaceAdapterIndexV0, bool, error) {
	goalRef := strings.TrimSpace(request.GoalRef)
	workspaceRef := strings.TrimSpace(request.WorkspaceRef)
	providerRef := strings.TrimSpace(request.ProviderRef)
	runtimeGenerationRef := strings.TrimSpace(request.RuntimeGenerationRef)
	externalGoalRef := strings.TrimSpace(request.ExternalGoalRef)
	intentManifestRef := strings.TrimSpace(request.IntentManifestRef)
	intentManifestSHA256 := strings.TrimSpace(request.IntentManifestSHA256)
	canonicalWorkspaceRef := orquestagoal.GoalWorkspaceRefForGoalV0(goalRef)
	requestHasAuthority := workspaceRef != "" && providerRef != "" && runtimeGenerationRef != ""
	requestHasExecution := requestHasAuthority && externalGoalRef != ""
	if requestHasExecution && (intentManifestRef != index.Identity.IntentManifestRef ||
		intentManifestSHA256 != index.Identity.IntentManifestSHA256) {
		return codexGoalWorkspaceAdapterIndexV0{}, false, errCodexGoalWorkspaceAdapterConflictV0
	}
	if index.Identity.WorkspaceRef != "" && index.Identity.WorkspaceRef != canonicalWorkspaceRef {
		return codexGoalWorkspaceAdapterIndexV0{}, false, errCodexGoalWorkspaceRefMismatchV0
	}
	if index.SchemaVersion == codexGoalWorkspaceAdapterIndexSchemaV0 {
		if requestHasAuthority || externalGoalRef != "" {
			// v0 lacks the manifest/write-set/rework identity needed by v2. Do
			// not manufacture a partially verified projection from the request.
			return codexGoalWorkspaceAdapterIndexV0{}, false, errCodexGoalWorkspaceAdapterConflictV0
		}
		return index, false, nil
	}
	if index.SchemaVersion == codexGoalWorkspaceAdapterIndexSchemaV1 {
		if !requestHasAuthority && externalGoalRef == "" {
			return index, false, nil
		}
		if !requestHasExecution {
			return codexGoalWorkspaceAdapterIndexV0{}, false, errCodexGoalWorkspaceAdapterConflictV0
		}
		bound := index
		bound.SchemaVersion = codexGoalWorkspaceAdapterIndexSchemaV2
		bound.Identity.WorkspaceRef = canonicalWorkspaceRef
		bound.Identity.ProviderRef = providerRef
		bound.Identity.RuntimeGenerationRef = runtimeGenerationRef
		bound.Identity.ExternalGoalRef = externalGoalRef
		return bound, true, nil
	}
	indexHasExecution := index.Identity.ProviderRef != "" ||
		index.Identity.RuntimeGenerationRef != "" || index.Identity.ExternalGoalRef != ""
	indexExecutionComplete := index.Identity.ProviderRef != "" &&
		index.Identity.RuntimeGenerationRef != "" && index.Identity.ExternalGoalRef != ""
	if indexHasExecution != indexExecutionComplete || !requestHasExecution {
		return codexGoalWorkspaceAdapterIndexV0{}, false, errCodexGoalWorkspaceAdapterConflictV0
	}
	if providerRef != "" && index.Identity.ProviderRef != "" && index.Identity.ProviderRef != providerRef {
		return codexGoalWorkspaceAdapterIndexV0{}, false, errCodexGoalWorkspaceProviderMismatchV0
	}
	if runtimeGenerationRef != "" && index.Identity.RuntimeGenerationRef != "" && index.Identity.RuntimeGenerationRef != runtimeGenerationRef {
		return codexGoalWorkspaceAdapterIndexV0{}, false, errCodexGoalWorkspaceGenerationMismatchV0
	}
	if index.Identity.ExternalGoalRef != "" && index.Identity.ExternalGoalRef != externalGoalRef {
		return codexGoalWorkspaceAdapterIndexV0{}, false, errCodexGoalWorkspaceExternalGoalMismatchV0
	}
	bound := index
	if !indexHasExecution {
		bound.Identity.ProviderRef = providerRef
		bound.Identity.RuntimeGenerationRef = runtimeGenerationRef
		bound.Identity.ExternalGoalRef = externalGoalRef
	}
	return bound, bound != index, nil
}

func codexGoalWorkspaceObservationAuthorityErrorV0(request orquestaruntimecodexgoal.CodexGoalObservationRequestV0) error {
	issues := orquestagoal.ValidateGoalObservationRequestV0(orquestagoal.GoalObservationRequestV0{
		GoalRef: request.GoalRef, ExternalGoalRef: request.ExternalGoalRef,
		IntentManifestRef: request.IntentManifestRef, IntentManifestSHA256: request.IntentManifestSHA256,
		WorkspaceAuthoritySchemaVersion: request.WorkspaceAuthoritySchemaVersion,
		WorkspaceRef:                    request.WorkspaceRef, ProviderRef: request.ProviderRef,
		RuntimeGenerationRef: request.RuntimeGenerationRef,
	})
	for _, issue := range issues {
		if issue.Code == orquestagoal.ErrGoalWorkspaceRefMismatchV0 {
			return errCodexGoalWorkspaceRefMismatchV0
		}
	}
	if len(issues) != 0 {
		return errCodexGoalWorkspaceAdapterConflictV0
	}
	return nil
}

func codexGoalWorkspaceBindingFromWorkspaceV0(workspace orquestaruntimeworktree.GoalWorkspaceV0) orquestaruntimecodexappserver.GoalWorkspaceBindingV0 {
	return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{
		ProjectWorkDir: workspace.ProjectWorkDir,
		EvidenceRefs:   append([]string(nil), workspace.EvidenceRefs...),
	}
}

func firstNonEmptyCodexGoalWorkspaceAdapterV0(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
