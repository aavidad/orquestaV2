package orquestadomainwork

import "testing"

func TestExpectedDomainWorkArtifactTypeForWorkKindV0(t *testing.T) {
	cases := map[string]string{
		"draft_content_block":             DomainWorkArtifactTypeContentBlockV0,
		"generate_block":                  DomainWorkArtifactTypeContentBlockV0,
		"generate_program_topic_draft":    DomainWorkArtifactTypeContentBlockV0,
		"generate_visual_asset":           DomainWorkArtifactTypeVisualAssetV0,
		"review_legal":                    DomainWorkArtifactTypeBlockRevisionV0,
		"review_pedagogical":              DomainWorkArtifactTypeBlockRevisionV0,
		"review_quality":                  DomainWorkArtifactTypeBlockRevisionV0,
		"validate_topic":                  DomainWorkArtifactTypeBlockRevisionV0,
		"research_sources":                DomainWorkArtifactTypeSourceV0,
		"download_source":                 DomainWorkArtifactTypeSourceV0,
		"verify_sources":                  DomainWorkArtifactTypeSourceV0,
		"research_exam_precedents":        DomainWorkArtifactTypeExamResearchReportV0,
		"research_exam_results":           DomainWorkArtifactTypeExamResearchReportV0,
		"buscar_examenes_ope":             DomainWorkArtifactTypeExamResearchReportV0,
		"split_syllabus_topic":            DomainWorkArtifactTypeTopicStructureV0,
		"draft_topic_outline":             DomainWorkArtifactTypeTopicOutlineV0,
		"create_exam_outline":             DomainWorkArtifactTypeTopicOutlineV0,
		"summarize_block":                 DomainWorkArtifactTypeTopicSummaryV0,
		"summarize_chapter":               DomainWorkArtifactTypeTopicSummaryV0,
		"summarize_topic":                 DomainWorkArtifactTypeTopicSummaryV0,
		"expand_topic_from_summary":       DomainWorkArtifactTypeTopicExpansionPackageV0,
		"plan_documento":                  DomainDocumentPlanArtifactTypeV0,
		"plan_tema":                       DomainDocumentPlanArtifactTypeV0,
		"plan_temario":                    DomainDocumentPlanArtifactTypeV0,
		"assemble_topic":                  DomainWorkArtifactTypeAssembledTopicV0,
		"generate_question_bank":          DomainWorkArtifactTypeQuestionBankV0,
		"generate_topic_tests":            DomainWorkArtifactTypeQuestionBankV0,
		"generate_html_site":              DomainWorkArtifactTypeLocalHTMLSiteV0,
		"assemble_local_html_site":        DomainWorkArtifactTypeLocalHTMLSiteV0,
		"generate_audio_asset":            DomainWorkArtifactTypeAudioAssetV0,
		"generate_topic_audio":            DomainWorkArtifactTypeAudioAssetV0,
		"audio_tema":                      DomainWorkArtifactTypeAudioAssetV0,
		"generacion_audio":                DomainWorkArtifactTypeAudioAssetV0,
		"tts_topic":                       DomainWorkArtifactTypeAudioAssetV0,
		"generate_tutor_assets":           DomainWorkArtifactTypeTutorBotPackageV0,
		"configure_temario_bots":          DomainWorkArtifactTypeTutorBotPackageV0,
		"generate_rag_assets":             DomainWorkArtifactTypeTutorBotPackageV0,
		"generate_interactive_practice":   DomainWorkArtifactTypeInteractivePracticeV0,
		"create_interactive_practice":     DomainWorkArtifactTypeInteractivePracticeV0,
		"generate_practice_package":       DomainWorkArtifactTypeInteractivePracticeV0,
		"generate_learning_games":         DomainWorkArtifactTypeInteractivePracticeV0,
		"generate_help_package":           DomainWorkArtifactTypeHelpPackageV0,
		"create_help_package":             DomainWorkArtifactTypeHelpPackageV0,
		"generate_help_manual_assets":     DomainWorkArtifactTypeHelpPackageV0,
		"review_agent_independent":        DomainWorkArtifactTypeAgentReviewReportV0,
		"review_codex":                    DomainWorkArtifactTypeAgentReviewReportV0,
		"review_gemini":                   DomainWorkArtifactTypeAgentReviewReportV0,
		"review_claude":                   DomainWorkArtifactTypeAgentReviewReportV0,
		"review_agent_pair":               DomainWorkArtifactTypeAgentPairReviewReportV0,
		"review_pair_codex_gemini":        DomainWorkArtifactTypeAgentPairReviewReportV0,
		"review_pair_codex_claude":        DomainWorkArtifactTypeAgentPairReviewReportV0,
		"review_pair_gemini_claude":       DomainWorkArtifactTypeAgentPairReviewReportV0,
		"generate_agent_candidate":        DomainWorkArtifactTypeAgentCandidateV0,
		"vote_agent_candidates":           DomainWorkArtifactTypeAgentCandidateVoteV0,
		"select_agent_candidate_director": DomainWorkArtifactTypeAgentCandidateSelectV0,
		"review_director_consolidation":   DomainWorkArtifactTypeDirectorReviewMatrixV0,
		"update_topic_registry":           DomainWorkArtifactTypeTopicRegistryUpdateV0,
		"claim_topic_registry":            DomainWorkArtifactTypeTopicRegistryUpdateV0,
		"release_topic_registry":          DomainWorkArtifactTypeTopicRegistryUpdateV0,
		"finalize_domain_package":         DomainWorkArtifactTypeFinalDomainPackageV0,
		"finalize_topic_package":          DomainWorkArtifactTypeFinalDomainPackageV0,
		"finalize_temario_package":        DomainWorkArtifactTypeFinalDomainPackageV0,
		"unknown_work_kind":               DomainWorkArtifactTypeGenericWorkDeliveryV0,
	}
	for workKind, want := range cases {
		if got := ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind); got != want {
			t.Fatalf("%s artifact=%s want=%s", workKind, got, want)
		}
	}
}

func TestExpectedDomainWorkArtifactContractForWorkKindV0ClasificaCanonicoDerivadoYEvidencia(t *testing.T) {
	cases := []struct {
		name                  string
		workKind              string
		artifactType          string
		sourceKind            string
		canonicality          string
		stage                 string
		materializationTarget string
	}{
		{
			name:                  "html canonico",
			workKind:              "generate_html_site",
			artifactType:          DomainWorkArtifactTypeLocalHTMLSiteV0,
			sourceKind:            DomainWorkArtifactSourceKindCanonicalSourceV0,
			canonicality:          DomainWorkArtifactCanonicalityCanonicalV0,
			stage:                 DomainWorkArtifactStageSourceV0,
			materializationTarget: DomainWorkArtifactMaterializationTargetDomainV0,
		},
		{
			name:                  "tests canonicos",
			workKind:              "generate_question_bank",
			artifactType:          DomainWorkArtifactTypeQuestionBankV0,
			sourceKind:            DomainWorkArtifactSourceKindCanonicalSourceV0,
			canonicality:          DomainWorkArtifactCanonicalityCanonicalV0,
			stage:                 DomainWorkArtifactStageSourceV0,
			materializationTarget: DomainWorkArtifactMaterializationTargetDomainV0,
		},
		{
			name:                  "audio derivado regenerable",
			workKind:              "generate_audio_asset",
			artifactType:          DomainWorkArtifactTypeAudioAssetV0,
			sourceKind:            DomainWorkArtifactSourceKindDerivedRegenerableV0,
			canonicality:          DomainWorkArtifactCanonicalityDerivedRegenerableV0,
			stage:                 DomainWorkArtifactStageDerivedV0,
			materializationTarget: DomainWorkArtifactMaterializationTargetRegenerateV0,
		},
		{
			name:                  "tutor rag derivado regenerable",
			workKind:              "generate_rag_assets",
			artifactType:          DomainWorkArtifactTypeTutorBotPackageV0,
			sourceKind:            DomainWorkArtifactSourceKindDerivedRegenerableV0,
			canonicality:          DomainWorkArtifactCanonicalityDerivedRegenerableV0,
			stage:                 DomainWorkArtifactStageDerivedV0,
			materializationTarget: DomainWorkArtifactMaterializationTargetRegenerateV0,
		},
		{
			name:                  "qa evidencia",
			workKind:              "review_agent_independent",
			artifactType:          DomainWorkArtifactTypeAgentReviewReportV0,
			sourceKind:            DomainWorkArtifactSourceKindEvidenceOnlyV0,
			canonicality:          DomainWorkArtifactCanonicalityEvidenceOnlyV0,
			stage:                 DomainWorkArtifactStageReviewV0,
			materializationTarget: DomainWorkArtifactMaterializationTargetEvidenceV0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExpectedDomainWorkArtifactContractForWorkKindV0(tc.workKind)
			if got.ArtifactType != tc.artifactType ||
				got.SourceKind != tc.sourceKind ||
				got.Canonicality != tc.canonicality ||
				got.Stage != tc.stage ||
				got.MaterializationTarget != tc.materializationTarget {
				t.Fatalf("contract=%+v", got)
			}
		})
	}
}
