package application

import (
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type AlcanceEspacioPreservacionEntornoAgente string

const (
	PreservacionEntornoConEspacioTrabajo AlcanceEspacioPreservacionEntornoAgente = "workspace_bound"
	PreservacionEntornoSinEspacioTrabajo AlcanceEspacioPreservacionEntornoAgente = "workspace_absent"
)

type ComprobantePreservacionEntornoAgente struct {
	Ref, ClaveIdempotencia, DigestBindingEspacio, BaseOID, DigestCambio string
	// Optional as an atomic pair for historical A06 compatibility. The B12
	// SQLite writer cannot persist this pair until its separate migration 034.
	ManifiestoFisicoRef, ManifiestoFisicoDigest string
	ProyectoRef                                 goal.ProjectRef
	ObjetivoRef                                 goal.GoalRef
	ItemRef                                     goal.WorkItemRef
	EjecucionRef                                goal.ExecutionRef
	EspacioTrabajoRef                           ports.ExecutionWorkspaceRef
	FormatoObjeto                               ports.GitObjectFormat
	CambioRef                                   ports.ChangeSetRef
	AlcanceEspacio                              AlcanceEspacioPreservacionEntornoAgente
	Resultado                                   ports.ResultadoPreservacionEntornoAgente
	RegistradoEn                                time.Time
}

func ValidarComprobantePreservacionEntornoAgente(comprobante ComprobantePreservacionEntornoAgente) error {
	cambioPresente := comprobante.CambioRef.String() != "" || comprobante.DigestCambio != ""
	manifiestoPresente := comprobante.ManifiestoFisicoRef != "" || comprobante.ManifiestoFisicoDigest != ""
	if !validApplicationRef(comprobante.Ref) || !validApplicationRef(comprobante.ClaveIdempotencia) ||
		comprobante.ProyectoRef.String() == "" || comprobante.ObjetivoRef.String() == "" || comprobante.ItemRef.String() == "" ||
		comprobante.EjecucionRef != comprobante.Resultado.EjecucionRef ||
		(manifiestoPresente && (!validApplicationRef(comprobante.ManifiestoFisicoRef) ||
			!validEffectDigest(comprobante.ManifiestoFisicoDigest))) ||
		validarEspacioPreservacionEntornoAgente(comprobante, cambioPresente) != nil ||
		ports.ValidarResultadoPreservacionEntornoAgente(comprobante.Resultado) != nil ||
		comprobante.RegistradoEn.Before(comprobante.Resultado.PreservadoEn) {
		return errors.New("application.agent_environment_receipt_invalid")
	}
	return nil
}

func ValidarCausalidadPreservacionEntornoAgente(comprobante ComprobantePreservacionEntornoAgente, registro GoalRecord) error {
	if ValidarComprobantePreservacionEntornoAgente(comprobante) != nil || registro.Goal.Ref() != comprobante.ObjetivoRef ||
		registro.Goal.Project() != comprobante.ProyectoRef {
		return errors.New("application.agent_environment_receipt_scope_invalid")
	}
	sinEspacio := comprobante.AlcanceEspacio == PreservacionEntornoSinEspacioTrabajo
	ejecutada, ligada, lanzada, cambio := false, sinEspacio,
		false, comprobante.CambioRef.String() == ""
	for _, ejecucion := range registro.Executions {
		if ejecucion.Ref == comprobante.EjecucionRef {
			ejecutada = ejecucion.GoalRef == comprobante.ObjetivoRef && ejecucion.WorkItemRef == comprobante.ItemRef &&
				ejecucion.AttemptNo == comprobante.Resultado.IntentoEjecucion && ejecucion.ExternalRef == comprobante.Resultado.IdentidadExterna &&
				ejecucion.LaunchReceiptRef != ""
		}
	}
	for _, binding := range registro.WorkspaceBindings {
		if sinEspacio && binding.ExecutionRef == comprobante.EjecucionRef {
			ligada = false
			continue
		}
		if binding.Ref == comprobante.EspacioTrabajoRef {
			ligada = binding.ProjectRef == comprobante.ProyectoRef && binding.GoalRef == comprobante.ObjetivoRef &&
				binding.WorkItemRef == comprobante.ItemRef && binding.ExecutionRef == comprobante.EjecucionRef &&
				binding.Digest() == comprobante.DigestBindingEspacio && binding.BaseOID == comprobante.BaseOID &&
				binding.ObjectFormat == comprobante.FormatoObjeto
		}
	}
	for _, consumo := range registro.ConsumptionReceipts {
		if consumo.Kind == ActionLaunchAgent && consumo.ExecutionRef == comprobante.EjecucionRef &&
			consumo.Fence == comprobante.Resultado.Cerca && consumo.Outcome == ActionConsumedCompleted {
			lanzada = true
		}
	}
	for _, candidato := range registro.ChangeSets {
		if sinEspacio && candidato.ExecutionRef == comprobante.EjecucionRef {
			cambio = false
			continue
		}
		if candidato.Ref == comprobante.CambioRef {
			cambio = candidato.ProjectRef == comprobante.ProyectoRef && candidato.GoalRef == comprobante.ObjetivoRef &&
				candidato.WorkItemRef == comprobante.ItemRef && candidato.ExecutionRef == comprobante.EjecucionRef &&
				candidato.Digest() == comprobante.DigestCambio
		}
	}
	if !ejecutada || !ligada || !lanzada || !cambio {
		return errors.New("application.agent_environment_receipt_causality_invalid")
	}
	return nil
}

func validarEspacioPreservacionEntornoAgente(comprobante ComprobantePreservacionEntornoAgente, cambioPresente bool) error {
	switch comprobante.AlcanceEspacio {
	case "", PreservacionEntornoConEspacioTrabajo:
		if comprobante.EspacioTrabajoRef.String() == "" || !validEffectDigest(comprobante.DigestBindingEspacio) ||
			ports.ValidateGitOID(comprobante.BaseOID, comprobante.FormatoObjeto) != nil ||
			(cambioPresente && (comprobante.CambioRef.String() == "" || !validEffectDigest(comprobante.DigestCambio))) {
			return errors.New("application.agent_environment_workspace_invalid")
		}
	case PreservacionEntornoSinEspacioTrabajo:
		if comprobante.EspacioTrabajoRef.String() != "" || comprobante.DigestBindingEspacio != "" ||
			comprobante.BaseOID != "" || comprobante.FormatoObjeto != "" || cambioPresente {
			return errors.New("application.agent_environment_workspace_absence_invalid")
		}
	default:
		return errors.New("application.agent_environment_workspace_scope_invalid")
	}
	return nil
}

func PermiteTerminalizarEntornoAgente(ejecucion ExecutionRecord, preservada bool) bool {
	return !ejecucion.RequierePreservacionEntorno || !terminalExecutionState(ejecucion.State) || preservada
}
