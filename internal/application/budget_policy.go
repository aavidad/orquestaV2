package application

import (
	"errors"
	"sort"
	"strconv"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
)

type BudgetPolicy struct {
	DeploymentEnvelope      governance.BudgetEnvelope
	ProjectEnvelopeTemplate governance.BudgetEnvelope
	GoalEnvelopeTemplate    governance.BudgetEnvelope
	DefaultWorkItemDemand   governance.ResourceVector
	QuotaRetryDelay         time.Duration
	EffectApprovalTTL       time.Duration
	PolicyHash              string
}

type effectPolicySnapshot struct {
	PolicyHash      string
	PolicyRevision  uint64
	QuotaRetryDelay time.Duration
	ApprovalTTL     time.Duration
	DefaultDemand   governance.ResourceVector
	GoalLimit       governance.ResourceVector
}

func legacyGovernanceRecord(record GoalRecord) bool {
	return len(record.BudgetEnvelopes) == 0 && len(record.WorkItemAuthorities) == 0
}

func (policy BudgetPolicy) effectPolicy() effectPolicySnapshot {
	return effectPolicySnapshot{
		PolicyHash: policy.PolicyHash, PolicyRevision: policy.GoalEnvelopeTemplate.Revision,
		QuotaRetryDelay: policy.QuotaRetryDelay, ApprovalTTL: policy.EffectApprovalTTL,
		DefaultDemand: policy.DefaultWorkItemDemand, GoalLimit: policy.GoalEnvelopeTemplate.Limit,
	}
}

func historicalEffectPolicy(record GoalRecord) (effectPolicySnapshot, error) {
	if len(record.BudgetEnvelopes) != 3 || len(record.EffectIntents) == 0 {
		return effectPolicySnapshot{}, errors.New("application.goal_effect_policy_invalid")
	}
	source := record.EffectIntents[0]
	policy := effectPolicySnapshot{
		PolicyHash: source.PolicyHash, PolicyRevision: source.PolicyRevision,
		QuotaRetryDelay: source.QuotaRetryDelay, ApprovalTTL: source.ApprovalTTL,
	}
	seen := make(map[governance.BudgetScope]bool, 3)
	for _, envelope := range record.BudgetEnvelopes {
		if governance.ValidateBudgetEnvelope(envelope) != nil || seen[envelope.Scope] ||
			(envelope.Scope == governance.BudgetScopeProject && envelope.SubjectRef != record.Goal.Project().String()) ||
			(envelope.Scope == governance.BudgetScopeGoal && envelope.SubjectRef != record.Goal.Ref().String()) ||
			envelope.PolicyHash != policy.PolicyHash || envelope.Revision != policy.PolicyRevision {
			return effectPolicySnapshot{}, errors.New("application.goal_effect_policy_invalid")
		}
		seen[envelope.Scope] = true
		if envelope.Scope == governance.BudgetScopeGoal {
			policy.GoalLimit = envelope.Limit
		}
	}
	for _, intent := range record.EffectIntents {
		if ValidateEffectIntent(intent) != nil || intent.Subject.GoalRef != record.Goal.Ref() ||
			intent.PolicyHash != policy.PolicyHash || intent.PolicyRevision != policy.PolicyRevision ||
			intent.QuotaRetryDelay != policy.QuotaRetryDelay || intent.ApprovalTTL != policy.ApprovalTTL {
			return effectPolicySnapshot{}, errors.New("application.goal_effect_policy_invalid")
		}
	}
	if len(seen) != 3 {
		return effectPolicySnapshot{}, errors.New("application.goal_effect_policy_invalid")
	}
	return policy, nil
}

// retryFitsIrreversibleGoalBudget distinguishes durable Goal consumption from
// capacity that another active reservation may later release. Only the former
// can prove that an automatic replacement will never fit its original demand.
func retryFitsIrreversibleGoalBudget(
	record GoalRecord,
	settlement *governance.BudgetSettlement,
	demand governance.BudgetDemand,
) (bool, error) {
	if settlement == nil {
		return true, nil
	}
	if governance.ValidateBudgetSettlement(*settlement) != nil ||
		governance.ValidateBudgetDemand(demand) != nil {
		return false, errors.New("application.execution_retry_budget_invalid")
	}
	settlements, err := validatedGoalBudgetSettlements(record, settlement)
	if err != nil {
		return false, err
	}
	envelope, err := historicalGoalBudgetEnvelope(record)
	if err != nil {
		return false, err
	}
	disposition, _, err := ClassifyGoalRetryBudget(envelope, settlements, demand)
	return disposition == RetryBudgetFits, err
}

type RetryBudgetDisposition string

const (
	RetryBudgetFits                   RetryBudgetDisposition = "fits"
	RetryBudgetIrreversible           RetryBudgetDisposition = "irreversible"
	RetryBudgetExhaustionMarkerPrefix                        = "budget.retry_irreversible:"
)

// ClassifyGoalRetryBudget is the single policy projection shared by retry
// creation and late SQLite admission. Settled tokens, money, active time and
// disk are cumulative. Process slots are current capacity, so only the new
// demand contributes slots after settlements.
func ClassifyGoalRetryBudget(
	envelope governance.BudgetEnvelope,
	settlements []governance.BudgetSettlement,
	demand governance.BudgetDemand,
) (RetryBudgetDisposition, RetryBudgetExhaustion, error) {
	if governance.ValidateBudgetEnvelope(envelope) != nil || envelope.Scope != governance.BudgetScopeGoal ||
		governance.ValidateBudgetDemand(demand) != nil {
		return "", RetryBudgetExhaustion{}, errors.New("application.execution_retry_budget_invalid")
	}
	ordered := append([]governance.BudgetSettlement(nil), settlements...)
	sort.Slice(ordered, func(left, right int) bool { return ordered[left].Ref < ordered[right].Ref })
	charged := governance.ResourceVector{}
	fields := []string{
		envelope.SubjectRef, envelope.Ref, envelope.PolicyHash, strconv.FormatUint(envelope.Revision, 10),
		envelope.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	fields = append(fields, resourceVectorFingerprintFields(envelope.Limit)...)
	fields = append(fields, demand.Ref)
	fields = append(fields, resourceVectorFingerprintFields(demand.Resources)...)
	for index, settlement := range ordered {
		if governance.ValidateBudgetSettlement(settlement) != nil ||
			(index > 0 && ordered[index-1].Ref == settlement.Ref) {
			return "", RetryBudgetExhaustion{}, errors.New("application.execution_retry_budget_invalid")
		}
		durableCharge := settlement.Charged
		// A settled process slot is released even when reconciliation retained
		// it conservatively because provider telemetry was unknown.
		durableCharge.ProcessSlots = 0
		var err error
		charged, err = governance.Add(charged, durableCharge)
		if err != nil {
			return "", RetryBudgetExhaustion{}, err
		}
		fields = append(fields, settlement.Ref, settlement.ReservationRef, settlement.CausalAttemptRef,
			settlement.SettledAt.UTC().Format(time.RFC3339Nano))
		fields = append(fields, resourceVectorFingerprintFields(settlement.Charged)...)
	}
	required, err := governance.Add(charged, demand.Resources)
	if err != nil {
		return "", RetryBudgetExhaustion{}, err
	}
	fits, err := governance.Fits(envelope.Limit, required)
	if err != nil {
		return "", RetryBudgetExhaustion{}, err
	}
	evidence := RetryBudgetExhaustion{
		EnvelopeRef: envelope.Ref, PolicyHash: envelope.PolicyHash, PolicyRevision: envelope.Revision,
		Limit: envelope.Limit, Settled: charged, Demand: demand, Required: required,
		FrontierDigest: fingerprintFields("orquesta.retry-budget.frontier.v1", fields...),
	}
	evidence.GoalRef, err = goal.NewGoalRef(envelope.SubjectRef)
	if err != nil {
		return "", RetryBudgetExhaustion{}, errors.New("application.execution_retry_budget_invalid")
	}
	if fits {
		return RetryBudgetFits, evidence, nil
	}
	return RetryBudgetIrreversible, evidence, nil
}

func resourceVectorFingerprintFields(vector governance.ResourceVector) []string {
	return []string{
		strconv.FormatInt(vector.Tokens, 10), strconv.FormatInt(vector.MoneyMicros, 10), string(vector.Currency),
		strconv.FormatInt(vector.ActiveTimeNS, 10), strconv.FormatInt(vector.ProcessSlots, 10),
		strconv.FormatInt(vector.DiskBytes, 10),
	}
}

func RetryBudgetExhaustionMarker(evidence RetryBudgetExhaustion) (string, error) {
	if evidence.GoalRef.String() == "" || evidence.EnvelopeRef == "" || !validEffectDigest(evidence.PolicyHash) ||
		evidence.PolicyRevision == 0 || governance.ValidateResourceVector(evidence.Limit) != nil ||
		governance.ValidateResourceVector(evidence.Settled) != nil ||
		governance.ValidateBudgetDemand(evidence.Demand) != nil ||
		governance.ValidateResourceVector(evidence.Required) != nil ||
		!validEffectDigest(evidence.FrontierDigest) {
		return "", errors.New("application.retry_budget_exhaustion_invalid")
	}
	required, err := governance.Add(evidence.Settled, evidence.Demand.Resources)
	if err != nil || required != evidence.Required {
		return "", errors.New("application.retry_budget_exhaustion_invalid")
	}
	fits, err := governance.Fits(evidence.Limit, evidence.Required)
	if err != nil || fits {
		return "", errors.New("application.retry_budget_exhaustion_invalid")
	}
	return RetryBudgetExhaustionMarkerPrefix + evidence.FrontierDigest, nil
}

func historicalGoalBudgetEnvelope(record GoalRecord) (governance.BudgetEnvelope, error) {
	if _, err := historicalEffectPolicy(record); err != nil {
		return governance.BudgetEnvelope{}, err
	}
	for _, envelope := range record.BudgetEnvelopes {
		if envelope.Scope == governance.BudgetScopeGoal {
			return envelope, nil
		}
	}
	return governance.BudgetEnvelope{}, errors.New("application.goal_effect_policy_invalid")
}

func validatedGoalBudgetSettlements(
	record GoalRecord,
	additional *governance.BudgetSettlement,
) ([]governance.BudgetSettlement, error) {
	result := append([]governance.BudgetSettlement(nil), record.BudgetSettlements...)
	additionalSeen := additional == nil
	for _, settlement := range record.BudgetSettlements {
		reservation, found := reservationByRef(record.BudgetReservations, settlement.ReservationRef)
		if !found || reservation.GoalRef != record.Goal.Ref().String() ||
			settlement.Reserved != reservation.Resources {
			return nil, errors.New("application.execution_retry_budget_invalid")
		}
		if additional != nil && settlement.ReservationRef == additional.ReservationRef {
			if settlement != *additional {
				return nil, errors.New("application.execution_retry_budget_conflict")
			}
			additionalSeen = true
		}
	}
	if !additionalSeen {
		if governance.ValidateBudgetSettlement(*additional) != nil {
			return nil, errors.New("application.execution_retry_budget_invalid")
		}
		reservation, found := reservationByRef(record.BudgetReservations, additional.ReservationRef)
		if !found || reservation.GoalRef != record.Goal.Ref().String() ||
			additional.Reserved != reservation.Resources {
			return nil, errors.New("application.execution_retry_budget_invalid")
		}
		result = append(result, *additional)
	}
	return result, nil
}

func retryBudgetExhaustionForRecord(
	record GoalRecord,
	demand governance.BudgetDemand,
) (RetryBudgetDisposition, RetryBudgetExhaustion, error) {
	envelope, err := historicalGoalBudgetEnvelope(record)
	if err != nil {
		return "", RetryBudgetExhaustion{}, err
	}
	settlements, err := validatedGoalBudgetSettlements(record, nil)
	if err != nil {
		return "", RetryBudgetExhaustion{}, err
	}
	return ClassifyGoalRetryBudget(envelope, settlements, demand)
}

func ValidateBudgetPolicy(policy BudgetPolicy) error {
	if !validEffectDigest(policy.PolicyHash) || policy.QuotaRetryDelay <= 0 || policy.EffectApprovalTTL <= 0 ||
		governance.ValidateResourceVector(policy.DefaultWorkItemDemand) != nil ||
		policy.DefaultWorkItemDemand.ProcessSlots == 0 {
		return errors.New("application.budget_policy_invalid")
	}
	envelopes := []struct {
		value governance.BudgetEnvelope
		scope governance.BudgetScope
	}{
		{policy.DeploymentEnvelope, governance.BudgetScopeDeployment},
		{policy.ProjectEnvelopeTemplate, governance.BudgetScopeProject},
		{policy.GoalEnvelopeTemplate, governance.BudgetScopeGoal},
	}
	for _, envelope := range envelopes {
		if governance.ValidateBudgetEnvelope(envelope.value) != nil || envelope.value.Scope != envelope.scope ||
			envelope.value.PolicyHash != policy.PolicyHash || envelope.value.Revision != policy.GoalEnvelopeTemplate.Revision {
			return errors.New("application.budget_policy_invalid")
		}
	}
	pairs := [][2]governance.ResourceVector{
		{policy.GoalEnvelopeTemplate.Limit, policy.DefaultWorkItemDemand},
		{policy.ProjectEnvelopeTemplate.Limit, policy.GoalEnvelopeTemplate.Limit},
		{policy.DeploymentEnvelope.Limit, policy.ProjectEnvelopeTemplate.Limit},
	}
	for _, pair := range pairs {
		fits, err := governance.Fits(pair[0], pair[1])
		if err != nil || !fits {
			return errors.New("application.budget_policy_invalid")
		}
	}
	return nil
}

func effectiveDemand(demand governance.BudgetDemand, fallback governance.ResourceVector) governance.BudgetDemand {
	if demand.Resources == (governance.ResourceVector{}) {
		demand.Resources = fallback
	}
	return demand
}

func (policy BudgetPolicy) envelopes(projectRef goal.ProjectRef, goalRef goal.GoalRef, at time.Time) []governance.BudgetEnvelope {
	deployment := policy.DeploymentEnvelope
	project := policy.ProjectEnvelopeTemplate
	project.Ref, project.SubjectRef = "budget-envelope:project:"+projectRef.String()+":"+policy.PolicyHash, projectRef.String()
	goalEnvelope := policy.GoalEnvelopeTemplate
	goalEnvelope.Ref, goalEnvelope.SubjectRef, goalEnvelope.CreatedAt =
		"budget-envelope:goal:"+goalRef.String()+":"+policy.PolicyHash, goalRef.String(), at.UTC()
	return []governance.BudgetEnvelope{deployment, project, goalEnvelope}
}
