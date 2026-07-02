package orquestaopesbridge

import (
	"context"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

type OPESRequiredTestPolicyV0 struct{}

var _ orquestadomainwork.DomainWorkRequiredTestPolicyPortV0 = OPESRequiredTestPolicyV0{}

func (OPESRequiredTestPolicyV0) BuildDomainWorkRequiredTestPlanV0(
	_ context.Context,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (orquestadomainwork.DomainWorkRequiredTestPlanV0, error) {
	request = orquestadomainwork.NormalizeDomainWorkJobRequestV0(request)
	tests := append([]orquestadomainwork.DomainWorkRequiredTestV0(nil), request.RequiredTests...)
	if strings.TrimSpace(request.DomainRef) == "opes" && len(tests) == 0 {
		tests = opesRequiredTestsForJobV0(request.WorkKind, opesRequiredTestJobRefV0(request), request.WorkRefs)
	}
	return orquestadomainwork.NormalizeDomainWorkRequiredTestPlanV0(
		orquestadomainwork.DomainWorkRequiredTestPlanV0{
			DomainRef:          request.DomainRef,
			WorkKind:           request.WorkKind,
			JobRef:             opesRequiredTestJobRefV0(request),
			AcceptanceCriteria: append([]string(nil), request.AcceptanceCriteria...),
			RequiredTests:      tests,
			ExternalRefs:       append([]orquestadomainwork.DomainWorkExternalRefV0(nil), request.ExternalRefs...),
			EvidenceRefs:       append([]string(nil), request.EvidenceRefs...),
		},
	), nil
}

func opesRequiredTestsForJobV0(
	jobType string,
	jobRef string,
	workRefs []string,
) []orquestadomainwork.DomainWorkRequiredTestV0 {
	safeJob := compactOPESBridgeRefV0(jobRef)
	workKind := compactOPESBridgeRefV0(jobType)
	artifactType := compactOPESBridgeRefV0(expectedArtifactTypeV0(jobType))
	tests := []orquestadomainwork.DomainWorkRequiredTestV0{{
		TestRef: "opes-domain-test-" + workKind + "-" + safeJob,
		AcceptanceCriteria: []string{
			"OPES acepta el artefacto por contrato publico submit_artifact.",
			"El receipt OPES conserva job_ref, delivery_ref y trazabilidad causal.",
		},
		AcceptanceCriteriaRefs: []string{"opes-required-artifact-" + artifactType},
		InputRefs:              compactStringsV0(append([]string{"opes-job-" + safeJob}, workRefs...)),
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "domain_ref", Ref: "opes"},
			{Kind: "job_ref", Ref: safeJob},
			{Kind: "artifact_type", Ref: artifactType},
		},
		EvidenceRefs: []string{
			"opes-job-" + safeJob,
			"opes-expected-artifact-" + artifactType,
		},
	}}
	tests = append(tests, opesArtifactSpecificRequiredTestsV0(jobType, safeJob, workRefs)...)
	if opesFinalPackageWorkKindV0(jobType) {
		tests = append(tests, opesFinalPackageRequiredTestsV0(safeJob, workRefs)...)
	}
	if opesAudioWorkKindV0(jobType) {
		tests = append(tests, opesAudioTTSResumableRequiredTestsV0(safeJob, workRefs)...)
	}
	if opesQuestionBankWorkKindV0(jobType) {
		tests = append(tests, opesQuestionBankQARequiredTestsV0(safeJob, workRefs)...)
	}
	if opesTutorWorkKindV0(jobType) {
		tests = append(tests, opesTutorAssetsQARequiredTestsV0(safeJob, workRefs)...)
	}
	return tests
}

func opesArtifactSpecificRequiredTestsV0(
	jobType string,
	safeJob string,
	workRefs []string,
) []orquestadomainwork.DomainWorkRequiredTestV0 {
	switch expectedArtifactTypeV0(jobType) {
	case orquestadomainwork.DomainDocumentPlanArtifactTypeV0:
		return []orquestadomainwork.DomainWorkRequiredTestV0{opesNamedRequiredTestV0(
			"document-plan-contract", safeJob, workRefs,
			[]string{
				"El plan documental declara plan_ref, domain_ref, work_kind, document_kind, title, objective y entregables requeridos.",
				"Cada section, visual y review_step declara work_kind permitido, objetivo, criterios de aceptacion y refs trazables.",
				"El plan incluye la secuencia OPES completa hasta finalize_temario_package y no omite HTML, audio, tests, tutor, visuales ni QA.",
			},
			[]string{"opes-required-document-plan-contract"},
			[]string{"document_plan", "required_plan_parts"},
			[]string{"opes-rule-document-plan-complete", "opes-final-evidence:document_plan_contract"},
		)}
	case orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
		orquestadomainwork.DomainWorkArtifactTypeTopicSummaryV0,
		orquestadomainwork.DomainWorkArtifactTypeTopicExpansionPackageV0,
		orquestadomainwork.DomainWorkArtifactTypeAssembledTopicV0:
		return []orquestadomainwork.DomainWorkRequiredTestV0{opesNamedRequiredTestV0(
			"topic-text-publicable", safeJob, workRefs,
			[]string{
				"El texto publico pasa OPESTopicQualityContractV0 con contador canonico, minimo por nivel y evidencias de QA por tema.",
				"No contiene anclas visibles {#...}, tablas colapsadas en encabezados, mojibake, metacomentarios de examen, notas internas ni trazabilidad editorial visible.",
				"Si hay artefactos recuperables pero no publicables, el resultado declara invalid_artifact_paths, valid_artifact_paths y followup/rework causal.",
			},
			[]string{"opes-required-topic-text-publicable"},
			[]string{"topic_quality_contract_result", "public_text_qa_report"},
			[]string{"opes-rule-topic-quality-contract", "opes-final-evidence:topic_quality_contract_pass"},
		)}
	case orquestadomainwork.DomainWorkArtifactTypeVisualAssetV0:
		return []orquestadomainwork.DomainWorkRequiredTestV0{opesNamedRequiredTestV0(
			"didactic-visual-publicable", safeJob, workRefs,
			[]string{
				"El visual declara funcion didactica, anchor_ref o placement_ref, alt_text y relacion con una seccion publicable.",
				"No se acepta raster decorativo, SVG/boceto heredado ni imagen sin funcion docente como visual final.",
				"Si el visual es insumo o borrador, queda como recuperable con rework y no como final publicable.",
			},
			[]string{"opes-required-didactic-visual-publicable"},
			[]string{"didactic_visual_report", "visual_anchor_manifest"},
			[]string{"opes-rule-didactic-visual-required", "opes-final-evidence:visual_didactic_publicable"},
		)}
	case orquestadomainwork.DomainWorkArtifactTypeBlockRevisionV0,
		orquestadomainwork.DomainWorkArtifactTypeAgentReviewReportV0,
		orquestadomainwork.DomainWorkArtifactTypeAgentPairReviewReportV0,
		orquestadomainwork.DomainWorkArtifactTypeDirectorReviewMatrixV0:
		return []orquestadomainwork.DomainWorkRequiredTestV0{opesNamedRequiredTestV0(
			"review-report-actionable", safeJob, workRefs,
			[]string{
				"La revision separa findings bloqueantes, recuperables y aceptados con refs de artefacto y tema.",
				"No marca ready por narrativa libre: toda aceptacion o rework conserva issue_refs, evidence_refs y siguiente accion.",
				"Las revisiones por modelo o pareja no sustituyen la consolidacion del Director cuando hay contradicciones.",
			},
			[]string{"opes-required-review-report-actionable"},
			[]string{"review_report", "issue_refs", "evidence_refs"},
			[]string{"opes-rule-review-actionable", "opes-final-evidence:review_report_actionable"},
		)}
	case orquestadomainwork.DomainWorkArtifactTypeExamResearchReportV0,
		orquestadomainwork.DomainWorkArtifactTypeSourceV0:
		return []orquestadomainwork.DomainWorkRequiredTestV0{opesNamedRequiredTestV0(
			"source-research-traceable", safeJob, workRefs,
			[]string{
				"La investigacion declara fuentes, administracion, fecha, convocatoria o relacion con examen equivalente cuando aplique.",
				"Las fuentes quedan como refs opacas verificables y no como URLs/rutas sueltas sin resumen de uso.",
				"No se usa investigacion parcial como canon de temario sin revision de relevancia y cobertura.",
			},
			[]string{"opes-required-source-research-traceable"},
			[]string{"source_research_report", "source_refs"},
			[]string{"opes-rule-source-research-traceable", "opes-final-evidence:source_research_traceable"},
		)}
	case orquestadomainwork.DomainWorkArtifactTypePracticalCasesV0:
		return []orquestadomainwork.DomainWorkRequiredTestV0{opesNamedRequiredTestV0(
			"practical-cases-publicable", safeJob, workRefs,
			[]string{
				"Los supuestos practicos declaran schema importable, cobertura por tema, preguntas, solucion y criterios de correccion.",
				"Los JSON/JSONL se validan estructuralmente y los casos parciales quedan en rework, no ready.",
				"La consolidacion conserva conteos esperados, entregados, faltantes y refs de reparacion.",
			},
			[]string{"opes-required-practical-cases-publicable"},
			[]string{"practical_cases_schema_report", "practical_cases_coverage_report"},
			[]string{"opes-rule-practical-cases-schema", "opes-final-evidence:practical_cases_publicable"},
		)}
	case orquestadomainwork.DomainWorkArtifactTypeLocalHTMLSiteV0:
		return []orquestadomainwork.DomainWorkRequiredTestV0{opesNamedRequiredTestV0(
			"html-site-publicable", safeJob, workRefs,
			[]string{
				"El HTML local genera paginas tema_*.html revisables, indice, navegacion, assets enlazados y reporte de enlaces.",
				"No se acepta index.html, portada o carcasa como manifest de tema ni como sustituto de paginas publicables.",
				"El HTML incorpora tests, tutor, audio, visuales y RAG solo desde artefactos canonicos o evidencia explicita de no aplicabilidad.",
			},
			[]string{"opes-required-html-site-publicable"},
			[]string{"html_validation_report", "html_topic_pages_manifest"},
			[]string{"opes-rule-html-local-publicable", "opes-final-evidence:html_site_publicable"},
		)}
	case orquestadomainwork.DomainWorkArtifactTypeInteractivePracticeV0, opesArtifactTypeLearningGamesPackageV0:
		return []orquestadomainwork.DomainWorkRequiredTestV0{opesNamedRequiredTestV0(
			"interactive-practice-publicable", safeJob, workRefs,
			[]string{
				"Los juegos o practicas declaran reglas, dataset, accesibilidad, integracion HTML y trazabilidad a tema/apartado.",
				"No se acepta demo visual sin contenido evaluable, sin fallback ni sin QA funcional.",
				"El paquete conserva build/test local o evidencia de validacion manual estructurada.",
			},
			[]string{"opes-required-interactive-practice-publicable"},
			[]string{"interactive_practice_manifest", "interactive_practice_qa_report"},
			[]string{"opes-rule-interactive-practice-publicable", "opes-final-evidence:interactive_practice_publicable"},
		)}
	case orquestadomainwork.DomainWorkArtifactTypeHelpPackageV0, opesArtifactTypeHelpManualPackageV0:
		return []orquestadomainwork.DomainWorkRequiredTestV0{opesNamedRequiredTestV0(
			"help-manual-publicable", safeJob, workRefs,
			[]string{
				"Los manuales de ayuda declaran YAML, capturas o evidencias visuales, index.html, manual.md o manual.pdf segun contrato.",
				"El manual se deriva del HTML local y no inventa opciones ni pantallas no existentes.",
				"Si faltan capturas, indice o validacion de enlaces, queda pendiente_continuar con refs causales.",
			},
			[]string{"opes-required-help-manual-publicable"},
			[]string{"help_manual_manifest", "help_manual_qa_report"},
			[]string{"opes-rule-help-manual-publicable", "opes-final-evidence:help_manual_publicable"},
		)}
	case opesArtifactTypeVisualReuseManifestV0:
		return []orquestadomainwork.DomainWorkRequiredTestV0{opesNamedRequiredTestV0(
			"visual-reuse-manifest", safeJob, workRefs,
			[]string{
				"El manifest de reutilizacion visual lista reutilizables, copiados, insertados, rechazados y no aplicables con motivo.",
				"Cada visual reutilizado mantiene ref opaca, placement_ref, alt_text y aparicion en html_final/html_ampliado.",
				"No se acepta visual_count=0 sin justificacion de no aplicabilidad.",
			},
			[]string{"opes-required-visual-reuse-manifest"},
			[]string{"visual_reuse_manifest"},
			[]string{"opes-rule-visual-reuse-common-assets", "opes-final-evidence:visual_reuse"},
		)}
	case orquestadomainwork.DomainWorkArtifactTypeTopicRegistryUpdateV0:
		return []orquestadomainwork.DomainWorkRequiredTestV0{opesNamedRequiredTestV0(
			"topic-registry-update", safeJob, workRefs,
			[]string{
				"El registro de tema conserva course_id, topic_id, settlement_status, operational_status, pending_refs y done_refs.",
				"Un tema goal-first no queda settled_text o settled_final sin checkpoint durable y refs de lifecycle.",
				"Si faltan permisos o conector, el registro publica bloqueo operativo y no marca ready.",
			},
			[]string{"opes-required-topic-registry-update"},
			[]string{"topic_registry_update", "settlement_status", "goal_first_checkpoint_refs"},
			[]string{"opes-rule-topic-registry-causal", "opes-final-evidence:topic_registry_update"},
		)}
	}
	return nil
}

func opesNamedRequiredTestV0(
	name string,
	safeJob string,
	workRefs []string,
	criteria []string,
	criteriaRefs []string,
	requiredEvidence []string,
	evidenceRefs []string,
) orquestadomainwork.DomainWorkRequiredTestV0 {
	inputRefs := compactStringsV0(append([]string{"opes-job-" + safeJob}, workRefs...))
	externalRefs := []orquestadomainwork.DomainWorkExternalRefV0{
		{Kind: "domain_ref", Ref: "opes"},
		{Kind: "job_ref", Ref: safeJob},
		{Kind: "required_test_name", Ref: name},
	}
	for _, evidence := range requiredEvidence {
		if evidence = strings.TrimSpace(evidence); evidence != "" {
			externalRefs = append(externalRefs, orquestadomainwork.DomainWorkExternalRefV0{
				Kind: "required_evidence",
				Ref:  evidence,
			})
		}
	}
	return orquestadomainwork.DomainWorkRequiredTestV0{
		TestRef:                "opes-" + compactOPESBridgeRefV0(name) + "-" + safeJob,
		AcceptanceCriteria:     append([]string(nil), criteria...),
		AcceptanceCriteriaRefs: compactStringsV0(criteriaRefs),
		InputRefs:              inputRefs,
		ExternalRefs:           externalRefs,
		EvidenceRefs:           compactStringsV0(evidenceRefs),
	}
}

func opesFinalPackageRequiredTestsV0(
	safeJob string,
	workRefs []string,
) []orquestadomainwork.DomainWorkRequiredTestV0 {
	inputRefs := compactStringsV0(append([]string{"opes-job-" + safeJob}, workRefs...))
	return []orquestadomainwork.DomainWorkRequiredTestV0{
		{
			TestRef: "opes-extension_pass-" + safeJob,
			AcceptanceCriteria: []string{
				"Existe informe_extension_temario.json y .md con conteo por tema y estado publico extension_pass.",
				"Cada ampliado publicable alcanza el minimo de su nivel con texto propio del tema: A1 20.250, A2 14.400, B 10.800, C1 7.200, C2 4.500 o AP 3.150 palabras.",
				"No se acepta extension conseguida por anexos ajenos, andamiaje de estudio, trazabilidad interna, contenido de otros temas o material no publicable.",
				"Si algun tema no llega, el estado es pendiente_continuar con needs_expansion_min_words_<nivel>.",
			},
			AcceptanceCriteriaRefs: []string{"opes-required-extension_pass"},
			InputRefs:              inputRefs,
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
				{Kind: "domain_ref", Ref: "opes"},
				{Kind: "job_ref", Ref: safeJob},
				{Kind: "required_test_name", Ref: "extension_pass"},
				{Kind: "required_evidence", Ref: "informe_extension_temario"},
			},
			EvidenceRefs: []string{
				"opes-rule-minimos-extension-temarios-2026-06-22",
				"opes-expected-evidence-informe-extension-temario",
				"opes-final-evidence:extension_pass",
			},
		},
		{
			TestRef: "opes-official_text_qa_pass-" + safeJob,
			AcceptanceCriteria: []string{
				"Existe informe oficial de texto publico con estado official_text_qa_pass por tema y por paquete.",
				"El texto publicable no contiene metanotas de examen, notas de autor, mojibake, rutas internas, refs tecnicas visibles ni placeholders de revision.",
				"Los validadores oficiales de texto publico OPES quedan verdes sobre el texto final que vera el alumnado, no sobre borradores auxiliares.",
				"Si falla la QA oficial de texto, el estado es pendiente_continuar con followup_refs causales y no listo_para_revision_operador.",
			},
			AcceptanceCriteriaRefs: []string{"opes-required-official_text_qa_pass"},
			InputRefs:              inputRefs,
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
				{Kind: "domain_ref", Ref: "opes"},
				{Kind: "job_ref", Ref: safeJob},
				{Kind: "required_test_name", Ref: "official_text_qa_pass"},
				{Kind: "required_evidence", Ref: "official_text_qa_report"},
			},
			EvidenceRefs: []string{
				"opes-rule-public-text-official-qa",
				"opes-expected-evidence-official-text-qa-report",
				"opes-final-evidence:official_text_qa_pass",
			},
		},
		{
			TestRef: "opes-strict_editorial_qa_pass-" + safeJob,
			AcceptanceCriteria: []string{
				"Existe informe estricto de QA editorial con estado strict_editorial_qa_pass por tema y por paquete.",
				"La QA estricta comprueba ausencia de andamiaje interno de estudio, calendarios, trazabilidad editorial visible, referencias a ficheros internos, SVG/Markdown visibles, bloques de canon/maestro y contaminacion cruzada entre temas.",
				"El informe usa una validacion equivalente a validate_public_text_no_study_scaffolding.py o una regla OPES publica versionada equivalente.",
				"Si falla andamiaje interno o contaminacion cruzada, el estado es pendiente_rework_editorial o needs_remove_study_scaffolding_and_cross_topic_contamination, nunca ready.",
			},
			AcceptanceCriteriaRefs: []string{"opes-required-strict_editorial_qa_pass"},
			InputRefs:              inputRefs,
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
				{Kind: "domain_ref", Ref: "opes"},
				{Kind: "job_ref", Ref: safeJob},
				{Kind: "required_test_name", Ref: "strict_editorial_qa_pass"},
				{Kind: "required_evidence", Ref: "strict_editorial_qa_report"},
			},
			EvidenceRefs: []string{
				"opes-rule-no-study-scaffolding-and-cross-topic-contamination",
				"opes-expected-evidence-strict-editorial-qa-report",
				"opes-final-evidence:strict_editorial_qa_pass",
			},
		},
		{
			TestRef: "opes-derivacion-comunes-maestro-" + safeJob,
			AcceptanceCriteria: []string{
				"Cada tema comun declara matriz de derivacion desde maestro comun A1/A1-A2 o superior validado.",
				"Si no hay temas comunes, existe evidencia explicita de no aplicabilidad.",
				"No se acepta reutilizar un curso vecino como canon cuando exista maestro comun superior.",
			},
			AcceptanceCriteriaRefs: []string{"opes-required-derivacion-comunes-maestro"},
			InputRefs:              inputRefs,
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
				{Kind: "domain_ref", Ref: "opes"},
				{Kind: "job_ref", Ref: safeJob},
				{Kind: "required_evidence", Ref: "matriz_reutilizacion_comunes"},
			},
			EvidenceRefs: []string{
				"opes-rule-comunes-a1-genericos",
				"opes-expected-evidence-matriz-reutilizacion-comunes",
			},
		},
		{
			TestRef: "opes-question-bank-publicable-" + safeJob,
			AcceptanceCriteria: []string{
				"Existe tests.json, banco_preguntas_i18n_es.json o question_bank equivalente con preguntas publicables por tema.",
				"Cada tema declara al menos 50 preguntas, 4 opciones A/B/C/D, una unica respuesta correcta exacta, distractores plausibles y explicacion tutor para aciertos y fallos.",
				"Existe informe estructural y de dificultad/proximidad del banco, mas revision 100% Codex/Gemini/Claude por lotes si hace falta.",
				"Un paquete con JSON parseable pero sin preguntas suficientes, opciones incompletas, respuesta ambigua, distractores triviales o sin revision triple queda pendiente_continuar.",
			},
			AcceptanceCriteriaRefs: []string{"opes-required-question-bank-publicable"},
			InputRefs:              inputRefs,
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
				{Kind: "domain_ref", Ref: "opes"},
				{Kind: "job_ref", Ref: safeJob},
				{Kind: "required_test_name", Ref: "question_bank_publicable"},
				{Kind: "required_evidence", Ref: "question_bank"},
				{Kind: "required_evidence", Ref: "question_bank_structural_report"},
				{Kind: "required_evidence", Ref: "question_bank_difficulty_report"},
				{Kind: "required_evidence", Ref: "question_bank_three_model_review"},
			},
			EvidenceRefs: []string{
				"opes-rule-question-bank-publicable",
				"opes-expected-evidence-question-bank",
				"opes-expected-evidence-question-bank-qa-report",
				"opes-final-evidence:question_bank_publicable",
			},
		},
		opesTutorAssetsQARequiredTestsV0(safeJob, workRefs)[0],
		{
			TestRef: "opes-final-package-manifest-" + safeJob,
			AcceptanceCriteria: []string{
				"Existe manifest_cierre.json del completed_syllabus_package con schema opes_final_package_evidence_manifest.v0.",
				"El manifest identifica package_ref, manifest_ref, checksum_refs, validation_report_ref y review_matrix_ref del paquete final.",
				"El manifest declara evidencias requeridas para HTML, RAG, audio, tests, tutor, visual, qa_passes y qa_report_refs separados para extension_pass, official_text_qa_pass, strict_editorial_qa_pass, question_bank_publicable y tutor_assets_publicable.",
				"El RAG final usa rag/corpus/chunks.jsonl, rag/corpus/summary.json y rag/manifest.json, con metadata course_id y source_variant o equivalentes en chunks y summary antes de aceptar ready.",
				"Si falta una evidencia obligatoria, el estado es pendiente_continuar con followup_refs causales y no listo_para_revision_operador.",
			},
			AcceptanceCriteriaRefs: []string{"opes-required-final-package-manifest"},
			InputRefs:              inputRefs,
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
				{Kind: "domain_ref", Ref: "opes"},
				{Kind: "job_ref", Ref: safeJob},
				{Kind: "artifact_type", Ref: opesArtifactTypeCompletedSyllabusPackageV0},
				{Kind: "required_evidence", Ref: "manifest_cierre"},
			},
			EvidenceRefs: []string{
				"opes-rule-final-package-manifest",
				"opes-expected-evidence-manifest-cierre",
				"opes-final-evidence:html",
				"opes-final-evidence:rag",
				"opes-final-evidence:audio",
				"opes-final-evidence:tests",
				"opes-final-evidence:tutor",
				"opes-final-evidence:visual",
				"opes-final-evidence:extension_pass",
				"opes-final-evidence:official_text_qa_pass",
				"opes-final-evidence:strict_editorial_qa_pass",
				"opes-final-evidence:question_bank_publicable",
				"opes-final-evidence:tutor_assets_publicable",
			},
		},
		opesAudioTTSResumableRequiredTestsV0(safeJob, workRefs)[0],
		{
			TestRef: "opes-visual-reuse-manifest-" + safeJob,
			AcceptanceCriteria: []string{
				"Existe visual_reuse_manifest o evidencia explicita de no aplicabilidad para cursos con comunes/assets visuales reutilizables.",
				"Si reusable_visual_count o common_visual_count es mayor que cero, copied_visual_count/inserted_visual_count reflejan assets importados y ubicados.",
				"Si visual_count=0, el manifest declara visual_requirement_status=not_applicable o visual_zero_justification_ref; no se acepta ready/html_validado sin esa evidencia.",
				"Los assets reutilizados conservan refs opacas, placement_ref/ancla, alt_text y motivo editorial; los rechazados conservan motivo de rechazo.",
				"Tras rebuild HTML, cada visual manifestado para html_final/html_ampliado sigue existiendo y esta referenciado desde las paginas tema_*.html correspondientes.",
			},
			AcceptanceCriteriaRefs: []string{"opes-required-visual-reuse-manifest"},
			InputRefs:              inputRefs,
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
				{Kind: "domain_ref", Ref: "opes"},
				{Kind: "job_ref", Ref: safeJob},
				{Kind: "required_evidence", Ref: "visual_reuse_manifest"},
			},
			EvidenceRefs: []string{
				"opes-rule-visual-reuse-common-assets",
				"opes-expected-evidence-visual-reuse-manifest",
				"opes-final-evidence:visual_reuse",
			},
		},
	}
}

func opesQuestionBankQARequiredTestsV0(
	safeJob string,
	workRefs []string,
) []orquestadomainwork.DomainWorkRequiredTestV0 {
	inputRefs := compactStringsV0(append([]string{"opes-job-" + safeJob}, workRefs...))
	return []orquestadomainwork.DomainWorkRequiredTestV0{{
		TestRef: "opes-question-bank-publicable-" + safeJob,
		AcceptanceCriteria: []string{
			"El banco entrega JSON por tema y HTML revisable, con metadata de curso/tema, fuente de contenido y validacion estructural.",
			"Cada tema contiene al menos 50 preguntas publicables, 4 opciones A/B/C/D, una unica respuesta correcta exacta, distractores plausibles y explicacion tutor para cada opcion.",
			"La QA cubre dificultad/proximidad de distractores y revision 100% Codex/Gemini/Claude por lotes si hace falta; una muestra o revision parcial no cierra el banco.",
			"Si faltan preguntas, opciones, explicaciones, metadata, informes o revision triple, el estado es pendiente_continuar con followup_refs causales, no ready.",
		},
		AcceptanceCriteriaRefs: []string{"opes-required-question-bank-publicable"},
		InputRefs:              inputRefs,
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "domain_ref", Ref: "opes"},
			{Kind: "job_ref", Ref: safeJob},
			{Kind: "required_test_name", Ref: "question_bank_publicable"},
			{Kind: "required_evidence", Ref: "question_bank"},
			{Kind: "required_evidence", Ref: "question_bank_html_review"},
			{Kind: "required_evidence", Ref: "question_bank_structural_report"},
			{Kind: "required_evidence", Ref: "question_bank_difficulty_report"},
			{Kind: "required_evidence", Ref: "question_bank_three_model_review"},
		},
		EvidenceRefs: []string{
			"opes-rule-question-bank-publicable",
			"opes-rule-review-tests-three-models",
			"opes-expected-evidence-question-bank",
			"opes-expected-evidence-question-bank-qa-report",
			"opes-final-evidence:question_bank_publicable",
		},
	}}
}

func opesTutorAssetsQARequiredTestsV0(
	safeJob string,
	workRefs []string,
) []orquestadomainwork.DomainWorkRequiredTestV0 {
	inputRefs := compactStringsV0(append([]string{"opes-job-" + safeJob}, workRefs...))
	return []orquestadomainwork.DomainWorkRequiredTestV0{{
		TestRef: "opes-tutor-assets-publicable-" + safeJob,
		AcceptanceCriteria: []string{
			"Existe paquete tutor/bots con fuentes canonicas aprobadas, prompt/guardas de alcance, mapa tema/apartado y fallback cuando no hay evidencia suficiente.",
			"Si genera RAG, el corpus final usa rag/corpus/chunks.jsonl, rag/corpus/summary.json y rag/manifest.json reconstruidos desde HTML final/local, tests y tutor fuente limpios.",
			"La QA del tutor comprueba respuestas por tema/apartado, explicacion de fallos de test, recomendacion de repaso, ausencia de invencion fuera de fuentes y cobertura de bloques clave.",
			"Un tutor con corpus suelto, fuentes regenerables usadas como canon, respuestas sin trazabilidad o sin QA por tema queda pendiente_continuar, no listo_para_revision_operador.",
		},
		AcceptanceCriteriaRefs: []string{"opes-required-tutor-assets-publicable"},
		InputRefs:              inputRefs,
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "domain_ref", Ref: "opes"},
			{Kind: "job_ref", Ref: safeJob},
			{Kind: "required_test_name", Ref: "tutor_assets_publicable"},
			{Kind: "required_evidence", Ref: "tutor_bot_package"},
			{Kind: "required_evidence", Ref: "tutor_scope_guard_report"},
			{Kind: "required_evidence", Ref: "tutor_qa_report"},
			{Kind: "required_evidence", Ref: "rag_corpus_manifest"},
		},
		EvidenceRefs: []string{
			"opes-rule-tutor-bots-canonical-sources",
			"opes-rule-rag-corpus-regenerable",
			"opes-expected-evidence-tutor-qa-report",
			"opes-final-evidence:tutor_assets_publicable",
		},
	}}
}

func opesAudioTTSResumableRequiredTestsV0(
	safeJob string,
	workRefs []string,
) []orquestadomainwork.DomainWorkRequiredTestV0 {
	inputRefs := compactStringsV0(append([]string{"opes-job-" + safeJob}, workRefs...))
	return []orquestadomainwork.DomainWorkRequiredTestV0{{
		TestRef: "opes-audio-tts-resumable-" + safeJob,
		AcceptanceCriteria: []string{
			"Antes de considerar audio listo, OPES declara supervision TTS con heartbeat de progreso y provider_timeout operativo o aporta evidencia de reanudacion segura.",
			"La evidencia de reanudacion identifica sidecar/manifest de audio, MP3 validos preservados, MP3 faltantes u obsoletos regenerados y contadores de skipped_valid_mp3_refs/generated_mp3_refs.",
			"No se duplican MP3 validos: cada section_ref narrable conserva un unico audio_ref final vigente, y los MP3 ya validos se reutilizan o se marcan como skipped_valid_mp3_refs.",
			"Si faltan heartbeat/provider_timeout y tambien falta reanudacion sin duplicar validos, el estado es pendiente_continuar con retry_from_phase=tts, no ready ni listo_para_revision_operador.",
		},
		AcceptanceCriteriaRefs: []string{"opes-required-audio-tts-resumable"},
		InputRefs:              inputRefs,
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "domain_ref", Ref: "opes"},
			{Kind: "job_ref", Ref: safeJob},
			{Kind: "required_test_name", Ref: "audio_tts_resumable"},
			{Kind: "required_evidence", Ref: "audio_tts_operational_manifest"},
			{Kind: "required_evidence", Ref: "progress_heartbeat"},
			{Kind: "required_evidence", Ref: "provider_timeout"},
			{Kind: "required_evidence", Ref: "resume_without_duplicate_valid_mp3"},
		},
		EvidenceRefs: []string{
			"opes-rule-audio-tts-heartbeat-timeout-or-resume",
			"opes-expected-evidence-audio-tts-operational-manifest",
			"opes-expected-evidence-resume-without-duplicate-valid-mp3",
			"opes-final-evidence:audio_tts_resumable",
		},
	}}
}

func opesAudioWorkKindV0(jobType string) bool {
	return expectedArtifactTypeV0(jobType) == orquestadomainwork.DomainWorkArtifactTypeAudioAssetV0
}

func opesQuestionBankWorkKindV0(jobType string) bool {
	return expectedArtifactTypeV0(jobType) == orquestadomainwork.DomainWorkArtifactTypeQuestionBankV0
}

func opesTutorWorkKindV0(jobType string) bool {
	return expectedArtifactTypeV0(jobType) == orquestadomainwork.DomainWorkArtifactTypeTutorBotPackageV0
}

func opesFinalPackageWorkKindV0(jobType string) bool {
	switch strings.TrimSpace(jobType) {
	case "finalize_topic_package",
		"finalize_temario_package",
		"close_temario_package",
		"finalize_syllabus_package",
		"close_syllabus_package":
		return true
	default:
		return false
	}
}

func opesRequiredTestJobRefV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) string {
	for _, ref := range request.WorkRefs {
		if strings.TrimSpace(ref) != "" {
			return ref
		}
	}
	return request.IdempotencyKey
}
