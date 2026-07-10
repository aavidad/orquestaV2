package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

const (
	stackRunControlEscalatorEvidenceCleanedV0      = "evidence-ref-run-control-backend-stop-cleaned"
	stackRunControlEscalatorEvidenceNoActiveWorkV0 = "evidence-ref-run-control-backend-stop-no-active-work"
)

// stackRunControlBackendStopEscalatorV0 implementa la escalada real de parada
// del backend goal sobre los puertos de shutdown: lee el trabajo activo,
// filtra por identidad (run_ref/work_ref) y solo limpia lo que pertenece al
// run escalado; confirma stopped reobservando, no por autodeclaracion.
type stackRunControlBackendStopEscalatorV0 struct {
	Reader  orquestaservershutdown.ActiveShutdownWorkReaderPortV0
	Cleaner orquestaservershutdown.ActiveShutdownWorkCleanerPortV0
}

var _ orquestamcp.MCPRunControlBackendStopEscalatorPortV0 = stackRunControlBackendStopEscalatorV0{}

func (escalator stackRunControlBackendStopEscalatorV0) EscalateBackendStopV0(
	ctx context.Context,
	request orquestamcp.MCPRunControlBackendStopEscalationRequestV0,
) (orquestamcp.MCPRunControlBackendStopEscalationResultV0, error) {
	if escalator.Reader == nil || escalator.Cleaner == nil {
		return orquestamcp.MCPRunControlBackendStopEscalationResultV0{}, errors.New("backend_stop_escalator_ports_unbound")
	}
	if strings.TrimSpace(request.RunRef) == "" {
		return orquestamcp.MCPRunControlBackendStopEscalationResultV0{}, errors.New("backend_stop_escalator_run_ref_required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	active, err := escalator.Reader.ReadActiveShutdownWorkV0(ctx, orquestaservershutdown.ActiveShutdownWorkRequestV0{
		EvidenceRefs: compactStringsV0(request.EvidenceRefs),
	})
	if err != nil {
		return orquestamcp.MCPRunControlBackendStopEscalationResultV0{}, err
	}
	matched := stackRunControlEscalatorMatchingWorksV0(active.ActiveWorks, request)
	if len(matched) == 0 {
		return orquestamcp.MCPRunControlBackendStopEscalationResultV0{
			Stopped: true,
			EvidenceRefs: compactStringsV0(append(
				[]string{stackRunControlEscalatorEvidenceNoActiveWorkV0},
				active.EvidenceRefs...,
			)),
		}, nil
	}
	cleaned, err := escalator.Cleaner.CleanupActiveShutdownWorkV0(ctx, orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0{
		EvidenceRefs:        compactStringsV0(request.EvidenceRefs),
		ActiveWorks:         matched,
		CleanupGoalBackends: true,
	})
	if err != nil {
		return orquestamcp.MCPRunControlBackendStopEscalationResultV0{}, err
	}
	evidenceRefs := compactStringsV0(append(
		[]string{stackRunControlEscalatorEvidenceCleanedV0},
		cleaned.EvidenceRefs...,
	))
	after, err := escalator.Reader.ReadActiveShutdownWorkV0(ctx, orquestaservershutdown.ActiveShutdownWorkRequestV0{
		EvidenceRefs: compactStringsV0(request.EvidenceRefs),
	})
	if err != nil {
		return orquestamcp.MCPRunControlBackendStopEscalationResultV0{}, err
	}
	residual := stackRunControlEscalatorMatchingWorksV0(after.ActiveWorks, request)
	if len(residual) > 0 {
		residualRefs := make([]string, 0, len(residual))
		for _, work := range residual {
			residualRefs = append(residualRefs, firstNonEmptyQueuedSourceV0(work.WorkRef, work.ExternalWorkRef, work.RunRef))
		}
		return orquestamcp.MCPRunControlBackendStopEscalationResultV0{
			Stopped:      false,
			ResidualRefs: compactStringsV0(residualRefs),
			EvidenceRefs: evidenceRefs,
		}, nil
	}
	return orquestamcp.MCPRunControlBackendStopEscalationResultV0{
		Stopped:      true,
		EvidenceRefs: evidenceRefs,
	}, nil
}

func stackRunControlEscalatorMatchingWorksV0(
	works []orquestaservershutdown.ActiveShutdownWorkV0,
	request orquestamcp.MCPRunControlBackendStopEscalationRequestV0,
) []orquestaservershutdown.ActiveShutdownWorkV0 {
	runRef := strings.TrimSpace(request.RunRef)
	goalRef := strings.TrimSpace(request.GoalRef)
	externalGoalRef := strings.TrimSpace(request.ExternalGoalRef)
	matched := make([]orquestaservershutdown.ActiveShutdownWorkV0, 0, len(works))
	for _, work := range works {
		switch {
		case strings.TrimSpace(work.RunRef) == runRef && runRef != "":
		case goalRef != "" && strings.TrimSpace(work.WorkRef) == goalRef:
		case externalGoalRef != "" && strings.TrimSpace(work.ExternalWorkRef) == externalGoalRef:
		default:
			continue
		}
		matched = append(matched, work)
	}
	return matched
}
