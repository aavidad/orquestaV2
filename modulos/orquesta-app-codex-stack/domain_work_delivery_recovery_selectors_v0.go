package orquestaappcodexstack

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func domainWorkTaskForDescriptorV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (orquestacoreworkflow.WorkflowTaskV0, bool) {
	taskRef := strings.TrimSpace(descriptor.Spec.AgentPacket.Task.TaskRef)
	for _, task := range tasks {
		if strings.TrimSpace(task.TaskID) == taskRef && workflowTaskHasDomainWorkContractV0(task) {
			return task, true
		}
	}
	return orquestacoreworkflow.WorkflowTaskV0{}, false
}

func domainWorkRecoveryCoreDeliveryEligibleV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	agentRef := strings.TrimSpace(descriptor.AgentRef)
	return !stringInDrainSetV0(run.StoppedAgents, agentRef) &&
		!stringInDrainSetV0(run.ConfirmedStoppedAgents, agentRef) &&
		!stringInDrainSetV0(run.FailedAgents, agentRef)
}

func domainWorkRecoveryDirectSubmitEligibleV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	agentRef := strings.TrimSpace(descriptor.AgentRef)
	ackRef := strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef)
	if agentRef == "" ||
		ackRef == "" ||
		stringInDrainSetV0(run.Deliveries, ackRef) ||
		stringInDrainSetV0(run.FailedAgents, agentRef) ||
		domainWorkRecoveryAgentIsAssessmentV0(agentRef) {
		return false
	}
	return stringInDrainSetV0(run.StoppedAgents, agentRef) ||
		stringInDrainSetV0(run.ConfirmedStoppedAgents, agentRef) ||
		stringInDrainSetV0(run.LostAgents, agentRef)
}

func domainWorkRecoveryAgentIsAssessmentV0(agentRef string) bool {
	return strings.Contains(
		strings.ToLower(strings.TrimSpace(agentRef)),
		"agent-ref-assessment-",
	)
}
