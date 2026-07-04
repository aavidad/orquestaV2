package orquestaopesdirector

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func followupRequestV0(
	record OPESCausalArtifactRecordV0,
	followupRef string,
) orquestadomainwork.DomainWorkJobRequestV0 {
	workKind := followupWorkKindV0(record)
	expectedArtifactType := orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind)
	key := causalJobIdempotencyKeyV0("followup", record, followupRef, workKind)
	fields := provenanceFieldsV0(record, followupRef, expectedArtifactType)
	fields = appendRequiredEvidenceReworkFieldsV0(record, followupRef, fields)
	fields = appendQuestionBankQualityReworkFieldsV0(record, followupRef, fields)
	return orquestadomainwork.NormalizeDomainWorkJobRequestV0(orquestadomainwork.DomainWorkJobRequestV0{
		RequestID:      "request-" + key,
		CorrelationID:  firstNonEmptyV0(record.CorrelationID, record.IdempotencyKey),
		IdempotencyKey: key,
		RequestedBy:    OPESCausalProducerDefaultRequestedByV0,
		DomainRef:      OPESCausalProducerDefaultDomainRefV0,
		InterfaceRefs:  []string{"orquesta.domain_work.v0", "opes.rest.v0"},
		WorkKind:       workKind,
		WorkRefs: []string{
			"source-job-" + safeRefV0(record.JobRef),
			"source-artifact-" + safeRefV0(record.ArtifactRef),
			"followup-" + safeRefV0(followupRef),
		},
		Objective: "Resolver seguimiento causal OPES sin cerrar el temario mientras queden tareas pendientes.",
		Constraints: []string{
			"Reutilizar el artefacto fuente como insumo; no rehacer material valido por formato reparable.",
			"No subir a produccion; entregar artefacto local revisable.",
		},
		AcceptanceCriteria: followupAcceptanceCriteriaV0(expectedArtifactType),
		InputFields:        fields,
		ExternalRefs:       provenanceExternalRefsV0(record, followupRef),
		EvidenceRefs:       provenanceEvidenceRefsV0(record),
	})
}

func appendRequiredEvidenceReworkFieldsV0(
	record OPESCausalArtifactRecordV0,
	followupRef string,
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	if !strings.HasPrefix(strings.TrimSpace(followupRef), "required-evidence-") {
		return fields
	}
	pending := topicRegistryRequiredEvidencePendingRefsForRecordV0(record)
	if len(pending) == 0 {
		pending = []string{followupRef}
	}
	return append(fields,
		orquestadomainwork.DomainWorkFieldV0{Name: "rework_reason", Value: "required_evidence_missing"},
		orquestadomainwork.DomainWorkFieldV0{Name: "required_evidence_missing_refs", Values: compactStringsV0(pending)},
		orquestadomainwork.DomainWorkFieldV0{Name: "publication_status", Value: "not_publicable_without_required_evidence"},
		orquestadomainwork.DomainWorkFieldV0{Name: "recommended_action", Value: "review_required_evidence"},
	)
}

func appendQuestionBankQualityReworkFieldsV0(
	record OPESCausalArtifactRecordV0,
	followupRef string,
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	if !strings.HasPrefix(strings.TrimSpace(followupRef), "question-bank-quality-") &&
		strings.TrimSpace(followupRef) != topicRegistryQuestionBankQualityNeedsReworkRefV0 {
		return fields
	}
	pending := topicRegistryQuestionBankQualityPendingRefsForRecordV0(record)
	result, ok := topicRegistryQuestionBankQualityResultForRecordV0(record)
	issueRefs := make([]string, 0, len(result.Issues))
	if ok {
		for _, issue := range result.Issues {
			if code := strings.TrimSpace(issue.Code); code != "" {
				issueRefs = append(issueRefs, code)
			}
		}
	}
	if len(pending) == 0 {
		pending = []string{followupRef}
	}
	return append(fields,
		orquestadomainwork.DomainWorkFieldV0{Name: "rework_reason", Value: "question_bank_quality_contract_failed"},
		orquestadomainwork.DomainWorkFieldV0{Name: "question_bank_quality_missing_refs", Values: compactStringsV0(pending)},
		orquestadomainwork.DomainWorkFieldV0{Name: "question_bank_quality_issue_refs", Values: compactStringsV0(issueRefs)},
		orquestadomainwork.DomainWorkFieldV0{Name: "publication_status", Value: "not_publicable_question_bank_quality_failed"},
		orquestadomainwork.DomainWorkFieldV0{Name: "recommended_action", Value: "review_question_bank_quality"},
	)
}

func rejectedArtifactCorrectionRequestV0(
	record OPESCausalArtifactRecordV0,
) orquestadomainwork.DomainWorkJobRequestV0 {
	workKind := firstNonEmptyV0(fieldStringV0(record.PayloadFields, "source_work_kind"), "review_director_consolidation")
	expectedArtifactType := orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind)
	followupRef := "rejected-" + safeRefV0(record.ArtifactRef)
	key := causalJobIdempotencyKeyV0("rejected", record, followupRef, workKind)
	fields := provenanceFieldsV0(record, followupRef, expectedArtifactType)
	fields = append(fields, orquestadomainwork.DomainWorkFieldV0{Name: "issue_refs", Values: compactStringsV0(record.IssueRefs)})
	return orquestadomainwork.NormalizeDomainWorkJobRequestV0(orquestadomainwork.DomainWorkJobRequestV0{
		RequestID:      "request-" + key,
		CorrelationID:  firstNonEmptyV0(record.CorrelationID, record.IdempotencyKey),
		IdempotencyKey: key,
		RequestedBy:    OPESCausalProducerDefaultRequestedByV0,
		DomainRef:      OPESCausalProducerDefaultDomainRefV0,
		InterfaceRefs:  []string{"orquesta.domain_work.v0", "opes.rest.v0"},
		WorkKind:       workKind,
		WorkRefs: []string{
			"source-job-" + safeRefV0(record.JobRef),
			"source-artifact-" + safeRefV0(record.ArtifactRef),
			followupRef,
		},
		Objective:   "Corregir una entrega OPES rechazada conservando el trabajo recuperable.",
		InputFields: fields,
		Constraints: []string{
			"Corregir solo la entrega rechazada o materializar una normalizacion recuperable.",
			"No descartar trabajo util por alias, formato o metadatos reparables.",
		},
		AcceptanceCriteria: []string{
			"Devolver artifact_type=" + expectedArtifactType + " con payload trazable.",
			"Explicar que se corrige y que se conserva como insumo.",
			"No cerrar el curso si quedan followup_refs abiertos.",
		},
		ExternalRefs: provenanceExternalRefsV0(record, followupRef),
		EvidenceRefs: provenanceEvidenceRefsV0(record),
	})
}

func planSequenceReworkRequestV0(
	record OPESCausalArtifactRecordV0,
	plan orquestadomainwork.DomainDocumentPlanV0,
	missing []string,
) orquestadomainwork.DomainWorkJobRequestV0 {
	followupRef := "missing-opes-sequence"
	key := causalJobIdempotencyKeyV0("plan-sequence", record, followupRef, plan.WorkKind)
	fields := provenanceFieldsV0(record, followupRef, orquestadomainwork.DomainDocumentPlanArtifactTypeV0)
	fields = append(fields,
		orquestadomainwork.DomainWorkFieldV0{Name: "plan_ref", Value: plan.PlanRef},
		orquestadomainwork.DomainWorkFieldV0{Name: "missing_work_kinds", Values: compactStringsV0(missing)},
	)
	return orquestadomainwork.NormalizeDomainWorkJobRequestV0(orquestadomainwork.DomainWorkJobRequestV0{
		RequestID:      "request-" + key,
		CorrelationID:  firstNonEmptyV0(record.CorrelationID, record.IdempotencyKey),
		IdempotencyKey: key,
		RequestedBy:    OPESCausalProducerDefaultRequestedByV0,
		DomainRef:      OPESCausalProducerDefaultDomainRefV0,
		InterfaceRefs:  []string{"orquesta.domain_work.v0", "opes.rest.v0"},
		WorkKind:       plan.WorkKind,
		WorkRefs: []string{
			"source-job-" + safeRefV0(record.JobRef),
			"source-artifact-" + safeRefV0(record.ArtifactRef),
			"plan-" + safeRefV0(plan.PlanRef),
		},
		Objective:   "Rehacer el plan OPES para cubrir la secuencia completa hasta paquete local terminado.",
		InputFields: fields,
		Constraints: []string{
			"Conservar el document_plan anterior como insumo.",
			"Crear plan completo con tests, visuales, HTML, RAG, audios, revisiones, manuales y paquete.",
		},
		AcceptanceCriteria: []string{
			"El document_plan debe declarar todos los trabajos faltantes en missing_work_kinds.",
			"No marcar listo si quedan fases obligatorias fuera del plan.",
			"Usar castellano correcto con tildes, eñes y signos de apertura y cierre.",
		},
		ExternalRefs: provenanceExternalRefsV0(record, followupRef),
		EvidenceRefs: provenanceEvidenceRefsV0(record),
	})
}

func followupWorkKindV0(record OPESCausalArtifactRecordV0) string {
	if workKind := firstNonEmptyV0(
		fieldStringV0(record.PayloadFields, "recommended_work_kind"),
		fieldStringV0(record.PayloadFields, "next_work_kind"),
	); workKind != "" {
		return workKind
	}
	if target := fieldStringV0(record.PayloadFields, "target_artifact_type"); target != "" {
		if workKind := workKindForArtifactTypeV0(target); workKind != "" {
			return workKind
		}
	}
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 &&
		record.CompleteJob &&
		!topicRegistryFinalPackageHasClosureEvidenceV0(record) {
		return "finalize_temario_package"
	}
	return "review_director_consolidation"
}

func workKindForArtifactTypeV0(artifactType string) string {
	switch strings.TrimSpace(artifactType) {
	case orquestadomainwork.DomainWorkArtifactTypeContentBlockV0:
		return "draft_content_block"
	case orquestadomainwork.DomainWorkArtifactTypeVisualAssetV0:
		return "generate_visual_asset"
	case orquestadomainwork.DomainWorkArtifactTypeQuestionBankV0:
		return "generate_question_bank"
	case orquestadomainwork.DomainWorkArtifactTypeAudioAssetV0:
		return "generate_audio_asset"
	case orquestadomainwork.DomainWorkArtifactTypeLocalHTMLSiteV0:
		return "generate_html_site"
	case orquestadomainwork.DomainWorkArtifactTypeTutorBotPackageV0:
		return "generate_tutor_assets"
	case orquestadomainwork.DomainWorkArtifactTypeInteractivePracticeV0:
		return "generate_learning_games"
	case orquestadomainwork.DomainWorkArtifactTypeHelpPackageV0:
		return "generate_help_manual_assets"
	case orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0:
		return "finalize_temario_package"
	default:
		return ""
	}
}

func followupRefsForRecordV0(record OPESCausalArtifactRecordV0) []string {
	refs := fieldStringsV0(
		record.PayloadFields,
		"followup_refs",
		"pending_refs",
		"pending_followup_refs",
		"rework_refs",
		"missing_required_refs",
	)
	refs = append(refs, topicRegistryLifecyclePendingRefsForRecordV0(record)...)
	refs = append(refs, topicRegistryQualityPendingRefsForRecordV0(record)...)
	refs = append(refs, topicRegistryQuestionBankQualityPendingRefsForRecordV0(record)...)
	refs = append(refs, topicRegistryRequiredEvidencePendingRefsForRecordV0(record)...)
	status := strings.ToLower(firstNonEmptyV0(
		fieldStringV0(record.PayloadFields, "status"),
		fieldStringV0(record.PayloadFields, "estado"),
		fieldStringV0(record.PayloadFields, "decision"),
	))
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 &&
		strings.Contains(status, "pendiente") &&
		len(refs) == 0 {
		refs = append(refs, "final-package-pendiente-continuar")
	}
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 &&
		record.CompleteJob &&
		len(refs) == 0 &&
		!topicRegistryFinalPackageHasClosureEvidenceV0(record) {
		refs = append(refs, "final-package-manifest-closure-evidence-required")
	}
	return compactStringsV0(refs)
}

func provenanceFieldsV0(
	record OPESCausalArtifactRecordV0,
	followupRef string,
	expectedArtifactType string,
) []orquestadomainwork.DomainWorkFieldV0 {
	fields := []orquestadomainwork.DomainWorkFieldV0{
		{Name: "expected_artifact_type", Value: expectedArtifactType},
		{Name: "source_job_ref", Value: record.JobRef},
		{Name: "source_artifact_ref", Value: record.ArtifactRef},
		{Name: "source_receipt_ref", Value: record.ReceiptRef},
		{Name: "source_artifact_type", Value: record.ArtifactType},
		{Name: "source_status", Value: record.Status},
	}
	if followupRef != "" {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{Name: "followup_ref", Value: followupRef})
	}
	for _, name := range []string{
		"program_id",
		"topic_id",
		"course_id",
		"plan_ref",
		"package_ref",
		"manifest_ref",
		"review_matrix_ref",
		"target_part_ref",
		"target_artifact_type",
		"language_code",
		"level",
	} {
		if value := fieldStringV0(record.PayloadFields, name); value != "" {
			fields = append(fields, orquestadomainwork.DomainWorkFieldV0{Name: name, Value: value})
		}
	}
	return cloneFieldsV0(fields)
}

func provenanceExternalRefsV0(
	record OPESCausalArtifactRecordV0,
	followupRef string,
) []orquestadomainwork.DomainWorkExternalRefV0 {
	refs := append([]orquestadomainwork.DomainWorkExternalRefV0(nil), record.ExternalRefs...)
	refs = append(refs,
		orquestadomainwork.DomainWorkExternalRefV0{Kind: "source_job_ref", Ref: safeRefV0(record.JobRef)},
		orquestadomainwork.DomainWorkExternalRefV0{Kind: "source_artifact_ref", Ref: safeRefV0(record.ArtifactRef)},
	)
	if record.ReceiptRef != "" {
		refs = append(refs, orquestadomainwork.DomainWorkExternalRefV0{Kind: "source_receipt_ref", Ref: safeRefV0(record.ReceiptRef)})
	}
	if followupRef != "" {
		refs = append(refs, orquestadomainwork.DomainWorkExternalRefV0{Kind: "followup_ref", Ref: safeRefV0(followupRef)})
	}
	return orquestadomainwork.NormalizeDomainWorkJobRequestV0(
		orquestadomainwork.DomainWorkJobRequestV0{
			RequestID:      "external-refs",
			CorrelationID:  "external-refs",
			IdempotencyKey: "external-refs",
			RequestedBy:    "external-refs",
			DomainRef:      OPESCausalProducerDefaultDomainRefV0,
			WorkKind:       "external_refs",
			Objective:      "external refs",
			ExternalRefs:   refs,
		},
	).ExternalRefs
}

func provenanceEvidenceRefsV0(record OPESCausalArtifactRecordV0) []string {
	refs := append([]string(nil), record.EvidenceRefs...)
	refs = append(refs, record.ReceiptRef, record.ArtifactRef, record.JobRef)
	return compactStringsV0(refs)
}

func followupAcceptanceCriteriaV0(expectedArtifactType string) []string {
	return []string{
		"Devolver artifact_type=" + expectedArtifactType + " con payload trazable.",
		"Resolver solo el followup_ref o target_part_ref indicado.",
		"Conservar source_artifact_ref como insumo y no rehacer material válido por formato recuperable.",
		"No cerrar ready si quedan followup_refs abiertos.",
		"Usar castellano correcto con tildes, eñes y signos de apertura y cierre.",
	}
}

func causalJobIdempotencyKeyV0(kind string, record OPESCausalArtifactRecordV0, followupRef string, workKind string) string {
	return "opes-causal-" + safeRefV0(kind) +
		"-job-" + safeRefV0(record.JobRef) +
		"-artifact-" + safeRefV0(record.ArtifactRef) +
		"-followup-" + safeRefV0(followupRef) +
		"-work-" + safeRefV0(workKind)
}

func artifactRecordSourceRefV0(record OPESCausalArtifactRecordV0) string {
	return "source-job-" + safeRefV0(record.JobRef) + "-artifact-" + safeRefV0(record.ArtifactRef)
}
