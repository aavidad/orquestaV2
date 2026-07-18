package bootstrap

import (
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/governance"
)

func TestBudgetPolicyUsesCanonicalGovernanceAndDerivedResourceLimits(t *testing.T) {
	snapshot, err := config.Resolve(config.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	policy, err := buildBudgetPolicy(snapshot, now)
	if err != nil {
		t.Fatalf("buildBudgetPolicy: %v", err)
	}
	if err := application.ValidateBudgetPolicy(policy); err != nil {
		t.Fatalf("ValidateBudgetPolicy: %v", err)
	}
	wantDefault := governance.ResourceVector{
		Tokens: 200000, MoneyMicros: 1000000, Currency: "USD",
		ActiveTimeNS: (45 * time.Minute).Nanoseconds(), ProcessSlots: 1, DiskBytes: 1048576,
	}
	if policy.DefaultWorkItemDemand != wantDefault {
		t.Fatalf("default demand = %+v want %+v", policy.DefaultWorkItemDemand, wantDefault)
	}
	wantLimit := governance.ResourceVector{
		Tokens: 14000000, MoneyMicros: 70000000, Currency: "USD",
		ActiveTimeNS: (45 * time.Minute).Nanoseconds() * 70, ProcessSlots: 70,
		DiskBytes: 1048576 * 70,
	}
	for _, envelope := range []governance.BudgetEnvelope{
		policy.DeploymentEnvelope, policy.ProjectEnvelopeTemplate, policy.GoalEnvelopeTemplate,
	} {
		if envelope.Limit != wantLimit || envelope.PolicyHash != policy.PolicyHash || envelope.CreatedAt != now ||
			envelope.Revision == 0 || !strings.HasSuffix(envelope.Ref, policy.PolicyHash) {
			t.Fatalf("budget envelope = %+v", envelope)
		}
	}
	if policy.QuotaRetryDelay != snapshot.SchedulerPollInterval() {
		t.Fatalf("quota retry delay = %s", policy.QuotaRetryDelay)
	}
	if policy.EffectApprovalTTL != snapshot.GovernanceEffectApprovalTTL() {
		t.Fatalf("effect approval TTL = %s", policy.EffectApprovalTTL)
	}
}

func TestBudgetPolicyHashSeparatesFanoutAndApprovalPolicy(t *testing.T) {
	base, err := config.Resolve(config.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	changedFanout, err := config.Resolve(config.ResolveOptions{TOML: []byte("[scheduler]\nmax_children_per_parent = 7\n")})
	if err != nil {
		t.Fatal(err)
	}
	changedApproval, err := config.Resolve(config.ResolveOptions{TOML: []byte("[governance]\neffect_approval_ttl = \"12h\"\n")})
	if err != nil {
		t.Fatal(err)
	}
	lowercaseCurrency, err := config.Resolve(config.ResolveOptions{TOML: []byte("[governance]\nbudget_currency = \"usd\"\n")})
	if err != nil {
		t.Fatal(err)
	}
	currency, _ := governance.NewCurrency(base.GovernanceBudgetCurrency())
	baseHash := budgetPolicyHash(base, base.SchedulerPollInterval(), currency)
	if baseHash != budgetPolicyHash(changedFanout, changedFanout.SchedulerPollInterval(), currency) {
		t.Fatal("scheduler fanout leaked into governance policy identity")
	}
	approvalHash := budgetPolicyHash(changedApproval, changedApproval.SchedulerPollInterval(), currency)
	if baseHash == approvalHash {
		t.Fatal("effect approval TTL retained governance policy identity")
	}
	lowerCurrency, _ := governance.NewCurrency(lowercaseCurrency.GovernanceBudgetCurrency())
	if baseHash != budgetPolicyHash(lowercaseCurrency, lowercaseCurrency.SchedulerPollInterval(), lowerCurrency) {
		t.Fatal("semantically equal currency produced a different policy hash")
	}
	changedBudget, err := config.Resolve(config.ResolveOptions{TOML: []byte("[governance]\nglobal_token_budget = 15000000\n")})
	if err != nil {
		t.Fatal(err)
	}
	changedHash := budgetPolicyHash(changedBudget, changedBudget.SchedulerPollInterval(), currency)
	if changedHash == baseHash {
		t.Fatal("changed budget retained policy identity")
	}
	baseRevision, baseErr := budgetPolicyRevision(baseHash)
	changedRevision, changedErr := budgetPolicyRevision(changedHash)
	approvalRevision, approvalErr := budgetPolicyRevision(approvalHash)
	if baseErr != nil || changedErr != nil || approvalErr != nil || baseRevision == 0 ||
		changedRevision == 0 || approvalRevision == 0 || baseRevision == changedRevision || baseRevision == approvalRevision {
		t.Fatalf("policy revisions = %d/%d/%d errors=%v/%v/%v",
			baseRevision, changedRevision, approvalRevision, baseErr, changedErr, approvalErr)
	}
}

func TestBudgetPolicyRejectsDerivedResourceOverflow(t *testing.T) {
	if _, ok := multiplyNonNegative(int64(^uint64(0)>>1), 2); ok {
		t.Fatal("overflow accepted")
	}
}
