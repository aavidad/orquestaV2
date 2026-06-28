package orquestaopesconnector

import orquestadomainwork "orquesta/modulos/orquesta-domain-work"

func opesJobRequestForTestV0() orquestadomainwork.DomainWorkJobRequestV0 {
	return orquestadomainwork.NormalizeDomainWorkJobRequestV0(orquestadomainwork.DomainWorkJobRequestV0{
		RequestID:      "req-opes-001",
		CorrelationID:  "corr-opes-001",
		IdempotencyKey: "idem-opes-001",
		RequestedBy:    "orquesta",
		DomainRef:      "opes",
		InterfaceRefs:  []string{"opes-rest-v0"},
		WorkKind:       "draft_content_block",
		Objective:      "Crear bloque editorial.",
		InputFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "program_id", Value: "program-ref-001"},
			{Name: "topic_id", Value: "topic-ref-001"},
			{Name: "chapter_id", Value: "chapter-ref-001"},
			{Name: "level", Value: "A1/A2"},
			{Name: "language_code", Value: "es"},
			{Name: "block_position", ValueJSON: []byte(`{"chapter_order":1,"block_order":2}`)},
			{Name: "source_refs", Values: []string{"boe-ref-001"}},
		},
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "run_ref", Ref: "run-ref-001"},
			{Kind: "task_ref", Ref: "task-ref-001"},
		},
	})
}

func opesArtifactSubmissionForTestV0() orquestadomainwork.DomainWorkArtifactSubmissionV0 {
	return orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(orquestadomainwork.DomainWorkArtifactSubmissionV0{
		RequestID:      "req-delivery-001",
		CorrelationID:  "corr-opes-001",
		IdempotencyKey: "idem-delivery-001",
		RequestedBy:    "orquesta",
		DomainRef:      "opes",
		JobRef:         "job-ref-opes-001",
		ArtifactRef:    "artifact-ref-001",
		ArtifactType:   "content_block",
		Summary:        "Bloque 1",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "title", Value: "Bloque 1"},
			{Name: "body", Value: "Contenido producido por Orquesta."},
		},
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "delivery_ref", Ref: "delivery-ref-001"},
		},
		CompleteJob: true,
	})
}

func opesVisualArtifactSubmissionForTestV0() orquestadomainwork.DomainWorkArtifactSubmissionV0 {
	return orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(orquestadomainwork.DomainWorkArtifactSubmissionV0{
		RequestID:      "req-visual-delivery-001",
		CorrelationID:  "corr-visual-001",
		IdempotencyKey: "idem-visual-delivery-001",
		RequestedBy:    "orquesta",
		DomainRef:      "opes",
		JobRef:         "job-ref-visual-001",
		ArtifactRef:    "artifact-ref-visual-001",
		ArtifactType:   "visual_asset",
		Summary:        "Visual de red en estrella",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "topic_id", Value: "topic-ref-001"},
			{Name: "chapter_id", Value: "chapter-ref-001"},
			{Name: "asset_type", Value: "vignette"},
			{Name: "format", Value: "svg"},
			{Name: "title", Value: "Red en estrella"},
			{Name: "caption", Value: "Topologia con nodo central."},
			{Name: "alt_text", Value: "Switch central conectado a equipos cliente."},
			{Name: "body", Value: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 420"></svg>`},
		},
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "delivery_ref", Ref: "delivery-ref-visual-001"},
		},
		CompleteJob: true,
	})
}

func opesDocumentPlanArtifactSubmissionForTestV0() orquestadomainwork.DomainWorkArtifactSubmissionV0 {
	return orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(orquestadomainwork.DomainWorkArtifactSubmissionV0{
		RequestID:      "req-plan-temario-delivery-001",
		CorrelationID:  "corr-plan-temario-001",
		IdempotencyKey: "idem-plan-temario-delivery-001",
		RequestedBy:    "orquesta",
		DomainRef:      "opes",
		JobRef:         "job-ref-plan-temario-001",
		ArtifactRef:    "artifact-ref-plan-temario-001",
		ArtifactType:   orquestadomainwork.DomainDocumentPlanArtifactTypeV0,
		Summary:        "Plan de temario operadores",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "schema_version", Value: orquestadomainwork.DomainDocumentPlanSchemaV0},
			{Name: "artifact_type", Value: orquestadomainwork.DomainDocumentPlanArtifactTypeV0},
			{Name: "plan_ref", Value: "plan-ref-operadores-001"},
			{Name: "domain_ref", Value: "opes"},
			{Name: "work_kind", Value: orquestadomainwork.DomainWorkKindPlanSyllabusV0},
			{Name: "document_kind", Value: "temario_oposicion"},
			{Name: "scope_ref", Value: "program-ref-operadores-001"},
			{Name: "language_code", Value: "es"},
			{Name: "title", Value: "Temario operadores"},
			{Name: "objective", Value: "Planificar temario sin redactarlo."},
			{Name: "sections", ValueJSON: []byte(`[{"section_ref":"section-operadores-001","order":1,"title":"Operadores","objective":"Cubrir operadores basicos.","work_kind":"draft_content_block"}]`)},
			{Name: "deliverables", ValueJSON: []byte(`[{"deliverable_ref":"deliverable-operadores-001","artifact_type":"assembled_topic","title":"Temario ensamblado","required":true}]`)},
			{Name: "quality_criteria", Values: []string{"derivacion desde maestro superior si existe", "sin placeholders"}},
		},
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "delivery_ref", Ref: "delivery-ref-plan-temario-001"},
		},
		CompleteJob: true,
	})
}

func opesAudioArtifactSubmissionForTestV0() orquestadomainwork.DomainWorkArtifactSubmissionV0 {
	return orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(orquestadomainwork.DomainWorkArtifactSubmissionV0{
		RequestID:      "req-audio-delivery-001",
		CorrelationID:  "corr-audio-001",
		IdempotencyKey: "idem-audio-delivery-001",
		RequestedBy:    "orquesta",
		DomainRef:      "opes",
		JobRef:         "job-ref-audio-001",
		ArtifactRef:    "artifact-ref-audio-001",
		ArtifactType:   orquestadomainwork.DomainWorkArtifactTypeAudioAssetV0,
		Summary:        "Audio accesible del tema",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "topic_id", Value: "topic-ref-001"},
			{Name: "assembled_topic_artifact_id", Value: "artifact-assembled-topic-001"},
			{Name: "language_code", Value: "es"},
			{Name: "format", Value: "mp3"},
			{Name: "mime_type", Value: "audio/mpeg"},
			{Name: "duration_seconds", Value: "1830"},
			{Name: "audio_ref", Value: "audio-ref-topic-001-mp3"},
			{Name: "manifest_ref", Value: "manifest-ref-topic-001-audio"},
			{Name: "source_artifact_ref", Value: "artifact-assembled-topic-001"},
			{Name: "source_refs", ValueJSON: []byte(`{"course_id":"course-ref-001","program_id":"program-ref-001","topic_id":"topic-ref-001","assembled_topic_artifact_id":"artifact-assembled-topic-001"}`)},
		},
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "delivery_ref", Ref: "delivery-ref-audio-001"},
		},
		CompleteJob: true,
	})
}
