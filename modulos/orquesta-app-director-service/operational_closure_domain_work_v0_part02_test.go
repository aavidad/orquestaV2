package orquestaappdirectorservice

import (
	"fmt"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func (source *serviceDomainWorkClosureSourceForTestV0) validateV0(
	request AppDirectorOperationalClosureRequestV0,
) error {
	if source.Task.WorkProfileKind != orquestacoreworkflow.WorkProfileDomainWorkV0 {
		return fmt.Errorf("task no es domain_work: %s", source.Task.WorkProfileKind)
	}
	if len(source.Task.RequiredTests) != 0 || len(request.RequiredTestEvidenceRefs) != 0 {
		return fmt.Errorf("domain_work fake no debe requerir required tests")
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(source.Artifact); len(issues) > 0 {
		return fmt.Errorf("artefacto domain_work invalido: %+v", issues)
	}
	for _, check := range []struct {
		values []string
		want   string
		field  string
	}{
		{values: request.Run.Tasks, want: source.Task.TaskID, field: "run.tasks"},
		{values: request.Run.Deliveries, want: source.DeliveryRef, field: "run.deliveries"},
		{values: request.Run.AcceptedReviews, want: source.AcceptedReviewRef, field: "run.accepted_reviews"},
		{values: source.Task.ContextRefs, want: source.Artifact.JobRef, field: "task.context_refs"},
	} {
		if !serviceStringInSetV0(check.values, check.want) {
			return fmt.Errorf("%s no contiene ref opaca %q", check.field, check.want)
		}
	}
	for _, ref := range []orquestadomainwork.DomainWorkExternalRefV0{
		{Kind: "run_ref", Ref: request.Run.RunID},
		{Kind: "task_ref", Ref: source.Task.TaskID},
		{Kind: "delivery_ref", Ref: source.DeliveryRef},
		{Kind: "accepted_review_ref", Ref: source.AcceptedReviewRef},
	} {
		if !serviceDomainWorkExternalRefInSetForTestV0(source.Artifact.ExternalRefs, ref) {
			return fmt.Errorf("artefacto sin external ref opaca %+v", ref)
		}
	}
	return nil
}

func serviceDomainWorkExternalRefInSetForTestV0(
	values []orquestadomainwork.DomainWorkExternalRefV0,
	want orquestadomainwork.DomainWorkExternalRefV0,
) bool {
	for _, value := range values {
		if value.Kind == want.Kind && value.Ref == want.Ref {
			return true
		}
	}
	return false
}
