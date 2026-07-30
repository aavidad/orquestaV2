package bootstrap

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/governance"
)

func buildBudgetPolicy(snapshot config.Snapshot, now time.Time) (application.BudgetPolicy, error) {
	if now.IsZero() {
		return application.BudgetPolicy{}, errors.New("bootstrap.budget_policy_time_required")
	}
	currency, err := governance.NewCurrency(snapshot.GovernanceBudgetCurrency())
	if err != nil {
		return application.BudgetPolicy{}, errors.New("bootstrap.budget_currency_invalid")
	}
	maxConcurrent := snapshot.GovernanceGlobalProcessSlotsBudget()
	executionTimeout := snapshot.SchedulerExecutionTimeout()
	maxOutputBytes := snapshot.RuntimeMaxOutputBytes()
	if maxConcurrent <= 0 || executionTimeout <= 0 || maxOutputBytes <= 0 {
		return application.BudgetPolicy{}, errors.New("bootstrap.budget_policy_input_invalid")
	}
	activeTimeLimit, ok := multiplyNonNegative(executionTimeout.Nanoseconds(), maxConcurrent)
	if !ok {
		return application.BudgetPolicy{}, errors.New("bootstrap.budget_active_time_overflow")
	}
	diskLimit, ok := multiplyNonNegative(maxOutputBytes, maxConcurrent)
	if !ok {
		return application.BudgetPolicy{}, errors.New("bootstrap.budget_disk_overflow")
	}

	limit := governance.ResourceVector{
		Tokens:      snapshot.GovernanceGlobalTokenBudget(),
		MoneyMicros: snapshot.GovernanceGlobalMoneyMicrosBudget(), Currency: currency,
		ActiveTimeNS: activeTimeLimit, ProcessSlots: maxConcurrent, DiskBytes: diskLimit,
	}
	defaultDemand := governance.ResourceVector{
		Tokens:      snapshot.GovernanceDefaultExecutionTokenBudget(),
		MoneyMicros: snapshot.GovernanceDefaultExecutionMoneyMicrosBudget(), Currency: currency,
		ActiveTimeNS: executionTimeout.Nanoseconds(), ProcessSlots: 1, DiskBytes: maxOutputBytes,
	}
	fits, err := governance.Fits(limit, defaultDemand)
	if err != nil || !fits {
		return application.BudgetPolicy{}, errors.New("bootstrap.default_execution_budget_exceeds_global_budget")
	}
	retryDelay := snapshot.SchedulerPollInterval()
	policyHash := budgetPolicyHash(snapshot, retryDelay, currency)
	policyRevision, err := budgetPolicyRevision(policyHash)
	if err != nil {
		return application.BudgetPolicy{}, err
	}
	createdAt := now.UTC()
	envelope := func(ref, subject string, scope governance.BudgetScope) governance.BudgetEnvelope {
		return governance.BudgetEnvelope{
			Ref: ref, SubjectRef: subject, Scope: scope, Limit: limit,
			Revision: policyRevision, PolicyHash: policyHash, CreatedAt: createdAt,
		}
	}
	policy := application.BudgetPolicy{
		DeploymentEnvelope: envelope(
			"budget-envelope:deployment:local:"+policyHash, "deployment:local", governance.BudgetScopeDeployment,
		),
		ProjectEnvelopeTemplate: envelope(
			"budget-envelope:project:template:"+policyHash, "project:template", governance.BudgetScopeProject,
		),
		GoalEnvelopeTemplate: envelope(
			"budget-envelope:goal:template:"+policyHash, "goal:template", governance.BudgetScopeGoal,
		),
		DefaultWorkItemDemand: defaultDemand, QuotaRetryDelay: retryDelay,
		EffectApprovalTTL: snapshot.GovernanceEffectApprovalTTL(), PolicyHash: policyHash,
	}
	if err := application.ValidateBudgetPolicy(policy); err != nil {
		return application.BudgetPolicy{}, err
	}
	return policy, nil
}

func budgetPolicyHash(snapshot config.Snapshot, retryDelay time.Duration, currency governance.Currency) string {
	fields := []string{
		"orquesta.bootstrap.governance-policy.v1",
		application.EffectRiskPolicyV1,
		string(currency),
		strconv.FormatInt(snapshot.GovernanceGlobalTokenBudget(), 10),
		strconv.FormatInt(snapshot.GovernanceGlobalMoneyMicrosBudget(), 10),
		strconv.FormatInt(snapshot.GovernanceDefaultExecutionTokenBudget(), 10),
		strconv.FormatInt(snapshot.GovernanceDefaultExecutionMoneyMicrosBudget(), 10),
		strconv.FormatInt(snapshot.GovernanceGlobalProcessSlotsBudget(), 10),
		strconv.FormatInt(snapshot.SchedulerExecutionTimeout().Nanoseconds(), 10),
		strconv.FormatInt(snapshot.RuntimeMaxOutputBytes(), 10),
		strconv.FormatInt(retryDelay.Nanoseconds(), 10),
		strconv.FormatInt(snapshot.GovernanceEffectApprovalTTL().Nanoseconds(), 10),
	}
	digest := sha256.Sum256([]byte(strings.Join(fields, "\x00")))
	return hex.EncodeToString(digest[:])
}

func budgetPolicyRevision(policyHash string) (uint64, error) {
	digest, err := hex.DecodeString(policyHash)
	if err != nil || len(digest) != sha256.Size {
		return 0, errors.New("bootstrap.budget_policy_hash_invalid")
	}
	revision := binary.BigEndian.Uint64(digest[:8]) & math.MaxInt64
	if revision == 0 {
		revision = 1
	}
	return revision, nil
}

func multiplyNonNegative(left, right int64) (int64, bool) {
	if left < 0 || right < 0 || left != 0 && right > math.MaxInt64/left {
		return 0, false
	}
	return left * right, true
}
