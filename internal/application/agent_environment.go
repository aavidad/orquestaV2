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
