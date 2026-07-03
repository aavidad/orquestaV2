package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type EvidenciaEstadoReceiptsV0 struct {
	GoalStateStore orquestagoal.GoalWorkStateStorePortV0
	ReceiptStore   orquestaruntimecodexdelivery.CodexReceiptDescriptorStorePortV0
}

var _ orquestaestadovivo.FuenteEvidenciaEstadoPortV0 = EvidenciaEstadoReceiptsV0{}

func (source EvidenciaEstadoReceiptsV0) ListarEvidenciasEstadoV0(
	ctx context.Context,
	filtro orquestaestadovivo.FiltroEvidenciaEstadoV0,
) ([]orquestaestadovivo.EvidenciaEstadoV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	filtro = normalizarFiltroEvidenciaEstadoV0(filtro)
	evidencias := []orquestaestadovivo.EvidenciaEstadoV0{}
	goalResults, err := source.listarGoalResultEvidenciasV0(ctx, filtro)
	if err != nil {
		return nil, err
	}
	evidencias = append(evidencias, goalResults...)
	if filtro.Limit > 0 && len(evidencias) >= filtro.Limit {
		return limitarEvidenciasEstadoV0(evidencias, filtro.Limit), nil
	}
	ackReceipts, err := source.listarCodexAckReceiptEvidenciasV0(ctx, filtro)
	if err != nil {
		return nil, err
	}
	evidencias = append(evidencias, ackReceipts...)
	return limitarEvidenciasEstadoV0(evidencias, filtro.Limit), nil
}

func (source EvidenciaEstadoReceiptsV0) listarGoalResultEvidenciasV0(
	ctx context.Context,
	filtro orquestaestadovivo.FiltroEvidenciaEstadoV0,
) ([]orquestaestadovivo.EvidenciaEstadoV0, error) {
	if source.GoalStateStore == nil {
		return nil, nil
	}
	states, err := (EvidenciaEstadoGoalStateV0{Store: source.GoalStateStore}).listarGoalStatesV0(ctx, filtro)
	if err != nil {
		return nil, err
	}
	evidencias := make([]orquestaestadovivo.EvidenciaEstadoV0, 0, len(states))
	for _, state := range states {
		state, err = orquestagoal.NewGoalWorkStateV0(state)
		if err != nil {
			return nil, err
		}
		if state.LastResult == nil {
			continue
		}
		result := orquestagoal.NormalizeGoalWorkResultV0(*state.LastResult)
		validado := len(orquestagoal.ValidateGoalWorkResultV0(result)) == 0
		terminal := validado && orquestagoal.GoalWorkResultTerminalV0(result.Status)
		aceptado := terminal && state.LastClosure != nil && state.LastClosure.Accepted
		evidenceRefs := evidenciaEstadoGoalResultRefsV0(result)
		if state.LastClosure != nil {
			evidenceRefs = compactStringsV0(append(evidenceRefs, state.LastClosure.EvidenceRefs...))
		}
		evidencia := normalizarEvidenciaEstadoV0(orquestaestadovivo.EvidenciaEstadoV0{
			RunRef:          state.RunRef,
			GoalRef:         firstNonEmptyQueuedSourceV0(result.GoalRef, state.GoalRef),
			ExternalGoalRef: firstNonEmptyQueuedSourceV0(result.ExternalGoalRef, state.ExternalGoalRef),
			Fuente:          evidenciaEstadoFuenteReceiptV0,
			Estado:          strings.TrimSpace(result.Status),
			Terminal:        terminal,
			Aceptado:        aceptado,
			EvidenceRefs:    evidenceRefs,
		})
		if !evidenciaEstadoPasaFiltroV0(evidencia, filtro) {
			continue
		}
		evidencias = append(evidencias, evidencia)
	}
	return limitarEvidenciasEstadoV0(evidencias, filtro.Limit), nil
}

func (source EvidenciaEstadoReceiptsV0) listarCodexAckReceiptEvidenciasV0(
	ctx context.Context,
	filtro orquestaestadovivo.FiltroEvidenciaEstadoV0,
) ([]orquestaestadovivo.EvidenciaEstadoV0, error) {
	if source.ReceiptStore == nil || filtro.GoalRef != "" {
		return nil, nil
	}
	descriptors, err := source.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: filtro.RunRef},
	)
	if err != nil {
		return nil, err
	}
	evidencias := make([]orquestaestadovivo.EvidenciaEstadoV0, 0, len(descriptors))
	for _, descriptor := range descriptors {
		ack, issues := orquestaruntimecodex.ReadAndValidateCodexDeliveryAckFileV0(
			strings.TrimSpace(descriptor.AckPath),
			descriptor.Spec,
		)
		if len(issues) > 0 {
			continue
		}
		evidencia := normalizarEvidenciaEstadoV0(orquestaestadovivo.EvidenciaEstadoV0{
			RunRef:       descriptor.RunID,
			Fuente:       evidenciaEstadoFuenteReceiptV0,
			Estado:       strings.TrimSpace(ack.Status),
			EvidenceRefs: evidenciaEstadoAckRefsV0(ack, descriptor),
		})
		if !evidenciaEstadoPasaFiltroV0(evidencia, filtro) {
			continue
		}
		evidencias = append(evidencias, evidencia)
		if filtro.Limit > 0 && len(evidencias) >= filtro.Limit {
			return limitarEvidenciasEstadoV0(evidencias, filtro.Limit), nil
		}
	}
	return limitarEvidenciasEstadoV0(evidencias, filtro.Limit), nil
}
