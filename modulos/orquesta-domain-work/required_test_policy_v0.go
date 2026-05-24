package orquestadomainwork

import "context"

type DeclaredDomainWorkRequiredTestPolicyV0 struct{}

var _ DomainWorkRequiredTestPolicyPortV0 = DeclaredDomainWorkRequiredTestPolicyV0{}

func (DeclaredDomainWorkRequiredTestPolicyV0) BuildDomainWorkRequiredTestPlanV0(
	_ context.Context,
	request DomainWorkJobRequestV0,
) (DomainWorkRequiredTestPlanV0, error) {
	request = NormalizeDomainWorkJobRequestV0(request)
	plan := NormalizeDomainWorkRequiredTestPlanV0(DomainWorkRequiredTestPlanV0{
		DomainRef:          request.DomainRef,
		WorkKind:           request.WorkKind,
		JobRef:             domainWorkRequiredTestPolicyJobRefV0(request),
		AcceptanceCriteria: append([]string(nil), request.AcceptanceCriteria...),
		RequiredTests:      append([]DomainWorkRequiredTestV0(nil), request.RequiredTests...),
		ExternalRefs:       append([]DomainWorkExternalRefV0(nil), request.ExternalRefs...),
		EvidenceRefs:       append([]string(nil), request.EvidenceRefs...),
	})
	if len(plan.RequiredTests) == 0 {
		plan.Issues = append(plan.Issues, DomainWorkIssueV0{
			Code:  ErrDomainWorkRequiredTestsMissingV0,
			Field: "required_tests",
		})
	}
	return plan, nil
}

func domainWorkRequiredTestPolicyJobRefV0(
	request DomainWorkJobRequestV0,
) string {
	for _, ref := range request.WorkRefs {
		if ref != "" {
			return ref
		}
	}
	return ""
}
