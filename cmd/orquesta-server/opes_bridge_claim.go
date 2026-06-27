package main

import (
	"context"
	"strings"
	"unicode"

	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

const (
	externalBridgeClaimFailedCodeV0      = "external_bridge_claim_failed"
	externalBridgeRecoveryRequiredCodeV0 = "external_bridge_recovery_required"
)

func opesBridgeSkipRecordedInputV0(
	ctx context.Context,
	ledger externalBridgeInputLedgerV0,
	job orquestaopesconnector.ExternalJobV0,
	summary *opesDrainSummaryV0,
	result *opesDrainJobResultV0,
) bool {
	if ledger == nil {
		return false
	}
	entry, ok, err := ledger.LookupExternalBridgeInputV0(ctx, opesBridgeInputLedgerKeyV0(job.ID))
	if err != nil {
		summary.Skipped++
		result.Status = "ledger_error"
		summary.Results = append(summary.Results, *result)
		appendOPESDrainErrorV0(summary, job.ID, externalBridgeClaimFailedCodeV0)
		return true
	}
	if !ok {
		return false
	}
	if entry.Status == externalBridgeInputStatusSubmitFailedV0 {
		result.RunRef = entry.RunRef
		result.ChangeRef = entry.ChangeRef
		return false
	}
	return opesBridgeRecordHitV0(entry, summary, result)
}

func opesBridgeClaimRunRequestV0(
	ctx context.Context,
	ledger externalBridgeInputLedgerV0,
	job orquestaopesconnector.ExternalJobV0,
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
	summary *opesDrainSummaryV0,
	result *opesDrainJobResultV0,
) (orquestaexternalworkrun.StartExternalWorkRunRequestV0, bool) {
	request.RunRef = opesBridgeExternalWorkRunRefV0(request)
	result.RunRef = request.RunRef
	if ledger == nil {
		return request, false
	}
	entry, acquired, err := externalBridgeClaimInputV0(ctx, ledger, externalBridgeInputLedgerEntryV0{
		Key:            opesBridgeInputLedgerKeyV0(job.ID),
		ExternalSystem: opesBridgeExternalSystemV0,
		ExternalJobRef: strings.TrimSpace(job.ID),
		Status:         externalBridgeInputStatusClaimedV0,
		ClaimRef:       opesBridgeInputClaimRefV0(job.ID),
		CorrelationID:  request.CorrelationID,
		IdempotencyKey: request.AppChangeRequest.ChangeRef,
		RunRef:         request.RunRef,
		ChangeRef:      request.AppChangeRequest.ChangeRef,
	})
	if err != nil {
		summary.Skipped++
		result.Status = "claim_error"
		summary.Results = append(summary.Results, *result)
		appendOPESDrainErrorV0(summary, job.ID, externalBridgeClaimFailedCodeV0)
		return request, true
	}
	if !acquired {
		return request, opesBridgeRecordHitV0(entry, summary, result)
	}
	return request, false
}

func opesBridgeRecordSubmitFailedV0(
	ctx context.Context,
	ledger externalBridgeInputLedgerV0,
	job orquestaopesconnector.ExternalJobV0,
	result opesDrainJobResultV0,
	errorCode string,
) error {
	if ledger == nil {
		return nil
	}
	return ledger.UpsertExternalBridgeInputV0(ctx, externalBridgeInputLedgerEntryV0{
		Key:            opesBridgeInputLedgerKeyV0(job.ID),
		ExternalSystem: opesBridgeExternalSystemV0,
		ExternalJobRef: strings.TrimSpace(job.ID),
		Status:         externalBridgeInputStatusSubmitFailedV0,
		ClaimRef:       opesBridgeInputClaimRefV0(job.ID),
		CorrelationID:  strings.TrimSpace(job.CorrelationID),
		IdempotencyKey: strings.TrimSpace(result.ChangeRef),
		RunRef:         strings.TrimSpace(result.RunRef),
		ChangeRef:      strings.TrimSpace(result.ChangeRef),
		LastError:      strings.TrimSpace(errorCode),
	})
}

func opesBridgeRecordHitV0(
	entry externalBridgeInputLedgerEntryV0,
	summary *opesDrainSummaryV0,
	result *opesDrainJobResultV0,
) bool {
	result.RunRef = entry.RunRef
	result.ChangeRef = entry.ChangeRef
	opesBridgeApplyRunMetadataV0(result, externalBridgeInputRunMetadataFromEntryV0(entry))
	switch entry.Status {
	case externalBridgeInputStatusSubmittedV0:
		summary.AlreadySubmitted++
		result.Status = "already_submitted"
	case externalBridgeInputStatusClaimedV0:
		summary.Claimed++
		summary.Skipped++
		result.Status = "claimed"
		appendOPESDrainErrorV0(summary, entry.ExternalJobRef, externalBridgeRecoveryRequiredCodeV0)
	default:
		summary.Skipped++
		result.Status = "ledger_error"
		appendOPESDrainErrorV0(summary, entry.ExternalJobRef, externalBridgeClaimFailedCodeV0)
	}
	summary.Results = append(summary.Results, *result)
	return true
}

func appendOPESDrainErrorV0(summary *opesDrainSummaryV0, jobRef string, code string) {
	summary.Errors = append(summary.Errors, opesDrainPublicErrorV0{
		JobRef: strings.TrimSpace(jobRef),
		Code:   strings.TrimSpace(code),
	})
}

func opesBridgeInputClaimRefV0(jobRef string) string {
	return "claim-ref-external-bridge-input-" + opesBridgeCompactRunPartV0(jobRef)
}

func opesBridgeExternalWorkRunRefV0(
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
) string {
	if strings.TrimSpace(request.RunRef) != "" {
		return strings.TrimSpace(request.RunRef)
	}
	parts := []string{request.ProjectRef}
	if request.AppChangeRequest.ExternalWork != nil {
		parts = append(parts, request.AppChangeRequest.ExternalWork.JobRef)
	}
	parts = append(parts, request.AppChangeRequest.ChangeRef)
	basis := strings.Join(compactOPESBridgeRunPartsV0(parts), "-")
	if basis == "" {
		basis = firstExternalBridgeInputValueV0(
			request.AppChangeRequest.RequestID,
			request.RequestID,
			"request",
		)
	}
	return "run-external-work-" + opesBridgeCompactRunPartV0(basis)
}

func compactOPESBridgeRunPartsV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func opesBridgeCompactRunPartV0(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '_', r == '.', r == '-':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "external-work"
	}
	return out
}
