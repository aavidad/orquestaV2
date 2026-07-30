package config

import (
	"encoding/json"
	"testing"
	"time"
)

func TestV15GovernanceValuesUseCanonicalInputsAndEffectiveProjection(t *testing.T) {
	snapshot := resolveTOML(t, `
[governance]
budget_currency = "EUR"
global_token_budget = 15000000
global_process_slots_budget = 80
global_money_micros_budget = 71000000
default_execution_token_budget = 210000
default_execution_money_micros_budget = 1100000
effect_approval_ttl = "12h"

[scheduler]
max_children_per_parent = 5
`, map[string]string{
		"ORQUESTA_GOVERNANCE_GLOBAL_TOKEN_BUDGET":         "16000000",
		"ORQUESTA_GOVERNANCE_GLOBAL_PROCESS_SLOTS_BUDGET": "81",
	})
	if snapshot.GovernanceBudgetCurrency() != "EUR" || snapshot.GovernanceGlobalTokenBudget() != 16000000 ||
		snapshot.GovernanceGlobalProcessSlotsBudget() != 81 ||
		snapshot.GovernanceGlobalMoneyMicrosBudget() != 71000000 ||
		snapshot.GovernanceDefaultExecutionTokenBudget() != 210000 ||
		snapshot.GovernanceDefaultExecutionMoneyMicrosBudget() != 1100000 ||
		snapshot.GovernanceEffectApprovalTTL() != 12*time.Hour || snapshot.SchedulerMaxChildrenPerParent() != 5 {
		t.Fatalf("typed governance projection is incomplete: hash=%s", snapshot.Hash())
	}
	assertSource(t, snapshot, KeyGovernanceBudgetCurrency, SourceFile)
	assertSource(t, snapshot, KeyGovernanceGlobalTokenBudget, SourceEnv)
	assertSource(t, snapshot, KeyGovernanceGlobalProcessSlotsBudget, SourceEnv)
	assertSource(t, snapshot, KeySchedulerMaxChildrenPerParent, SourceFile)

	content, err := snapshot.EffectiveJSON()
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Entries []struct {
			Key   Key `json:"key"`
			Value any `json:"value"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	want := map[Key]any{
		KeyGovernanceBudgetCurrency:                    "EUR",
		KeyGovernanceGlobalTokenBudget:                 float64(16000000),
		KeyGovernanceGlobalProcessSlotsBudget:          float64(81),
		KeyGovernanceGlobalMoneyMicrosBudget:           float64(71000000),
		KeyGovernanceDefaultExecutionTokenBudget:       float64(210000),
		KeyGovernanceDefaultExecutionMoneyMicrosBudget: float64(1100000),
		KeyGovernanceEffectApprovalTTL:                 "12h0m0s",
		KeySchedulerMaxChildrenPerParent:               float64(5),
	}
	for _, entry := range document.Entries {
		if expected, exists := want[entry.Key]; exists {
			if entry.Value != expected {
				t.Errorf("effective %s=%v want=%v", entry.Key, entry.Value, expected)
			}
			delete(want, entry.Key)
		}
	}
	if len(want) != 0 {
		t.Fatalf("effective governance entries missing: %v", want)
	}
}

func TestV15GovernanceBoundsRejectZeroWithoutAddingDerivedResourceKeys(t *testing.T) {
	for _, test := range []struct {
		name string
		toml string
		key  Key
	}{
		{"global tokens", "[governance]\nglobal_token_budget = 0\n", KeyGovernanceGlobalTokenBudget},
		{"global process slots", "[governance]\nglobal_process_slots_budget = 0\n", KeyGovernanceGlobalProcessSlotsBudget},
		{"global process slots max", "[governance]\nglobal_process_slots_budget = 4097\n", KeyGovernanceGlobalProcessSlotsBudget},
		{"global money", "[governance]\nglobal_money_micros_budget = 0\n", KeyGovernanceGlobalMoneyMicrosBudget},
		{"execution tokens", "[governance]\ndefault_execution_token_budget = 0\n", KeyGovernanceDefaultExecutionTokenBudget},
		{"execution money", "[governance]\ndefault_execution_money_micros_budget = 0\n", KeyGovernanceDefaultExecutionMoneyMicrosBudget},
		{"approval ttl", "[governance]\neffect_approval_ttl = \"0s\"\n", KeyGovernanceEffectApprovalTTL},
		{"children", "[scheduler]\nmax_children_per_parent = 0\n", KeySchedulerMaxChildrenPerParent},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Resolve(ResolveOptions{TOML: []byte(test.toml)})
			assertConfigError(t, err, ErrorValueInvalid, test.key)
		})
	}
	for _, forbidden := range []Key{
		"governance.active_time_budget", "governance.disk_bytes_budget", "governance.process_slots_budget",
	} {
		if _, found := Definition(forbidden); found {
			t.Errorf("duplicated derived resource key exists: %s", forbidden)
		}
	}
}
