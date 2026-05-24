package orquestadomainwork

import (
	"context"
	"testing"
)

func TestDeclaredDomainWorkRequiredTestPolicyV0UsaRefsDeclaradas(t *testing.T) {
	policy := DeclaredDomainWorkRequiredTestPolicyV0{}
	request := validDomainWorkJobRequestForTestV0()
	request.WorkRefs = []string{"job-ref-domain-001", "work-ref-domain-001"}
	request.AcceptanceCriteria = []string{"criterio del dominio propietario"}
	request.RequiredTests = []DomainWorkRequiredTestV0{{
		TestRef:                "domain-test-ref-contract",
		AcceptanceCriteriaRefs: []string{"criteria-ref-contract"},
		InputRefs:              []string{"input-ref-domain-001"},
		ExternalRefs: []DomainWorkExternalRefV0{{
			Kind: "domain_validator",
			Ref:  "validator-ref-contract",
		}},
		EvidenceRefs: []string{"artifact-ref-domain-001"},
	}}

	plan, err := policy.BuildDomainWorkRequiredTestPlanV0(context.Background(), request)

	if err != nil {
		t.Fatalf("BuildDomainWorkRequiredTestPlanV0: %v", err)
	}
	if plan.JobRef != "job-ref-domain-001" ||
		len(plan.Issues) != 0 ||
		len(plan.RequiredTests) != 1 ||
		plan.RequiredTests[0].TestRef != "domain-test-ref-contract" ||
		plan.RequiredTests[0].ExternalRefs[0].Ref != "validator-ref-contract" {
		t.Fatalf("plan=%+v", plan)
	}
	if issues := ValidateDomainWorkRequiredTestPlanV0(plan); len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestDeclaredDomainWorkRequiredTestPolicyV0RechazaEvidenciaInterna(t *testing.T) {
	policy := DeclaredDomainWorkRequiredTestPolicyV0{}
	request := validDomainWorkJobRequestForTestV0()
	request.RequiredTests = []DomainWorkRequiredTestV0{{
		TestRef:      "domain-test-ref-contract",
		EvidenceRefs: []string{"postgres://internal/jobs/1"},
	}}

	plan, err := policy.BuildDomainWorkRequiredTestPlanV0(context.Background(), request)

	if err != nil {
		t.Fatalf("BuildDomainWorkRequiredTestPlanV0: %v", err)
	}
	issues := ValidateDomainWorkRequiredTestPlanV0(plan)
	if len(issues) == 0 ||
		issues[0].Code != ErrDomainWorkRefInvalidV0 ||
		issues[0].Field != "required_tests.evidence_refs" {
		t.Fatalf("issues=%+v", issues)
	}
}
