package orquestacionnucleoapp

import (
	"strings"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

func ValidateOperationalDirectorPlanStateV0(state OperationalDirectorPlanStateV0) error {
	if state.SchemaVersion != OperationalDirectorPlanStateSchemaVersionV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "schema_version", "schema_version invalida")
	}
	for field, value := range map[string]string{
		"state_ref":   state.StateRef,
		"plan_ref":    state.PlanRef,
		"run_ref":     state.RunRef,
		"observed_at": state.ObservedAt,
	} {
		if strings.TrimSpace(value) == "" {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, field, field+" requerido")
		}
	}
	if state.Mode != orquestadirectoroperativo.OperationalDirectorModeProgrammingV0 &&
		state.Mode != orquestadirectoroperativo.OperationalDirectorModeDomainWorkV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "mode", "mode invalido")
	}
	if state.Status != OperationalDirectorPlanStateActiveV0 &&
		state.Status != OperationalDirectorPlanStateBlockedV0 &&
		state.Status != OperationalDirectorPlanStateClosedV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "status", "status invalido")
	}
	if state.ReplanAttempts < 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "replan_attempts", "replan_attempts invalido")
	}
	if len(state.Steps) == 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps", "steps requerido")
	}
	steps := map[string]bool{}
	acceptedReviewRefs := map[string]bool{}
	var activeStep OperationalDirectorPlanStepStateV0
	activeStepFound := false
	for _, step := range state.Steps {
		if err := ValidateOperationalDirectorPlanStepStateV0(step); err != nil {
			return err
		}
		if steps[step.StepID] {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.step_id", "step_id duplicado")
		}
		steps[step.StepID] = true
		if state.ActiveStepID != "" && step.StepID == state.ActiveStepID {
			activeStep = step
			activeStepFound = true
		}
		for _, acceptedReviewRef := range step.AcceptedReviewRefs {
			if acceptedReviewRefs[acceptedReviewRef] {
				return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.accepted_review_refs", "accepted_review_ref duplicado")
			}
			acceptedReviewRefs[acceptedReviewRef] = true
		}
	}
	if state.ActiveStepID != "" && !activeStepFound {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "active_step_id", "active_step_id no existe")
	}
	if activeStepFound {
		if err := validateOperationalDirectorPlanStateActiveStepV0(state, activeStep); err != nil {
			return err
		}
	}
	return nil
}

func validateOperationalDirectorPlanStateActiveStepV0(
	state OperationalDirectorPlanStateV0,
	step OperationalDirectorPlanStepStateV0,
) error {
	if state.ActiveWaveRef != "" && step.WaveRef != "" && state.ActiveWaveRef != step.WaveRef {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "active_wave_ref", "active_wave_ref inconsistente")
	}
	if state.ActiveCohortRef != "" && step.CohortRef != "" && state.ActiveCohortRef != step.CohortRef {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "active_cohort_ref", "active_cohort_ref inconsistente")
	}
	if state.ActiveParentTaskRef != "" && step.ParentTaskRef != "" && state.ActiveParentTaskRef != step.ParentTaskRef {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "active_parent_task_ref", "active_parent_task_ref inconsistente")
	}
	if step.Kind == orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 &&
		step.Status == orquestadirectoroperativo.OperationalDirectorStepRunningV0 &&
		len(step.WaitRefs) != 1 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.wait_refs", "wait_subagents activo requiere wait_ref unico")
	}
	return nil
}

func ValidateOperationalDirectorPlanStepStateV0(step OperationalDirectorPlanStepStateV0) error {
	if strings.TrimSpace(step.StepID) == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.step_id", "step_id requerido")
	}
	if !operationalDirectorPlanStepKindValidV0(step.Kind) {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.kind", "kind invalido")
	}
	if !operationalDirectorPlanStepStatusValidV0(step.Status) {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.status", "status invalido")
	}
	if step.Attempts < 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.attempts", "attempts invalido")
	}
	if len(step.AcceptedReviewRefs) > 0 &&
		step.Kind != orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.accepted_review_refs", "accepted_review_refs solo aplica a review_deliveries")
	}
	if step.Kind == orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0 &&
		(step.Status == orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
			step.Status == orquestadirectoroperativo.OperationalDirectorStepClosedV0) &&
		(len(step.DeliveryRefs) == 0 || len(step.ReviewResultRefs) == 0 || len(step.AcceptedReviewRefs) == 0) {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.accepted_review_refs", "review_deliveries aceptado requiere refs causales")
	}
	return nil
}

func operationalDirectorPlanStepKindValidV0(kind orquestadirectoroperativo.OperationalDirectorStepKindV0) bool {
	switch kind {
	case orquestadirectoroperativo.OperationalDirectorStepGatherContextV0,
		orquestadirectoroperativo.OperationalDirectorStepRequestDomainContextV0,
		orquestadirectoroperativo.OperationalDirectorStepSplitWorkV0,
		orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
		orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
		orquestadirectoroperativo.OperationalDirectorStepGovernDelegationV0,
		orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
		orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
		orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0:
		return true
	default:
		return false
	}
}

func operationalDirectorPlanStepStatusValidV0(status orquestadirectoroperativo.OperationalDirectorStepStatusV0) bool {
	switch status {
	case orquestadirectoroperativo.OperationalDirectorStepPendingV0,
		orquestadirectoroperativo.OperationalDirectorStepRunningV0,
		orquestadirectoroperativo.OperationalDirectorStepBlockedV0,
		orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0,
		orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
		orquestadirectoroperativo.OperationalDirectorStepClosedV0:
		return true
	default:
		return false
	}
}
