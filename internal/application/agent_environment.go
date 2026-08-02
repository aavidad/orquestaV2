package application

import (
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type ComprobantePreservacionEntornoAgente struct {
	Ref, ClaveIdempotencia, DigestBindingEspacio, BaseOID, DigestCambio string
	ProyectoRef                                                         goal.ProjectRef
	ObjetivoRef                                                         goal.GoalRef
	ItemRef                                                             goal.WorkItemRef
	EjecucionRef                                                        goal.ExecutionRef
	EspacioTrabajoRef                                                   ports.ExecutionWorkspaceRef
	FormatoObjeto                                                       ports.GitObjectFormat
	CambioRef                                                           ports.ChangeSetRef
	Resultado                                                           ports.ResultadoPreservacionEntornoAgente
	RegistradoEn                                                        time.Time
}

func ValidarComprobantePreservacionEntornoAgente(comprobante ComprobantePreservacionEntornoAgente) error {
	cambioPresente := comprobante.CambioRef.String() != "" || comprobante.DigestCambio != ""
	if !validApplicationRef(comprobante.Ref) || !validApplicationRef(comprobante.ClaveIdempotencia) ||
		comprobante.ProyectoRef.String() == "" || comprobante.ObjetivoRef.String() == "" || comprobante.ItemRef.String() == "" ||
		comprobante.EjecucionRef != comprobante.Resultado.EjecucionRef ||
		comprobante.EspacioTrabajoRef.String() == "" || !validEffectDigest(comprobante.DigestBindingEspacio) ||
		ports.ValidateGitOID(comprobante.BaseOID, comprobante.FormatoObjeto) != nil ||
		(cambioPresente && (comprobante.CambioRef.String() == "" || !validEffectDigest(comprobante.DigestCambio))) ||
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
	ejecutada, ligada, lanzada, cambio := false, false, false, comprobante.CambioRef.String() == ""
	for _, ejecucion := range registro.Executions {
		if ejecucion.Ref == comprobante.EjecucionRef {
			ejecutada = ejecucion.GoalRef == comprobante.ObjetivoRef && ejecucion.WorkItemRef == comprobante.ItemRef &&
				ejecucion.AttemptNo == comprobante.Resultado.IntentoEjecucion && ejecucion.ExternalRef == comprobante.Resultado.IdentidadExterna &&
				ejecucion.LaunchReceiptRef != ""
		}
	}
	for _, binding := range registro.WorkspaceBindings {
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
