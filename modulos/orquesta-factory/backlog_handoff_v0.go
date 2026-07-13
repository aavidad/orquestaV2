package orquestafactory

import "strings"

const (
	BacklogInicialEstadoPreviewNoEjecutableV0 = "preview_no_ejecutable"
	BacklogDirectorHandoffStatusPendienteV0   = "pendiente_director_v2"
	BacklogDirectorHandoffContractV0          = "orquesta.apps.arrancar_director.v0"
)

type BacklogFreshnessV0 struct {
	SourceKind      string `json:"source_kind"`
	SourceRef       string `json:"source_ref"`
	SourceCreatedAt string `json:"source_created_at,omitempty"`
	GeneratedFrom   string `json:"generated_from"`
}

type BacklogDirectorHandoffV0 struct {
	Status            string   `json:"status"`
	RequiredContract  string   `json:"required_contract"`
	RequiredInputRef  string   `json:"required_input_ref"`
	Reason            string   `json:"reason"`
	HandoffEvidenceRefs []string `json:"handoff_evidence_refs"`
}

func buildBacklogFreshnessV0(spec AppSpecV0) BacklogFreshnessV0 {
	specID := strings.TrimSpace(spec.SpecID)
	return BacklogFreshnessV0{
		SourceKind:      "AppSpecV0",
		SourceRef:       "app_spec:" + specID,
		SourceCreatedAt: strings.TrimSpace(spec.CreatedAt),
		GeneratedFrom:   "SolicitarNuevaApp v0",
	}
}

func buildBacklogDirectorHandoffV0(spec AppSpecV0) BacklogDirectorHandoffV0 {
	specID := strings.TrimSpace(spec.SpecID)
	return BacklogDirectorHandoffV0{
		Status:             BacklogDirectorHandoffStatusPendienteV0,
		RequiredContract:   BacklogDirectorHandoffContractV0,
		RequiredInputRef:   "app_spec:" + specID,
		Reason:             "backlog_preview_requires_director_handoff",
		HandoffEvidenceRefs: compactUniqueV0([]string{"evidence-ref-backlog-preview-" + specID}),
	}
}
