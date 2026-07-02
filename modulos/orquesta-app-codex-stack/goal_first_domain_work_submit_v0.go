package orquestaappcodexstack

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

const (
	goalFirstDomainWorkDescriptorRefV0 = "descriptor-ref-goal-first-domain-work"
	goalFirstDomainWorkAgentRefV0      = "agent-ref-goal-first-domain-work"
	goalFirstDomainWorkTaskPrefixV0    = "task-ref-goal-first-domain-work-"
)

func (stack StackV0) submitDomainWorkArtifactAfterGoalObservationV0(
	ctx context.Context,
	request orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0,
	result orquestaappdirectorservice.ObserveAppDirectorGoalResultV0,
) (bool, error) {
	if !stack.goalFirstDomainWorkSubmitPortsReadyV0(result) {
		return false, nil
	}
	state, err := stack.Ports.GoalStateStore.LoadGoalWorkStateV0(ctx, result.RunRef)
	if err != nil {
		return false, nil
	}
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return false, nil
	}
	spec := orquestagoal.NormalizeGoalWorkSpecV0(state.Spec)
	if !goalFirstDomainWorkSubmitReadyForSpecV0(spec, result) {
		return false, nil
	}
	record, ok, err := stack.goalFirstDomainWorkAppChangeRecordV0(ctx, result.RunRef)
	if err != nil || !ok {
		return false, err
	}
	contract, ok := goalFirstDomainWorkContractV0(spec, result.GoalResult)
	if !ok {
		return false, nil
	}
	fileRef, ok, err := stack.goalFirstDomainWorkArtifactFileRefV0(spec, contract.ArtifactType)
	if err != nil || !ok {
		return false, err
	}
	run, err := stack.goalFirstDomainWorkRunV0(ctx, result)
	if err != nil {
		return false, err
	}
	task := goalFirstDomainWorkTaskV0(spec, contract)
	observation := goalFirstDomainWorkObservationV0(spec, contract, result.GoalResult)
	descriptor := goalFirstDomainWorkDescriptorV0(stack, spec)
	ack := goalFirstDomainWorkAckV0(request, spec, task, contract, fileRef)
	input := DomainWorkArtifactSubmissionBuildInputV0{
		Run:         run,
		Task:        task,
		Record:      record,
		Descriptor:  descriptor,
		Ack:         ack,
		Observation: observation,
		OccurredAt:  request.OccurredAt,
	}
	submission, ok, err := stack.DomainDelivery.Builder.BuildDomainWorkArtifactSubmissionV0(ctx, input)
	if err != nil || !ok {
		return false, err
	}
	submission = enrichGoalFirstDomainWorkSubmissionLifecycleV0(submission, result.GoalResult)
	if submitted, err := stack.domainWorkSubmissionAlreadyRecordedV0(ctx, run, task, observation, submission); err != nil || submitted {
		return submitted, err
	}
	return stack.submitGoalFirstDomainWorkArtifactV0(ctx, request, run, task, observation, submission, result.GoalResult)
}

func (stack StackV0) goalFirstDomainWorkSubmitPortsReadyV0(
	result orquestaappdirectorservice.ObserveAppDirectorGoalResultV0,
) bool {
	return stack.DomainWork != nil &&
		stack.DomainDelivery.Enabled &&
		stack.DomainDelivery.Builder != nil &&
		stack.DomainDelivery.Ledger != nil &&
		stack.Ports.GoalStateStore != nil &&
		stack.Stores.AppChangeStore != nil &&
		strings.TrimSpace(stack.Codex.ProjectWorkDir) != "" &&
		result.GoalResult.Status == orquestagoal.GoalStatusCompleteV0 &&
		!result.Closure.Accepted
}

func goalFirstDomainWorkSubmitReadyForSpecV0(
	spec orquestagoal.GoalWorkSpecV0,
	result orquestaappdirectorservice.ObserveAppDirectorGoalResultV0,
) bool {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	if !spec.ClosurePolicy.RequireDomainReceipt {
		return false
	}
	if goalFirstDomainWorkClosureNeedsReceiptV0(result.Closure) {
		return true
	}
	if goalFirstDomainWorkClosureNeedsDomainTestEvidenceV0(spec, result.GoalResult, result.Closure) {
		return true
	}
	return goalFirstDomainWorkClosureNeedsArtifactReconciliationV0(spec, result.Closure)
}

func goalFirstDomainWorkClosureNeedsReceiptV0(
	closure orquestagoal.GoalClosureValidationV0,
) bool {
	for _, issue := range closure.Issues {
		switch issue.Code {
		case goalDomainReceiptLedgerAcceptedMissingIssueV0,
			goalDomainReceiptLedgerRequiredArtifactIssueV0,
			goalDomainReceiptLedgerIncompleteArtifactV0:
			return true
		case orquestagoal.ErrGoalClosureInvalidV0:
			if strings.TrimSpace(issue.Field) == "domain_receipt_refs" {
				return true
			}
		}
	}
	return false
}

func goalFirstDomainWorkClosureNeedsDomainTestEvidenceV0(
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
) bool {
	if !goalFirstDomainWorkClosureHasIssueFieldV0(closure, "required_tests") ||
		!goalFirstDomainWorkHasDomainOnlyRequiredTestsV0(spec.RequiredTests) {
		return false
	}
	return goalFirstDomainWorkCommandRequiredTestsPassedV0(spec.RequiredTests, result.RequiredTestResults)
}

func goalFirstDomainWorkClosureHasIssueFieldV0(
	closure orquestagoal.GoalClosureValidationV0,
	field string,
) bool {
	field = strings.TrimSpace(field)
	if field == "" {
		return false
	}
	for _, issue := range closure.Issues {
		if strings.TrimSpace(issue.Field) == field {
			return true
		}
	}
	return false
}

func goalFirstDomainWorkClosureNeedsArtifactReconciliationV0(
	spec orquestagoal.GoalWorkSpecV0,
	closure orquestagoal.GoalClosureValidationV0,
) bool {
	if !goalFirstDomainWorkClosureHasIssueFieldV0(closure, "artifact_refs") &&
		!goalFirstDomainWorkClosureHasIssueFieldV0(closure, "artifact_paths") {
		return false
	}
	_, ok := goalFirstDomainWorkSingleRequiredContractV0(spec)
	return ok
}

func goalFirstDomainWorkHasDomainOnlyRequiredTestsV0(
	required []orquestagoal.GoalRequiredTestV0,
) bool {
	for _, test := range required {
		if strings.TrimSpace(test.TestRef) != "" &&
			strings.TrimSpace(test.Command) == "" &&
			strings.TrimSpace(test.CommandRef) == "" {
			return true
		}
	}
	return false
}

func goalFirstDomainWorkCommandRequiredTestsPassedV0(
	required []orquestagoal.GoalRequiredTestV0,
	results []orquestagoal.GoalRequiredTestResultV0,
) bool {
	for _, test := range required {
		if strings.TrimSpace(test.Command) == "" && strings.TrimSpace(test.CommandRef) == "" {
			continue
		}
		if !goalFirstDomainWorkRequiredTestPassedV0(test.TestRef, results) {
			return false
		}
	}
	return true
}

func goalFirstDomainWorkRequiredTestPassedV0(
	testRef string,
	results []orquestagoal.GoalRequiredTestResultV0,
) bool {
	testRef = strings.TrimSpace(testRef)
	if testRef == "" {
		return false
	}
	for _, result := range results {
		if strings.TrimSpace(result.TestRef) != testRef {
			continue
		}
		status := strings.TrimSpace(result.Status)
		if status == orquestagoal.GoalStatusAcceptedV0 || status == "passed" {
			return true
		}
	}
	return false
}

func (stack StackV0) goalFirstDomainWorkAppChangeRecordV0(
	ctx context.Context,
	runRef string,
) (orquestaappchange.AppChangeRecordV0, bool, error) {
	records, err := stack.Stores.AppChangeStore.ListAppChangeRecordsV0(
		ctx,
		orquestaappchange.AppChangeRecordFilterV0{RunRef: strings.TrimSpace(runRef)},
	)
	if err != nil {
		return orquestaappchange.AppChangeRecordV0{}, false, err
	}
	for _, record := range records {
		if record.Request.ExternalWork != nil {
			return record, true, nil
		}
	}
	return orquestaappchange.AppChangeRecordV0{}, false, nil
}

func goalFirstDomainWorkContractV0(
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
) (orquestagoal.GoalArtifactContractV0, bool) {
	resultRefs := compactStringsV0(result.ArtifactRefs)
	var required []orquestagoal.GoalArtifactContractV0
	for _, contract := range spec.ArtifactContracts {
		contract.ArtifactRef = strings.TrimSpace(contract.ArtifactRef)
		contract.ArtifactType = strings.TrimSpace(contract.ArtifactType)
		if !contract.Required || contract.ArtifactRef == "" || contract.ArtifactType == "" {
			continue
		}
		required = append(required, contract)
		if len(resultRefs) == 0 || codexStackStringInSetV0(resultRefs, contract.ArtifactRef) {
			return contract, true
		}
	}
	if len(required) == 1 {
		return required[0], true
	}
	return orquestagoal.GoalArtifactContractV0{}, false
}

func goalFirstDomainWorkSingleRequiredContractV0(
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalArtifactContractV0, bool) {
	var found orquestagoal.GoalArtifactContractV0
	for _, contract := range spec.ArtifactContracts {
		contract.ArtifactRef = strings.TrimSpace(contract.ArtifactRef)
		contract.ArtifactType = strings.TrimSpace(contract.ArtifactType)
		if !contract.Required || contract.ArtifactRef == "" || contract.ArtifactType == "" {
			continue
		}
		if found.ArtifactRef != "" {
			return orquestagoal.GoalArtifactContractV0{}, false
		}
		found = contract
	}
	return found, found.ArtifactRef != ""
}

func (stack StackV0) goalFirstDomainWorkArtifactFileRefV0(
	spec orquestagoal.GoalWorkSpecV0,
	artifactType string,
) (string, bool, error) {
	artifactType = strings.TrimSpace(artifactType)
	if artifactType == "" {
		return "", false, nil
	}
	candidates := goalFirstDomainWorkArtifactCandidateNamesV0(artifactType)
	for _, scope := range spec.WriteSet {
		scopePath := filepath.ToSlash(strings.Trim(filepath.Clean(strings.TrimSpace(scope.Path)), "/"))
		if scopePath == "" || scopePath == "." {
			continue
		}
		for _, candidate := range candidates {
			fileRef := filepath.ToSlash(filepath.Join(scopePath, candidate))
			path, ok := safeDomainWorkDeliveryFilePathV0(stack.Codex.ProjectWorkDir, fileRef)
			if !ok {
				return "", false, nil
			}
			info, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return "", false, err
			}
			if !info.IsDir() && info.Mode().IsRegular() {
				return fileRef, true, nil
			}
		}
		fileRef, ok, err := stack.goalFirstDomainWorkArtifactFileRefInScopeV0(scopePath, artifactType)
		if err != nil || ok {
			return fileRef, ok, err
		}
	}
	return "", false, nil
}

func goalFirstDomainWorkArtifactCandidateNamesV0(artifactType string) []string {
	aliases := domainWorkDeliveryArtifactTypeAliasKeysV0(artifactType)
	candidates := make([]string, 0, len(aliases)*4+2)
	for _, alias := range aliases {
		candidates = append(candidates,
			alias+".json",
			alias+".md",
			"artifact_"+alias+".json",
			"artifact_"+alias+".md",
		)
	}
	candidates = append(candidates, "artifact.json", "artifact.md")
	return compactStringsV0(candidates)
}

func (stack StackV0) goalFirstDomainWorkArtifactFileRefInScopeV0(
	scopePath string,
	artifactType string,
) (string, bool, error) {
	scopeAbs, ok := safeDomainWorkDeliveryFilePathV0(stack.Codex.ProjectWorkDir, scopePath)
	if !ok {
		return "", false, nil
	}
	info, err := os.Stat(scopeAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if !info.IsDir() {
		if info.Mode().IsRegular() && domainWorkDeliveryArtifactFileRefMatchesV0(scopePath, artifactType) {
			return scopePath, true, nil
		}
		return "", false, nil
	}
	found := ""
	err = filepath.WalkDir(scopeAbs, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if found != "" || entry.IsDir() {
			return nil
		}
		if entry.Type() != 0 && !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(stack.Codex.ProjectWorkDir, path)
		if err != nil {
			return err
		}
		fileRef := filepath.ToSlash(rel)
		if domainWorkDeliveryArtifactFileRefMatchesV0(fileRef, artifactType) {
			found = fileRef
		}
		return nil
	})
	if err != nil {
		return "", false, err
	}
	return found, found != "", nil
}

func (stack StackV0) goalFirstDomainWorkRunV0(
	ctx context.Context,
	result orquestaappdirectorservice.ObserveAppDirectorGoalResultV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if strings.TrimSpace(result.Run.RunID) != "" {
		return result.Run, nil
	}
	if stack.Stores.RunStore == nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, fmt.Errorf("goal_first_domain_work_run_store_unavailable")
	}
	return stack.Stores.RunStore.LoadRunV0(ctx, result.RunRef)
}

func goalFirstDomainWorkTaskV0(
	spec orquestagoal.GoalWorkSpecV0,
	contract orquestagoal.GoalArtifactContractV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	suffix := codexStackOperationalClosureSafeRefV0(firstNonEmptyQueuedSourceV0(
		contract.ArtifactRef,
		spec.RunRef,
		spec.GoalRef,
	))
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             goalFirstDomainWorkTaskPrefixV0 + suffix,
		RunID:              spec.RunRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileDomainWorkV0,
		Title:              firstNonEmptyQueuedSourceV0(spec.Objective, "Entrega goal-first de dominio"),
		Summary:            "Entrega materializada desde resultado goal-first.",
		AcceptanceCriteria: append([]string(nil), spec.AcceptanceCriteria...),
		WriteSet:           goalFirstDomainWorkWriteSetV0(spec),
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
}

func goalFirstDomainWorkWriteSetV0(spec orquestagoal.GoalWorkSpecV0) []string {
	out := make([]string, 0, len(spec.WriteSet))
	for _, scope := range spec.WriteSet {
		out = append(out, scope.Path)
	}
	return compactStringsV0(out)
}

func goalFirstDomainWorkObservationV0(
	spec orquestagoal.GoalWorkSpecV0,
	contract orquestagoal.GoalArtifactContractV0,
	result orquestagoal.GoalWorkResultV0,
) orquestacionnucleoapp.AgentDeliveryObservationV0 {
	return orquestacionnucleoapp.AgentDeliveryObservationV0{
		DeliveryRef:  contract.ArtifactRef,
		ArtifactRef:  contract.ArtifactRef,
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskID:       goalFirstDomainWorkTaskV0(spec, contract).TaskID,
		AgentRef:     goalFirstDomainWorkAgentRefV0,
		Summary:      firstNonEmptyQueuedSourceV0(result.Summary, "Artefacto goal-first listo para entrega de dominio."),
		EvidenceRefs: compactStringsV0(append([]string{contract.ArtifactRef}, result.EvidenceRefs...)),
	}
}

func goalFirstDomainWorkDescriptorV0(
	stack StackV0,
	spec orquestagoal.GoalWorkSpecV0,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef:  goalFirstDomainWorkDescriptorRefV0 + "-" + codexStackOperationalClosureSafeRefV0(spec.RunRef),
		RunID:          spec.RunRef,
		AgentRef:       goalFirstDomainWorkAgentRefV0,
		ProjectWorkDir: strings.TrimSpace(stack.Codex.ProjectWorkDir),
	}
}

func goalFirstDomainWorkAckV0(
	request orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0,
	spec orquestagoal.GoalWorkSpecV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	contract orquestagoal.GoalArtifactContractV0,
	fileRef string,
) orquestaruntimecodex.CodexAgentAckV0 {
	return orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     firstNonEmptyQueuedSourceV0(request.RequestedBy, goalFirstDomainWorkAgentRefV0),
		CorrelationID: firstNonEmptyQueuedSourceV0(request.CorrelationID, spec.RequestRef, spec.RunRef),
		AckRef:        contract.ArtifactRef,
		TargetModule:  spec.ProjectRef,
		TaskRef:       task.TaskID,
		Status:        "completed",
		Files:         orquestaruntimecodex.EvidenceListV0{fileRef},
	}
}

func (stack StackV0) submitGoalFirstDomainWorkArtifactV0(
	ctx context.Context,
	request orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
	result orquestagoal.GoalWorkResultV0,
) (bool, error) {
	occurredAt := request.OccurredAt
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		ctx,
		domainWorkClaimedSubmissionRecordV0(run, task, observation, submission, occurredAt),
	); err != nil {
		return false, fmt.Errorf("domain_work_submit_claim_failed")
	}
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		ctx,
		domainWorkSubmittingSubmissionRecordV0(run, task, observation, submission, occurredAt),
	); err != nil {
		return false, fmt.Errorf("domain_work_submit_claim_failed")
	}
	toolResult, err := stack.DomainWork.Execute(ctx, orquestamcp.MCPDomainWorkToolInputV0{
		RequestID:          submission.RequestID,
		CorrelationID:      submission.CorrelationID,
		Action:             orquestamcp.MCPDomainWorkActionSubmitArtifactV0,
		ArtifactSubmission: submission,
	})
	if err != nil {
		if recordErr := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
			ctx,
			domainWorkRejectedSubmissionRecordWithIssueRefsV0(
				run,
				task,
				observation,
				submission,
				[]string{"domain-work-submit-execute-error"},
				occurredAt,
			),
		); recordErr != nil {
			return false, fmt.Errorf("domain_work_submit_recovery_required")
		}
		return false, nil
	}
	if toolResult.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 || toolResult.Receipt == nil {
		if recordErr := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
			ctx,
			domainWorkRejectedSubmissionRecordV0(run, task, observation, submission, toolResult, occurredAt),
		); recordErr != nil {
			return false, fmt.Errorf("domain_work_submit_recovery_required")
		}
		return false, nil
	}
	aliasedSubmission, aliasedResult := goalFirstDomainWorkAliasReceiptV0(submission, toolResult, result)
	if err := stack.recordDomainWorkSuccessfulSubmissionV0(
		ctx,
		run,
		task,
		observation,
		aliasedSubmission,
		aliasedResult,
		occurredAt,
	); err != nil {
		return false, err
	}
	return true, nil
}

func goalFirstDomainWorkAliasReceiptV0(
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
	toolResult orquestamcp.MCPDomainWorkToolResultV0,
	result orquestagoal.GoalWorkResultV0,
) (orquestadomainwork.DomainWorkArtifactSubmissionV0, orquestamcp.MCPDomainWorkToolResultV0) {
	receiptRef := goalFirstDomainWorkReceiptRefV0(result)
	if receiptRef == "" || toolResult.Receipt == nil {
		return submission, toolResult
	}
	actualReceipt := strings.TrimSpace(toolResult.Receipt.ReceiptRef)
	if actualReceipt != "" && actualReceipt != receiptRef {
		submission.ExternalRefs = append(submission.ExternalRefs, orquestadomainwork.DomainWorkExternalRefV0{
			Kind: "actual_domain_receipt_ref",
			Ref:  actualReceipt,
		})
		submission.EvidenceRefs = append(submission.EvidenceRefs, "domain-work-actual-receipt-"+safeDomainWorkEvidenceRefV0(actualReceipt))
	}
	receipt := *toolResult.Receipt
	receipt.ReceiptRef = receiptRef
	receipt.EvidenceRefs = compactStringsV0(append(
		receipt.EvidenceRefs,
		"domain-work-goal-receipt-alias-"+safeDomainWorkEvidenceRefV0(receiptRef),
	))
	toolResult.Receipt = &receipt
	return orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(submission), toolResult
}

func goalFirstDomainWorkReceiptRefV0(
	result orquestagoal.GoalWorkResultV0,
) string {
	refs := compactStringsV0(result.DomainReceiptRefs)
	if len(refs) == 0 {
		return ""
	}
	return refs[0]
}
