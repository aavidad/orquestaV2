package orquestaappcodexstack

import (
	"reflect"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

const (
	evidenciaEstadoFuenteRunStoreV0        = "run_store"
	evidenciaEstadoFuenteGoalStateV0       = "goal_state"
	evidenciaEstadoFuenteRunMarkerV0       = "run_marker"
	evidenciaEstadoFuenteProcessRegistryV0 = "process_registry"
	evidenciaEstadoFuenteProcessSnapshotV0 = "process_snapshot"
	evidenciaEstadoFuenteReceiptV0         = "receipt"
)

func normalizarFiltroEvidenciaEstadoV0(
	filtro orquestaestadovivo.FiltroEvidenciaEstadoV0,
) orquestaestadovivo.FiltroEvidenciaEstadoV0 {
	filtro.RunRef = strings.TrimSpace(filtro.RunRef)
	filtro.GoalRef = strings.TrimSpace(filtro.GoalRef)
	if filtro.Limit < 0 {
		filtro.Limit = 0
	}
	return filtro
}

func normalizarEvidenciaEstadoV0(
	evidencia orquestaestadovivo.EvidenciaEstadoV0,
) orquestaestadovivo.EvidenciaEstadoV0 {
	evidencia.RunRef = strings.TrimSpace(evidencia.RunRef)
	evidencia.GoalRef = strings.TrimSpace(evidencia.GoalRef)
	evidencia.ExternalGoalRef = strings.TrimSpace(evidencia.ExternalGoalRef)
	evidencia.Fuente = strings.TrimSpace(evidencia.Fuente)
	evidencia.Estado = strings.TrimSpace(evidencia.Estado)
	evidencia.ObservadoEn = strings.TrimSpace(evidencia.ObservadoEn)
	evidencia.EvidenceRefs = compactStringsV0(evidencia.EvidenceRefs)
	return evidencia
}

func evidenciaEstadoPasaFiltroV0(
	evidencia orquestaestadovivo.EvidenciaEstadoV0,
	filtro orquestaestadovivo.FiltroEvidenciaEstadoV0,
) bool {
	filtro = normalizarFiltroEvidenciaEstadoV0(filtro)
	if filtro.RunRef != "" && strings.TrimSpace(evidencia.RunRef) != filtro.RunRef {
		return false
	}
	if filtro.GoalRef != "" && strings.TrimSpace(evidencia.GoalRef) != filtro.GoalRef {
		return false
	}
	return true
}

func limitarEvidenciasEstadoV0(
	evidencias []orquestaestadovivo.EvidenciaEstadoV0,
	limit int,
) []orquestaestadovivo.EvidenciaEstadoV0 {
	if limit <= 0 || len(evidencias) <= limit {
		return evidencias
	}
	return evidencias[:limit]
}

func fuenteEvidenciaEstadoNilV0(
	fuente orquestaestadovivo.FuenteEvidenciaEstadoPortV0,
) bool {
	if fuente == nil {
		return true
	}
	value := reflect.ValueOf(fuente)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func evidenciaEstadoRunRefsV0(run orquestacoreworkflow.OrchestrationRunV0) []string {
	refs := []string{run.LastEventID}
	refs = append(refs, run.Blockers...)
	refs = append(refs, run.Agents...)
	refs = append(refs, run.StartedAgents...)
	refs = append(refs, run.FailedAgents...)
	refs = append(refs, run.LostAgents...)
	refs = append(refs, run.StoppedAgents...)
	refs = append(refs, run.ConfirmedStoppedAgents...)
	refs = append(refs, run.PhaseArtifacts...)
	refs = append(refs, run.Deliveries...)
	refs = append(refs, run.DeliveredTasks...)
	refs = append(refs, run.DeliveredAgents...)
	refs = append(refs, run.Reviews...)
	refs = append(refs, run.ReviewResults...)
	refs = append(refs, run.ReworkRequests...)
	refs = append(refs, run.ReplanDecisions...)
	refs = append(refs, run.AcceptedReviews...)
	refs = append(refs, run.ClosedTasks...)
	refs = append(refs, run.Validations...)
	refs = append(refs, run.Closures...)
	return compactStringsV0(refs)
}

func evidenciaEstadoGoalStateRefsV0(state orquestagoal.GoalWorkStateV0) []string {
	refs := append([]string{}, state.EvidenceRefs...)
	refs = append(refs, state.Spec.EvidenceRefs...)
	refs = append(refs, state.LaunchReceipt.EvidenceRefs...)
	if state.LastResult != nil {
		refs = append(refs, evidenciaEstadoGoalResultRefsV0(*state.LastResult)...)
	}
	if state.LastClosure != nil {
		refs = append(refs, state.LastClosure.EvidenceRefs...)
	}
	return compactStringsV0(refs)
}

func evidenciaEstadoGoalMarkerRefsV0(marker orquestagoal.GoalWorkRunMarkerV0) []string {
	refs := append([]string{}, marker.EvidenceRefs...)
	if marker.Spec != nil {
		refs = append(refs, marker.Spec.EvidenceRefs...)
	}
	if marker.LaunchReceipt != nil {
		refs = append(refs, marker.LaunchReceipt.EvidenceRefs...)
	}
	return compactStringsV0(refs)
}

func evidenciaEstadoGoalResultRefsV0(result orquestagoal.GoalWorkResultV0) []string {
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	refs := append([]string{}, result.EvidenceRefs...)
	refs = append(refs, result.ArtifactRefs...)
	refs = append(refs, result.DomainReceiptRefs...)
	refs = append(refs, result.ReworkPlanRefs...)
	refs = append(refs, result.Checklist.EvidenceRefs...)
	for _, artifact := range result.MaterializedArtifacts {
		refs = append(refs, artifact.ArtifactRef)
		refs = append(refs, artifact.EvidenceRefs...)
	}
	for _, test := range result.RequiredTestResults {
		refs = append(refs, test.EvidenceRefs...)
	}
	return compactStringsV0(refs)
}

func evidenciaEstadoProcessRecordRefsV0(
	record orquestacionnucleoapp.AgentProcessRegistryRecordV0,
) []string {
	return compactStringsV0(append([]string{
		record.AgentRequestID,
		record.ProcessRef,
		record.SessionRef,
		record.LaunchRef,
		record.ReadinessRef,
	}, record.EvidenceRefs...))
}

func evidenciaEstadoSnapshotRefsV0(
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
) []string {
	return compactStringsV0(orquestaruntime.ProcessRuntimeSnapshotEvidenceRefsV0(snapshot))
}

func evidenciaEstadoProcessRecordMatchesSnapshotV0(
	record orquestacionnucleoapp.AgentProcessRegistryRecordV0,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
) bool {
	if strings.TrimSpace(record.ProcessRef) == "" ||
		strings.TrimSpace(record.ProcessRef) != strings.TrimSpace(snapshot.ProcessRef) {
		return false
	}
	if strings.TrimSpace(record.SessionRef) != "" &&
		strings.TrimSpace(snapshot.SessionRef) != "" &&
		strings.TrimSpace(record.SessionRef) != strings.TrimSpace(snapshot.SessionRef) {
		return false
	}
	if strings.TrimSpace(record.LaunchRef) != "" &&
		strings.TrimSpace(snapshot.LaunchRef) != "" &&
		strings.TrimSpace(record.LaunchRef) != strings.TrimSpace(snapshot.LaunchRef) {
		return false
	}
	return true
}

func evidenciaEstadoProcesoVivoConfirmadoV0(
	record orquestacionnucleoapp.AgentProcessRegistryRecordV0,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
) bool {
	if !evidenciaEstadoProcessRecordMatchesSnapshotV0(record, snapshot) {
		return false
	}
	switch snapshot.Status {
	case orquestaruntime.ProcessRuntimeRunningV0, orquestaruntime.ProcessRuntimeStoppingV0:
		return true
	default:
		return false
	}
}

func evidenciaEstadoReceiptDescriptorRefsV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) []string {
	packet := descriptor.Spec.AgentPacket
	return compactStringsV0([]string{
		descriptor.DescriptorRef,
		descriptor.AgentRef,
		packet.DeliveryRefs.AckRef,
		packet.DeliveryRefs.MailboxRef,
		packet.DeliveryRefs.ReadinessRef,
		packet.DeliveryRefs.CheckpointRef,
		descriptor.WorktreeBaselineRef,
	})
}

func evidenciaEstadoAckRefsV0(
	ack orquestaruntimecodex.CodexAgentAckV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) []string {
	refs := evidenciaEstadoReceiptDescriptorRefsV0(descriptor)
	refs = append(refs, ack.AckRef, ack.TaskRef, ack.RequestID)
	refs = append(refs, orquestaruntimecodex.CodexRequiredTestReceiptEvidenceRefsV0(ack.TestReceipts)...)
	return compactStringsV0(refs)
}
