package orquestaruntimecodexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"syscall"
	"time"

	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

// CleanupOwnedGenerationAfterStartupFailureV0 cleans only the generation
// described by the owner marker beside this backend's configured socket.
func (backend serverCodexAppServerTmuxBackendV0) CleanupOwnedGenerationAfterStartupFailureV0(
	ctx context.Context,
) error {
	marker, present, err := backend.readTmuxGenerationMarkerForCleanupV0()
	if err != nil || !present {
		return err
	}
	return backend.shutdownTmuxGenerationLeaseV0(ctx, marker, true)
}

func (backend serverCodexAppServerTmuxBackendV0) readTmuxGenerationMarkerForCleanupV0() (
	codexAppServerTmuxOwnerMarkerV0,
	bool,
	error,
) {
	path := strings.TrimSpace(backend.tmuxOwnerMarkerPathV0())
	if path == "" {
		return codexAppServerTmuxOwnerMarkerV0{}, false, codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_config_missing", Err: errors.New("codex_app_server_tmux_config_missing"),
		}
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return codexAppServerTmuxOwnerMarkerV0{}, false, nil
	}
	if err != nil {
		return codexAppServerTmuxOwnerMarkerV0{}, false, codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_owner_marker_read_failed", Err: err,
		}
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return codexAppServerTmuxOwnerMarkerV0{}, false,
			codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return codexAppServerTmuxOwnerMarkerV0{}, false, codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_owner_marker_read_failed", Err: err,
		}
	}
	var marker codexAppServerTmuxOwnerMarkerV0
	if err := json.Unmarshal(raw, &marker); err != nil {
		return codexAppServerTmuxOwnerMarkerV0{}, false, codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_owner_marker_invalid", Err: err,
		}
	}
	if !backend.tmuxGenerationMarkerMatchesBackendV0(marker) {
		return codexAppServerTmuxOwnerMarkerV0{}, false,
			codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	current, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || !reflect.DeepEqual(current, marker) {
		return codexAppServerTmuxOwnerMarkerV0{}, false,
			codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	return marker, true, nil
}

func (backend serverCodexAppServerTmuxBackendV0) stopExactTmuxGenerationProcessV0(
	ctx context.Context,
	guard *codexAppServerTmuxLeaseGuardV0,
	marker codexAppServerTmuxOwnerMarkerV0,
) error {
	if err := backend.requireExactTmuxGenerationForSignalV0(guard, marker); err != nil {
		return err
	}
	if !codexAppServerTmuxProcessIdentityAliveV0(marker.AppServerPID, marker.AppServerStartRef) {
		return nil
	}
	if err := backend.signalExactTmuxGenerationProcessV0(marker, syscall.SIGTERM); err != nil {
		return err
	}
	termTimeout := backend.shutdownCleanupTimeoutV0() / 2
	if termTimeout > 500*time.Millisecond {
		termTimeout = 500 * time.Millisecond
	}
	if termTimeout < 50*time.Millisecond {
		termTimeout = 50 * time.Millisecond
	}
	termCtx, termCancel := context.WithTimeout(ctx, termTimeout)
	termErr := waitCodexAppServerTmuxProcessIdentityGoneV0(termCtx, marker.AppServerPID, marker.AppServerStartRef)
	termCancel()
	if termErr == nil {
		return nil
	}
	if err := backend.requireExactTmuxGenerationForSignalV0(guard, marker); err != nil {
		return err
	}
	if !codexAppServerTmuxProcessIdentityAliveV0(marker.AppServerPID, marker.AppServerStartRef) {
		return nil
	}
	if err := backend.signalExactTmuxGenerationProcessV0(marker, syscall.SIGKILL); err != nil {
		return err
	}
	return waitCodexAppServerTmuxProcessIdentityGoneV0(ctx, marker.AppServerPID, marker.AppServerStartRef)
}

func (backend serverCodexAppServerTmuxBackendV0) requireExactTmuxGenerationForSignalV0(
	guard *codexAppServerTmuxLeaseGuardV0,
	marker codexAppServerTmuxOwnerMarkerV0,
) error {
	if err := backend.requireTmuxLeaseV0(guard); err != nil {
		return err
	}
	current, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || !reflect.DeepEqual(current, marker) || !backend.tmuxGenerationMarkerMatchesBackendV0(current) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) signalExactTmuxGenerationProcessV0(
	marker codexAppServerTmuxOwnerMarkerV0,
	signal syscall.Signal,
) error {
	if !codexAppServerTmuxProcessIdentityAliveV0(marker.AppServerPID, marker.AppServerStartRef) {
		return nil
	}
	if backend.signalProcessIdentityV0 != nil {
		if err := backend.signalProcessIdentityV0(marker.AppServerPID, marker.AppServerStartRef, signal); err != nil {
			return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_process_signal_failed", Err: err}
		}
		return nil
	}
	if err := syscall.Kill(marker.AppServerPID, signal); err != nil {
		if errors.Is(err, syscall.ESRCH) && !codexAppServerTmuxProcessIdentityAliveV0(marker.AppServerPID, marker.AppServerStartRef) {
			return nil
		}
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_process_signal_failed", Err: err}
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) CleanupActiveShutdownWorkV0(
	ctx context.Context,
	command orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0,
) (orquestaservershutdown.ActiveShutdownWorkCleanupResultV0, error) {
	if !command.CleanupGoalBackends {
		return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}, nil
	}
	evidenceRefs := compactStringsV0(append(
		append([]string(nil), command.EvidenceRefs...),
		"evidence-ref-codex-app-server-tmux-cleanup-requested",
	))
	cleaned := 0
	if backend.detectTmuxResidueV0(ctx).Active {
		if err := backend.shutdownTmuxSessionForCleanupV0(ctx); err != nil {
			if codexAppServerTmuxIsGenerationConflictV0(err) || codexAppServerTmuxIsObservationTransientV0(err) {
				residualEvidence := "evidence-ref-codex-app-server-tmux-generation-observation-residual"
				if codexAppServerTmuxIsGenerationConflictV0(err) {
					residualEvidence = "evidence-ref-codex-app-server-tmux-generation-conflict-residual"
				}
				return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{
					EvidenceRefs: compactStringsV0(append(
						evidenceRefs,
						residualEvidence,
					)),
				}, nil
			}
			return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}, err
		}
		if !backend.detectTmuxResidueV0(ctx).Active {
			cleaned++
			evidenceRefs = append(evidenceRefs, "evidence-ref-codex-app-server-tmux-configured-cleaned")
		}
	}
	return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{
		CleanedWorkCount: cleaned,
		EvidenceRefs:     compactStringsV0(evidenceRefs),
	}, nil
}
