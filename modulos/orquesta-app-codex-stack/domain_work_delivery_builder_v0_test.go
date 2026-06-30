package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestDefaultDomainWorkArtifactSubmissionBuilderV0NoFiltraPathsComoPayloadRefs(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "draft_content_block")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(bodyPath, []byte("# Bloque OPES\n\nContenido."), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run: orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{
					TaskID: "task-ref-001",
					Title:  "Redactar bloque documental OPES",
				},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-001",
						ChangeRef:     "change-ref-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-001",
							WorkKind:   "draft_content_block",
							InputFields: []orquestadomainwork.DomainWorkFieldV0{
								{Name: "topic_id", Value: "topic-ref-001"},
								{Name: "existing_operario_legacy_package_ref", Value: "/home/alberto/Trabajo/OPES/opes-salidas/codex_directo/operario/AP/produccion_externa_2026-05-19"},
							},
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-001",
					ProjectWorkDir: projectDir,
					Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
						AgentPacket: orquestaruntime.AgentStartPacketV0{
							DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
								MailboxRef:   "mailbox-ref-001",
								ReadinessRef: "readiness-ref-001",
							},
						},
					},
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/draft_content_block"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-001",
					AgentRef:     "agent-ref-001",
					Summary:      "Entrega validada.",
					EvidenceRefs: []string{"ack-ref-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if len(submission.PayloadRefs) != 0 {
		t.Fatalf("payload_refs no debe filtrar rutas internas: %+v", submission.PayloadRefs)
	}
	if !domainWorkFieldValueForTestV0(submission.PayloadFields, "body", "# Bloque OPES\n\nContenido.") {
		t.Fatalf("payload_fields=%+v", submission.PayloadFields)
	}
	encoded, err := json.Marshal(submission)
	if err != nil {
		t.Fatalf("marshal submission: %v", err)
	}
	if !strings.Contains(string(encoded), "/home/alberto/Trabajo/OPES/opes-salidas/codex_directo/operario/AP/produccion_externa_2026-05-19") {
		t.Fatalf("ref local no conservada: %s", string(encoded))
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0EntregaTopicSummaryOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "summarize_topic")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"markdown":"Resumen contrastado del tema.",
		"coverage":["prevencion","factores de riesgo"],
		"key_concepts":["prevencion primaria","epidemiologia"],
		"cross_topic_links":["tema-88"],
		"source_refs":["fuente-ref-001"]
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-summary-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-summary-001", Title: "Resumir tema OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-summary-001",
						ChangeRef:     "change-ref-summary-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-summary-001",
							WorkKind:   "summarize_topic",
							InputFields: []orquestadomainwork.DomainWorkFieldV0{
								{Name: "topic_id", Value: "topic-ref-001"},
							},
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-summary-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/summarize_topic"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-summary-001",
					AgentRef:     "agent-ref-summary-001",
					Summary:      "Resumen validado.",
					EvidenceRefs: []string{"ack-ref-summary-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "topic_summary" ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "markdown", "Resumen contrastado del tema.") ||
		!domainWorkFieldValuesForTestV0(submission.PayloadFields, "coverage", []string{"prevencion", "factores de riesgo"}) ||
		domainWorkFieldValueForTestV0(submission.PayloadFields, "body", "Resumen contrastado del tema.") {
		t.Fatalf("submission=%+v", submission)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0NormalizaSourceRefsRicosContentBlockOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "draft_content_block")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"artifact_type":"content_block",
		"payload_json":{
			"topic_id":"topic-ref-001",
			"chapter_id":"chapter-ref-001",
			"type":"doctrine",
			"title":"Fundamentos",
			"markdown":"Contenido con fuentes trazables.",
			"language_code":"es",
			"source_refs":[
				{"source_ref":"beck-1976","title":"Cognitive Therapy"},
				{"source_ref":"beck-1976","title":"duplicada"},
				{"source_ref":"ley-41-2002","title":"Autonomia del paciente"}
			],
			"citations":[{"ref":"beck-1976","locator":"cap. 1","claim":"apoyo doctrinal","url":"https://example.invalid/beck"}]
		}
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-content-rich-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-content-rich-001", Title: "Redactar bloque OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-content-rich-001",
						ChangeRef:     "change-ref-content-rich-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-content-rich-001",
							WorkKind:   "draft_content_block",
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-content-rich-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/draft_content_block"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-content-rich-001",
					AgentRef:     "agent-ref-content-rich-001",
					Summary:      "Bloque validado.",
					EvidenceRefs: []string{"ack-ref-content-rich-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "content_block" ||
		!domainWorkFieldValuesForTestV0(submission.PayloadFields, "source_refs", []string{"beck-1976", "ley-41-2002"}) ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "source_ref_details") ||
		!domainWorkCitationSourceRefForTestV0(submission.PayloadFields, "beck-1976") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "citation_details") {
		t.Fatalf("submission=%+v", submission)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0DerivaSourceRefsDesdeCitasOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "draft_content_block")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"artifact_type":"content_block",
		"payload_json":{
			"topic_id":"topic-ref-001",
			"chapter_id":"chapter-ref-001",
			"type":"doctrine",
			"title":"Fuentes oficiales",
			"markdown":"Contenido con citas oficiales y trazabilidad por fuente.",
			"language_code":"es",
			"citations":[
				{"ref":"boe-ley-41-2002","locator":"art. 2","claim":"normativa oficial"},
				{"source_ref":"boe-ley-41-2002","locator":"art. 3"},
				{"source_ref":"manual-doctrina-001","claim":"apoyo doctrinal"}
			]
		}
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-content-citations-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-content-citations-001", Title: "Redactar bloque OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-content-citations-001",
						ChangeRef:     "change-ref-content-citations-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-content-citations-001",
							WorkKind:   "draft_content_block",
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-content-citations-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/draft_content_block"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-content-citations-001",
					AgentRef:     "agent-ref-content-citations-001",
					Summary:      "Bloque con fuentes validado.",
					EvidenceRefs: []string{"ack-ref-content-citations-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if !domainWorkFieldValuesForTestV0(
		submission.PayloadFields,
		"source_refs",
		[]string{"boe-ley-41-2002", "manual-doctrina-001"},
	) || !domainWorkCitationSourceRefForTestV0(submission.PayloadFields, "manual-doctrina-001") {
		t.Fatalf("submission=%+v", submission)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0AceptaEnvelopeConNombreLibreOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "draft_content_block")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"artifact_type":"el_director_lo_llamo_como_quiso",
		"payload_json":{
			"tema_id":"topic-ref-001",
			"id_capitulo":"chapter-ref-001",
			"tipo_bloque":"doctrine",
			"titulo":"Fundamentos",
			"contenido":"Contenido materializable aunque cambien los nombres.",
			"idioma":"es",
			"fuentes":["fuente-ref-001"]
		}
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-content-alias-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-content-alias-001", Title: "Redactar bloque OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-content-alias-001",
						ChangeRef:     "change-ref-content-alias-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-content-alias-001",
							WorkKind:   "draft_content_block",
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-content-alias-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/draft_content_block"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-content-alias-001",
					AgentRef:     "agent-ref-content-alias-001",
					Summary:      "Bloque validado.",
					EvidenceRefs: []string{"ack-ref-content-alias-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "content_block" ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "topic_id", "topic-ref-001") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "chapter_id", "chapter-ref-001") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "block_type", "doctrine") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "title", "Fundamentos") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "body", "Contenido materializable aunque cambien los nombres.") ||
		!domainWorkFieldValuesForTestV0(submission.PayloadFields, "source_refs", []string{"fuente-ref-001"}) {
		t.Fatalf("submission=%+v", submission)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0EntregaExpansionPackageOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "expand_topic_from_summary")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"topic_id":"topic-ref-001",
		"language_code":"es",
		"chapters":[
			{"title":"Capitulo 1","order":1,"blocks":[
				{"block_type":"doctrine","title":"Bloque 1","markdown":"Contenido ampliado.","source_refs":["fuente-ref-001"]}
			]}
		]
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-expansion-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-expansion-001", Title: "Ampliar tema OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-expansion-001",
						ChangeRef:     "change-ref-expansion-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-expansion-001",
							WorkKind:   "expand_topic_from_summary",
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-expansion-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/expand_topic_from_summary"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-expansion-001",
					AgentRef:     "agent-ref-expansion-001",
					Summary:      "Expansion validada.",
					EvidenceRefs: []string{"ack-ref-expansion-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "topic_expansion_package" ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "topic_id", "topic-ref-001") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "chapters") {
		t.Fatalf("submission=%+v", submission)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0EntregaAssembledTopicOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "assemble_topic")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"topic_id":"topic-ref-001",
		"title":"Tema ensamblado",
		"language_code":"es",
		"sections":[{"title":"Introduccion","markdown":"Contenido ensamblado."}],
		"source_refs":["fuente-ref-001"]
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-assemble-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-assemble-001", Title: "Ensamblar tema OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-assemble-001",
						ChangeRef:     "change-ref-assemble-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-assemble-001",
							WorkKind:   "assemble_topic",
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-assemble-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/assemble_topic"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-assemble-001",
					AgentRef:     "agent-ref-assemble-001",
					Summary:      "Tema ensamblado validado.",
					EvidenceRefs: []string{"ack-ref-assemble-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "assembled_topic" ||
		submission.ArtifactType == "work_delivery" ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "topic_id", "topic-ref-001") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "sections") {
		t.Fatalf("submission=%+v", submission)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDomainWorkArtifactTypeForWorkKindV0NormalizaAliasPedagogicosYEnsamblado(t *testing.T) {
	cases := map[string]string{
		"revision_pedagogica":      "block_revision",
		"revision calidad":         "block_revision",
		"validacion_tema":          "block_revision",
		"ensamblado_y_exportacion": "assembled_topic",
		"audio_tema":               "audio_asset",
		"generacion audio":         "audio_asset",
		"tts_topic":                "audio_asset",
		"redaccion documental":     "content_block",
		"plan visual":              "visual_asset",
	}
	for workKind, want := range cases {
		if got := domainWorkArtifactTypeForWorkKindV0(workKind); got != want {
			t.Fatalf("work_kind %q artifact=%q, want %q", workKind, got, want)
		}
		if !domainWorkDeliveryArtifactTypeMatchesV0(workKind, want) {
			t.Fatalf("work_kind %q no matchea artifact %q", workKind, want)
		}
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0EntregaAudioAssetOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "generate_audio_asset")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"tema_id":"topic-ref-001",
		"tema_ensamblado_ref":"artifact-assembled-topic-001",
		"idioma":"es",
		"formato":"mp3",
		"tipo_mime":"audio/mpeg",
		"duracion_segundos":"1830",
		"ref_audio":"audio-ref-topic-001-mp3",
		"ref_manifest":"manifest-ref-topic-001-audio",
		"sha256":"sha256-ref-audio-001"
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-audio-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-audio-001", Title: "Generar audio accesible OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-audio-001",
						ChangeRef:     "change-ref-audio-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-audio-001",
							WorkKind:   "audio_tema",
							InputFields: []orquestadomainwork.DomainWorkFieldV0{
								{Name: "audio_profile_ref", Value: "audio-profile-accessible-es-001"},
							},
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-audio-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/generate_audio_asset"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-audio-001",
					AgentRef:     "agent-ref-audio-001",
					Summary:      "Audio accesible generado.",
					EvidenceRefs: []string{"ack-ref-audio-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "audio_asset" ||
		submission.ArtifactType == "work_delivery" ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "content_type", "application/json") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "topic_id", "topic-ref-001") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "assembled_topic_artifact_id", "artifact-assembled-topic-001") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "language_code", "es") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "format", "mp3") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "mime_type", "audio/mpeg") ||
		!domainWorkFieldJSONIntForTestV0(submission.PayloadFields, "duration_seconds", 1830) ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "audio_ref", "audio-ref-topic-001-mp3") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "manifest_ref", "manifest-ref-topic-001-audio") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "checksum", "sha256-ref-audio-001") {
		t.Fatalf("submission=%+v", submission)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0NormalizaAudioPorApartadosOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "generate_audio_asset", "job-ref-audio-sections-001")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"topic_id":"topic-ref-001",
		"assembled_topic_ref":"artifact-assembled-topic-001",
		"language":"es",
		"format":"mp3",
		"mime_type":"audio/mpeg",
		"audio_ref":"audio-ref-topic-001",
		"manifest_ref":"manifest-ref-topic-001",
		"section_audio_links":[{
			"section_ref":"sec-tema-01-constitucion",
			"audio_ref":"audio-ref-section-001",
			"manifest_ref":"manifest-ref-topic-001",
			"duration_seconds":42,
			"text_hash":"hash-section-001"
		}]
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-audio-sections-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-audio-sections-001", Title: "Generar audio por apartados OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-audio-sections-001",
						ChangeRef:     "change-ref-audio-sections-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-audio-sections-001",
							WorkKind:   "generate_audio_asset",
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-audio-sections-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/generate_audio_asset/job-ref-audio-sections-001"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef: "ack-ref-audio-sections-001",
					AgentRef:    "agent-ref-audio-sections-001",
					Summary:     "Manifest de audio por apartados generado.",
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if !domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "segments") ||
		domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "section_audio_links") {
		t.Fatalf("payload_fields=%+v", submission.PayloadFields)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0ElevaAudioManifestOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "generate_audio_asset", "job-ref-audio-manifest-001")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"schema_version":"opes.audio_asset.v1",
		"source_refs":{
			"course_id":"course-ref-001",
			"program_id":"program-ref-001",
			"topic_id":"topic-ref-001",
			"assembled_topic_artifact_id":"artifact-assembled-topic-001"
		},
		"audio_manifest":{
			"audio_profile_ref":"accessible-es",
			"audio_ref":"audio-ref-topic-v1",
			"manifest_ref":"audio-manifest-ref-v1",
			"language":"es",
			"duration_seconds_approx":{"global_topic":34,"package_total":62,"segments_total":28},
			"formats":[{"format":"mp3","media_type":"audio/mpeg","role":"primary_delivery"}],
			"segments":[{"section_ref":"sec-1","audio_ref":"audio-ref-sec-1","duration_seconds_approx":8}]
		}
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-audio-manifest-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-audio-manifest-001", Title: "Generar audio desde manifest OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-audio-manifest-001",
						ChangeRef:     "change-ref-audio-manifest-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-audio-manifest-001",
							WorkKind:   "generate_audio_asset",
							InputFields: []orquestadomainwork.DomainWorkFieldV0{
								{Name: "topic_id", Value: "topic-ref-001"},
								{Name: "assembled_topic_artifact_id", Value: "artifact-assembled-topic-001"},
							},
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-audio-manifest-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/generate_audio_asset/job-ref-audio-manifest-001"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef: "ack-ref-audio-manifest-001",
					AgentRef:    "agent-ref-audio-manifest-001",
					Summary:     "Manifest de audio generado.",
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if !domainWorkFieldValueForTestV0(submission.PayloadFields, "topic_id", "topic-ref-001") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "assembled_topic_artifact_id", "artifact-assembled-topic-001") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "audio_ref", "audio-ref-topic-v1") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "manifest_ref", "audio-manifest-ref-v1") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "audio_profile_ref", "accessible-es") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "language_code", "es") ||
		!domainWorkFieldJSONIntForTestV0(submission.PayloadFields, "duration_seconds", 62) ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "format", "mp3") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "mime_type", "audio/mpeg") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "segments") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "source_ref_details") ||
		!domainWorkFieldValuesForTestV0(submission.PayloadFields, "source_refs", []string{
			"artifact-assembled-topic-001",
			"topic-ref-001",
			"program-ref-001",
			"course-ref-001",
		}) {
		t.Fatalf("payload_fields=%+v", submission.PayloadFields)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0CanonicalizaDocumentPlanAliasesOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "plan_tema", "job-ref-plan-001")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"schema_version":"domain_document_plan.v0",
		"artifact_type":"document_plan",
		"work_kind":"plan_tema",
		"job_id":"job-ref-plan-001",
		"topic_id":"topic-ref-080",
		"topic_title":"Evaluacion diagnostica en psicologia",
		"document_kind":"tema_oposicion",
		"language_code":"es",
		"target_pages":{"min":45,"max":50},
		"sections":[{
			"section_id":"sec-01-presentacion",
			"title":"Presentacion",
			"objective":"Situar el proceso diagnostico.",
			"work_kind":"draft_content_block",
			"planned_pages":2,
			"required_points":["definiciones"],
			"acceptance_criteria":["lectura facil"]
		}],
		"review_steps":[{
			"review_id":"rev-01-legal",
			"work_kind":"review_legal",
			"scope":"Normativa y proteccion de datos."
		}],
		"deliverables":[{
			"deliverable_id":"del-01-topic-expansion-package",
			"name":"topic_expansion_package",
			"description":"Tema grande completo."
		}]
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-plan-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-plan-001", Title: "Planificar tema"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-plan-001",
						UserIntent:    "Planificar tema OPES.",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-plan-001",
							WorkKind:   "plan_tema",
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/plan_tema/job-ref-plan-001"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef: "ack-ref-plan-001",
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != orquestadomainwork.DomainDocumentPlanArtifactTypeV0 ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "plan_ref", "plan-job-ref-plan-001") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "sections") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "deliverables") {
		t.Fatalf("submission=%+v", submission)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0IdempotenciaEstablePorJobYArtefacto(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "draft_content_block")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(bodyPath, []byte("# Entrega\n"), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}
	baseInput := DomainWorkArtifactSubmissionBuildInputV0{
		Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-idem-001"},
		Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-idem-001", Title: "Redactar bloque"},
		Record: orquestaappchange.AppChangeRecordV0{
			Request: orquestaappchange.AppChangeRequestV0{
				CorrelationID: "corr-ref-idem-001",
				ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
					ProjectRef: "opes",
					JobRef:     "job-ref-idem-001",
					WorkKind:   "draft_content_block",
				},
			},
		},
		Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{ProjectWorkDir: projectDir},
		Ack:        orquestaruntimecodex.CodexAgentAckV0{Files: []string{"external/opes/draft_content_block"}},
	}
	firstInput := baseInput
	firstInput.Observation = orquestacionnucleoapp.AgentDeliveryObservationV0{DeliveryRef: "ack-ref-original"}
	secondInput := baseInput
	secondInput.Observation = orquestacionnucleoapp.AgentDeliveryObservationV0{DeliveryRef: "ack-ref-rework"}

	first, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(context.Background(), firstInput)
	if err != nil || !ok {
		t.Fatalf("first ok=%v err=%v", ok, err)
	}
	second, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(context.Background(), secondInput)
	if err != nil || !ok {
		t.Fatalf("second ok=%v err=%v", ok, err)
	}
	if first.IdempotencyKey != second.IdempotencyKey ||
		first.IdempotencyKey != "idem-domain-work-artifact-opes-job-ref-idem-001-content_block" {
		t.Fatalf("idempotency first=%s second=%s", first.IdempotencyKey, second.IdempotencyKey)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0DesenvuelveExpansionPackageOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "expand_topic_from_summary")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"artifact_type":"topic_expansion_package",
		"payload_json":{
			"topic_id":"topic-ref-001",
			"language_code":"es",
			"chapters":[
				{"title":"Capitulo 1","order":1,"blocks":[
					{"block_type":"doctrine","title":"Bloque 1","markdown":"uno dos tres cuatro cinco seis siete ocho","source_refs":["fuente-ref-001"]}
				]}
			]
		}
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-expansion-envelope-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-expansion-envelope-001", Title: "Ampliar tema OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-expansion-envelope-001",
						ChangeRef:     "change-ref-expansion-envelope-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-expansion-envelope-001",
							WorkKind:   "expand_topic_from_summary",
							InputFields: []orquestadomainwork.DomainWorkFieldV0{
								{Name: "target_words_min", Value: "8"},
							},
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-expansion-envelope-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/expand_topic_from_summary"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-expansion-envelope-001",
					AgentRef:     "agent-ref-expansion-envelope-001",
					Summary:      "Expansion validada.",
					EvidenceRefs: []string{"ack-ref-expansion-envelope-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "topic_expansion_package" ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "topic_id", "topic-ref-001") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "chapters") ||
		domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "payload_json") {
		t.Fatalf("submission=%+v", submission)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0NoBloqueaExpansionCortaOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "expand_topic_from_summary")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"topic_id":"topic-ref-001",
		"chapters":[{"title":"Capitulo 1","blocks":[{"title":"Bloque 1","markdown":"Texto corto."}]}]
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-expansion-short-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-expansion-short-001", Title: "Ampliar tema OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-expansion-short-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-expansion-short-001",
							WorkKind:   "expand_topic_from_summary",
							InputFields: []orquestadomainwork.DomainWorkFieldV0{
								{Name: "target_words_min", Value: "100"},
							},
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/expand_topic_from_summary"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("la expansion corta debe pasar a revision/rework, no bloquear submit: ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != orquestadomainwork.DomainWorkArtifactTypeTopicExpansionPackageV0 {
		t.Fatalf("submission=%+v", submission)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0NormalizaRevisionesProveedorOPES(t *testing.T) {
	cases := []struct {
		name         string
		workKind     string
		fileRef      string
		artifactType string
	}{
		{
			name:         "revision codex independiente",
			workKind:     "review_codex",
			fileRef:      "external/opes/review_codex.json",
			artifactType: orquestadomainwork.DomainWorkArtifactTypeAgentReviewReportV0,
		},
		{
			name:         "revision por pares codex gemini",
			workKind:     "review_pair_codex_gemini",
			fileRef:      "external/opes/review_pair_codex_gemini.json",
			artifactType: orquestadomainwork.DomainWorkArtifactTypeAgentPairReviewReportV0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			projectDir := t.TempDir()
			bodyPath := filepath.Join(projectDir, filepath.FromSlash(tc.fileRef))
			if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			payload := `{
				"artifact_type":"` + tc.artifactType + `",
				"payload_json":{
					"verdict":"needs_rework",
					"findings":["falta trazabilidad en dos preguntas"],
					"risks":["tests incompletos"],
					"rework_refs":["rework-tests-001"],
					"evidence_refs":["mini-package-manifest"]
				}
			}`
			if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
				t.Fatalf("write body: %v", err)
			}

			submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
				BuildDomainWorkArtifactSubmissionV0(
					context.Background(),
					DomainWorkArtifactSubmissionBuildInputV0{
						Run: orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-review-001"},
						Task: orquestacoreworkflow.WorkflowTaskV0{
							TaskID: "task-ref-review-001",
							Title:  "Revisar temario OPES",
						},
						Record: orquestaappchange.AppChangeRecordV0{
							Request: orquestaappchange.AppChangeRequestV0{
								CorrelationID: "corr-ref-review-001",
								ChangeRef:     "change-ref-review-001",
								ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
									ProjectRef: "opes",
									JobRef:     "job-ref-review-001",
									WorkKind:   tc.workKind,
									InputFields: []orquestadomainwork.DomainWorkFieldV0{
										{Name: "package_ref", Value: "mini-package-ref"},
									},
								},
							},
						},
						Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
							DescriptorRef:  "descriptor-ref-review-001",
							ProjectWorkDir: projectDir,
						},
						Ack: orquestaruntimecodex.CodexAgentAckV0{
							Files: []string{tc.fileRef},
						},
						Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
							DeliveryRef:  "ack-ref-review-001",
							AgentRef:     "agent-ref-review-001",
							Summary:      "Revision OPES validada.",
							EvidenceRefs: []string{"ack-ref-review-001"},
						},
					},
				)

			if err != nil || !ok {
				t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
			}
			if submission.ArtifactType != tc.artifactType ||
				!domainWorkFieldValueForTestV0(submission.PayloadFields, "verdict", "needs_rework") ||
				!domainWorkFieldValuesForTestV0(submission.PayloadFields, "findings", []string{"falta trazabilidad en dos preguntas"}) ||
				!domainWorkFieldValueForTestV0(submission.PayloadFields, "package_ref", "mini-package-ref") {
				t.Fatalf("submission=%+v", submission)
			}
			if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
				t.Fatalf("submission invalida: %+v", issues)
			}
		})
	}
}

func domainWorkFieldValuesForTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	values []string,
) bool {
	for _, field := range fields {
		if field.Name != name || len(field.Values) != len(values) {
			continue
		}
		matches := true
		for index := range values {
			if field.Values[index] != values[index] {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

func domainWorkFieldHasJSONForTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
) bool {
	for _, field := range fields {
		if field.Name == name && len(field.ValueJSON) > 0 && json.Valid(field.ValueJSON) {
			return true
		}
	}
	return false
}

func domainWorkFieldJSONIntForTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	want int,
) bool {
	for _, field := range fields {
		if field.Name != name || len(field.ValueJSON) == 0 {
			continue
		}
		var got int
		if err := json.Unmarshal(field.ValueJSON, &got); err == nil && got == want {
			return true
		}
	}
	return false
}

func domainWorkCitationSourceRefForTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	sourceRef string,
) bool {
	for _, field := range fields {
		if field.Name != "citations" || len(field.ValueJSON) == 0 {
			continue
		}
		var citations []struct {
			SourceRef string `json:"source_ref"`
			Note      string `json:"note"`
		}
		if err := json.Unmarshal(field.ValueJSON, &citations); err != nil {
			continue
		}
		for _, citation := range citations {
			if citation.SourceRef == sourceRef && strings.Contains(citation.Note, "claim: apoyo doctrinal") {
				return true
			}
		}
	}
	return false
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0EntregaVisualAssetOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "generate_visual_asset")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 420" role="img" aria-label="Red en estrella"></svg>`
	if err := os.WriteFile(bodyPath, []byte(svg), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-visual-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-visual-001", Title: "Generar recurso visual OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-visual-001",
						ChangeRef:     "change-ref-visual-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-visual-001",
							WorkKind:   "generate_visual_asset",
							InputFields: []orquestadomainwork.DomainWorkFieldV0{
								{Name: "topic_id", Value: "topic-ref-001"},
								{Name: "chapter_id", Value: "chapter-ref-001"},
								{Name: "asset_type", Value: "vignette"},
								{Name: "format", Value: "svg"},
								{Name: "title", Value: "Red en estrella"},
								{Name: "caption", Value: "Topologia con nodo central."},
								{Name: "alt_text", Value: "Switch central conectado a equipos cliente."},
								{Name: "placement", Value: "after_block"},
								{Name: "language_code", Value: "es"},
							},
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-visual-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/generate_visual_asset"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-visual-001",
					AgentRef:     "agent-ref-visual-001",
					Summary:      "Visual validado.",
					EvidenceRefs: []string{"ack-ref-visual-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "visual_asset" ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "content_type", "image/svg+xml") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "body", svg) ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "title", "Red en estrella") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "caption", "Topologia con nodo central.") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "alt_text", "Switch central conectado a equipos cliente.") {
		t.Fatalf("submission=%+v", submission)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0NormalizaVisualAssetRicoOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "generate_visual_asset", "visual_asset.json")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 420" role="img" aria-label="Mapa conceptual"></svg>`
	payload := `{
		"artifact_type":"visual_asset",
		"payload_json":{
			"schema_version":"opes.visual_asset.v1",
			"contract_status":"complete",
			"source_refs":{
				"course_id":"course-ref-001",
				"program_id":"program-ref-001",
				"topic_id":"topic-ref-001",
				"visual_ref":"visual-ref-001",
				"official_order":1
			},
			"input_contract":{
				"expected_artifact_type":"visual_asset",
				"asset_type":"diagram",
				"aspect_ratio":"16:9"
			},
			"assets":[{
				"asset_id":"asset-ref-001",
				"artifact_type":"visual_asset",
				"asset_type":"diagram",
				"visual_kind":"full_topic_concept_map_prompt",
				"placement_ref":"topic-ref-001#visual-ref-001#official_order=1",
				"program_id":"program-ref-001",
				"topic_id":"topic-ref-001",
				"course_id":"course-ref-001",
				"title":"Mapa conceptual del tema completo",
				"didactic_objective":"Relacionar los conceptos principales del tema.",
				"editorial_rationale":"Ayuda a revisar el bloque antes del test.",
				"recommended_placement":{"placement_ref":"topic-ref-001#after-introduction"},
				"alt_text":"Mapa conceptual con los conceptos principales conectados.",
				"generation_prompt":{
					"language":"es",
					"prompt":"Genera un mapa conceptual limpio para topic-ref-001.",
					"negative_prompt":"No incluir marcas de agua ni datos no citados."
				},
				"diagram_blueprint":{
					"format":"svg_wireframe",
					"purpose":"Mapa conceptual accesible.",
					"svg":` + strconv.Quote(svg) + `
				}
			}]
		}
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-visual-rich-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-visual-rich-001", Title: "Generar recurso visual OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-visual-rich-001",
						ChangeRef:     "change-ref-visual-rich-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-visual-rich-001",
							WorkKind:   "generate_visual_asset",
							InputFields: []orquestadomainwork.DomainWorkFieldV0{
								{Name: "chapter_id", Value: "chapter-ref-001"},
								{Name: "language_code", Value: "es"},
							},
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-visual-rich-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/generate_visual_asset/visual_asset.json"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-visual-rich-001",
					AgentRef:     "agent-ref-visual-rich-001",
					Summary:      "Visual validado.",
					EvidenceRefs: []string{"ack-ref-visual-rich-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "visual_asset" ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "content_type", "image/svg+xml") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "topic_id", "topic-ref-001") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "asset_type", "diagram") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "format", "svg") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "title", "Mapa conceptual del tema completo") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "caption", "Relacionar los conceptos principales del tema.") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "alt_text", "Mapa conceptual con los conceptos principales conectados.") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "body", svg) ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "svg", svg) ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "prompt", "Genera un mapa conceptual limpio para topic-ref-001.") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "negative_prompt", "No incluir marcas de agua ni datos no citados.") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "assets") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "source_ref_details") {
		t.Fatalf("submission=%+v", submission)
	}
	if domainWorkFieldHasNameV0(submission.PayloadFields, "source_refs") {
		t.Fatalf("source_refs debe llegar compacto: %+v", submission.PayloadFields)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0NormalizaVisualAssetsPromptOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "generate_visual_asset", "visual_asset.json")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	prompt := "Create a clean instructional horizontal flow diagram in Spanish."
	payload := `{
		"artifact_type":"visual_asset",
		"payload_json":{
			"expected_artifact_type":"visual_asset",
			"topic_id":"topic-ref-001",
			"visual_assets":[{
				"asset_ref":"visual-ref-001",
				"asset_type":"diagram",
				"placement_ref":"topic-ref-001::visual-ref-001::official_order-1",
				"title":"Diagrama de referencia visual solicitada",
				"didactic_objective":"Mostrar en un unico esquema la relacion entre curso, programa, tema y visual solicitado.",
				"alt_text":"Diagrama de flujo que conecta curso, programa, tema y visual.",
				"diagram_spec":{"recommended_format":"svg","layout":"left_to_right_flow"},
				"render_prompt":` + strconv.Quote(prompt) + `
			}]
		}
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-visual-prompt-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-visual-prompt-001", Title: "Generar recurso visual OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-visual-prompt-001",
						ChangeRef:     "change-ref-visual-prompt-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-visual-prompt-001",
							WorkKind:   "generate_visual_asset",
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-visual-prompt-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/generate_visual_asset/visual_asset.json"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-visual-prompt-001",
					AgentRef:     "agent-ref-visual-prompt-001",
					Summary:      "Visual validado.",
					EvidenceRefs: []string{"ack-ref-visual-prompt-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "visual_asset" ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "content_type", "text/markdown") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "topic_id", "topic-ref-001") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "asset_type", "diagram") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "format", "markdown") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "caption", "Mostrar en un unico esquema la relacion entre curso, programa, tema y visual solicitado.") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "alt_text", "Diagrama de flujo que conecta curso, programa, tema y visual.") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "body", prompt) ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "prompt", prompt) ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "visual_assets") {
		t.Fatalf("submission=%+v", submission)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0NormalizaVisualAssetsPromptObjetoOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "generate_visual_asset", "visual_asset.json")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	instruction := "Genera un diagrama didactico para el tema usando solo el paquete de dominio recibido."
	payload := `{
		"artifact_type":"visual_asset",
		"payload_json":{
			"course_id":"course-ref-001",
			"program_id":"program-ref-001",
			"topic_id":"topic-ref-001",
			"expected_artifact_type":"visual_asset",
			"visual_assets":[{
				"asset_ref":"visual-ref-001",
				"asset_type":"diagram",
				"placement_ref":"topic-ref-001:official_order:1:visual-ref-001",
				"didactic_objective":"Sintetizar visualmente un punto importante del tema completo.",
				"alt_text":"Diagrama didactico del tema vinculado a topic-ref-001.",
				"prompt":{
					"language":"es",
					"instruction":` + strconv.Quote(instruction) + `,
					"composition":{
						"format":"diagram",
						"style":"administrativo claro, alto contraste"
					}
				}
			}]
		}
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-visual-prompt-object-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-visual-prompt-object-001", Title: "Generar recurso visual OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-visual-prompt-object-001",
						ChangeRef:     "change-ref-visual-prompt-object-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-visual-prompt-object-001",
							WorkKind:   "generate_visual_asset",
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-visual-prompt-object-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/generate_visual_asset/visual_asset.json"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-visual-prompt-object-001",
					AgentRef:     "agent-ref-visual-prompt-object-001",
					Summary:      "Visual validado.",
					EvidenceRefs: []string{"ack-ref-visual-prompt-object-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "visual_asset" ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "content_type", "text/markdown") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "topic_id", "topic-ref-001") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "asset_type", "diagram") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "format", "markdown") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "caption", "Sintetizar visualmente un punto importante del tema completo.") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "alt_text", "Diagrama didactico del tema vinculado a topic-ref-001.") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "body", instruction) ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "prompt", instruction) ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "image_prompt", instruction) ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "style", "administrativo claro, alto contraste") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "language_code", "es") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "visual_assets") {
		t.Fatalf("submission=%+v", submission)
	}
	if domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "prompt") {
		t.Fatalf("prompt debe llegar plano para OPES: %+v", submission.PayloadFields)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDomainWorkDeliveryContentTypeV0VisualHTML(t *testing.T) {
	got := domainWorkDeliveryContentTypeV0(
		[]orquestadomainwork.DomainWorkFieldV0{{Name: "format", Value: "html"}},
		"visual_asset",
	)
	if got != "text/html" {
		t.Fatalf("content_type=%q", got)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0DerivaManifestCierreOPESFinal(t *testing.T) {
	projectDir := t.TempDir()
	fileRef := "external/opes/finalize_temario_package/completed_syllabus_package.json"
	bodyPath := filepath.Join(projectDir, filepath.FromSlash(fileRef))
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"artifact_type":"completed_syllabus_package",
		"payload_json":{
			"package_ref":"package-ref-final-001",
			"manifest_cierre":{
				"schema_version":"opes_final_package_evidence_manifest.v0",
				"package_ref":"package-ref-final-001",
				"manifest_ref":"manifest-cierre-ref-001",
				"checksum_refs":["checksum-ref-001"],
				"validation_report_ref":"validation-report-ref-001",
				"review_matrix_ref":"review-matrix-ref-001",
				"required_evidence_refs":{
					"html":"opes-final-evidence:html:001",
					"rag":"opes-final-evidence:rag:001",
					"audio":"opes-final-evidence:audio:001",
					"tests":"opes-final-evidence:tests:001",
					"visual":"opes-final-evidence:visual:001",
					"qa":"opes-final-evidence:qa:001"
				}
			}
		}
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run: orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-final-package-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{
					TaskID: "task-ref-final-package-001",
					Title:  "Cerrar paquete final OPES",
				},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-final-package-001",
						ChangeRef:     "change-ref-final-package-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-final-package-001",
							WorkKind:   "finalize_temario_package",
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-final-package-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{fileRef},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-final-package-001",
					AgentRef:     "agent-ref-final-package-001",
					Summary:      "Paquete final OPES validado.",
					EvidenceRefs: []string{"ack-ref-final-package-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0 ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "package_ref", "package-ref-final-001") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "manifest_cierre") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "validation_report_ref", "validation-report-ref-001") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "review_matrix_ref", "review-matrix-ref-001") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "required_evidence_refs") {
		t.Fatalf("submission=%+v", submission)
	}
	for _, want := range []string{
		"manifest-cierre-ref-001",
		"checksum-ref-001",
		"validation-report-ref-001",
		"review-matrix-ref-001",
		"opes-final-evidence:html:001",
		"opes-final-evidence:rag:001",
		"opes-final-evidence:audio:001",
		"opes-final-evidence:tests:001",
		"opes-final-evidence:visual:001",
		"opes-final-evidence:qa:001",
	} {
		if !codexStackStringInSetV0(submission.EvidenceRefs, want) {
			t.Fatalf("evidence_refs=%+v falta %s", submission.EvidenceRefs, want)
		}
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0NoCompletaOPESFinalSinManifest(t *testing.T) {
	projectDir := t.TempDir()
	fileRef := "external/opes/finalize_temario_package/completed_syllabus_package.json"
	bodyPath := filepath.Join(projectDir, filepath.FromSlash(fileRef))
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"artifact_type":"completed_syllabus_package",
		"payload_json":{
			"package_ref":"package-ref-final-incomplete-001"
		}
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run: orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-final-package-incomplete-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{
					TaskID: "task-ref-final-package-incomplete-001",
					Title:  "Cerrar paquete final OPES",
				},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-final-package-incomplete-001",
						ChangeRef:     "change-ref-final-package-incomplete-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-final-package-incomplete-001",
							WorkKind:   "finalize_temario_package",
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-final-package-incomplete-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{fileRef},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-final-package-incomplete-001",
					AgentRef:     "agent-ref-final-package-incomplete-001",
					Summary:      "Paquete final OPES incompleto.",
					EvidenceRefs: []string{"ack-ref-final-package-incomplete-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.CompleteJob ||
		!domainWorkFieldValuesForTestV0(submission.PayloadFields, "validation_issue_refs", []string{codexStackOPESFinalPackageEvidenceIncompleteIssueV0}) ||
		!codexStackStringInSetV0(submission.EvidenceRefs, codexStackOPESFinalPackageEvidenceIncompleteIssueV0) {
		t.Fatalf("submission final incompleta debe quedar no terminal: %+v", submission)
	}
}
