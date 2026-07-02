package orquestaopesdirector

import (
	"encoding/json"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const opesTopicRegistryUpdateWorkKindV0 = "update_topic_registry"

func shouldRequestTopicRegistryUpdateV0(record OPESCausalArtifactRecordV0) bool {
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeTopicRegistryUpdateV0 {
		return false
	}
	if fieldStringV0(record.PayloadFields, "source_work_kind", "work_kind") == opesTopicRegistryUpdateWorkKindV0 {
		return false
	}
	return fieldStringV0(record.PayloadFields, "course_id") != "" &&
		fieldStringV0(record.PayloadFields, "topic_id") != ""
}

func topicRegistryUpdateRequestV0(
	record OPESCausalArtifactRecordV0,
) orquestadomainwork.DomainWorkJobRequestV0 {
	courseID := fieldStringV0(record.PayloadFields, "course_id")
	topicID := fieldStringV0(record.PayloadFields, "topic_id")
	scopeRef := "topic-registry-" + safeRefV0(courseID) + "-" + safeRefV0(topicID)
	expectedArtifactType := orquestadomainwork.DomainWorkArtifactTypeTopicRegistryUpdateV0
	key := causalJobIdempotencyKeyV0("topic-registry", record, scopeRef, opesTopicRegistryUpdateWorkKindV0)
	fields := provenanceFieldsV0(record, scopeRef, expectedArtifactType)
	pendingRefs := topicRegistryPendingRefsForRecordV0(record)
	fields = append(fields,
		orquestadomainwork.DomainWorkFieldV0{Name: "registry_scope", Value: "topic"},
		orquestadomainwork.DomainWorkFieldV0{Name: "registry_action", Value: topicRegistryActionForRecordV0(record)},
		orquestadomainwork.DomainWorkFieldV0{Name: "registry_tool_ref", Value: "opes-registro-trabajo-temas"},
		orquestadomainwork.DomainWorkFieldV0{Name: "proposed_status", Value: topicRegistryStatusForRecordV0(record)},
		orquestadomainwork.DomainWorkFieldV0{Name: "operational_status", Value: topicRegistryOperationalStatusForRecordV0(record)},
		orquestadomainwork.DomainWorkFieldV0{Name: "operational_status_contract", Value: "working|waiting|needs_rework|blocked|complete"},
		orquestadomainwork.DomainWorkFieldV0{Name: "done_refs", Values: compactStringsV0([]string{record.ArtifactRef, record.ReceiptRef})},
		orquestadomainwork.DomainWorkFieldV0{Name: "pending_refs", Values: pendingRefs},
	)
	fields = append(fields, topicRegistryLifecycleFieldsForRecordV0(record)...)
	fields = append(fields, topicRegistrySettlementFieldsForRecordV0(record)...)
	fields = append(fields, topicRegistryQualityFieldsForRecordV0(record)...)
	if sourceWorkKind := fieldStringV0(record.PayloadFields, "source_work_kind", "work_kind"); sourceWorkKind != "" {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{Name: "source_work_kind", Value: sourceWorkKind})
	}
	if record.Summary != "" {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{Name: "source_summary", Value: record.Summary})
	}
	return orquestadomainwork.NormalizeDomainWorkJobRequestV0(orquestadomainwork.DomainWorkJobRequestV0{
		RequestID:      "request-" + key,
		CorrelationID:  firstNonEmptyV0(record.CorrelationID, record.IdempotencyKey),
		IdempotencyKey: key,
		RequestedBy:    OPESCausalProducerDefaultRequestedByV0,
		DomainRef:      OPESCausalProducerDefaultDomainRefV0,
		InterfaceRefs:  []string{"orquesta.domain_work.v0", "opes.topic_registry.v0"},
		WorkKind:       opesTopicRegistryUpdateWorkKindV0,
		WorkRefs: []string{
			"source-job-" + safeRefV0(record.JobRef),
			"source-artifact-" + safeRefV0(record.ArtifactRef),
			scopeRef,
		},
		Objective:   "Actualizar el registro global OPES del tema tras una entrega aceptada sin abrir el write-set del agente de contenido.",
		InputFields: fields,
		Constraints: []string{
			"Usar el conector o herramienta oficial del registro OPES; no editar el JSON bruto a mano si existe herramienta.",
			"Actualizar solo el topic_id indicado dentro del course_id indicado.",
			"No marcar ready si quedan pending_refs o followup_refs abiertos.",
		},
		AcceptanceCriteria: []string{
			"Devolver artifact_type=" + expectedArtifactType + " con payload trazable.",
			"Registrar course_id, topic_id, registry_action, proposed_status, done_refs, pending_refs y evidencias.",
			"Si falta permiso o conector, devolver bloqueo público con comando exacto necesario y no cerrar el tema.",
			"Usar castellano correcto con tildes, eñes y signos de apertura y cierre.",
		},
		ExternalRefs: provenanceExternalRefsV0(record, scopeRef),
		EvidenceRefs: provenanceEvidenceRefsV0(record),
	})
}

func topicRegistryActionForRecordV0(record OPESCausalArtifactRecordV0) string {
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 &&
		record.CompleteJob &&
		len(topicRegistryPendingRefsForRecordV0(record)) == 0 &&
		topicRegistryFinalPackageHasClosureEvidenceV0(record) {
		return "release"
	}
	return "update"
}

func topicRegistryStatusForRecordV0(record OPESCausalArtifactRecordV0) string {
	status := firstNonEmptyV0(
		fieldStringV0(record.PayloadFields, "status"),
		fieldStringV0(record.PayloadFields, "estado"),
		fieldStringV0(record.PayloadFields, "decision"),
	)
	if topicRegistryExplicitPartialStatusV0(status) {
		return strings.TrimSpace(status)
	}
	if explicit, ok := topicRegistryExplicitOperationalStatusV0(status); ok {
		return explicit
	}
	normalized := strings.ToLower(strings.TrimSpace(status))
	if refs := topicRegistryQualityPendingRefsForRecordV0(record); len(refs) > 0 {
		return "pendiente_rework_editorial"
	}
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 && !record.CompleteJob {
		return "pendiente_continuar"
	}
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 &&
		!topicRegistryFinalPackageHasClosureEvidenceV0(record) {
		return "pendiente_validacion_paquete_final"
	}
	if len(followupRefsForRecordV0(record)) > 0 || strings.Contains(normalized, "pendiente") {
		return "pendiente_continuar"
	}
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 {
		return "paquete_final_local_verificable"
	}
	return "en_progreso_orquesta"
}

func topicRegistryOperationalStatusForRecordV0(record OPESCausalArtifactRecordV0) string {
	status := firstNonEmptyV0(
		fieldStringV0(record.PayloadFields, "operational_status"),
		fieldStringV0(record.PayloadFields, "status"),
		fieldStringV0(record.PayloadFields, "estado"),
		fieldStringV0(record.PayloadFields, "decision"),
	)
	if refs := topicRegistryQualityPendingRefsForRecordV0(record); len(refs) > 0 {
		return "needs_rework"
	}
	explicit, hasExplicit := topicRegistryExplicitOperationalStatusV0(status)
	if hasExplicit && (explicit == "blocked" || explicit == "needs_rework") {
		return explicit
	}
	if refs := topicRegistryLifecyclePendingRefsForRecordV0(record); len(refs) > 0 {
		return "waiting"
	}
	if hasExplicit {
		return explicit
	}
	normalized := strings.ToLower(strings.TrimSpace(status))
	if topicRegistryExplicitPartialStatusV0(status) ||
		len(followupRefsForRecordV0(record)) > 0 ||
		strings.Contains(normalized, "pendiente") {
		return "waiting"
	}
	if strings.Contains(normalized, "blocked") || strings.Contains(normalized, "bloqueado") {
		return "blocked"
	}
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 &&
		record.CompleteJob &&
		len(topicRegistryPendingRefsForRecordV0(record)) == 0 &&
		topicRegistryFinalPackageHasClosureEvidenceV0(record) {
		return "complete"
	}
	return "working"
}

func topicRegistryPendingRefsForRecordV0(record OPESCausalArtifactRecordV0) []string {
	refs := followupRefsForRecordV0(record)
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 &&
		record.CompleteJob &&
		len(refs) == 0 &&
		!topicRegistryFinalPackageHasClosureEvidenceV0(record) {
		refs = append(refs, "final-package-manifest-closure-evidence-required")
	}
	return compactStringsV0(refs)
}

func topicRegistryFinalPackageHasClosureEvidenceV0(record OPESCausalArtifactRecordV0) bool {
	refs := topicRegistryFinalPackageEvidenceRefsV0(record)
	hasCompatibleManifest := topicRegistryFinalPackageHasCompatibleManifestV0(record.PayloadFields)
	if !hasCompatibleManifest &&
		!topicRegistryEvidenceContainsAnyV0(refs,
			"manifest_cierre",
			"manifest_cierre.json",
			"completed_syllabus_package_manifest",
			"opes-expected-evidence-manifest-cierre",
			"opes-rule-final-package-manifest",
		) {
		return false
	}
	required := [][]string{
		{"opes-final-evidence:html", "html_evidence_ref", "local_html_site", "html/index.html"},
		{"opes-final-evidence:rag", "rag_evidence_ref", "rag/manifest.json", "tutor_rag_manifest"},
		{"opes-final-evidence:audio", "audio_evidence_ref", "audio/guion_audio.md", "audio_manifest"},
		{"opes-final-evidence:tests", "tests_evidence_ref", "question_bank", "tests.json"},
		{"opes-final-evidence:tutor", "tutor_evidence_ref", "tutor_bot_package", "tutor_assets_publicable"},
		{"opes-final-evidence:visual", "visual_evidence_ref", "visuales_plan.md", "visual_validation_report_ref"},
	}
	for _, options := range required {
		if !topicRegistryEvidenceContainsAnyV0(refs, options...) {
			return false
		}
	}
	return hasCompatibleManifest || topicRegistryFinalPackageHasStrictQAEvidenceV0(refs)
}

func topicRegistryFinalPackageEvidenceRefsV0(record OPESCausalArtifactRecordV0) []string {
	refs := append([]string(nil), record.EvidenceRefs...)
	refs = append(refs, fieldStringsV0(record.PayloadFields, "evidence_refs", "validation_refs", "required_test_evidence_refs")...)
	refs = append(refs, record.PayloadRefs...)
	refs = append(refs, topicRegistryFinalPackageManifestEvidenceRefsV0(record.PayloadFields)...)
	return compactStringsV0(refs)
}

func topicRegistryFinalPackageHasStrictQAEvidenceV0(refs []string) bool {
	required := [][]string{
		{"opes-final-evidence:extension_pass", "extension_pass"},
		{"opes-final-evidence:official_text_qa_pass", "official_text_qa_pass"},
		{"opes-final-evidence:strict_editorial_qa_pass", "strict_editorial_qa_pass"},
	}
	for _, options := range required {
		if !topicRegistryEvidenceContainsAnyV0(refs, options...) {
			return false
		}
	}
	return true
}

type topicRegistryFinalPackageManifestV0 struct {
	SchemaVersion        string                     `json:"schema_version"`
	PackageRef           string                     `json:"package_ref"`
	ManifestRef          string                     `json:"manifest_ref"`
	ChecksumRefs         []string                   `json:"checksum_refs"`
	ValidationReportRef  string                     `json:"validation_report_ref"`
	ReviewMatrixRef      string                     `json:"review_matrix_ref"`
	RequiredEvidenceRefs map[string]json.RawMessage `json:"required_evidence_refs"`
	QAPasses             struct {
		ExtensionPass              bool `json:"extension_pass"`
		OfficialTextQAPass         bool `json:"official_text_qa_pass"`
		StrictEditorialQAPass      bool `json:"strict_editorial_qa_pass"`
		QuestionBankPublicablePass bool `json:"question_bank_publicable"`
		TutorAssetsPublicablePass  bool `json:"tutor_assets_publicable"`
	} `json:"qa_passes"`
	QAReportRefs map[string]json.RawMessage `json:"qa_report_refs"`
}

func topicRegistryFinalPackageHasCompatibleManifestV0(fields []orquestadomainwork.DomainWorkFieldV0) bool {
	manifest, ok := topicRegistryFinalPackageManifestV0FromFields(fields)
	if !ok {
		return false
	}
	for _, category := range []string{"html", "rag", "audio", "tests", "tutor", "visual", "qa"} {
		if len(topicRegistryJSONRawRefsV0(manifest.RequiredEvidenceRefs[category])) == 0 {
			return false
		}
	}
	if !manifest.QAPasses.ExtensionPass ||
		!manifest.QAPasses.OfficialTextQAPass ||
		!manifest.QAPasses.StrictEditorialQAPass ||
		!manifest.QAPasses.QuestionBankPublicablePass ||
		!manifest.QAPasses.TutorAssetsPublicablePass {
		return false
	}
	for _, category := range []string{"extension", "official_text", "strict_editorial", "question_bank_publicable", "tutor_assets_publicable"} {
		if len(topicRegistryJSONRawRefsV0(manifest.QAReportRefs[category])) == 0 {
			return false
		}
	}
	return true
}

func topicRegistryFinalPackageManifestEvidenceRefsV0(fields []orquestadomainwork.DomainWorkFieldV0) []string {
	manifest, ok := topicRegistryFinalPackageManifestV0FromFields(fields)
	if !ok {
		return nil
	}
	refs := []string{"manifest_cierre", manifest.ManifestRef, manifest.ValidationReportRef, manifest.ReviewMatrixRef}
	refs = append(refs, manifest.ChecksumRefs...)
	for _, raw := range manifest.RequiredEvidenceRefs {
		refs = append(refs, topicRegistryJSONRawRefsV0(raw)...)
	}
	return compactStringsV0(refs)
}

func topicRegistryFinalPackageManifestV0FromFields(
	fields []orquestadomainwork.DomainWorkFieldV0,
) (topicRegistryFinalPackageManifestV0, bool) {
	for _, field := range fields {
		name := strings.TrimSpace(field.Name)
		if name != "manifest_cierre" && name != "manifest_cierre.json" && name != "completed_syllabus_package_manifest" {
			continue
		}
		manifest, ok := topicRegistryFinalPackageManifestV0FromRaw(field.ValueJSON)
		if ok {
			return manifest, true
		}
		manifest, ok = topicRegistryFinalPackageManifestV0FromRaw([]byte(field.Value))
		if ok {
			return manifest, true
		}
	}
	return topicRegistryFinalPackageManifestV0{}, false
}

func topicRegistryFinalPackageManifestV0FromRaw(raw json.RawMessage) (topicRegistryFinalPackageManifestV0, bool) {
	if len(raw) == 0 || !json.Valid(raw) {
		return topicRegistryFinalPackageManifestV0{}, false
	}
	var manifest topicRegistryFinalPackageManifestV0
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return topicRegistryFinalPackageManifestV0{}, false
	}
	if manifest.SchemaVersion != "opes_final_package_evidence_manifest.v0" ||
		strings.TrimSpace(manifest.PackageRef) == "" ||
		strings.TrimSpace(manifest.ManifestRef) == "" ||
		len(compactStringsV0(manifest.ChecksumRefs)) == 0 ||
		strings.TrimSpace(manifest.ValidationReportRef) == "" ||
		strings.TrimSpace(manifest.ReviewMatrixRef) == "" {
		return topicRegistryFinalPackageManifestV0{}, false
	}
	return manifest, true
}

func topicRegistryJSONRawRefsV0(raw json.RawMessage) []string {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err == nil {
		return compactStringsV0(values)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return compactStringsV0([]string{value})
	}
	return nil
}

func topicRegistryEvidenceContainsAnyV0(refs []string, accepted ...string) bool {
	for _, ref := range refs {
		normalizedRef := strings.TrimSpace(ref)
		for _, want := range accepted {
			normalizedWant := strings.TrimSpace(want)
			if normalizedWant != "" && strings.Contains(normalizedRef, normalizedWant) {
				return true
			}
		}
	}
	return false
}

func topicRegistryExplicitPartialStatusV0(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return false
	}
	if strings.HasPrefix(status, "texto_minimo_") && strings.Contains(status, "_pendiente_") {
		return true
	}
	switch {
	case strings.HasPrefix(status, "pendiente_reintento_orquesta"):
		return true
	case strings.HasPrefix(status, "stale_lock_no_process"):
		return true
	case strings.HasPrefix(status, "needs_reconcile"):
		return true
	default:
		return false
	}
}

func topicRegistryExplicitOperationalStatusV0(status string) (string, bool) {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "working", "in_progress", "in-progress", "en_progreso", "en_progreso_orquesta":
		return "working", true
	case "waiting", "pending", "pendiente", "pendiente_continuar":
		return "waiting", true
	case "needs_rework", "needs-rework", "rework", "pendiente_rework_editorial":
		return "needs_rework", true
	case "blocked", "bloqueado", "stale_lock_no_process":
		return "blocked", true
	case "complete", "completed", "done", "paquete_final_local_verificable":
		return "complete", true
	default:
		return "", false
	}
}
