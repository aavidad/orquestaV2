package orquestaopesdirector

import (
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestValidateOPESArtifactQualityContractV0CubreWorkKindsMinimosV0(t *testing.T) {
	cases := []struct {
		name          string
		workKind      string
		artifactType  string
		goodFields    []orquestadomainwork.DomainWorkFieldV0
		finalEvidence string
		wantIssue     string
	}{
		{
			name:         "visual",
			workKind:     "generate_visual_asset",
			artifactType: orquestadomainwork.DomainWorkArtifactTypeVisualAssetV0,
			goodFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "didactic_function", Value: "comparar pasos del procedimiento"},
				{Name: "placement_ref", Value: "tema-001-apartado-2"},
				{Name: "alt_text", Value: "Diagrama del procedimiento"},
			},
			finalEvidence: "opes-final-evidence:visual_didactic_publicable",
			wantIssue:     ErrOPESArtifactQualityVisualDidacticFunctionRequiredV0,
		},
		{
			name:         "audio",
			workKind:     "generate_audio_asset",
			artifactType: orquestadomainwork.DomainWorkArtifactTypeAudioAssetV0,
			goodFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "audio_manifest_ref", Value: "audio/manifest.json"},
				{Name: "generated_mp3_refs", Values: []string{"audio/tema-001.mp3"}},
			},
			finalEvidence: "opes-final-evidence:audio_tts_resumable",
			wantIssue:     ErrOPESArtifactQualityAudioManifestRequiredV0,
		},
		{
			name:         "html",
			workKind:     "generate_html_site",
			artifactType: orquestadomainwork.DomainWorkArtifactTypeLocalHTMLSiteV0,
			goodFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "html_topic_pages_manifest", Value: "html/topic_pages_manifest.json"},
				{Name: "html_validation_report", Value: "validacion/html_links.json"},
			},
			finalEvidence: "opes-final-evidence:html_site_publicable",
			wantIssue:     ErrOPESArtifactQualityHTMLTopicPagesRequiredV0,
		},
		{
			name:         "tutor",
			workKind:     "generate_tutor_assets",
			artifactType: orquestadomainwork.DomainWorkArtifactTypeTutorBotPackageV0,
			goodFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "rag_corpus_manifest", Value: "rag/manifest.json"},
				{Name: "tutor_qa_report", Value: "validacion/tutor_qa.json"},
			},
			finalEvidence: "opes-final-evidence:tutor_assets_publicable",
			wantIssue:     ErrOPESArtifactQualityTutorManifestRequiredV0,
		},
		{
			name:         "sources",
			workKind:     "research_exam_precedents",
			artifactType: orquestadomainwork.DomainWorkArtifactTypeExamResearchReportV0,
			goodFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "source_refs", Values: []string{"source-ref-boe-001"}},
				{Name: "source_research_report", Value: "fuentes/reporte.json"},
			},
			finalEvidence: "opes-final-evidence:source_research_traceable",
			wantIssue:     ErrOPESArtifactQualitySourceRefsRequiredV0,
		},
		{
			name:         "review",
			workKind:     "review_quality",
			artifactType: orquestadomainwork.DomainWorkArtifactTypeBlockRevisionV0,
			goodFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "issue_refs", Values: []string{"issue-ref-001"}},
				{Name: "review_evidence_refs", Values: []string{"artifact-ref-001"}},
			},
			finalEvidence: "opes-final-evidence:review_report_actionable",
			wantIssue:     ErrOPESArtifactQualityReviewFindingsRequiredV0,
		},
		{
			name:         "cases",
			workKind:     "generate_practical_cases",
			artifactType: orquestadomainwork.DomainWorkArtifactTypePracticalCasesV0,
			goodFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "practical_cases_schema_report", Value: "validacion/schema.json"},
				{Name: "practical_cases_coverage_report", Value: "validacion/coverage.json"},
			},
			finalEvidence: "opes-final-evidence:practical_cases_publicable",
			wantIssue:     ErrOPESArtifactQualityCasesSchemaRequiredV0,
		},
		{
			name:         "games",
			workKind:     "generate_learning_games",
			artifactType: orquestadomainwork.DomainWorkArtifactTypeInteractivePracticeV0,
			goodFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "interactive_practice_manifest", Value: "juegos/manifest.json"},
				{Name: "interactive_practice_qa_report", Value: "validacion/juegos.json"},
			},
			finalEvidence: "opes-final-evidence:interactive_practice_publicable",
			wantIssue:     ErrOPESArtifactQualityInteractiveManifestRequiredV0,
		},
		{
			name:         "help",
			workKind:     "generate_help_manual_assets",
			artifactType: orquestadomainwork.DomainWorkArtifactTypeHelpPackageV0,
			goodFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "help_manual_manifest", Value: "ayuda/manifest.json"},
				{Name: "help_manual_qa_report", Value: "validacion/ayuda.json"},
			},
			finalEvidence: "opes-final-evidence:help_manual_publicable",
			wantIssue:     ErrOPESArtifactQualityHelpManifestRequiredV0,
		},
		{
			name:         "visual_reuse",
			workKind:     "visual_asset_reuse",
			artifactType: "visual_reuse_manifest",
			goodFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "visual_reuse_manifest", Value: "visuales/reuse_manifest.json"},
				{Name: "inserted_visual_count", Value: "4"},
			},
			finalEvidence: "opes-final-evidence:visual_reuse",
			wantIssue:     ErrOPESArtifactQualityVisualReuseManifestRequiredV0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name+"_bad", func(t *testing.T) {
			result := ValidateOPESArtifactQualityContractV0(OPESArtifactQualityContractRequestV0{
				WorkKind:     tc.workKind,
				ArtifactType: tc.artifactType,
				EvidenceRefs: []string{tc.finalEvidence},
			})
			if result.Status != OPESArtifactQualityStatusNeedsReworkV0 ||
				!artifactQualityIssueCodeForTestV0(result.Issues, tc.wantIssue) {
				t.Fatalf("result=%+v want_issue=%s", result, tc.wantIssue)
			}
		})
		t.Run(tc.name+"_good", func(t *testing.T) {
			result := ValidateOPESArtifactQualityContractV0(OPESArtifactQualityContractRequestV0{
				WorkKind:     tc.workKind,
				ArtifactType: tc.artifactType,
				Fields:       tc.goodFields,
				EvidenceRefs: []string{tc.finalEvidence},
			})
			if result.Status != OPESArtifactQualityStatusCompleteV0 || len(result.Issues) != 0 {
				t.Fatalf("result=%+v", result)
			}
		})
	}
}

func TestValidateOPESArtifactQualityContractV0NormalizaArtefactosOPESExtendidosV0(t *testing.T) {
	cases := []struct {
		name         string
		artifactType string
		goodFields   []orquestadomainwork.DomainWorkFieldV0
		wantIssue    string
	}{
		{
			name:         "learning_games_package",
			artifactType: opesDirectorArtifactTypeLearningGamesPackageV0,
			goodFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "learning_games_manifest", Value: "juegos/manifest.json"},
				{Name: "learning_games_qa_report", Value: "validacion/juegos.json"},
			},
			wantIssue: ErrOPESArtifactQualityInteractiveManifestRequiredV0,
		},
		{
			name:         "help_manual_package",
			artifactType: opesDirectorArtifactTypeHelpManualPackageV0,
			goodFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "help_manual_manifest", Value: "ayuda/manifest.json"},
				{Name: "help_manual_qa_report", Value: "validacion/ayuda.json"},
			},
			wantIssue: ErrOPESArtifactQualityHelpManifestRequiredV0,
		},
		{
			name:         "quality_audit_report",
			artifactType: opesDirectorArtifactTypeQualityAuditReportV0,
			goodFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "decision_global", Value: "rework_menor"},
				{Name: "rework_task_requests", ValueJSON: []byte(`[{"topic_ref":"tema-001"}]`)},
			},
			wantIssue: ErrOPESArtifactQualityAuditDecisionRequiredV0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name+"_bad", func(t *testing.T) {
			result := ValidateOPESArtifactQualityContractV0(OPESArtifactQualityContractRequestV0{
				ArtifactType: tc.artifactType,
			})
			if result.Status != OPESArtifactQualityStatusNeedsReworkV0 ||
				!artifactQualityIssueCodeForTestV0(result.Issues, tc.wantIssue) {
				t.Fatalf("result=%+v want_issue=%s", result, tc.wantIssue)
			}
		})
		t.Run(tc.name+"_good", func(t *testing.T) {
			result := ValidateOPESArtifactQualityContractV0(OPESArtifactQualityContractRequestV0{
				ArtifactType: tc.artifactType,
				Fields:       tc.goodFields,
			})
			if result.Status != OPESArtifactQualityStatusCompleteV0 || len(result.Issues) != 0 {
				t.Fatalf("result=%+v", result)
			}
		})
	}
}

func artifactQualityIssueCodeForTestV0(issues []OPESArtifactQualityIssueV0, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
