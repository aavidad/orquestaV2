package orquestaappcodexstack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type DomainWorkRequiredTestRunnerV0 struct {
	Inner            orquestacionnucleoapp.RequiredTestRunnerPortV0
	Policy           orquestadomainwork.DomainWorkRequiredTestPolicyPortV0
	AppChangeStore   orquestaappchange.AppChangeRecordSourcePortV0
	SubmissionLedger DomainWorkArtifactSubmissionRecordReaderPortV0
	EvidenceReader   orquestacionnucleoapp.RequiredTestEvidenceReaderPortV0
	EvidenceWriter   orquestacionnucleoapp.RequiredTestEvidenceWriterPortV0
}

func (runner DomainWorkRequiredTestRunnerV0) RunRequiredTestsV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
) (orquestacionnucleoapp.RequiredTestExecutionResultV0, error) {
	plan, ok, err := runner.domainWorkPlanForTaskV0(ctx, request)
	if err != nil || !ok {
		return runner.runInnerRequiredTestsV0(ctx, request, err)
	}
	return runner.runDomainWorkRequiredTestsV0(ctx, request, plan)
}

func (runner DomainWorkRequiredTestRunnerV0) runInnerRequiredTestsV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
	err error,
) (orquestacionnucleoapp.RequiredTestExecutionResultV0, error) {
	if err != nil || runner.Inner == nil {
		return orquestacionnucleoapp.RequiredTestExecutionResultV0{}, err
	}
	return runner.Inner.RunRequiredTestsV0(ctx, request)
}

func (runner DomainWorkRequiredTestRunnerV0) domainWorkPlanForTaskV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
) (orquestadomainwork.DomainWorkRequiredTestPlanV0, bool, error) {
	if runner.Policy == nil || runner.AppChangeStore == nil {
		return orquestadomainwork.DomainWorkRequiredTestPlanV0{}, false, nil
	}
	records, err := runner.AppChangeStore.ListAppChangeRecordsV0(
		ctx,
		orquestaappchange.AppChangeRecordFilterV0{RunRef: request.RunRef},
	)
	if err != nil {
		return orquestadomainwork.DomainWorkRequiredTestPlanV0{}, false, err
	}
	for _, record := range records {
		if orquestaappchangedirectorsource.AppChangeTaskRefV0(record.Request.ChangeRef) != request.TaskRef {
			continue
		}
		job, ok := orquestaappchange.DomainWorkJobRequestFromAppChangeV0(record.Request)
		if !ok {
			return orquestadomainwork.DomainWorkRequiredTestPlanV0{}, false, nil
		}
		plan, err := runner.Policy.BuildDomainWorkRequiredTestPlanV0(ctx, job)
		if err != nil {
			return orquestadomainwork.DomainWorkRequiredTestPlanV0{}, false, err
		}
		return orquestadomainwork.NormalizeDomainWorkRequiredTestPlanV0(plan), true, nil
	}
	return orquestadomainwork.DomainWorkRequiredTestPlanV0{}, false, nil
}

func (runner DomainWorkRequiredTestRunnerV0) runDomainWorkRequiredTestsV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
	plan orquestadomainwork.DomainWorkRequiredTestPlanV0,
) (orquestacionnucleoapp.RequiredTestExecutionResultV0, error) {
	record, ok, err := runner.domainWorkSubmissionRecordV0(ctx, request)
	if err != nil || !ok || runner.EvidenceWriter == nil {
		return orquestacionnucleoapp.RequiredTestExecutionResultV0{}, err
	}
	out := orquestacionnucleoapp.RequiredTestExecutionResultV0{}
	allowed := domainWorkRequiredTestPlanRefSetV0(plan)
	for _, command := range compactCodexStackStringsV0(request.TestCommands) {
		if !allowed[command] {
			continue
		}
		evidence, err := runner.domainWorkRequiredTestEvidenceV0(request, command, record)
		if err != nil {
			return out, err
		}
		if err := runner.EvidenceWriter.SaveRequiredTestEvidenceV0(ctx, evidence); err != nil {
			return out, err
		}
		out = domainWorkRequiredTestResultWithEvidenceV0(out, evidence)
	}
	return out, nil
}

func (runner DomainWorkRequiredTestRunnerV0) domainWorkSubmissionRecordV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
) (DomainWorkArtifactSubmissionRecordV0, bool, error) {
	if runner.SubmissionLedger == nil {
		return DomainWorkArtifactSubmissionRecordV0{}, false, nil
	}
	records, err := runner.SubmissionLedger.ListDomainWorkArtifactSubmissionsV0(
		ctx,
		DomainWorkArtifactSubmissionRecordFilterV0{
			RunRef:      request.RunRef,
			TaskRef:     request.TaskRef,
			DeliveryRef: request.DeliveryRef,
		},
	)
	if err != nil || len(records) == 0 {
		return DomainWorkArtifactSubmissionRecordV0{}, false, err
	}
	return normalizeDomainWorkArtifactSubmissionRecordV0(records[0]), true, nil
}

func (runner DomainWorkRequiredTestRunnerV0) domainWorkRequiredTestEvidenceV0(
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
	command string,
	record DomainWorkArtifactSubmissionRecordV0,
) (orquestacionnucleoapp.RequiredTestEvidenceV0, error) {
	ref := domainWorkRequiredTestEvidenceRefV0(request, command)
	status := orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0
	if record.Status == DomainWorkArtifactSubmissionStatusRejectedV0 {
		status = orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0
	}
	evidenceRefs := domainWorkRequiredTestEvidenceRefsV0(request, record, status)
	return orquestacionnucleoapp.NewRequiredTestEvidenceV0(orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       ref,
		RunRef:            request.RunRef,
		TaskRef:           request.TaskRef,
		TestCommand:       command,
		Status:            status,
		DeliveryRef:       request.DeliveryRef,
		ReviewRequestID:   request.ReviewRequestID,
		ReviewResultRef:   request.ReviewResultRef,
		AcceptedReviewRef: request.AcceptedReviewRef,
		OccurredAt:        request.OccurredAt,
		EvidenceRefs:      evidenceRefs,
	})
}

func domainWorkRequiredTestEvidenceRefsV0(
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
	record DomainWorkArtifactSubmissionRecordV0,
	status orquestacionnucleoapp.RequiredTestEvidenceStatusV0,
) []string {
	evidenceRefs := compactCodexStackStringsV0(append(
		append([]string(nil), request.EvidenceRefs...),
		append(append([]string(nil), record.EvidenceRefs...), record.ReceiptRef)...,
	))
	if status == orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0 {
		evidenceRefs = compactCodexStackStringsV0(append(evidenceRefs, record.IssueRefs...))
	}
	return evidenceRefs
}

func domainWorkRequiredTestResultWithEvidenceV0(
	result orquestacionnucleoapp.RequiredTestExecutionResultV0,
	evidence orquestacionnucleoapp.RequiredTestEvidenceV0,
) orquestacionnucleoapp.RequiredTestExecutionResultV0 {
	result.EvidenceRefs = compactCodexStackStringsV0(append(result.EvidenceRefs, evidence.EvidenceRef))
	if evidence.Status == orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 {
		result.PassedEvidenceRefs = compactCodexStackStringsV0(append(result.PassedEvidenceRefs, evidence.EvidenceRef))
	} else {
		result.FailedEvidenceRefs = compactCodexStackStringsV0(append(result.FailedEvidenceRefs, evidence.EvidenceRef))
	}
	return result
}

func domainWorkRequiredTestPlanRefSetV0(
	plan orquestadomainwork.DomainWorkRequiredTestPlanV0,
) map[string]bool {
	out := map[string]bool{}
	for _, ref := range domainWorkRequiredTestRefsFromPlanV0(plan) {
		out[ref] = true
	}
	return out
}

func domainWorkRequiredTestEvidenceRefV0(
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
	command string,
) string {
	hash := sha256.Sum256([]byte(strings.Join([]string{
		request.RunRef,
		request.TaskRef,
		command,
		request.DeliveryRef,
		request.ReviewRequestID,
		request.ReviewResultRef,
		request.AcceptedReviewRef,
	}, "\x00")))
	return "domain-test-evidence-ref-v0-" + hex.EncodeToString(hash[:])[:24]
}
