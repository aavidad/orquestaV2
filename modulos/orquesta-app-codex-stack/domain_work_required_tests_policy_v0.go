package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func (source compositeDirectorDecisionSourceV0) normalizeDomainWorkRequiredTestsV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if source.DomainRequiredPolicy == nil || source.AppChangeStore == nil {
		return decisions, nil
	}
	records, err := source.AppChangeStore.ListAppChangeRecordsV0(
		ctx,
		orquestaappchange.AppChangeRecordFilterV0{RunRef: run.RunID},
	)
	if err != nil {
		return nil, err
	}
	plans, err := domainWorkRequiredTestPlansByTaskV0(ctx, source.DomainRequiredPolicy, records)
	if err != nil {
		return nil, err
	}
	out := append([]orquestadirectoragent.DirectorAgentDecisionV0(nil), decisions...)
	for i := range out {
		if out[i].CreateMicrotask == nil {
			continue
		}
		task := &out[i].CreateMicrotask.Task
		plan, ok := plans[strings.TrimSpace(task.TaskID)]
		if !ok || len(plan.RequiredTests) == 0 || !codexStackDirectorTaskIsDomainWorkV0(*task) {
			continue
		}
		task.RequiredTests = domainWorkRequiredTestRefsFromPlanV0(plan)
		task.AcceptanceCriteria = compactCodexStackStringsV0(
			append(task.AcceptanceCriteria, plan.AcceptanceCriteria...),
		)
	}
	return out, nil
}

func domainWorkRequiredTestPlansByTaskV0(
	ctx context.Context,
	policy orquestadomainwork.DomainWorkRequiredTestPolicyPortV0,
	records []orquestaappchange.AppChangeRecordV0,
) (map[string]orquestadomainwork.DomainWorkRequiredTestPlanV0, error) {
	out := map[string]orquestadomainwork.DomainWorkRequiredTestPlanV0{}
	for _, record := range records {
		job, ok := orquestaappchange.DomainWorkJobRequestFromAppChangeV0(record.Request)
		if !ok {
			continue
		}
		plan, err := policy.BuildDomainWorkRequiredTestPlanV0(ctx, job)
		if err != nil {
			return nil, err
		}
		plan = orquestadomainwork.NormalizeDomainWorkRequiredTestPlanV0(plan)
		if len(plan.Issues) > 0 {
			return nil, fmt.Errorf("domain_work_required_test_policy_issues")
		}
		if issues := orquestadomainwork.ValidateDomainWorkRequiredTestPlanV0(plan); len(issues) > 0 {
			return nil, fmt.Errorf("domain_work_required_test_policy_invalid")
		}
		out[orquestaappchangedirectorsource.AppChangeTaskRefV0(record.Request.ChangeRef)] = plan
	}
	return out, nil
}

func codexStackDirectorTaskIsDomainWorkV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
) bool {
	for _, ref := range task.FunctionContractRefs {
		if strings.TrimSpace(ref.FunctionName) == "ApplyExternalDomainWorkV0" {
			return true
		}
	}
	return false
}

func domainWorkRequiredTestRefsFromPlanV0(
	plan orquestadomainwork.DomainWorkRequiredTestPlanV0,
) []string {
	refs := make([]string, 0, len(plan.RequiredTests))
	for _, test := range plan.RequiredTests {
		refs = append(refs, test.TestRef)
	}
	return compactCodexStackStringsV0(refs)
}

func requiredTestRunnerV0(
	config ConfigV0,
) orquestacionnucleoapp.RequiredTestRunnerPortV0 {
	inner := codexAckRequiredTestRunnerFromConfigV0(config, config.RequiredTests)
	if config.DomainTests.Policy == nil {
		return inner
	}
	return DomainWorkRequiredTestRunnerV0{
		Inner:            inner,
		Policy:           config.DomainTests.Policy,
		AppChangeStore:   config.Stores.AppChangeStore,
		SubmissionLedger: domainWorkSubmissionRecordReaderV0(config.DomainDelivery.Ledger),
		EvidenceReader:   requiredTestEvidenceStoreV0(config),
		EvidenceWriter:   requiredTestEvidenceStoreV0(config),
	}
}

func domainWorkSubmissionRecordReaderV0(
	ledger DomainWorkArtifactSubmissionLedgerPortV0,
) DomainWorkArtifactSubmissionRecordReaderPortV0 {
	reader, _ := ledger.(DomainWorkArtifactSubmissionRecordReaderPortV0)
	return reader
}
