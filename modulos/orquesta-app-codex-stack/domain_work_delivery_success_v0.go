package orquestaappcodexstack

import (
	"context"
	"fmt"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func (stack StackV0) recordDomainWorkSuccessfulSubmissionV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
	result orquestamcp.MCPDomainWorkToolResultV0,
	occurredAt string,
) error {
	submitted := domainWorkSubmittedSubmissionRecordV0(run, task, observation, submission, result, occurredAt)
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(ctx, submitted); err != nil {
		return fmt.Errorf("domain_work_submit_recovery_required")
	}
	accepted := domainWorkAcceptedSubmissionRecordV0(run, task, observation, submission, result, occurredAt)
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(ctx, accepted); err != nil {
		return fmt.Errorf("domain_work_submit_recovery_required")
	}
	return nil
}
