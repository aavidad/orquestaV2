package orquestacontext

import (
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

const contextBundleRequestRailBoundaryV0 = "context_bundle_request"

func ValidateContextBundleRequestV0(request ContextBundleRequestV0) []ContextBundleIssueV0 {
	request = normalizeContextBundleRequestV0(request)
	var issues []ContextBundleIssueV0
	if request.SchemaVersion != ContextBundleRequestSchemaVersionV0 {
		issues = append(issues, contextBundleIssueV0(ErrContextBundleSchemaNoSoportadoV0, "schema_version", "schema_version no soportada"))
	}
	for _, required := range []struct {
		field string
		value string
	}{
		{"bundle_ref", request.BundleRef},
		{"work_order_ref", request.WorkOrderRef},
		{"target_module", request.TargetModule},
		{"phase", request.Phase},
		{"task_kind", request.TaskKind},
		{"objective", request.Objective},
		{"capacity_level", request.CapacityLevel},
	} {
		if required.value == "" {
			issues = append(issues, contextBundleIssueV0(ErrContextBundleCampoRequeridoV0, required.field, "campo requerido"))
		}
	}
	if len(request.WriteSet) == 0 {
		issues = append(issues, contextBundleIssueV0(ErrContextBundleCampoRequeridoV0, "write_set", "write_set requerido"))
	}
	if !validContextModuleV0(request.TargetModule) {
		issues = append(issues, contextBundleIssueV0(ErrContextBundleModuloInvalidoV0, "target_module", "modulo no soportado"))
	}
	if !validContextPhaseV0(request.Phase) {
		issues = append(issues, contextBundleIssueV0(ErrContextBundleFaseNoSoportadaV0, "phase", "fase no soportada"))
	}
	if !validContextCapacityV0(request.CapacityLevel) {
		issues = append(issues, contextBundleIssueV0(ErrContextBundleCapacidadInvalidaV0, "capacity_level", "capacidad no soportada"))
	}
	if request.MaxEntries < 8 || request.MaxEntries > 30 {
		issues = append(issues, contextBundleIssueV0(ErrContextBundleTamanoInvalidoV0, "max_entries", "limite de entradas invalido"))
	}
	if request.MaxTotalBytes < 4000 || request.MaxTotalBytes > 64000 {
		issues = append(issues, contextBundleIssueV0(ErrContextBundleTamanoInvalidoV0, "max_total_bytes", "limite de bytes invalido"))
	}
	return append(issues, forbiddenContextDetailsV0(request)...)
}

func validContextModuleV0(module string) bool {
	return strings.HasPrefix(module, "orquesta-") && !strings.Contains(module, "/") && !strings.Contains(module, "..")
}

func validContextPhaseV0(phase string) bool {
	switch phase {
	case "descubrimiento", "brainstorming_arquitectura", "votacion_y_decision",
		"planificacion_microtareas", "programacion", "documentacion",
		"integracion", "revision", "validacion_final", "cierre":
		return true
	default:
		return false
	}
}

func validContextCapacityV0(level string) bool {
	switch level {
	case "low", "medium", "high", "xhigh":
		return true
	default:
		return false
	}
}

func forbiddenContextDetailsV0(request ContextBundleRequestV0) []ContextBundleIssueV0 {
	values := []struct {
		field string
		value string
	}{
		{"bundle_ref", request.BundleRef},
		{"work_order_ref", request.WorkOrderRef},
		{"target_module", request.TargetModule},
		{"objective", request.Objective},
		{"task_kind", request.TaskKind},
	}
	values = appendContextValuesV0(values, "read_set", request.ReadSet)
	values = appendContextValuesV0(values, "write_set", request.WriteSet)
	values = appendContextValuesV0(values, "contract_refs", request.ContractRefs)
	values = appendContextValuesV0(values, "cross_module_refs", request.CrossModuleRefs)
	values = appendContextValuesV0(values, "evidence_refs", request.EvidenceRefs)

	var issues []ContextBundleIssueV0
	for _, item := range values {
		if contextBundleRequestValueHasForbiddenDetailV0(item.field, item.value) {
			issues = append(issues, contextBundleIssueV0(ErrContextBundleDetalleProhibidoV0, item.field, "detalle prohibido"))
		}
	}
	return issues
}

func contextBundleRequestValueHasForbiddenDetailV0(field string, value string) bool {
	if contextBundleRequestRawFieldV0(field) {
		return orquestarails.TextContainsOperationalRawDetailForFieldV0(
			contextBundleRequestRailBoundaryV0,
			field,
			value,
		)
	}
	return orquestarails.TextContainsOperationalSensitiveDetailForFieldV0(
		contextBundleRequestRailBoundaryV0,
		field,
		value,
	)
}

func contextBundleRequestRawFieldV0(field string) bool {
	switch field {
	case "read_set", "write_set", "contract_refs", "cross_module_refs", "evidence_refs":
		return true
	default:
		return false
	}
}

func appendContextValuesV0(values []struct {
	field string
	value string
}, field string, refs []string) []struct {
	field string
	value string
} {
	for _, ref := range refs {
		values = append(values, struct {
			field string
			value string
		}{field: field, value: ref})
	}
	return values
}
