package orquestacore

import (
	"strings"
)

func construirProyectoPlanBorradorV0(cmd RegistrarProyectoDesdeAppSpecCommandV0) ProyectoPlanBorradorV0 {
	items := mapMicrotareasV0(cmd.Backlog.Microtareas)
	return ProyectoPlanBorradorV0{
		ProyectoIDPropuesto:    stableIDV0("proyecto", cmd.IdempotencyKey),
		Nombre:                 strings.TrimSpace(cmd.AppSpec.App.Nombre),
		Estado:                 EstadoProyectoPlanBorradorV0,
		AppSpecRef:             strings.TrimSpace(cmd.AppSpec.SpecID),
		RequestID:              firstNonEmptyV0(cmd.RequestID, cmd.AppSpec.RequestID),
		FasesIniciales:         mapFasesV0(cmd.Backlog.Fases),
		Microtareas:            items,
		BacklogNormalizado:     items,
		ContratosRequeridos:    compactStringsV0(cmd.Backlog.ContratosRequeridos),
		CriteriosCierre:        criteriosCierreV0(cmd),
		DependenciasPendientes: compactStringsV0(append(cmd.Backlog.Riesgos, cmd.Backlog.PreguntasAbiertas...)),
	}
}

func mapFasesV0(fases []RegistrarFaseInicialV0) []FasePlanificadaV0 {
	result := make([]FasePlanificadaV0, 0, len(fases))
	for _, fase := range fases {
		result = append(result, FasePlanificadaV0{
			ID:               strings.TrimSpace(fase.ID),
			Nombre:           strings.TrimSpace(fase.Nombre),
			Orden:            fase.Orden,
			Estado:           EstadoFasePlanificadaPendienteV0,
			CriteriosEntrada: []string{},
			CriteriosSalida:  compactStringsV0([]string{fase.Objetivo}),
		})
	}
	return result
}

func mapMicrotareasV0(tasks []RegistrarMicrotareaV0) []ItemBacklogCoreV0 {
	result := make([]ItemBacklogCoreV0, 0, len(tasks))
	for index, task := range tasks {
		result = append(result, ItemBacklogCoreV0{
			ID:                  stableBacklogItemIDV0(index, task.ID),
			Titulo:              strings.TrimSpace(task.Objetivo),
			Descripcion:         strings.TrimSpace(task.Objetivo),
			FaseID:              strings.TrimSpace(task.Fase),
			ModuloSugerido:      strings.TrimSpace(task.ModuloSugerido),
			Prioridad:           prioridadV0(index),
			Dependencias:        compactStringsV0(task.Bloqueos),
			CriteriosAceptacion: compactStringsV0([]string{task.Validacion}),
			ContratoRequerido:   strings.TrimSpace(task.Contrato),
			WriteSetPrevisto:    compactStringsV0(task.WriteSetPrevisto),
			FuenteBacklogID:     strings.TrimSpace(task.ID),
		})
	}
	return result
}

func proyectoRegistradoEventoV0(cmd RegistrarProyectoDesdeAppSpecCommandV0, plan ProyectoPlanBorradorV0) EventoDominioCoreV0 {
	return EventoDominioCoreV0{
		Tipo:           EventoProyectoRegistradoEnBorradorV0,
		ProyectoID:     plan.ProyectoIDPropuesto,
		OccurredAt:     occurredAtV0(cmd),
		PayloadVersion: payloadVersionProyectoBorradorEventoV0,
		Payload: map[string]any{
			"app_spec_ref":         plan.AppSpecRef,
			"fases":                len(plan.FasesIniciales),
			"microtareas":          len(plan.Microtareas),
			"contratos_requeridos": plan.ContratosRequeridos,
		},
		CorrelationID: strings.TrimSpace(cmd.CorrelationID),
	}
}

func criteriosCierreV0(cmd RegistrarProyectoDesdeAppSpecCommandV0) []string {
	criterios := []string{"Microtareas propuestas revisadas y convertibles a FunctionContract v0."}
	criterios = append(criterios, cmd.Backlog.ContratosRequeridos...)
	return compactStringsV0(criterios)
}

func prioridadV0(index int) string {
	if index < 3 {
		return "alta"
	}
	return "normal"
}
