package orquestatoolcapability

import (
	"context"
	"crypto/sha256"
	"fmt"
)

func PrepareGeneratedAppToolAttachV0(
	ctx context.Context,
	request GeneratedAppToolAttachRequestV0,
	ports ToolCapabilityPortsV0,
) (GeneratedAppToolAttachPlanV0, error) {
	request.Operation = ToolOperationPrepareV0
	return buildGeneratedAppToolAttachPlanV0(ctx, request, ports, true)
}

func AttachGeneratedAppToolV0(
	ctx context.Context,
	request GeneratedAppToolAttachRequestV0,
	ports ToolCapabilityPortsV0,
) (ToolOperationReceiptV0, error) {
	request.Operation = ToolOperationAttachV0
	return executeGeneratedAppToolOperationV0(ctx, request, ports)
}

func UpgradeGeneratedAppToolV0(
	ctx context.Context,
	request GeneratedAppToolAttachRequestV0,
	ports ToolCapabilityPortsV0,
) (ToolOperationReceiptV0, error) {
	request.Operation = ToolOperationUpgradeV0
	return executeGeneratedAppToolOperationV0(ctx, request, ports)
}

func UninstallGeneratedAppToolV0(
	ctx context.Context,
	request GeneratedAppToolAttachRequestV0,
	ports ToolCapabilityPortsV0,
) (ToolOperationReceiptV0, error) {
	request.Operation = ToolOperationUninstallV0
	return executeGeneratedAppToolOperationV0(ctx, request, ports)
}

func StatusGeneratedAppToolV0(
	ctx context.Context,
	filter ToolOperationReceiptFilterV0,
	store ToolOperationReceiptStorePortV0,
) ([]ToolOperationReceiptV0, error) {
	ctx = toolCapabilityContextV0(ctx)
	if store == nil {
		return nil, fmt.Errorf("%s", ErrToolCapabilityReceiptStoreMissingV0)
	}
	if filter.AppRef == "" || filter.ToolRef == "" {
		return nil, fmt.Errorf("%s", ErrToolCapabilityStatusScopeRequiredV0)
	}
	return store.ListToolOperationReceiptsV0(ctx, filter)
}

func ReconcileGeneratedAppToolOperationV0(
	ctx context.Context,
	operationRef string,
	ports ToolCapabilityPortsV0,
) (ToolOperationReceiptV0, error) {
	return reconcileOrResumeGeneratedAppToolOperationV0(ctx, operationRef, ports, false)
}

func ResumeGeneratedAppToolOperationV0(
	ctx context.Context,
	operationRef string,
	ports ToolCapabilityPortsV0,
) (ToolOperationReceiptV0, error) {
	return reconcileOrResumeGeneratedAppToolOperationV0(ctx, operationRef, ports, true)
}

func buildGeneratedAppToolAttachPlanV0(
	ctx context.Context,
	request GeneratedAppToolAttachRequestV0,
	ports ToolCapabilityPortsV0,
	dryRun bool,
) (GeneratedAppToolAttachPlanV0, error) {
	ctx = toolCapabilityContextV0(ctx)
	plan := GeneratedAppToolAttachPlanV0{
		SchemaVersion:      ToolAttachPlanSchemaV0,
		OperationRef:       toolCapabilityOperationRefV0(request),
		RequestRef:         request.RequestRef,
		Operation:          request.Operation,
		IdempotencyKey:     request.IdempotencyKey,
		IntegrationMode:    request.IntegrationMode,
		ConfigRef:          request.ConfigRef,
		WriteSet:           append([]string(nil), request.WriteSet...),
		PreviousReceiptRef: request.PreviousReceiptRef,
		DryRun:             dryRun,
	}
	plan.Issues = append(plan.Issues, ValidateGeneratedAppToolAttachRequestV0(request)...)
	for field, unavailable := range map[string]bool{
		"binding_authority": ports.BindingAuthority == nil,
		"registry":          ports.Registry == nil,
		"snapshot_resolver": ports.SnapshotResolver == nil,
		"validator":         ports.Validator == nil,
	} {
		if unavailable {
			plan.Issues = append(plan.Issues, toolCapabilityIssueV0(ErrToolCapabilityPortUnavailableV0, field))
		}
	}
	if len(plan.Issues) > 0 {
		return plan, nil
	}

	binding, err := ports.BindingAuthority.ResolveGeneratedAppToolBindingV0(ctx, request.AppRef, request.BindingRef)
	if err != nil {
		return plan, err
	}
	manifest, err := ports.Registry.GetCapabilityManifestV0(ctx, request.ManifestRef)
	if err != nil {
		return plan, err
	}
	resolved, err := ports.SnapshotResolver.ResolveVerifiedToolBundleSnapshotV0(ctx, manifest, request.BundleRef)
	if err != nil {
		return plan, err
	}
	plan.Binding = binding
	plan.Manifest = manifest
	plan.Bundle = resolved.Bundle
	plan.Snapshot = resolved.Snapshot

	if binding.AppRef != request.AppRef || binding.BindingRef != request.BindingRef {
		plan.Issues = append(plan.Issues, toolCapabilityIssueV0(ErrToolCapabilityAuthorityMismatchV0, "binding"))
	}
	if manifest.ManifestRef != request.ManifestRef {
		plan.Issues = append(plan.Issues, toolCapabilityIssueV0(ErrToolCapabilityAuthorityMismatchV0, "manifest_ref"))
	}
	if resolved.Bundle.BundleRef != request.BundleRef {
		plan.Issues = append(plan.Issues, toolCapabilityIssueV0(ErrToolCapabilityAuthorityMismatchV0, "bundle_ref"))
	}
	plan.Issues = append(plan.Issues, ValidateGeneratedAppToolBindingV0(binding)...)
	plan.Issues = append(plan.Issues, ValidateCapabilityManifestV0(manifest)...)
	plan.Issues = append(plan.Issues, ValidateToolBundleV0(manifest, resolved.Bundle)...)
	plan.Issues = append(plan.Issues, ValidateToolBundleSnapshotV0(manifest, resolved.Bundle, resolved.Snapshot)...)
	plan.Issues = append(plan.Issues, validateToolAttachCompatibilityV0(plan)...)
	plan.TargetRef = toolCapabilityTargetRefV0(binding.AppRef, manifest.ToolRef, binding.BindingRef)
	plan.PlanFingerprint = CalculateGeneratedAppToolPlanFingerprintV0(plan)
	plan.PlanRef = ToolAttachPlanRefV0(plan.PlanFingerprint)
	plan.Issues = append(plan.Issues, ports.Validator.ValidateGeneratedAppToolAttachPlanV0(ctx, plan)...)
	plan.EffectAuthorized = !hasToolCapabilityIssueV0(plan.Issues, ErrToolCapabilityEffectUnauthorizedV0)
	return plan, nil
}

func executeGeneratedAppToolOperationV0(
	ctx context.Context,
	request GeneratedAppToolAttachRequestV0,
	ports ToolCapabilityPortsV0,
) (ToolOperationReceiptV0, error) {
	ctx = toolCapabilityContextV0(ctx)
	if ports.ReceiptStore == nil {
		return ToolOperationReceiptV0{}, fmt.Errorf("%s", ErrToolCapabilityReceiptStoreMissingV0)
	}
	plan, err := buildGeneratedAppToolAttachPlanV0(ctx, request, ports, false)
	if err != nil {
		return ToolOperationReceiptV0{}, err
	}
	if plan.PlanFingerprint == "" || plan.PlanRef == "" || plan.TargetRef == "" {
		return ToolOperationReceiptV0{}, fmt.Errorf("%s", ErrToolCapabilityPlanUnresolvedV0)
	}
	var previous *ToolOperationRecordV0
	if len(plan.Issues) == 0 && request.Operation == ToolOperationUpgradeV0 {
		loaded, loadIssues, loadErr := loadPreviousUpgradeRecordV0(ctx, plan, ports.ReceiptStore)
		if loadErr != nil {
			return ToolOperationReceiptV0{}, loadErr
		}
		plan.Issues = append(plan.Issues, loadIssues...)
		if len(loadIssues) == 0 {
			previous = &loaded
		}
	}
	if len(plan.Issues) > 0 || !plan.EffectAuthorized {
		record := newToolOperationRecordV0(plan, ToolReceiptStatusBlockedV0)
		claimed, _, claimErr := claimToolOperationV0(ctx, ports.ReceiptStore, record)
		return claimed.Receipt, claimErr
	}
	if ports.Installer == nil {
		return ToolOperationReceiptV0{}, fmt.Errorf("%s", ErrToolCapabilityPortUnavailableV0)
	}
	record := newToolOperationRecordV0(plan, ToolReceiptStatusClaimedV0)
	claimed, acquired, err := claimToolOperationV0(ctx, ports.ReceiptStore, record)
	if err != nil || !acquired {
		return claimed.Receipt, err
	}
	return executeClaimedToolOperationV0(ctx, claimed, previous, ports)
}

func claimToolOperationV0(
	ctx context.Context,
	store ToolOperationReceiptStorePortV0,
	record ToolOperationRecordV0,
) (ToolOperationRecordV0, bool, error) {
	claimed, acquired, err := store.ClaimToolOperationV0(ctx, record)
	if err != nil {
		return claimed, false, err
	}
	if !acquired && claimed.Plan.PlanFingerprint != record.Plan.PlanFingerprint {
		return claimed, false, fmt.Errorf("%s", ErrToolCapabilityIdempotencyConflictV0)
	}
	return claimed, acquired, nil
}

func executeClaimedToolOperationV0(
	ctx context.Context,
	record ToolOperationRecordV0,
	previous *ToolOperationRecordV0,
	ports ToolCapabilityPortsV0,
) (ToolOperationReceiptV0, error) {
	var (
		result ToolMaterializationResultV0
		err    error
	)
	if record.Plan.Operation == ToolOperationUninstallV0 {
		result, err = ports.Installer.UninstallToolBundleV0(ctx, record.Plan)
	} else {
		result, err = ports.Installer.MaterializeToolBundleV0(ctx, record.Plan)
	}
	if err != nil {
		if record.Plan.Operation == ToolOperationUpgradeV0 && previous != nil {
			return rollbackFailedUpgradeV0(ctx, record, *previous, ports, err)
		}
		return transitionToolOperationRecoveryV0(ctx, record, ports.ReceiptStore, ErrToolCapabilityMaterializationFailedV0, err)
	}
	if issues := validateToolCapabilityRefsV0(result.EvidenceRefs, "materialization.evidence_refs", false); len(issues) > 0 {
		return transitionToolOperationRecoveryV0(ctx, record, ports.ReceiptStore, ErrToolCapabilityMaterializationFailedV0, fmt.Errorf("%s", issues[0].Code))
	}
	completed := successfulToolOperationReceiptV0(record.Receipt, result.EvidenceRefs)
	return completeToolOperationV0(ctx, ports.ReceiptStore, record, completed)
}

func rollbackFailedUpgradeV0(
	ctx context.Context,
	record ToolOperationRecordV0,
	previous ToolOperationRecordV0,
	ports ToolCapabilityPortsV0,
	materializeErr error,
) (ToolOperationReceiptV0, error) {
	rollback, rollbackErr := ports.Installer.RollbackToolBundleV0(ctx, ToolRollbackRequestV0{
		FailedPlan: record.Plan, PreviousPlan: previous.Plan, PreviousReceipt: previous.Receipt,
	})
	if rollbackErr != nil {
		return transitionToolOperationRecoveryV0(ctx, record, ports.ReceiptStore, ErrToolCapabilityRollbackFailedV0, rollbackErr)
	}
	if issues := validateToolCapabilityRefsV0(rollback.EvidenceRefs, "rollback.evidence_refs", false); len(issues) > 0 {
		return transitionToolOperationRecoveryV0(ctx, record, ports.ReceiptStore, ErrToolCapabilityRollbackFailedV0, fmt.Errorf("%s", issues[0].Code))
	}
	completed := successfulToolOperationReceiptV0(record.Receipt, rollback.EvidenceRefs)
	completed.Status = ToolReceiptStatusRolledBackV0
	completed.Issues = []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityUpgradeFailedV0, "upgrade")}
	receipt, err := completeToolOperationV0(ctx, ports.ReceiptStore, record, completed)
	if err != nil {
		return receipt, err
	}
	_ = materializeErr
	return receipt, nil
}

func reconcileOrResumeGeneratedAppToolOperationV0(
	ctx context.Context,
	operationRef string,
	ports ToolCapabilityPortsV0,
	resume bool,
) (ToolOperationReceiptV0, error) {
	ctx = toolCapabilityContextV0(ctx)
	if ports.ReceiptStore == nil {
		return ToolOperationReceiptV0{}, fmt.Errorf("%s", ErrToolCapabilityReceiptStoreMissingV0)
	}
	record, found, err := ports.ReceiptStore.GetToolOperationRecordV0(ctx, operationRef)
	if err != nil {
		return ToolOperationReceiptV0{}, err
	}
	if !found {
		return ToolOperationReceiptV0{}, fmt.Errorf("%s", ErrToolCapabilityOperationNotFoundV0)
	}
	if ToolReceiptStatusIsTargetTerminalV0(record.Receipt.Status) {
		return record.Receipt, nil
	}
	if record.Receipt.Status != ToolReceiptStatusClaimedV0 &&
		record.Receipt.Status != ToolReceiptStatusResumingV0 &&
		record.Receipt.Status != ToolReceiptStatusRecoveryRequiredV0 {
		return record.Receipt, fmt.Errorf("%s", ErrToolCapabilityOperationStateInvalidV0)
	}
	if issues := ValidateToolOperationRecordV0(record); len(issues) > 0 {
		return transitionToolOperationRecoveryV0(ctx, record, ports.ReceiptStore, ErrToolCapabilityPlanFingerprintMismatchV0, fmt.Errorf("%s", issues[0].Code))
	}
	if ports.Installer == nil {
		return transitionToolOperationRecoveryV0(ctx, record, ports.ReceiptStore, ErrToolCapabilityPortUnavailableV0, fmt.Errorf("%s", ErrToolCapabilityPortUnavailableV0))
	}
	reconciliation, err := ports.Installer.ReconcileToolBundleV0(ctx, record.Plan)
	if err != nil {
		return transitionToolOperationRecoveryV0(ctx, record, ports.ReceiptStore, ErrToolCapabilityReconciliationUnknownV0, err)
	}
	if issues := validateToolCapabilityRefsV0(reconciliation.EvidenceRefs, "reconciliation.evidence_refs", false); len(issues) > 0 {
		return transitionToolOperationRecoveryV0(ctx, record, ports.ReceiptStore, ErrToolCapabilityReconciliationUnknownV0, fmt.Errorf("%s", issues[0].Code))
	}
	switch reconciliation.Disposition {
	case ToolReconcileDispositionAppliedV0:
		completed := successfulToolOperationReceiptV0(record.Receipt, reconciliation.EvidenceRefs)
		return completeToolOperationV0(ctx, ports.ReceiptStore, record, completed)
	case ToolReconcileDispositionUnknownV0:
		return transitionToolOperationRecoveryV0(ctx, record, ports.ReceiptStore, ErrToolCapabilityReconciliationUnknownV0, fmt.Errorf("%s", ErrToolCapabilityReconciliationUnknownV0))
	case ToolReconcileDispositionNotAppliedV0:
		if !resume {
			return record.Receipt, nil
		}
	default:
		return transitionToolOperationRecoveryV0(ctx, record, ports.ReceiptStore, ErrToolCapabilityReconciliationUnknownV0, fmt.Errorf("%s", ErrToolCapabilityReconciliationUnknownV0))
	}

	resuming := record.Receipt
	resuming.Status = ToolReceiptStatusResumingV0
	resumingRecord, swapped, casErr := ports.ReceiptStore.CompareAndSwapToolOperationV0(ctx, record.Plan.OperationRef, record.Receipt.StateVersion, withNextToolReceiptVersionV0(record.Receipt, resuming))
	if casErr != nil {
		return record.Receipt, casErr
	}
	if !swapped {
		return resumingRecord.Receipt, fmt.Errorf("%s", ErrToolCapabilityReceiptCASConflictV0)
	}
	var previous *ToolOperationRecordV0
	if resumingRecord.Plan.Operation == ToolOperationUpgradeV0 {
		loaded, issues, loadErr := loadPreviousUpgradeRecordV0(ctx, resumingRecord.Plan, ports.ReceiptStore)
		if loadErr != nil {
			return transitionToolOperationRecoveryV0(ctx, resumingRecord, ports.ReceiptStore, ErrToolCapabilityPreviousReceiptInvalidV0, loadErr)
		}
		if len(issues) > 0 {
			return transitionToolOperationRecoveryV0(ctx, resumingRecord, ports.ReceiptStore, issues[0].Code, fmt.Errorf("%s", issues[0].Code))
		}
		previous = &loaded
	}
	return executeClaimedToolOperationV0(ctx, resumingRecord, previous, ports)
}

func loadPreviousUpgradeRecordV0(
	ctx context.Context,
	plan GeneratedAppToolAttachPlanV0,
	store ToolOperationReceiptStorePortV0,
) (ToolOperationRecordV0, []ToolCapabilityIssueV0, error) {
	if plan.PreviousReceiptRef == "" {
		return ToolOperationRecordV0{}, []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityPreviousReceiptInvalidV0, "previous_receipt_ref")}, nil
	}
	previous, found, err := store.GetToolOperationRecordByReceiptRefV0(ctx, plan.PreviousReceiptRef)
	if err != nil {
		return ToolOperationRecordV0{}, nil, err
	}
	if !found || previous.Receipt.ReceiptRef != plan.PreviousReceiptRef {
		return ToolOperationRecordV0{}, []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityPreviousReceiptInvalidV0, "previous_receipt_ref")}, nil
	}
	if previous.Receipt.Status != ToolReceiptStatusAttachedV0 && previous.Receipt.Status != ToolReceiptStatusUpgradedV0 {
		return previous, []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityPreviousReceiptInvalidV0, "previous_receipt.status")}, nil
	}
	if len(ValidateToolOperationRecordV0(previous)) > 0 ||
		previous.Receipt.AppRef != plan.Binding.AppRef || previous.Receipt.BindingRef != plan.Binding.BindingRef ||
		previous.Receipt.ToolRef != plan.Manifest.ToolRef || previous.Receipt.CapabilityRef != plan.Manifest.CapabilityRef {
		return previous, []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityPreviousReceiptInvalidV0, "previous_receipt")}, nil
	}
	if !containsToolCapabilityStringV0(plan.Bundle.Compatibility.UpgradeFromRefs, previous.Receipt.BundleRef) {
		return previous, []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityUpgradeIncompatibleV0, "compatibility.upgrade_from_refs")}, nil
	}
	return previous, nil, nil
}

func newToolOperationRecordV0(plan GeneratedAppToolAttachPlanV0, status string) ToolOperationRecordV0 {
	recoveryRef := toolCapabilityRecoveryRefV0(plan.OperationRef)
	if ToolReceiptStatusIsTargetTerminalV0(status) {
		recoveryRef = ""
	}
	record := ToolOperationRecordV0{
		SchemaVersion: ToolOperationRecordSchemaV0,
		Plan:          plan,
		Receipt: ToolOperationReceiptV0{
			SchemaVersion: ToolOperationReceiptSchemaV0,
			ReceiptRef:    toolCapabilityReceiptRefV0(plan.OperationRef), OperationRef: plan.OperationRef,
			TargetRef: plan.TargetRef, PlanRef: plan.PlanRef, PlanFingerprint: plan.PlanFingerprint,
			Operation: plan.Operation, Status: status, StateVersion: 1,
			IdempotencyKey: plan.IdempotencyKey,
			AppRef:         plan.Binding.AppRef, BindingRef: plan.Binding.BindingRef,
			ToolRef: plan.Manifest.ToolRef, CapabilityRef: plan.Manifest.CapabilityRef,
			ManifestRef: plan.Manifest.ManifestRef, BundleRef: plan.Bundle.BundleRef,
			ContentHash: plan.Bundle.ContentHash, ConfigRef: plan.ConfigRef,
			SnapshotRef: plan.Snapshot.SnapshotRef, SnapshotHandleRef: plan.Snapshot.HandleRef,
			PreviousReceiptRef: plan.PreviousReceiptRef, RecoveryRef: recoveryRef,
		},
	}
	if status == ToolReceiptStatusBlockedV0 {
		record.Receipt.Issues = append([]ToolCapabilityIssueV0(nil), plan.Issues...)
	}
	return record
}

func successfulToolOperationReceiptV0(receipt ToolOperationReceiptV0, evidenceRefs []string) ToolOperationReceiptV0 {
	status := ToolReceiptStatusAttachedV0
	switch receipt.Operation {
	case ToolOperationUpgradeV0:
		status = ToolReceiptStatusUpgradedV0
	case ToolOperationUninstallV0:
		status = ToolReceiptStatusUninstalledV0
	}
	receipt.Status = status
	receipt.RecoveryRef = ""
	receipt.EvidenceRefs = append([]string(nil), evidenceRefs...)
	return receipt
}

func transitionToolOperationRecoveryV0(
	ctx context.Context,
	record ToolOperationRecordV0,
	store ToolOperationReceiptStorePortV0,
	code string,
	cause error,
) (ToolOperationReceiptV0, error) {
	recovery := record.Receipt
	recovery.Status = ToolReceiptStatusRecoveryRequiredV0
	recovery.RecoveryRef = toolCapabilityRecoveryRefV0(record.Plan.OperationRef)
	recovery.Issues = []ToolCapabilityIssueV0{toolCapabilityIssueV0(code, "operation")}
	receipt, err := completeToolOperationV0(ctx, store, record, recovery)
	if err != nil {
		return receipt, err
	}
	return receipt, cause
}

func completeToolOperationV0(
	ctx context.Context,
	store ToolOperationReceiptStorePortV0,
	record ToolOperationRecordV0,
	completed ToolOperationReceiptV0,
) (ToolOperationReceiptV0, error) {
	completed = withNextToolReceiptVersionV0(record.Receipt, completed)
	current, swapped, err := store.CompareAndSwapToolOperationV0(ctx, record.Plan.OperationRef, record.Receipt.StateVersion, completed)
	if err != nil {
		return record.Receipt, err
	}
	if !swapped {
		return current.Receipt, fmt.Errorf("%s", ErrToolCapabilityReceiptCASConflictV0)
	}
	return current.Receipt, nil
}

func withNextToolReceiptVersionV0(current, next ToolOperationReceiptV0) ToolOperationReceiptV0 {
	next.StateVersion = current.StateVersion + 1
	return next
}

func hasToolCapabilityIssueV0(issues []ToolCapabilityIssueV0, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func toolCapabilityContextV0(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func toolCapabilityOperationRefV0(request GeneratedAppToolAttachRequestV0) string {
	return "tool-operation-" + toolCapabilityIdentityHashV0(request.AppRef, request.BindingRef, request.Operation, request.IdempotencyKey)
}

func toolCapabilityTargetRefV0(appRef, toolRef, bindingRef string) string {
	return "tool-target-" + toolCapabilityIdentityHashV0(appRef, toolRef, bindingRef)
}

func toolCapabilityReceiptRefV0(operationRef string) string {
	return operationRef + "-receipt"
}

func toolCapabilityRecoveryRefV0(operationRef string) string {
	return operationRef + "-recovery"
}

func toolCapabilityIdentityHashV0(parts ...string) string {
	digest := sha256.New()
	for _, part := range parts {
		fmt.Fprintf(digest, "%d:%s", len(part), part)
	}
	return fmt.Sprintf("%x", digest.Sum(nil))
}
