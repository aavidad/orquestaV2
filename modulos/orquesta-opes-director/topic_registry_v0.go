package orquestaopesdirector

import (
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
		orquestadomainwork.DomainWorkFieldV0{Name: "done_refs", Values: compactStringsV0([]string{record.ArtifactRef, record.ReceiptRef})},
		orquestadomainwork.DomainWorkFieldV0{Name: "pending_refs", Values: pendingRefs},
	)
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
		topicRegistryFinalPackageHasDeterministicEvidenceV0(record) {
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
	normalized := strings.ToLower(strings.TrimSpace(status))
	if len(followupRefsForRecordV0(record)) > 0 || strings.Contains(normalized, "pendiente") {
		return "pendiente_continuar"
	}
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 && !record.CompleteJob {
		return "pendiente_continuar"
	}
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 &&
		!topicRegistryFinalPackageHasDeterministicEvidenceV0(record) {
		return "pendiente_validacion_paquete_final"
	}
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 {
		return "paquete_final_local_verificable"
	}
	return "en_progreso_orquesta"
}

func topicRegistryPendingRefsForRecordV0(record OPESCausalArtifactRecordV0) []string {
	refs := followupRefsForRecordV0(record)
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 &&
		record.CompleteJob &&
		len(refs) == 0 &&
		!topicRegistryFinalPackageHasDeterministicEvidenceV0(record) {
		refs = append(refs, "final-package-deterministic-validation-required")
	}
	return compactStringsV0(refs)
}

func topicRegistryFinalPackageHasDeterministicEvidenceV0(record OPESCausalArtifactRecordV0) bool {
	refs := append([]string(nil), record.EvidenceRefs...)
	refs = append(refs, fieldStringsV0(record.PayloadFields, "evidence_refs", "validation_refs", "required_test_evidence_refs")...)
	refs = append(refs, record.PayloadRefs...)
	return topicRegistryHasAnyEvidenceRefV0(refs,
		"evidence-ref-opes-finalpkg-deterministic-package-contract",
		"evidence-ref-opes-finalpkg-question-bank-present",
		"opes-required-question-bank-publicable",
		"opes-question-bank-publicable",
		"opes-final-package-validated",
		"opes_validate_topic_package_v1:passed",
	)
}

func topicRegistryHasAnyEvidenceRefV0(refs []string, accepted ...string) bool {
	for _, ref := range refs {
		for _, want := range accepted {
			if strings.TrimSpace(ref) == strings.TrimSpace(want) {
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
