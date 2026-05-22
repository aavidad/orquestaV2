package orquestacore

import (
	"strings"
)

func validarRegistrarProyectoCommandV0(cmd RegistrarProyectoDesdeAppSpecCommandV0) error {
	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		return registrarErrorV0(ErrIdempotencyKeyRequeridaV0, "idempotency_key requerida", "idempotency_key", cmd.CorrelationID)
	}
	if appSpecVaciaV0(cmd.AppSpec) {
		return registrarErrorV0(ErrAppSpecRequeridaV0, "app_spec requerida", "app_spec", cmd.CorrelationID)
	}
	if backlogVacioCompletoV0(cmd.Backlog) {
		return registrarErrorV0(ErrBacklogRequeridoV0, "backlog requerido", "backlog", cmd.CorrelationID)
	}
	if !versionesFactorySoportadasV0(cmd) {
		return registrarErrorV0(ErrContratoFactoryNoSoportadoV0, "contrato factory no soportado", "schema_version", cmd.CorrelationID)
	}
	if cmd.AppSpec.Validation.Estado != "valida" {
		return registrarErrorV0(ErrAppSpecNoValidadaV0, "app_spec no validada", "app_spec.validation.estado", cmd.CorrelationID)
	}
	if len(cmd.Backlog.Microtareas) == 0 {
		return registrarErrorV0(ErrBacklogVacioV0, "backlog sin microtareas", "backlog.microtareas", cmd.CorrelationID)
	}
	if strings.TrimSpace(cmd.Backlog.SpecID) != strings.TrimSpace(cmd.AppSpec.SpecID) {
		return registrarErrorV0(ErrBacklogIncompatibleConAppSpecV0, "backlog incompatible con app_spec", "backlog.spec_id", cmd.CorrelationID)
	}
	return nil
}

func versionesFactorySoportadasV0(cmd RegistrarProyectoDesdeAppSpecCommandV0) bool {
	if nonEmptyNotEqualV0(cmd.AppSpecVersion, RegistrarProyectoDesdeAppSpecVersionV0) {
		return false
	}
	if nonEmptyNotEqualV0(cmd.BacklogVersion, RegistrarProyectoDesdeAppSpecVersionV0) {
		return false
	}
	return cmd.AppSpec.SchemaVersion == RegistrarAppSpecSchemaV0 &&
		cmd.Backlog.SchemaVersion == RegistrarBacklogInicialSchemaV0
}

func appSpecVaciaV0(spec RegistrarAppSpecV0) bool {
	return strings.TrimSpace(spec.SchemaVersion) == "" &&
		strings.TrimSpace(spec.SpecID) == "" &&
		strings.TrimSpace(spec.App.Nombre) == ""
}

func backlogVacioCompletoV0(backlog RegistrarBacklogInicialV0) bool {
	return strings.TrimSpace(backlog.SchemaVersion) == "" &&
		strings.TrimSpace(backlog.SpecID) == "" &&
		len(backlog.Fases) == 0 &&
		len(backlog.Microtareas) == 0
}

func registrarErrorV0(code, message, field, correlationID string) RegistrarProyectoDesdeAppSpecErrorV0 {
	return RegistrarProyectoDesdeAppSpecErrorV0{
		Code:          code,
		Message:       message,
		Field:         field,
		Retryable:     false,
		CorrelationID: strings.TrimSpace(correlationID),
	}
}
