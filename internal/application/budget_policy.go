package application

import (
	"errors"
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
	policy, err := historicalEffectPolicy(record)
	if err != nil {
		return false, err
	}
	charged := governance.ResourceVector{}
	currentSeen := false
	for _, prior := range record.BudgetSettlements {
		reservation, found := reservationByRef(record.BudgetReservations, prior.ReservationRef)
		if !found || reservation.GoalRef != record.Goal.Ref().String() {
			return false, errors.New("application.execution_retry_budget_invalid")
		}
		if prior.ReservationRef == settlement.ReservationRef {
			if prior != *settlement {
				return false, errors.New("application.execution_retry_budget_conflict")
			}
			currentSeen = true
		}
		durableCharge := prior.Charged
		// Process slots are concurrent capacity, not cumulative consumption.
		// A settled reservation releases its slot even when telemetry was
		// unknown and reconciliation conservatively recorded it as charged.
		durableCharge.ProcessSlots = 0
		charged, err = governance.Add(charged, durableCharge)
		if err != nil {
			return false, err
		}
	}
	if !currentSeen {
		reservation, found := reservationByRef(record.BudgetReservations, settlement.ReservationRef)
		if !found || reservation.GoalRef != record.Goal.Ref().String() ||
			settlement.Reserved != reservation.Resources {
			return false, errors.New("application.execution_retry_budget_invalid")
		}
		durableCharge := settlement.Charged
		durableCharge.ProcessSlots = 0
		charged, err = governance.Add(charged, durableCharge)
		if err != nil {
			return false, err
		}
	}
	required, err := governance.Add(charged, demand.Resources)
	if err != nil {
		return false, err
	}
	return governance.Fits(policy.GoalLimit, required)
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
