package acceptance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
)

type v38ElasticFixture struct {
	SchemaVersion           int              `json:"schema_version"`
	FixtureID               string           `json:"fixture_id"`
	ContractID              string           `json:"contract_id"`
	Status                  string           `json:"status"`
	Priority                string           `json:"priority"`
	CapabilityOwnership     []string         `json:"capability_ownership"`
	NonOwnedRequirements    v38NonOwned      `json:"non_owned_requirements"`
	AccreditedPrerequisites []string         `json:"accredited_prerequisites"`
	CausalDependencies      []string         `json:"causal_dependencies"`
	Subgates                v38Subgates      `json:"subgates"`
	Capacity                v38Capacity      `json:"capacity"`
	Demand                  v38Demand        `json:"demand"`
	Scheduling              v38Scheduling    `json:"scheduling"`
	Shutdown                v38Shutdown      `json:"shutdown"`
	Preservation            v38Preservation  `json:"preservation"`
	Recovery                v38Recovery      `json:"recovery"`
	MeasuredLatencies       []string         `json:"measured_latencies"`
	RequiredTests           v38RequiredTests `json:"required_tests"`
	CatalogGuards           v38CatalogGuards `json:"catalog_guards"`
}

type v38NonOwned struct {
	Capabilities                 []string `json:"capabilities"`
	MessagesContinuity           string   `json:"messages_continuity"`
	ExactAgentStop               string   `json:"exact_agent_stop"`
	AgentEnvironmentPreservation string   `json:"agent_environment_preservation"`
}

type v38Subgates struct {
	Order               []string `json:"order"`
	GlobalAccreditation string   `json:"global_accreditation"`
	A                   struct {
		Scope        string `json:"scope"`
		KVM          string `json:"kvm"`
		Firecracker  string `json:"firecracker"`
		StatusEffect string `json:"status_effect"`
	} `json:"a"`
	B struct {
		Scope        string `json:"scope"`
		Selection    string `json:"selection"`
		Isolation    string `json:"isolation"`
		StatusEffect string `json:"status_effect"`
	} `json:"b"`
	C struct {
		Scope        string `json:"scope"`
		Subject      string `json:"subject"`
		StatusEffect string `json:"status_effect"`
	} `json:"c"`
}

type v38Capacity struct {
	Observation  string `json:"observation"`
	KnownQuota   string `json:"known_quota"`
	UnknownQuota string `json:"unknown_quota"`
	Reservation  string `json:"reservation"`
	Consumption  string `json:"consumption"`
	Release      string `json:"release"`
	Exhaustion   string `json:"exhaustion"`
	Restart      string `json:"restart"`
}

type v38Demand struct {
	LogicalCohorts      []int  `json:"logical_cohorts"`
	PhysicalSteps       []int  `json:"physical_steps"`
	CompleteReadySet    bool   `json:"complete_ready_set"`
	HiddenGlobalCeiling string `json:"hidden_global_ceiling"`
	PartialCapacity     string `json:"partial_capacity"`
	LargePhysicalClaim  string `json:"large_physical_claim"`
}

type v38Scheduling struct {
	Dispatcher                string `json:"dispatcher"`
	ClaimSelection            string `json:"claim_selection"`
	StopPriority              string `json:"stop_priority"`
	ObserveProgress           string `json:"observe_progress"`
	ConcurrentActionKind      string `json:"concurrent_action_kind"`
	NonLaunchActions          string `json:"non_launch_actions"`
	PrivateLaunchOnlySelector string `json:"private_launch_only_selector"`
	IdleWorkerGoroutines      string `json:"idle_worker_goroutines"`
}

type v38Shutdown struct {
	Scope       string `json:"scope"`
	Cooperative string `json:"cooperative"`
	Forced      string `json:"forced"`
	HungAgent   string `json:"hung_agent"`
}

type v38Preservation struct {
	SealBeforeUnmount        bool   `json:"seal_before_unmount"`
	InventoryBeforeUnmount   bool   `json:"inventory_before_unmount"`
	TerminalEnvironmentState string `json:"terminal_environment_state"`
	AutomaticDeletion        bool   `json:"automatic_deletion"`
	Removal                  string `json:"removal"`
}

type v38Recovery struct {
	BeforeAttempt        string `json:"before_attempt"`
	AttemptBeforeEffect  string `json:"attempt_before_effect"`
	EffectWithoutReceipt string `json:"effect_without_receipt"`
	UnknownApplied       string `json:"unknown_applied"`
	ReceiptBeforeObserve string `json:"receipt_before_observe"`
	StopReceiptLost      string `json:"stop_receipt_lost"`
	Retry                string `json:"retry"`
	DuplicateLaunch      string `json:"duplicate_launch"`
	StaleFenceWrite      string `json:"stale_fence_write"`
}

type v38RequiredTests struct {
	Names              []string `json:"names"`
	EventEvidence      string   `json:"event_evidence"`
	RejectNoTestsToRun bool     `json:"reject_no_tests_to_run"`
}

type v38CatalogGuards struct {
	CatalogSize                int  `json:"catalog_size"`
	VerticalCount              int  `json:"vertical_count"`
	AcceptanceContractCount    int  `json:"acceptance_contract_count"`
	V39Forbidden               bool `json:"v39_forbidden"`
	ReceiptAbsentWhilePlanned  bool `json:"receipt_absent_while_planned"`
	EvidenceAbsentWhilePlanned bool `json:"evidence_absent_while_planned"`
}

func v38ReadFixtureBytes(t *testing.T, path string) []byte {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("fixture V38 no regular: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func v38DecodeFixture(data []byte) (v38ElasticFixture, error) {
	var fixture v38ElasticFixture
	if err := v38RejectDuplicateJSON(data); err != nil {
		return fixture, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return fixture, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fixture, fmt.Errorf("datos posteriores al JSON: %v", err)
	}
	canonical, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		return fixture, err
	}
	if !bytes.Equal(data, append(canonical, '\n')) {
		return fixture, fmt.Errorf("JSON V38 no canónico")
	}
	return fixture, nil
}

func v38RejectDuplicateJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := v38WalkJSON(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("datos JSON posteriores: %v", err)
	}
	return nil
}

func v38WalkJSON(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	seen := map[string]struct{}{}
	for decoder.More() {
		if delim == '{' {
			keyToken, keyErr := decoder.Token()
			if keyErr != nil {
				return keyErr
			}
			key := keyToken.(string)
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("clave JSON duplicada: %s", key)
			}
			seen[key] = struct{}{}
		}
		if err := v38WalkJSON(decoder); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}

func v38CanonicalBytes(t *testing.T, fixture v38ElasticFixture) []byte {
	t.Helper()
	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}
