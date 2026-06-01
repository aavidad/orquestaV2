package orquestadirectoroperativo

import "fmt"

func addOperationalDirectorPlanStepDetailsV0(
	plan OperationalDirectorPlanV0,
) OperationalDirectorPlanV0 {
	launchStepCount := operationalDirectorStepKindCountV0(plan.Steps, OperationalDirectorStepLaunchSubagentsV0)
	launchStepIndex := 0
	for i := range plan.Steps {
		plan.Steps[i].WorkProfileKind = operationalDirectorWorkProfileKindV0(plan, plan.Steps[i].Kind)
		switch plan.Steps[i].Kind {
		case OperationalDirectorStepGatherContextV0,
			OperationalDirectorStepRequestDomainContextV0:
			plan.Steps[i].DomainRefs = append([]string(nil), plan.DomainRefs...)
			plan.Steps[i].EvidenceRefs = append([]string(nil), plan.MissingContext...)
		case OperationalDirectorStepSplitWorkV0,
			OperationalDirectorStepWaitSubagentsV0:
			plan.Steps[i].WriteSet = append([]string(nil), plan.WriteSet...)
			plan.Steps[i].DomainRefs = append([]string(nil), plan.DomainRefs...)
			plan.Steps[i].RequiredTests = append([]string(nil), plan.RequiredTests...)
			plan.Steps[i].AcceptanceCriteria = operationalDirectorAcceptanceCriteriaV0(plan)
		case OperationalDirectorStepLaunchSubagentsV0:
			launchStepIndex++
			plan.Steps[i].WriteSet = operationalDirectorShardedWriteSetV0(plan.WriteSet, launchStepIndex, launchStepCount)
			plan.Steps[i].DomainRefs = append([]string(nil), plan.DomainRefs...)
			plan.Steps[i].RequiredTests = append([]string(nil), plan.RequiredTests...)
			plan.Steps[i].AcceptanceCriteria = operationalDirectorAcceptanceCriteriaV0(plan)
			if launchStepCount > 1 {
				plan.Steps[i].AcceptanceCriteria = append(
					plan.Steps[i].AcceptanceCriteria,
					fmt.Sprintf("Coordinar como subagente %d de %d de la ola; el write-set asignado es ownership inicial, no limite duro.", launchStepIndex, launchStepCount),
				)
			}
		case OperationalDirectorStepGovernDelegationV0:
			plan.Steps[i].ParentStepID = "step-launch-subagents"
			plan.Steps[i].DelegationDepth = 1
			plan.Steps[i].MaxChildAgents = plan.MaxSubagentsPerAgent
			plan.Steps[i].ChildStepIDs = []string{"step-review-deliveries"}
			plan.Steps[i].DomainRefs = append([]string(nil), plan.DomainRefs...)
			plan.Steps[i].AcceptanceCriteria = []string{
				"Cada subagente hijo conserva parent_ref, objetivo, presupuesto y criterio de review.",
				"No lanzar hijos fuera de MaxDelegationDepth, MaxSubagentsPerAgent ni MaxRecursiveAgents.",
			}
		case OperationalDirectorStepReviewDeliveriesV0,
			OperationalDirectorStepRunRequiredTestsV0,
			OperationalDirectorStepReplanOrCloseV0:
			plan.Steps[i].RequiredTests = append([]string(nil), plan.RequiredTests...)
			plan.Steps[i].AcceptanceCriteria = operationalDirectorAcceptanceCriteriaV0(plan)
		}
	}
	return plan
}

func operationalDirectorStepKindCountV0(
	steps []OperationalDirectorStepV0,
	kind OperationalDirectorStepKindV0,
) int {
	count := 0
	for _, step := range steps {
		if step.Kind == kind {
			count++
		}
	}
	return count
}

func operationalDirectorShardedWriteSetV0(
	writeSet []string,
	index int,
	total int,
) []string {
	writeSet = compactStringsV0(writeSet)
	if total <= 1 || len(writeSet) <= 1 || index <= 0 {
		return append([]string(nil), writeSet...)
	}
	out := make([]string, 0, len(writeSet)/total+1)
	for pathIndex, path := range writeSet {
		if pathIndex%total == index-1 {
			out = append(out, path)
		}
	}
	if len(out) == 0 {
		return []string{writeSet[(index-1)%len(writeSet)]}
	}
	return out
}
