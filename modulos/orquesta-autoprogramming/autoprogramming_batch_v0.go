package orquestaautoprogramming

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

const (
	AutoprogrammingBatchSchemaVersionV0 = "autoprogramming_batch.v0"

	AutoprogrammingBatchStatusPreparedV0           = "prepared"
	AutoprogrammingBatchStatusGoalsRunningV0       = "goals_running"
	AutoprogrammingBatchStatusPendingIntegrationV0 = "pending_integration"
	AutoprogrammingBatchStatusPendingBatchGateV0   = "pending_batch_gate"
	AutoprogrammingBatchStatusBatchGateRunningV0   = "batch_gate_running"
	AutoprogrammingBatchStatusBatchGatePassedV0    = "batch_gate_passed"
	AutoprogrammingBatchStatusReworkPendingV0      = "rework_pending"
	AutoprogrammingBatchStatusPromotionPendingV0   = "promotion_pending"
	AutoprogrammingBatchStatusClosedV0             = "closed"
	AutoprogrammingBatchStatusBlockedV0            = "blocked"

	AutoprogrammingBatchFocalPendingV0 = "pending"
	AutoprogrammingBatchFocalRunningV0 = "running"
	AutoprogrammingBatchFocalClosedV0  = "closed"

	AutoprogrammingBatchIntegrationPendingV0    = "pending"
	AutoprogrammingBatchIntegrationIntegratedV0 = "integrated"

	AutoprogrammingBatchReworkNoneV0      = "none"
	AutoprogrammingBatchReworkRequestedV0 = "requested"
	AutoprogrammingBatchReworkCompletedV0 = "completed"

	AutoprogrammingBatchTestReceiptPassedV0 = "passed"
	AutoprogrammingBatchTestReceiptFailedV0 = "failed"
)

type AutoprogrammingBatchPlanV0 struct {
	BatchRef     string                         `json:"batch_ref"`
	RequestRef   string                         `json:"request_ref"`
	ProjectRef   string                         `json:"project_ref"`
	BaseRevision string                         `json:"base_revision"`
	Members      []AutoprogrammingBatchMemberV0 `json:"members"`
	FrozenTests  []AutoprogrammingBatchTestV0   `json:"frozen_tests"`
}

type AutoprogrammingBatchV0 struct {
	SchemaVersion      string                                 `json:"schema_version"`
	StoreVersion       uint64                                 `json:"store_version"`
	BatchRef           string                                 `json:"batch_ref"`
	RequestRef         string                                 `json:"request_ref"`
	ProjectRef         string                                 `json:"project_ref"`
	PlanHash           string                                 `json:"plan_hash"`
	BaseRevision       string                                 `json:"base_revision"`
	Members            []AutoprogrammingBatchMemberV0         `json:"members"`
	FrozenTests        []AutoprogrammingBatchTestV0           `json:"frozen_tests"`
	IntegratedRevision string                                 `json:"integrated_revision,omitempty"`
	TestClaims         []AutoprogrammingBatchTestClaimV0      `json:"test_claims,omitempty"`
	TestReceipts       []AutoprogrammingBatchTestReceiptV0    `json:"test_receipts,omitempty"`
	PromotionReceipt   AutoprogrammingBatchPromotionReceiptV0 `json:"promotion_receipt,omitempty"`
	Status             string                                 `json:"status"`
	BlockRef           string                                 `json:"block_ref,omitempty"`
	ActionReceipts     []AutoprogrammingBatchActionReceiptV0  `json:"action_receipts,omitempty"`
}

type AutoprogrammingBatchMemberV0 struct {
	TaskRef               string   `json:"task_ref"`
	GoalRef               string   `json:"goal_ref"`
	RunRef                string   `json:"run_ref"`
	WorkspaceRef          string   `json:"workspace_ref"`
	WriteSet              []string `json:"write_set"`
	SourceRevision        string   `json:"source_revision"`
	IntegrationRevision   string   `json:"integration_revision,omitempty"`
	IntegrationReceiptRef string   `json:"integration_receipt_ref,omitempty"`
	FocalStatus           string   `json:"focal_status"`
	IntegrationStatus     string   `json:"integration_status"`
	ReworkStatus          string   `json:"rework_status"`
}

type AutoprogrammingBatchTestV0 struct {
	Command string `json:"command"`
	SHA256  string `json:"sha256"`
}

type AutoprogrammingBatchTestClaimV0 struct {
	Revision string `json:"revision"`
	TestHash string `json:"test_hash"`
	ClaimRef string `json:"claim_ref"`
}

type AutoprogrammingBatchTestReceiptV0 struct {
	Revision   string `json:"revision"`
	TestHash   string `json:"test_hash"`
	ClaimRef   string `json:"claim_ref"`
	ReceiptRef string `json:"receipt_ref"`
	Status     string `json:"status"`
}

type AutoprogrammingBatchPromotionReceiptV0 struct {
	Revision   string `json:"revision"`
	ReceiptRef string `json:"receipt_ref"`
}

type AutoprogrammingBatchActionReceiptV0 struct {
	IdempotencyKey string `json:"idempotency_key"`
	ActionHash     string `json:"action_hash"`
	StoreVersion   uint64 `json:"store_version"`
}

type AutoprogrammingBatchValidationResultV0 struct {
	Accepted bool                            `json:"accepted"`
	Batch    AutoprogrammingBatchV0          `json:"batch"`
	Issues   []AutoprogrammingRequestIssueV0 `json:"issues,omitempty"`
}

// AutoprogrammingBatchStorePortV0 persists the aggregate; implementations own CAS storage.
type AutoprogrammingBatchStorePortV0 interface {
	LoadAutoprogrammingBatchV0(context.Context, string) (AutoprogrammingBatchV0, error)
	CompareAndSwapAutoprogrammingBatchV0(context.Context, uint64, AutoprogrammingBatchV0) (AutoprogrammingBatchV0, error)
}

type AutoprogrammingBatchTransitionResultV0 struct {
	Accepted bool                            `json:"accepted"`
	Replay   bool                            `json:"replay,omitempty"`
	Batch    AutoprogrammingBatchV0          `json:"batch"`
	Issues   []AutoprogrammingRequestIssueV0 `json:"issues,omitempty"`
}

func NewAutoprogrammingBatchV0(plan AutoprogrammingBatchPlanV0) AutoprogrammingBatchValidationResultV0 {
	for index := range plan.Members {
		plan.Members[index].SourceRevision = ""
		plan.Members[index].IntegrationRevision = ""
		plan.Members[index].IntegrationReceiptRef = ""
		plan.Members[index].FocalStatus = AutoprogrammingBatchFocalPendingV0
		plan.Members[index].IntegrationStatus = AutoprogrammingBatchIntegrationPendingV0
		plan.Members[index].ReworkStatus = AutoprogrammingBatchReworkNoneV0
	}
	batch := AutoprogrammingBatchV0{
		SchemaVersion: AutoprogrammingBatchSchemaVersionV0,
		StoreVersion:  1,
		BatchRef:      plan.BatchRef,
		RequestRef:    plan.RequestRef,
		ProjectRef:    plan.ProjectRef,
		BaseRevision:  plan.BaseRevision,
		Members:       plan.Members,
		FrozenTests:   plan.FrozenTests,
		Status:        AutoprogrammingBatchStatusPreparedV0,
	}
	batch = NormalizeAutoprogrammingBatchV0(batch)
	batch.PlanHash = AutoprogrammingBatchPlanHashV0(batch)
	return ValidateAutoprogrammingBatchV0(batch)
}

func NormalizeAutoprogrammingBatchV0(batch AutoprogrammingBatchV0) AutoprogrammingBatchV0 {
	batch.SchemaVersion = strings.TrimSpace(batch.SchemaVersion)
	batch.BatchRef = autoprogrammingBatchRefV0(batch.BatchRef)
	batch.RequestRef = autoprogrammingBatchRefV0(batch.RequestRef)
	batch.ProjectRef = autoprogrammingBatchRefV0(batch.ProjectRef)
	batch.PlanHash = strings.ToLower(strings.TrimSpace(batch.PlanHash))
	batch.BaseRevision = autoprogrammingBatchRefV0(batch.BaseRevision)
	batch.IntegratedRevision = autoprogrammingBatchRefV0(batch.IntegratedRevision)
	batch.Status = strings.TrimSpace(batch.Status)
	batch.BlockRef = autoprogrammingBatchRefV0(batch.BlockRef)
	batch.Members = autoprogrammingBatchMembersV0(batch.Members)
	batch.FrozenTests = autoprogrammingBatchTestsV0(batch.FrozenTests)
	batch.TestClaims = autoprogrammingBatchClaimsV0(batch.TestClaims)
	batch.TestReceipts = autoprogrammingBatchReceiptsV0(batch.TestReceipts)
	batch.PromotionReceipt.Revision = autoprogrammingBatchRefV0(batch.PromotionReceipt.Revision)
	batch.PromotionReceipt.ReceiptRef = autoprogrammingBatchRefV0(batch.PromotionReceipt.ReceiptRef)
	batch.ActionReceipts = autoprogrammingBatchActionReceiptsV0(batch.ActionReceipts)
	return batch
}

func ValidateAutoprogrammingBatchV0(batch AutoprogrammingBatchV0) AutoprogrammingBatchValidationResultV0 {
	batch = NormalizeAutoprogrammingBatchV0(batch)
	issues := autoprogrammingBatchStaticIssuesV0(batch)
	issues = append(issues, autoprogrammingBatchPlanIssuesV0(batch)...)
	issues = append(issues, autoprogrammingBatchDynamicIssuesV0(batch)...)
	return AutoprogrammingBatchValidationResultV0{Accepted: len(issues) == 0, Batch: batch, Issues: issues}
}

func AutoprogrammingBatchPlanHashV0(batch AutoprogrammingBatchV0) string {
	batch = NormalizeAutoprogrammingBatchV0(batch)
	if len(autoprogrammingBatchStaticIssuesV0(batch)) > 0 || len(autoprogrammingBatchPlanContentIssuesV0(batch)) > 0 {
		return ""
	}
	parts := []string{"autoprogramming-batch-plan.v0", batch.BatchRef, batch.RequestRef, batch.ProjectRef, batch.BaseRevision}
	for _, member := range batch.Members {
		parts = append(parts, strings.Join([]string{
			"member", member.TaskRef, member.GoalRef, member.RunRef, member.WorkspaceRef, strings.Join(member.WriteSet, "\x00"),
		}, "\n"))
	}
	for _, test := range batch.FrozenTests {
		parts = append(parts, strings.Join([]string{"test", test.Command, test.SHA256}, "\n"))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return "autoprogramming-batch-plan-v0:" + hex.EncodeToString(sum[:])
}

func AutoprogrammingBatchTestHashV0(test AutoprogrammingBatchTestV0) string {
	test = autoprogrammingBatchTestV0(test)
	if strings.TrimSpace(test.Command) == "" || !frozenRequiredTestSHA256ValidV0(test.SHA256) {
		return ""
	}
	sum := sha256.Sum256([]byte("autoprogramming-batch-test.v0\n" + test.Command + "\n" + test.SHA256))
	return "autoprogramming-batch-test-v0:" + hex.EncodeToString(sum[:])
}

func RegisterAutoprogrammingBatchLaunchV0(batch AutoprogrammingBatchV0, expectedStoreVersion uint64, idempotencyKey, taskRef string) AutoprogrammingBatchTransitionResultV0 {
	return transitionAutoprogrammingBatchV0(batch, expectedStoreVersion, idempotencyKey, "launch", []string{taskRef}, func(next *AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
		if next.Status != AutoprogrammingBatchStatusPreparedV0 && next.Status != AutoprogrammingBatchStatusGoalsRunningV0 {
			return autoprogrammingBatchTransitionIssueV0("batch_launch_not_allowed", "status", "launch solo permitido antes del cierre focal")
		}
		member := autoprogrammingBatchMemberIndexV0(next.Members, taskRef)
		if member < 0 {
			return autoprogrammingBatchTransitionIssueV0("batch_member_unknown", "task_ref", "miembro no pertenece al plan congelado")
		}
		if next.Members[member].FocalStatus != AutoprogrammingBatchFocalPendingV0 {
			return autoprogrammingBatchTransitionIssueV0("batch_launch_already_recorded", "task_ref", "launch focal ya registrado")
		}
		next.Members[member].FocalStatus = AutoprogrammingBatchFocalRunningV0
		next.Status = AutoprogrammingBatchStatusGoalsRunningV0
		return nil
	})
}

func RegisterAutoprogrammingBatchFocalCloseV0(batch AutoprogrammingBatchV0, expectedStoreVersion uint64, idempotencyKey, taskRef string) AutoprogrammingBatchTransitionResultV0 {
	return transitionAutoprogrammingBatchV0(batch, expectedStoreVersion, idempotencyKey, "focal_close", []string{taskRef}, func(next *AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
		if next.Status != AutoprogrammingBatchStatusGoalsRunningV0 && next.Status != AutoprogrammingBatchStatusReworkPendingV0 {
			return autoprogrammingBatchTransitionIssueV0("batch_focal_close_not_allowed", "status", "cierre focal no permitido en este estado")
		}
		member := autoprogrammingBatchMemberIndexV0(next.Members, taskRef)
		if member < 0 || next.Members[member].FocalStatus != AutoprogrammingBatchFocalRunningV0 {
			return autoprogrammingBatchTransitionIssueV0("batch_focal_close_invalid", "task_ref", "solo un focal en ejecucion puede cerrar")
		}
		next.Members[member].FocalStatus = AutoprogrammingBatchFocalClosedV0
		if next.Members[member].ReworkStatus == AutoprogrammingBatchReworkRequestedV0 {
			next.Members[member].ReworkStatus = AutoprogrammingBatchReworkCompletedV0
		}
		if autoprogrammingBatchAllFocalClosedV0(next.Members) {
			next.Status = AutoprogrammingBatchStatusPendingIntegrationV0
		}
		return nil
	})
}

func RegisterAutoprogrammingBatchIntegrationV0(batch AutoprogrammingBatchV0, expectedStoreVersion uint64, idempotencyKey, taskRef, sourceRevision, integrationRevision, receiptRef string) AutoprogrammingBatchTransitionResultV0 {
	return transitionAutoprogrammingBatchV0(batch, expectedStoreVersion, idempotencyKey, "integration", []string{taskRef, sourceRevision, integrationRevision, receiptRef}, func(next *AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
		if next.Status != AutoprogrammingBatchStatusPendingIntegrationV0 {
			return autoprogrammingBatchTransitionIssueV0("batch_integration_not_allowed", "status", "integracion requiere todos los focales cerrados")
		}
		sourceRevision = autoprogrammingBatchRefV0(sourceRevision)
		integrationRevision = autoprogrammingBatchRefV0(integrationRevision)
		if !autoprogrammingBatchRefValidV0(sourceRevision) || !autoprogrammingBatchRefValidV0(integrationRevision) {
			return autoprogrammingBatchTransitionIssueV0("batch_integrated_revision_invalid", "revision", "revisiones fuente e integrada requeridas")
		}
		member := autoprogrammingBatchMemberIndexV0(next.Members, taskRef)
		if member < 0 || next.Members[member].IntegrationStatus != AutoprogrammingBatchIntegrationPendingV0 || !autoprogrammingBatchRefValidV0(receiptRef) {
			return autoprogrammingBatchTransitionIssueV0("batch_integration_invalid", "task_ref", "miembro sin integracion pendiente")
		}
		next.Members[member].IntegrationStatus = AutoprogrammingBatchIntegrationIntegratedV0
		next.Members[member].SourceRevision = sourceRevision
		next.Members[member].IntegrationRevision = integrationRevision
		next.Members[member].IntegrationReceiptRef = autoprogrammingBatchRefV0(receiptRef)
		if autoprogrammingBatchAllIntegratedV0(next.Members) {
			next.IntegratedRevision = integrationRevision
			next.Status = AutoprogrammingBatchStatusPendingBatchGateV0
		}
		return nil
	})
}

func ClaimAutoprogrammingBatchTestV0(batch AutoprogrammingBatchV0, expectedStoreVersion uint64, idempotencyKey, revision, testHash, claimRef string) AutoprogrammingBatchTransitionResultV0 {
	return transitionAutoprogrammingBatchV0(batch, expectedStoreVersion, idempotencyKey, "claim_test", []string{revision, testHash, claimRef}, func(next *AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
		if next.Status != AutoprogrammingBatchStatusPendingBatchGateV0 && next.Status != AutoprogrammingBatchStatusBatchGateRunningV0 {
			return autoprogrammingBatchTransitionIssueV0("batch_test_claim_not_allowed", "status", "claim requiere batch gate pendiente")
		}
		if revision = autoprogrammingBatchRefV0(revision); revision != next.IntegratedRevision {
			return autoprogrammingBatchTransitionIssueV0("batch_test_claim_revision_invalid", "revision", "claim debe usar la revision integrada")
		}
		if !autoprogrammingBatchHasTestHashV0(next.FrozenTests, testHash) || !autoprogrammingBatchRefValidV0(claimRef) {
			return autoprogrammingBatchTransitionIssueV0("batch_test_claim_invalid", "test_hash", "test congelado y claim ref requeridos")
		}
		for _, claim := range next.TestClaims {
			if claim.Revision == revision && claim.TestHash == testHash {
				return autoprogrammingBatchTransitionIssueV0("batch_test_claim_duplicate", "test_hash", "solo existe un claim por revision y test")
			}
		}
		next.TestClaims = append(next.TestClaims, AutoprogrammingBatchTestClaimV0{Revision: revision, TestHash: testHash, ClaimRef: autoprogrammingBatchRefV0(claimRef)})
		next.Status = AutoprogrammingBatchStatusBatchGateRunningV0
		return nil
	})
}

func RecordAutoprogrammingBatchTestReceiptV0(batch AutoprogrammingBatchV0, expectedStoreVersion uint64, idempotencyKey, revision, testHash, claimRef, receiptRef, status string) AutoprogrammingBatchTransitionResultV0 {
	return transitionAutoprogrammingBatchV0(batch, expectedStoreVersion, idempotencyKey, "test_receipt", []string{revision, testHash, claimRef, receiptRef, status}, func(next *AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
		if next.Status != AutoprogrammingBatchStatusBatchGateRunningV0 {
			return autoprogrammingBatchTransitionIssueV0("batch_test_receipt_not_allowed", "status", "receipt requiere batch gate en curso")
		}
		revision, testHash, claimRef, receiptRef, status = autoprogrammingBatchRefV0(revision), strings.TrimSpace(testHash), autoprogrammingBatchRefV0(claimRef), autoprogrammingBatchRefV0(receiptRef), strings.TrimSpace(status)
		if revision != next.IntegratedRevision || !autoprogrammingBatchClaimExistsV0(next.TestClaims, revision, testHash, claimRef) || !autoprogrammingBatchRefValidV0(receiptRef) || !autoprogrammingBatchTestReceiptStatusValidV0(status) {
			return autoprogrammingBatchTransitionIssueV0("batch_test_receipt_invalid", "test_receipt", "receipt debe corresponder a un claim valido de la revision integrada")
		}
		if autoprogrammingBatchReceiptExistsV0(next.TestReceipts, revision, testHash) {
			return autoprogrammingBatchTransitionIssueV0("batch_test_receipt_duplicate", "test_hash", "solo existe un receipt por revision y test")
		}
		next.TestReceipts = append(next.TestReceipts, AutoprogrammingBatchTestReceiptV0{Revision: revision, TestHash: testHash, ClaimRef: claimRef, ReceiptRef: receiptRef, Status: status})
		if status == AutoprogrammingBatchTestReceiptFailedV0 {
			next.Status = AutoprogrammingBatchStatusReworkPendingV0
		} else if autoprogrammingBatchAllTestsPassedV0(*next) {
			next.Status = AutoprogrammingBatchStatusBatchGatePassedV0
		}
		return nil
	})
}

func RequestAutoprogrammingBatchReworkV0(batch AutoprogrammingBatchV0, expectedStoreVersion uint64, idempotencyKey, taskRef string) AutoprogrammingBatchTransitionResultV0 {
	return transitionAutoprogrammingBatchV0(batch, expectedStoreVersion, idempotencyKey, "request_rework", []string{taskRef}, func(next *AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
		if next.Status != AutoprogrammingBatchStatusReworkPendingV0 && next.Status != AutoprogrammingBatchStatusBatchGatePassedV0 {
			return autoprogrammingBatchTransitionIssueV0("batch_rework_not_allowed", "status", "rework requiere fallo de gate o gate pasado pendiente de promocion")
		}
		member := autoprogrammingBatchMemberIndexV0(next.Members, taskRef)
		if member < 0 {
			return autoprogrammingBatchTransitionIssueV0("batch_member_unknown", "task_ref", "miembro no pertenece al plan congelado")
		}
		next.Members[member].FocalStatus = AutoprogrammingBatchFocalRunningV0
		next.Members[member].ReworkStatus = AutoprogrammingBatchReworkRequestedV0
		for index := range next.Members {
			next.Members[index].IntegrationStatus = AutoprogrammingBatchIntegrationPendingV0
			next.Members[index].SourceRevision = ""
			next.Members[index].IntegrationRevision = ""
			next.Members[index].IntegrationReceiptRef = ""
		}
		next.IntegratedRevision = ""
		next.PromotionReceipt = AutoprogrammingBatchPromotionReceiptV0{}
		next.Status = AutoprogrammingBatchStatusGoalsRunningV0
		return nil
	})
}

func RegisterAutoprogrammingBatchPromotionV0(batch AutoprogrammingBatchV0, expectedStoreVersion uint64, idempotencyKey, revision, receiptRef string) AutoprogrammingBatchTransitionResultV0 {
	return transitionAutoprogrammingBatchV0(batch, expectedStoreVersion, idempotencyKey, "promotion", []string{revision, receiptRef}, func(next *AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
		if next.Status != AutoprogrammingBatchStatusBatchGatePassedV0 || autoprogrammingBatchRefV0(revision) != next.IntegratedRevision || !autoprogrammingBatchRefValidV0(receiptRef) {
			return autoprogrammingBatchTransitionIssueV0("batch_promotion_invalid", "promotion_receipt", "promocion requiere gate pasado, revision integrada y receipt")
		}
		next.PromotionReceipt = AutoprogrammingBatchPromotionReceiptV0{Revision: next.IntegratedRevision, ReceiptRef: autoprogrammingBatchRefV0(receiptRef)}
		next.Status = AutoprogrammingBatchStatusPromotionPendingV0
		return nil
	})
}

func CloseAutoprogrammingBatchV0(batch AutoprogrammingBatchV0, expectedStoreVersion uint64, idempotencyKey string) AutoprogrammingBatchTransitionResultV0 {
	return transitionAutoprogrammingBatchV0(batch, expectedStoreVersion, idempotencyKey, "close", nil, func(next *AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
		if next.Status != AutoprogrammingBatchStatusPromotionPendingV0 || !autoprogrammingBatchAllTestsPassedV0(*next) || next.PromotionReceipt.Revision != next.IntegratedRevision || !autoprogrammingBatchRefValidV0(next.PromotionReceipt.ReceiptRef) {
			return autoprogrammingBatchTransitionIssueV0("batch_close_invalid", "status", "cierre requiere todos los tests passed y promocion registrada")
		}
		next.Status = AutoprogrammingBatchStatusClosedV0
		return nil
	})
}

func BlockAutoprogrammingBatchV0(batch AutoprogrammingBatchV0, expectedStoreVersion uint64, idempotencyKey, blockRef string) AutoprogrammingBatchTransitionResultV0 {
	return transitionAutoprogrammingBatchV0(batch, expectedStoreVersion, idempotencyKey, "block", []string{blockRef}, func(next *AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
		if next.Status == AutoprogrammingBatchStatusClosedV0 || next.Status == AutoprogrammingBatchStatusBlockedV0 || !autoprogrammingBatchRefValidV0(blockRef) {
			return autoprogrammingBatchTransitionIssueV0("batch_block_invalid", "block_ref", "bloqueo requiere batch activo y ref")
		}
		next.Status, next.BlockRef = AutoprogrammingBatchStatusBlockedV0, autoprogrammingBatchRefV0(blockRef)
		return nil
	})
}

func transitionAutoprogrammingBatchV0(batch AutoprogrammingBatchV0, expectedStoreVersion uint64, idempotencyKey, action string, values []string, apply func(*AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0) AutoprogrammingBatchTransitionResultV0 {
	batch = NormalizeAutoprogrammingBatchV0(batch)
	key := strings.TrimSpace(idempotencyKey)
	actionHash := autoprogrammingBatchActionHashV0(action, values)
	if key == "" || actionHash == "" {
		return autoprogrammingBatchTransitionRejectedV0(batch, "batch_idempotency_key_invalid", "idempotency_key", "clave idempotente requerida")
	}
	for _, receipt := range batch.ActionReceipts {
		if receipt.IdempotencyKey != key {
			continue
		}
		if receipt.ActionHash == actionHash {
			return AutoprogrammingBatchTransitionResultV0{Accepted: true, Replay: true, Batch: batch}
		}
		return autoprogrammingBatchTransitionRejectedV0(batch, "batch_idempotency_key_reused", "idempotency_key", "clave ya usada por otra transicion")
	}
	if batch.StoreVersion != expectedStoreVersion {
		return autoprogrammingBatchTransitionRejectedV0(batch, "batch_cas_conflict", "expected_store_version", "store_version no coincide")
	}
	if validation := ValidateAutoprogrammingBatchV0(batch); !validation.Accepted {
		return AutoprogrammingBatchTransitionResultV0{Batch: validation.Batch, Issues: validation.Issues}
	}
	next := batch
	if issues := apply(&next); len(issues) > 0 {
		return AutoprogrammingBatchTransitionResultV0{Batch: batch, Issues: issues}
	}
	next.StoreVersion++
	next.ActionReceipts = append(next.ActionReceipts, AutoprogrammingBatchActionReceiptV0{IdempotencyKey: key, ActionHash: actionHash, StoreVersion: next.StoreVersion})
	validation := ValidateAutoprogrammingBatchV0(next)
	return AutoprogrammingBatchTransitionResultV0{Accepted: validation.Accepted, Batch: validation.Batch, Issues: validation.Issues}
}

func autoprogrammingBatchStaticIssuesV0(batch AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	if batch.SchemaVersion != AutoprogrammingBatchSchemaVersionV0 {
		issues = append(issues, autoprogrammingRequestIssueV0("batch_schema_version_invalid", "schema_version", "schema version requerida"))
	}
	if batch.StoreVersion == 0 {
		issues = append(issues, autoprogrammingRequestIssueV0("batch_store_version_invalid", "store_version", "store version positiva requerida"))
	}
	for _, item := range []struct{ field, value string }{{"batch_ref", batch.BatchRef}, {"request_ref", batch.RequestRef}, {"project_ref", batch.ProjectRef}, {"base_revision", batch.BaseRevision}} {
		if !autoprogrammingBatchRefValidV0(item.value) {
			issues = append(issues, autoprogrammingRequestIssueV0("batch_"+item.field+"_invalid", item.field, "ref compacta requerida"))
		}
	}
	if !autoprogrammingBatchStatusValidV0(batch.Status) {
		issues = append(issues, autoprogrammingRequestIssueV0("batch_status_invalid", "status", "status de batch invalido"))
	}
	return issues
}

func autoprogrammingBatchPlanIssuesV0(batch AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
	issues := autoprogrammingBatchPlanContentIssuesV0(batch)
	if want := autoprogrammingBatchPlanHashUncheckedV0(batch); want == "" || batch.PlanHash != want {
		issues = append(issues, autoprogrammingRequestIssueV0("batch_plan_hash_invalid", "plan_hash", "plan congelado no coincide con su hash"))
	}
	return issues
}

func autoprogrammingBatchPlanContentIssuesV0(batch AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	if len(batch.Members) == 0 {
		issues = append(issues, autoprogrammingRequestIssueV0("batch_members_missing", "members", "miembros congelados requeridos"))
	}
	seenMembers := map[string]bool{}
	for _, member := range batch.Members {
		if !autoprogrammingBatchMemberPlanValidV0(member) || seenMembers[member.TaskRef] {
			issues = append(issues, autoprogrammingRequestIssueV0("batch_member_invalid", "members", "miembro congelado invalido o duplicado"))
		}
		seenMembers[member.TaskRef] = true
	}
	if len(batch.FrozenTests) == 0 {
		issues = append(issues, autoprogrammingRequestIssueV0("batch_frozen_tests_missing", "frozen_tests", "tests batch congelados requeridos"))
	}
	seenTests := map[string]bool{}
	for _, test := range batch.FrozenTests {
		hash := AutoprogrammingBatchTestHashV0(test)
		if hash == "" || seenTests[hash] {
			issues = append(issues, autoprogrammingRequestIssueV0("batch_frozen_test_invalid", "frozen_tests", "test congelado invalido o duplicado"))
		}
		seenTests[hash] = true
	}
	return issues
}

func autoprogrammingBatchDynamicIssuesV0(batch AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	if batch.Status == AutoprogrammingBatchStatusBlockedV0 && !autoprogrammingBatchRefValidV0(batch.BlockRef) {
		issues = append(issues, autoprogrammingRequestIssueV0("batch_block_ref_invalid", "block_ref", "batch bloqueado requiere ref"))
	}
	if batch.Status != AutoprogrammingBatchStatusBlockedV0 && batch.BlockRef != "" {
		issues = append(issues, autoprogrammingRequestIssueV0("batch_block_ref_unexpected", "block_ref", "block ref solo aplica a estado blocked"))
	}
	if batch.IntegratedRevision != "" && !autoprogrammingBatchRefValidV0(batch.IntegratedRevision) {
		issues = append(issues, autoprogrammingRequestIssueV0("batch_integrated_revision_invalid", "integrated_revision", "revision integrada compacta requerida"))
	}
	for _, member := range batch.Members {
		if !autoprogrammingBatchMemberStateValidV0(member) {
			issues = append(issues, autoprogrammingRequestIssueV0("batch_member_state_invalid", "members", "estado de miembro invalido"))
		}
	}
	issues = append(issues, autoprogrammingBatchEvidenceIssuesV0(batch)...)
	issues = append(issues, autoprogrammingBatchStatusIssuesV0(batch)...)
	return issues
}

func autoprogrammingBatchEvidenceIssuesV0(batch AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	claims := map[string]AutoprogrammingBatchTestClaimV0{}
	for _, claim := range batch.TestClaims {
		key := claim.Revision + "\x00" + claim.TestHash
		if !autoprogrammingBatchRefValidV0(claim.Revision) || !autoprogrammingBatchHasTestHashV0(batch.FrozenTests, claim.TestHash) || !autoprogrammingBatchRefValidV0(claim.ClaimRef) || claims[key].ClaimRef != "" {
			issues = append(issues, autoprogrammingRequestIssueV0("batch_test_claim_invalid", "test_claims", "claim debe ser unico y pertenecer a test congelado"))
		}
		claims[key] = claim
	}
	receipts := map[string]bool{}
	for _, receipt := range batch.TestReceipts {
		key := receipt.Revision + "\x00" + receipt.TestHash
		claim, claimed := claims[key]
		if !claimed || claim.ClaimRef != receipt.ClaimRef || !autoprogrammingBatchRefValidV0(receipt.ReceiptRef) || !autoprogrammingBatchTestReceiptStatusValidV0(receipt.Status) || receipts[key] {
			issues = append(issues, autoprogrammingRequestIssueV0("batch_test_receipt_invalid", "test_receipts", "receipt debe corresponder a claim unico"))
		}
		receipts[key] = true
	}
	seenKeys := map[string]bool{}
	for _, receipt := range batch.ActionReceipts {
		if receipt.IdempotencyKey == "" || !autoprogrammingBatchSHA256ValidV0(receipt.ActionHash) || receipt.StoreVersion == 0 || seenKeys[receipt.IdempotencyKey] {
			issues = append(issues, autoprogrammingRequestIssueV0("batch_action_receipt_invalid", "action_receipts", "receipt idempotente invalido"))
		}
		seenKeys[receipt.IdempotencyKey] = true
	}
	return issues
}

func autoprogrammingBatchStatusIssuesV0(batch AutoprogrammingBatchV0) []AutoprogrammingRequestIssueV0 {
	allClosed, allIntegrated := autoprogrammingBatchAllFocalClosedV0(batch.Members), autoprogrammingBatchAllIntegratedV0(batch.Members)
	if (batch.Status == AutoprogrammingBatchStatusPendingBatchGateV0 || batch.Status == AutoprogrammingBatchStatusBatchGateRunningV0 || batch.Status == AutoprogrammingBatchStatusBatchGatePassedV0 || batch.Status == AutoprogrammingBatchStatusPromotionPendingV0 || batch.Status == AutoprogrammingBatchStatusClosedV0) && (!allClosed || !allIntegrated || batch.IntegratedRevision == "") {
		return autoprogrammingBatchTransitionIssueV0("batch_gate_before_integration", "status", "batch gate requiere todos los miembros integrados")
	}
	if batch.Status == AutoprogrammingBatchStatusPendingIntegrationV0 && !allClosed {
		return autoprogrammingBatchTransitionIssueV0("batch_pending_integration_invalid", "status", "pending integration requiere focales cerrados")
	}
	if (batch.Status == AutoprogrammingBatchStatusBatchGatePassedV0 || batch.Status == AutoprogrammingBatchStatusPromotionPendingV0 || batch.Status == AutoprogrammingBatchStatusClosedV0) && !autoprogrammingBatchAllTestsPassedV0(batch) {
		return autoprogrammingBatchTransitionIssueV0("batch_tests_not_passed", "test_receipts", "estado requiere todos los tests batch passed")
	}
	if (batch.Status == AutoprogrammingBatchStatusPromotionPendingV0 || batch.Status == AutoprogrammingBatchStatusClosedV0) && (batch.PromotionReceipt.Revision != batch.IntegratedRevision || !autoprogrammingBatchRefValidV0(batch.PromotionReceipt.ReceiptRef)) {
		return autoprogrammingBatchTransitionIssueV0("batch_promotion_receipt_invalid", "promotion_receipt", "estado requiere promotion receipt de la revision integrada")
	}
	if batch.Status == AutoprogrammingBatchStatusClosedV0 && (!autoprogrammingBatchAllTestsPassedV0(batch) || !autoprogrammingBatchRefValidV0(batch.PromotionReceipt.ReceiptRef)) {
		return autoprogrammingBatchTransitionIssueV0("batch_close_invalid", "status", "close requiere tests passed y promotion receipt")
	}
	return nil
}

func autoprogrammingBatchMembersV0(members []AutoprogrammingBatchMemberV0) []AutoprogrammingBatchMemberV0 {
	out := append([]AutoprogrammingBatchMemberV0(nil), members...)
	for index := range out {
		out[index].TaskRef = autoprogrammingBatchRefV0(out[index].TaskRef)
		out[index].GoalRef = autoprogrammingBatchRefV0(out[index].GoalRef)
		out[index].RunRef = autoprogrammingBatchRefV0(out[index].RunRef)
		out[index].WorkspaceRef = autoprogrammingBatchRefV0(out[index].WorkspaceRef)
		out[index].WriteSet = compactStringsV0(out[index].WriteSet)
		out[index].SourceRevision = autoprogrammingBatchRefV0(out[index].SourceRevision)
		out[index].IntegrationRevision = autoprogrammingBatchRefV0(out[index].IntegrationRevision)
		out[index].IntegrationReceiptRef = autoprogrammingBatchRefV0(out[index].IntegrationReceiptRef)
		out[index].FocalStatus = strings.TrimSpace(out[index].FocalStatus)
		out[index].IntegrationStatus = strings.TrimSpace(out[index].IntegrationStatus)
		out[index].ReworkStatus = strings.TrimSpace(out[index].ReworkStatus)
	}
	sort.SliceStable(out, func(left, right int) bool { return out[left].TaskRef < out[right].TaskRef })
	return out
}

func autoprogrammingBatchTestsV0(tests []AutoprogrammingBatchTestV0) []AutoprogrammingBatchTestV0 {
	out := make([]AutoprogrammingBatchTestV0, 0, len(tests))
	for _, test := range tests {
		out = append(out, autoprogrammingBatchTestV0(test))
	}
	sort.SliceStable(out, func(left, right int) bool {
		return AutoprogrammingBatchTestHashV0(out[left]) < AutoprogrammingBatchTestHashV0(out[right])
	})
	return out
}

func autoprogrammingBatchTestV0(test AutoprogrammingBatchTestV0) AutoprogrammingBatchTestV0 {
	return AutoprogrammingBatchTestV0{Command: strings.TrimSpace(test.Command), SHA256: strings.ToLower(strings.TrimSpace(test.SHA256))}
}

func autoprogrammingBatchClaimsV0(claims []AutoprogrammingBatchTestClaimV0) []AutoprogrammingBatchTestClaimV0 {
	out := append([]AutoprogrammingBatchTestClaimV0(nil), claims...)
	for index := range out {
		out[index].Revision, out[index].TestHash, out[index].ClaimRef = autoprogrammingBatchRefV0(out[index].Revision), strings.TrimSpace(out[index].TestHash), autoprogrammingBatchRefV0(out[index].ClaimRef)
	}
	sort.SliceStable(out, func(left, right int) bool {
		return out[left].Revision+"\x00"+out[left].TestHash < out[right].Revision+"\x00"+out[right].TestHash
	})
	return out
}

func autoprogrammingBatchReceiptsV0(receipts []AutoprogrammingBatchTestReceiptV0) []AutoprogrammingBatchTestReceiptV0 {
	out := append([]AutoprogrammingBatchTestReceiptV0(nil), receipts...)
	for index := range out {
		out[index].Revision, out[index].TestHash, out[index].ClaimRef, out[index].ReceiptRef, out[index].Status = autoprogrammingBatchRefV0(out[index].Revision), strings.TrimSpace(out[index].TestHash), autoprogrammingBatchRefV0(out[index].ClaimRef), autoprogrammingBatchRefV0(out[index].ReceiptRef), strings.TrimSpace(out[index].Status)
	}
	sort.SliceStable(out, func(left, right int) bool {
		return out[left].Revision+"\x00"+out[left].TestHash < out[right].Revision+"\x00"+out[right].TestHash
	})
	return out
}

func autoprogrammingBatchActionReceiptsV0(receipts []AutoprogrammingBatchActionReceiptV0) []AutoprogrammingBatchActionReceiptV0 {
	out := append([]AutoprogrammingBatchActionReceiptV0(nil), receipts...)
	for index := range out {
		out[index].IdempotencyKey, out[index].ActionHash = strings.TrimSpace(out[index].IdempotencyKey), strings.ToLower(strings.TrimSpace(out[index].ActionHash))
	}
	sort.SliceStable(out, func(left, right int) bool { return out[left].IdempotencyKey < out[right].IdempotencyKey })
	return out
}

func autoprogrammingBatchPlanHashUncheckedV0(batch AutoprogrammingBatchV0) string {
	parts := []string{"autoprogramming-batch-plan.v0", batch.BatchRef, batch.RequestRef, batch.ProjectRef, batch.BaseRevision}
	for _, member := range batch.Members {
		parts = append(parts, strings.Join([]string{"member", member.TaskRef, member.GoalRef, member.RunRef, member.WorkspaceRef, strings.Join(member.WriteSet, "\x00")}, "\n"))
	}
	for _, test := range batch.FrozenTests {
		parts = append(parts, strings.Join([]string{"test", test.Command, test.SHA256}, "\n"))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return "autoprogramming-batch-plan-v0:" + hex.EncodeToString(sum[:])
}

func autoprogrammingBatchMemberPlanValidV0(member AutoprogrammingBatchMemberV0) bool {
	if !autoprogrammingBatchRefValidV0(member.TaskRef) || !autoprogrammingBatchRefValidV0(member.GoalRef) || !autoprogrammingBatchRefValidV0(member.RunRef) || !autoprogrammingBatchRefValidV0(member.WorkspaceRef) || len(member.WriteSet) == 0 {
		return false
	}
	for _, path := range member.WriteSet {
		if !autoprogrammingRequestWriteSetPathAllowedV0(path) {
			return false
		}
	}
	return true
}

func autoprogrammingBatchMemberStateValidV0(member AutoprogrammingBatchMemberV0) bool {
	if member.FocalStatus != AutoprogrammingBatchFocalPendingV0 && member.FocalStatus != AutoprogrammingBatchFocalRunningV0 && member.FocalStatus != AutoprogrammingBatchFocalClosedV0 {
		return false
	}
	if member.ReworkStatus != AutoprogrammingBatchReworkNoneV0 && member.ReworkStatus != AutoprogrammingBatchReworkRequestedV0 && member.ReworkStatus != AutoprogrammingBatchReworkCompletedV0 {
		return false
	}
	if member.IntegrationStatus == AutoprogrammingBatchIntegrationPendingV0 {
		return member.SourceRevision == "" && member.IntegrationRevision == "" && member.IntegrationReceiptRef == ""
	}
	return member.IntegrationStatus == AutoprogrammingBatchIntegrationIntegratedV0 && autoprogrammingBatchRefValidV0(member.SourceRevision) && autoprogrammingBatchRefValidV0(member.IntegrationRevision) && autoprogrammingBatchRefValidV0(member.IntegrationReceiptRef)
}

func autoprogrammingBatchRefV0(value string) string { return strings.TrimSpace(value) }
func autoprogrammingBatchRefValidV0(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= 600 && !strings.ContainsAny(value, " /\\\t\r\n\x00")
}
func autoprogrammingBatchSHA256ValidV0(value string) bool {
	return len(value) == 64 && frozenRequiredTestSHA256ValidV0(value)
}
func autoprogrammingBatchStatusValidV0(status string) bool {
	switch status {
	case AutoprogrammingBatchStatusPreparedV0, AutoprogrammingBatchStatusGoalsRunningV0, AutoprogrammingBatchStatusPendingIntegrationV0, AutoprogrammingBatchStatusPendingBatchGateV0, AutoprogrammingBatchStatusBatchGateRunningV0, AutoprogrammingBatchStatusBatchGatePassedV0, AutoprogrammingBatchStatusReworkPendingV0, AutoprogrammingBatchStatusPromotionPendingV0, AutoprogrammingBatchStatusClosedV0, AutoprogrammingBatchStatusBlockedV0:
		return true
	}
	return false
}
func autoprogrammingBatchTestReceiptStatusValidV0(status string) bool {
	return status == AutoprogrammingBatchTestReceiptPassedV0 || status == AutoprogrammingBatchTestReceiptFailedV0
}
func autoprogrammingBatchMemberIndexV0(members []AutoprogrammingBatchMemberV0, taskRef string) int {
	taskRef = autoprogrammingBatchRefV0(taskRef)
	for index, member := range members {
		if member.TaskRef == taskRef {
			return index
		}
	}
	return -1
}
func autoprogrammingBatchAllFocalClosedV0(members []AutoprogrammingBatchMemberV0) bool {
	for _, member := range members {
		if member.FocalStatus != AutoprogrammingBatchFocalClosedV0 {
			return false
		}
	}
	return len(members) > 0
}
func autoprogrammingBatchAllIntegratedV0(members []AutoprogrammingBatchMemberV0) bool {
	for _, member := range members {
		if member.IntegrationStatus != AutoprogrammingBatchIntegrationIntegratedV0 {
			return false
		}
	}
	return len(members) > 0
}
func autoprogrammingBatchHasTestHashV0(tests []AutoprogrammingBatchTestV0, want string) bool {
	for _, test := range tests {
		if AutoprogrammingBatchTestHashV0(test) == strings.TrimSpace(want) {
			return true
		}
	}
	return false
}
func autoprogrammingBatchClaimExistsV0(claims []AutoprogrammingBatchTestClaimV0, revision, testHash, claimRef string) bool {
	for _, claim := range claims {
		if claim.Revision == revision && claim.TestHash == testHash && claim.ClaimRef == claimRef {
			return true
		}
	}
	return false
}
func autoprogrammingBatchReceiptExistsV0(receipts []AutoprogrammingBatchTestReceiptV0, revision, testHash string) bool {
	for _, receipt := range receipts {
		if receipt.Revision == revision && receipt.TestHash == testHash {
			return true
		}
	}
	return false
}
func autoprogrammingBatchAllTestsPassedV0(batch AutoprogrammingBatchV0) bool {
	if batch.IntegratedRevision == "" {
		return false
	}
	for _, test := range batch.FrozenTests {
		found := false
		hash := AutoprogrammingBatchTestHashV0(test)
		for _, receipt := range batch.TestReceipts {
			if receipt.Revision == batch.IntegratedRevision && receipt.TestHash == hash && receipt.Status == AutoprogrammingBatchTestReceiptPassedV0 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return len(batch.FrozenTests) > 0
}
func autoprogrammingBatchActionHashV0(action string, values []string) string {
	action = strings.TrimSpace(action)
	if action == "" {
		return ""
	}
	normalized := make([]string, len(values))
	for index, value := range values {
		normalized[index] = strings.TrimSpace(value)
	}
	sum := sha256.Sum256([]byte("autoprogramming-batch-action.v0\n" + action + "\n" + strings.Join(normalized, "\n")))
	return hex.EncodeToString(sum[:])
}
func autoprogrammingBatchTransitionIssueV0(code, field, message string) []AutoprogrammingRequestIssueV0 {
	return []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0(code, field, message)}
}
func autoprogrammingBatchTransitionRejectedV0(batch AutoprogrammingBatchV0, code, field, message string) AutoprogrammingBatchTransitionResultV0 {
	return AutoprogrammingBatchTransitionResultV0{Batch: batch, Issues: autoprogrammingBatchTransitionIssueV0(code, field, message)}
}
