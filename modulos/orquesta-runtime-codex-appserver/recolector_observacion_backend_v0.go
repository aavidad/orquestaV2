package orquestaruntimecodexappserver

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
)

type ResultadoObservacionBackendV0 struct {
	Observacion                       ObservacionBackendV0
	Estado                            EstadoBackendAppServerV0
	Issues                            []string
	EvidenceRefs                      []string
	PanePID                           string
	ProcessByRuntimeWorkdirObserved   bool
	ProcessByRuntimeOwnedPathObserved bool
}

type solicitudObservacionBackendV0 struct {
	Actual                EstadoBackendAppServerV0
	ActionRequested       string
	TmuxPath              string
	Preflight             serverCodexAppServerProbePortV0
	CheckPreflight        bool
	CheckProcesses        bool
	RequireTmux           bool
	SessionUnownedIsIssue bool
}

func (backend serverCodexAppServerTmuxBackendV0) recolectarObservacionBackendV0(
	ctx context.Context,
	request solicitudObservacionBackendV0,
) (ResultadoObservacionBackendV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	snapshot := SnapshotObservacionBackendV0{
		ActionRequested:     request.ActionRequested,
		OwnerMarkerObserved: backend.tmuxOwnerMarkerObservedV0(),
		OwnerMarkerValid:    backend.tmuxOwnerMarkerExistsV0(),
		SocketObserved:      codexAppServerTmuxSocketPresentV0(backend.SocketPath),
		IssueCode:           codexAppServerIssueCodeFromLogFileV0(backend.tmuxLogPathV0()),
	}
	tmuxPath, tmuxErr := backend.tmuxPathForObservationV0(request.TmuxPath)
	if tmuxErr != nil && request.RequireTmux {
		return ResultadoObservacionBackendV0{}, tmuxErr
	}
	result := ResultadoObservacionBackendV0{}
	if tmuxErr == nil {
		hasSession, err := backend.tmuxHasSessionV0(ctx, tmuxPath)
		if err != nil {
			if request.RequireTmux {
				return ResultadoObservacionBackendV0{}, err
			}
			snapshot.IssueCode = firstNonEmptyServerStackV0(
				snapshot.IssueCode,
				CodexAppServerIssueCodeForErrorV0(err, "codex_app_server_tmux_has_session_failed"),
			)
		}
		if hasSession {
			snapshot.SessionObserved = true
			result.PanePID = backend.tmuxPanePIDV0(ctx, tmuxPath)
			snapshot.PanePIDLive = codexAppServerTmuxPIDAliveV0(result.PanePID)
			if request.SessionUnownedIsIssue && !snapshot.OwnerMarkerValid {
				snapshot.IssueCode = firstNonEmptyServerStackV0(
					snapshot.IssueCode,
					"codex_app_server_tmux_session_unowned",
				)
			}
		}
	}
	if request.CheckPreflight && snapshot.SocketObserved && backend.ensureTmuxSocketPrivateV0() == nil && request.Preflight != nil {
		snapshot.PreflightOK = request.Preflight.ProbeV0(ctx) == nil
	}
	if request.CheckProcesses {
		snapshot.ProcessBySocketObserved = backend.detectCodexAppServerConfiguredSocketProcessV0(ctx)
		result.ProcessByRuntimeWorkdirObserved = backend.detectCodexAppServerRuntimeWorkdirProcessV0(ctx)
		result.ProcessByRuntimeOwnedPathObserved = backend.detectCodexAppServerRuntimeOwnedProcessV0(ctx)
		snapshot.ProcessByRuntimeObserved = result.ProcessByRuntimeWorkdirObserved || result.ProcessByRuntimeOwnedPathObserved
	}
	result.Observacion = RecolectarObservacionBackendV0(snapshot)
	result.Estado, result.Issues = TransicionBackendV0(request.Actual, result.Observacion)
	result.EvidenceRefs = result.evidenceRefsV0()
	return result, nil
}

func (result ResultadoObservacionBackendV0) evidenceRefsV0() []string {
	refs := []string{}
	obs := result.Observacion
	if obs.OwnerMarkerObserved {
		refs = append(refs, "evidence-ref-codex-app-server-tmux-owner-marker")
	}
	if obs.SocketObserved {
		refs = append(refs, "evidence-ref-codex-app-server-tmux-socket")
	}
	if obs.SessionObserved {
		refs = append(refs, "evidence-ref-codex-app-server-tmux-session")
	}
	if obs.PanePIDLive {
		refs = append(refs, "evidence-ref-codex-app-server-tmux-process")
	}
	if obs.ProcessBySocketObserved {
		refs = append(refs, "evidence-ref-codex-app-server-tmux-process-cmdline")
	}
	if result.ProcessByRuntimeWorkdirObserved {
		refs = append(refs, "evidence-ref-codex-app-server-tmux-process-runtime-workdir")
	}
	if result.ProcessByRuntimeOwnedPathObserved {
		refs = append(refs, "evidence-ref-codex-app-server-tmux-process-runtime-owned")
	}
	if obs.IssueCode != "" {
		refs = append(refs, codexAppServerIssueEvidenceRefV0(obs.IssueCode))
	}
	return compactStringsV0(refs)
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxOwnerMarkerObservedV0() bool {
	path := backend.tmuxOwnerMarkerPathV0()
	if path == "" {
		return false
	}
	info, err := os.Lstat(path)
	return err == nil && !info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxPathForObservationV0(tmuxPath string) (string, error) {
	tmuxPath = strings.TrimSpace(tmuxPath)
	if tmuxPath != "" {
		return tmuxPath, nil
	}
	return codexAppServerTmuxCommandPathV0(backend.PathEnv)
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxHasSessionV0(
	ctx context.Context,
	tmuxPath string,
) (bool, error) {
	return backend.tmuxHasSessionTargetV0(ctx, tmuxPath, backend.tmuxExactSessionTargetV0())
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxHasSessionTargetV0(
	ctx context.Context,
	tmuxPath string,
	target string,
) (bool, error) {
	target = strings.TrimSpace(target)
	output, err := backend.runTmuxCommandV0(ctx, tmuxPath, "has-session", "-t", target)
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	message := strings.TrimSpace(output)
	wantName := strings.TrimPrefix(target, "=")
	sessionName := strings.TrimSuffix(wantName, ":")
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 &&
		(message == "can't find session: "+sessionName || message == "can't find session: "+wantName ||
			message == "can't find session: "+target ||
			strings.HasPrefix(message, "no server running on ") ||
			(strings.HasPrefix(message, "error connecting to ") && strings.HasSuffix(message, "(No such file or directory)"))) {
		return false, nil
	}
	return false, codexAppServerTmuxCommandErrorV0("codex_app_server_tmux_has_session_failed", output, err)
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxPanePIDV0(
	ctx context.Context,
	tmuxPath string,
) string {
	output, err := backend.runTmuxCommandV0(
		ctx,
		tmuxPath,
		"display-message",
		"-p",
		"-t",
		backend.tmuxExactSessionTargetV0(),
		"#{pane_pid}",
	)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(output)
}
