package orquestadomainwork

import "testing"

func TestExpectedDomainWorkArtifactTypeForWorkKindV0(t *testing.T) {
	cases := map[string]string{
		"draft_content_block":          DomainWorkArtifactTypeContentBlockV0,
		"generate_block":               DomainWorkArtifactTypeContentBlockV0,
		"generate_program_topic_draft": DomainWorkArtifactTypeContentBlockV0,
		"generate_visual_asset":        DomainWorkArtifactTypeVisualAssetV0,
		"review_legal":                 DomainWorkArtifactTypeBlockRevisionV0,
		"review_pedagogical":           DomainWorkArtifactTypeBlockRevisionV0,
		"review_quality":               DomainWorkArtifactTypeBlockRevisionV0,
		"validate_topic":               DomainWorkArtifactTypeBlockRevisionV0,
		"research_sources":             DomainWorkArtifactTypeSourceV0,
		"download_source":              DomainWorkArtifactTypeSourceV0,
		"verify_sources":               DomainWorkArtifactTypeSourceV0,
		"split_syllabus_topic":         DomainWorkArtifactTypeTopicStructureV0,
		"draft_topic_outline":          DomainWorkArtifactTypeTopicOutlineV0,
		"create_exam_outline":          DomainWorkArtifactTypeTopicOutlineV0,
		"summarize_block":              DomainWorkArtifactTypeTopicSummaryV0,
		"summarize_chapter":            DomainWorkArtifactTypeTopicSummaryV0,
		"summarize_topic":              DomainWorkArtifactTypeTopicSummaryV0,
		"expand_topic_from_summary":    DomainWorkArtifactTypeTopicExpansionPackageV0,
		"plan_documento":               DomainDocumentPlanArtifactTypeV0,
		"plan_tema":                    DomainDocumentPlanArtifactTypeV0,
		"plan_temario":                 DomainDocumentPlanArtifactTypeV0,
		"assemble_topic":               DomainWorkArtifactTypeAssembledTopicV0,
		"unknown_work_kind":            DomainWorkArtifactTypeGenericWorkDeliveryV0,
	}
	for workKind, want := range cases {
		if got := ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind); got != want {
			t.Fatalf("%s artifact=%s want=%s", workKind, got, want)
		}
	}
}
