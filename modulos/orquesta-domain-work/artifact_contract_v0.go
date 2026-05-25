package orquestadomainwork

import "strings"

const (
	DomainWorkArtifactTypeContentBlockV0          = "content_block"
	DomainWorkArtifactTypeVisualAssetV0           = "visual_asset"
	DomainWorkArtifactTypeBlockRevisionV0         = "block_revision"
	DomainWorkArtifactTypeSourceV0                = "source"
	DomainWorkArtifactTypeTopicStructureV0        = "topic_structure"
	DomainWorkArtifactTypeTopicOutlineV0          = "topic_outline"
	DomainWorkArtifactTypeTopicSummaryV0          = "topic_summary"
	DomainWorkArtifactTypeTopicExpansionPackageV0 = "topic_expansion_package"
	DomainWorkArtifactTypeAssembledTopicV0        = "assembled_topic"
	DomainWorkArtifactTypeGenericWorkDeliveryV0   = "work_delivery"
)

func ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind string) string {
	switch strings.TrimSpace(workKind) {
	case "draft_content_block", "generate_block", "generate_program_topic_draft":
		return DomainWorkArtifactTypeContentBlockV0
	case "generate_visual_asset":
		return DomainWorkArtifactTypeVisualAssetV0
	case "review_legal", "review_pedagogical", "review_quality", "validate_topic":
		return DomainWorkArtifactTypeBlockRevisionV0
	case "research_sources", "download_source", "verify_sources":
		return DomainWorkArtifactTypeSourceV0
	case "split_syllabus_topic":
		return DomainWorkArtifactTypeTopicStructureV0
	case "draft_topic_outline", "create_exam_outline":
		return DomainWorkArtifactTypeTopicOutlineV0
	case "summarize_block", "summarize_chapter", "summarize_topic":
		return DomainWorkArtifactTypeTopicSummaryV0
	case "expand_topic_from_summary":
		return DomainWorkArtifactTypeTopicExpansionPackageV0
	case "plan_documento", "plan_tema", "plan_temario":
		return DomainDocumentPlanArtifactTypeV0
	case "assemble_topic":
		return DomainWorkArtifactTypeAssembledTopicV0
	default:
		return DomainWorkArtifactTypeGenericWorkDeliveryV0
	}
}
