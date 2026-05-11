package orquestacoreconcurrency

type WorksetClaimIssueCodeV0 string

const (
	ErrWorksetClaimSchemaNoSoportadoV0 WorksetClaimIssueCodeV0 = "workset_claim_schema_no_soportado"
	ErrWorksetClaimCampoRequeridoV0    WorksetClaimIssueCodeV0 = "workset_claim_campo_requerido"
	ErrWorksetClaimWriteSetVacioV0     WorksetClaimIssueCodeV0 = "workset_claim_write_set_vacio"
	ErrScopeRefVacioV0                 WorksetClaimIssueCodeV0 = "scope_ref_vacio"
	ErrScopeRefAbsolutoV0              WorksetClaimIssueCodeV0 = "scope_ref_absoluto"
	ErrScopeRefTraversalV0             WorksetClaimIssueCodeV0 = "scope_ref_traversal"
	ErrScopeRefHomeV0                  WorksetClaimIssueCodeV0 = "scope_ref_home"
	ErrScopeRefReservadoV0             WorksetClaimIssueCodeV0 = "scope_ref_reservado"
)

type WorksetClaimIssueV0 struct {
	Code  WorksetClaimIssueCodeV0 `json:"code"`
	Field string                  `json:"field"`
	Index int                     `json:"index,omitempty"`
	Value string                  `json:"value,omitempty"`
}

func validateNormalizedWorksetClaimV0(claim WorksetClaimV0) []WorksetClaimIssueV0 {
	var issues []WorksetClaimIssueV0
	if claim.SchemaVersion != WorksetClaimSchemaVersionV0 {
		issues = append(issues, newWorksetClaimIssueV0(ErrWorksetClaimSchemaNoSoportadoV0, "schema_version", claim.SchemaVersion))
	}
	if claim.ClaimRef == "" {
		issues = append(issues, newWorksetClaimIssueV0(ErrWorksetClaimCampoRequeridoV0, "claim_ref", ""))
	}
	if claim.RunRef == "" {
		issues = append(issues, newWorksetClaimIssueV0(ErrWorksetClaimCampoRequeridoV0, "run_ref", ""))
	}
	if claim.TaskRef == "" {
		issues = append(issues, newWorksetClaimIssueV0(ErrWorksetClaimCampoRequeridoV0, "task_ref", ""))
	}
	if len(claim.WriteSet) == 0 {
		issues = append(issues, newWorksetClaimIssueV0(ErrWorksetClaimWriteSetVacioV0, "write_set", ""))
	}
	return issues
}

func newWorksetClaimIssueV0(code WorksetClaimIssueCodeV0, field string, value string) WorksetClaimIssueV0 {
	return WorksetClaimIssueV0{
		Code:  code,
		Field: field,
		Index: -1,
		Value: value,
	}
}
