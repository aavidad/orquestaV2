package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type EvidenciaEstadoProcesosV0 struct {
	Registry       orquestacionnucleoapp.AgentProcessRegistryListPortV0
	SnapshotSource orquestaruntimecodexdelivery.CodexProcessSnapshotSourcePortV0
}

var _ orquestaestadovivo.FuenteEvidenciaEstadoPortV0 = EvidenciaEstadoProcesosV0{}

func (source EvidenciaEstadoProcesosV0) ListarEvidenciasEstadoV0(
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
	if source.Registry == nil || filtro.GoalRef != "" {
		return nil, nil
	}
	records, err := source.Registry.ListAgentProcessesV0(
		ctx,
		orquestacionnucleoapp.AgentProcessRegistryListFilterV0{RunID: filtro.RunRef},
	)
	if err != nil {
		return nil, err
	}
	evidencias := make([]orquestaestadovivo.EvidenciaEstadoV0, 0, len(records)*2)
	for _, record := range records {
		registryEvidence := normalizarEvidenciaEstadoV0(orquestaestadovivo.EvidenciaEstadoV0{
			RunRef:               record.RunID,
			Fuente:               evidenciaEstadoFuenteProcessRegistryV0,
			Estado:               "registered",
			Scope:                orquestaestadovivo.ScopeGoalExecutionV0,
			RuntimeIdentityRef:   strings.TrimSpace(record.ProcessRef),
			RuntimeGenerationRef: evidenciaEstadoRuntimeGenerationRefV0(record.SessionRef, record.LaunchRef),
			EvidenceRefs:         evidenciaEstadoProcessRecordRefsV0(record),
		})
		if evidenciaEstadoPasaFiltroV0(registryEvidence, filtro) {
			evidencias = append(evidencias, registryEvidence)
		}
		if source.SnapshotSource == nil || strings.TrimSpace(record.ProcessRef) == "" {
			continue
		}
		snapshot, err := source.SnapshotSource.SnapshotV0(strings.TrimSpace(record.ProcessRef))
		if err != nil {
			indeterminate := normalizarEvidenciaEstadoV0(orquestaestadovivo.EvidenciaEstadoV0{
				RunRef:                      record.RunID,
				Fuente:                      evidenciaEstadoFuenteProcessSnapshotV0,
				Estado:                      "observation_indeterminate",
				Scope:                       orquestaestadovivo.ScopeGoalExecutionV0,
				RuntimeIdentityRef:          strings.TrimSpace(record.ProcessRef),
				RuntimeGenerationRef:        evidenciaEstadoRuntimeGenerationRefV0(record.SessionRef, record.LaunchRef),
				RuntimeObservationAttempted: true,
				EvidenceRefs:                evidenciaEstadoProcessRecordRefsV0(record),
			})
			if evidenciaEstadoPasaFiltroV0(indeterminate, filtro) {
				evidencias = append(evidencias, indeterminate)
			}
			continue
		}
		snapshotEvidence := normalizarEvidenciaEstadoV0(orquestaestadovivo.EvidenciaEstadoV0{
			RunRef:                      record.RunID,
			Fuente:                      evidenciaEstadoFuenteProcessSnapshotV0,
			Estado:                      strings.TrimSpace(string(snapshot.Status)),
			Scope:                       orquestaestadovivo.ScopeGoalExecutionV0,
			RuntimeIdentityRef:          strings.TrimSpace(snapshot.ProcessRef),
			RuntimeGenerationRef:        evidenciaEstadoRuntimeGenerationRefV0(snapshot.SessionRef, snapshot.LaunchRef),
			RuntimeIdentityMismatch:     !evidenciaEstadoProcessRecordMatchesSnapshotV0(record, snapshot),
			RuntimeObservationAttempted: true,
			RuntimeObservado:            true,
			ProcesoVivo:                 evidenciaEstadoProcesoVivoConfirmadoV0(record, snapshot),
			EvidenceRefs:                compactStringsV0(append(evidenciaEstadoProcessRecordRefsV0(record), evidenciaEstadoSnapshotRefsV0(snapshot)...)),
		})
		if evidenciaEstadoPasaFiltroV0(snapshotEvidence, filtro) {
			evidencias = append(evidencias, snapshotEvidence)
		}
		if filtro.Limit > 0 && len(evidencias) >= filtro.Limit {
			return limitarEvidenciasEstadoV0(evidencias, filtro.Limit), nil
		}
	}
	return limitarEvidenciasEstadoV0(evidencias, filtro.Limit), nil
}

func evidenciaEstadoRuntimeGenerationRefV0(sessionRef, launchRef string) string {
	if launchRef = strings.TrimSpace(launchRef); launchRef != "" {
		return launchRef
	}
	return strings.TrimSpace(sessionRef)
}
