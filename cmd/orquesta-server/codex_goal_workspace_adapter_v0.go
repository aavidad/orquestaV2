package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	orquestaruntimecodexappserver "orquesta/modulos/orquesta-runtime-codex-appserver"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

const codexGoalWorkspaceAdapterIndexSchemaV0 = "codex_goal_workspace_adapter_index.v0"

var errCodexGoalWorkspaceAdapterUnavailableV0 = errors.New("codex_goal_workspace_unavailable")
var errCodexGoalWorkspaceAdapterConflictV0 = errors.New("codex_goal_workspace_conflict")

// codexGoalWorkspaceAdapterV0 persists the start request identity because goal
// observation only carries GoalRef and must survive process restarts.
type codexGoalWorkspaceAdapterV0 struct {
	SourceWorkDir       string
	WorkspaceRoot       string
	ProjectRefFallback  string
	WorktreeRefFallback string
}

type codexGoalWorkspaceAdapterIndexV0 struct {
	SchemaVersion string                              `json:"schema_version"`
	GoalRef       string                              `json:"goal_ref"`
	Identity      codexGoalWorkspaceRequestIdentityV0 `json:"identity"`
}

type codexGoalWorkspaceRequestIdentityV0 struct {
	RunRef      string `json:"run_ref"`
	GoalRef     string `json:"goal_ref"`
	ProjectRef  string `json:"project_ref"`
	WorktreeRef string `json:"worktree_ref"`
}

var _ orquestaruntimecodexappserver.GoalWorkspaceRouterPortV0 = codexGoalWorkspaceAdapterV0{}

type serverCodexGoalWorkspaceBindingLookupV0 interface {
	HasCodexGoalWorkspaceBindingV0(context.Context, string) (bool, error)
	ResolveCodexGoalWorkspaceV0(context.Context, orquestaruntimecodexgoal.CodexGoalObservationRequestV0) (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, error)
}

func (adapter codexGoalWorkspaceAdapterV0) HasCodexGoalWorkspaceBindingV0(
	ctx context.Context,
	goalRef string,
) (bool, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return false, err
		}
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
	var err error
	adapter, err = adapter.normalizedV0()
	if err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
	}
	request := adapter.requestForStartV0(packet)
	if strings.TrimSpace(request.GoalRef) == "" {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	return adapter.withGoalIndexLockV0(request.GoalRef, func() (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, error) {
		if indexed, found, err := adapter.loadGoalIndexV0(request.GoalRef); err != nil {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
		} else if found && indexed.Identity != codexGoalWorkspaceRequestIdentityFromRequestV0(request) {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterConflictV0
		}

		workspace, issues := (orquestaruntimeworktree.GitGoalWorkspaceProvisionerV0{}).PrepareGoalWorkspaceV0(ctx, request)
		if len(issues) > 0 {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
		}
		if err := adapter.writeGoalIndexV0(request.GoalRef, request); err != nil {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
		}
		return codexGoalWorkspaceBindingFromWorkspaceV0(workspace), nil
	})
}

func (adapter codexGoalWorkspaceAdapterV0) ResolveCodexGoalWorkspaceV0(
	ctx context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
) (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, error) {
	var err error
	adapter, err = adapter.normalizedV0()
	if err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
	}
	goalRef := strings.TrimSpace(request.GoalRef)
	if goalRef == "" {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	return adapter.withGoalIndexLockV0(goalRef, func() (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, error) {
		indexed, found, err := adapter.loadGoalIndexV0(goalRef)
		if err != nil {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, err
		}
		if !found {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
		}
		workspace, issues := (orquestaruntimeworktree.GitGoalWorkspaceProvisionerV0{}).ResolveGoalWorkspaceV0(ctx, adapter.requestFromIdentityV0(indexed.Identity))
		if len(issues) > 0 {
			return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
		}
		return codexGoalWorkspaceBindingFromWorkspaceV0(workspace), nil
	})
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
	goalRef string,
	fn func() (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, error),
) (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, error) {
	lockPath := adapter.goalIndexLockPathV0(goalRef)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	lock, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	return fn()
}

func (adapter codexGoalWorkspaceAdapterV0) loadGoalIndexV0(goalRef string) (codexGoalWorkspaceAdapterIndexV0, bool, error) {
	raw, err := os.ReadFile(adapter.goalIndexPathV0(goalRef))
	if errors.Is(err, os.ErrNotExist) {
		return codexGoalWorkspaceAdapterIndexV0{}, false, nil
	}
	if err != nil {
		return codexGoalWorkspaceAdapterIndexV0{}, false, errCodexGoalWorkspaceAdapterUnavailableV0
	}
	var index codexGoalWorkspaceAdapterIndexV0
	if json.Unmarshal(raw, &index) != nil || index.SchemaVersion != codexGoalWorkspaceAdapterIndexSchemaV0 || strings.TrimSpace(index.GoalRef) != strings.TrimSpace(goalRef) || strings.TrimSpace(index.Identity.GoalRef) != strings.TrimSpace(goalRef) {
		return codexGoalWorkspaceAdapterIndexV0{}, false, errCodexGoalWorkspaceAdapterConflictV0
	}
	return index, true, nil
}

func (adapter codexGoalWorkspaceAdapterV0) writeGoalIndexV0(goalRef string, request orquestaruntimeworktree.GoalWorkspaceRequestV0) error {
	path := adapter.goalIndexPathV0(goalRef)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	data, err := json.Marshal(codexGoalWorkspaceAdapterIndexV0{
		SchemaVersion: codexGoalWorkspaceAdapterIndexSchemaV0,
		GoalRef:       strings.TrimSpace(goalRef),
		Identity:      codexGoalWorkspaceRequestIdentityFromRequestV0(request),
	})
	if err != nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	if err := writeCommandDurableFileV0(path, append(data, '\n'), "codex_goal_workspace_index"); err != nil {
		return errCodexGoalWorkspaceAdapterUnavailableV0
	}
	return nil
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
