package orquestaopesdirector

import (
	"context"
	"strings"

	orquestadocumentplanexpander "orquesta/modulos/orquesta-document-plan-expander"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
	orquestaopestopicregistry "orquesta/modulos/orquesta-opes-topic-registry"
)

const defaultOPESCausalProducerMaxActionsV0 = 20

const opesTopicRegistryAppliedEvidenceV0 = "evidence-ref-opes-topic-registry-applied"

func ProduceOPESCausalJobsV0(
	ctx context.Context,
	request OPESCausalProducerRequestV0,
	ports OPESCausalProducerPortsV0,
) (OPESCausalProducerResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeProducerRequestV0(request)
	result := OPESCausalProducerResultV0{
		SchemaVersion: OPESCausalProducerResultSchemaV0,
		Status:        OPESCausalProducerStatusInvalidV0,
		EvidenceRefs:  compactStringsV0(request.EvidenceRefs),
	}
	if ports.ArtifactSource == nil {
		result.Issues = append(result.Issues, issueV0(ErrOPESCausalArtifactSourceRequiredV0, "artifact_source"))
		return result, nil
	}
	if ports.JobCreator == nil {
		result.Issues = append(result.Issues, issueV0(ErrOPESCausalJobCreatorRequiredV0, "job_creator"))
		return result, nil
	}
	records, err := ports.ArtifactSource.ListOPESCausalArtifactRecordsV0(
		ctx,
		OPESCausalArtifactRecordFilterV0{DomainRef: request.DomainRef, CorrelationID: request.CorrelationID},
	)
	if err != nil {
		return result, err
	}
	actions := 0
	for _, record := range records {
		if actions >= request.MaxActions {
			break
		}
		record = normalizeArtifactRecordV0(record, request.DomainRef)
		if !artifactRecordMatchesProducerScopeV0(record, request) ||
			record.DomainRef != request.DomainRef ||
			record.JobRef == "" ||
			record.ArtifactRef == "" {
			continue
		}
		result.ProcessedRefs = compactStringsV0(append(result.ProcessedRefs, artifactRecordSourceRefV0(record)))
		requests, recordIssues := causalRequestsForArtifactRecordV0(record)
		result.Issues = append(result.Issues, recordIssues...)
		for _, jobRequest := range requests {
			if actions >= request.MaxActions {
				break
			}
			jobRequest = orquestadomainwork.NormalizeDomainWorkJobRequestV0(jobRequest)
			if existing, ok, err := existingCausalJobV0(ctx, ports.JobRecords, jobRequest); err != nil {
				return result, err
			} else if ok {
				result.SkippedRefs = compactStringsV0(append(result.SkippedRefs, existing.Job.JobRef))
				if shouldRetryExistingTopicRegistryUpdateV0(existing, jobRequest, ports.TopicRegistryUpdater) {
					var attempted bool
					result, _, attempted, _ = applyTopicRegistryUpdateIfConfiguredV0(ctx, result, ports.TopicRegistryUpdater, jobRequest)
					if attempted {
						actions++
					}
				}
				continue
			}
			result.RequestedJobs = append(result.RequestedJobs, jobRequest)
			var attemptedTopicRegistryUpdate bool
			var appliedTopicRegistryUpdate bool
			result, jobRequest, attemptedTopicRegistryUpdate, appliedTopicRegistryUpdate = applyTopicRegistryUpdateIfConfiguredV0(
				ctx,
				result,
				ports.TopicRegistryUpdater,
				jobRequest,
			)
			if attemptedTopicRegistryUpdate && !appliedTopicRegistryUpdate {
				actions++
				continue
			}
			job, err := ports.JobCreator.CreateDomainWorkJobV0(ctx, jobRequest)
			if err != nil {
				return result, err
			}
			if job.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 {
				result.Issues = append(result.Issues, append(job.Issues, issueV0(ErrOPESCausalJobRejectedV0, "created_jobs.status"))...)
				continue
			}
			result.CreatedJobs = append(result.CreatedJobs, job)
			result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, job.EvidenceRefs...))
			actions++
		}
	}
	result.Status = OPESCausalProducerStatusCompletedV0
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-opes-causal-producer"))
	return result, nil
}

func applyTopicRegistryUpdateIfConfiguredV0(
	ctx context.Context,
	result OPESCausalProducerResultV0,
	updater OPESCausalTopicRegistryUpdaterPortV0,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (OPESCausalProducerResultV0, orquestadomainwork.DomainWorkJobRequestV0, bool, bool) {
	if updater == nil || request.WorkKind != opesTopicRegistryUpdateWorkKindV0 {
		return result, request, false, true
	}
	update, err := updater.ApplyOPESCausalTopicRegistryUpdateV0(ctx, request)
	result.TopicRegistryUpdates = append(result.TopicRegistryUpdates, update)
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, update.EvidenceRefs...))
	for _, issue := range update.Issues {
		result.Issues = append(result.Issues, issueV0(issue.Code, "topic_registry."+issue.Field))
	}
	if err != nil || update.Status != orquestaopestopicregistry.TopicRegistryUpdateStatusAppliedV0 {
		result.Issues = append(result.Issues, issueV0(ErrOPESCausalTopicRegistryFailedV0, "topic_registry.command"))
		return result, request, true, false
	}
	request.EvidenceRefs = compactStringsV0(append(
		append([]string(nil), request.EvidenceRefs...),
		append(update.EvidenceRefs, opesTopicRegistryAppliedEvidenceV0)...,
	))
	return result, request, true, true
}

func normalizeProducerRequestV0(request OPESCausalProducerRequestV0) OPESCausalProducerRequestV0 {
	request.DomainRef = strings.TrimSpace(request.DomainRef)
	if request.DomainRef == "" {
		request.DomainRef = OPESCausalProducerDefaultDomainRefV0
	}
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	if request.MaxActions <= 0 {
		request.MaxActions = defaultOPESCausalProducerMaxActionsV0
	}
	request.EvidenceRefs = compactStringsV0(request.EvidenceRefs)
	return request
}

func causalRequestsForArtifactRecordV0(
	record OPESCausalArtifactRecordV0,
) ([]orquestadomainwork.DomainWorkJobRequestV0, []orquestadomainwork.DomainWorkIssueV0) {
	switch strings.TrimSpace(record.Status) {
	case "accepted":
		return acceptedArtifactRequestsV0(record)
	case "rejected":
		if rejectedArtifactTransientRetryPendingV0(record.IssueRefs) {
			return nil, []orquestadomainwork.DomainWorkIssueV0{
				issueV0(ErrOPESCausalRejectedTransientRetryV0, "artifact.issue_refs"),
			}
		}
		return []orquestadomainwork.DomainWorkJobRequestV0{rejectedArtifactCorrectionRequestV0(record)}, nil
	default:
		return nil, nil
	}
}

func artifactRecordMatchesProducerScopeV0(
	record OPESCausalArtifactRecordV0,
	request OPESCausalProducerRequestV0,
) bool {
	if request.CorrelationID == "" {
		return true
	}
	return strings.TrimSpace(record.CorrelationID) == request.CorrelationID
}

func rejectedArtifactTransientRetryPendingV0(issueRefs []string) bool {
	for _, issue := range issueRefs {
		switch strings.TrimSpace(issue) {
		case "domain-work-submit-execute-error",
			"domain_work_port_no_disponible",
			"opes_http_request_failed",
			"opes_http_timeout",
			"opes_http_cancelled",
			"retry_budget_exhausted",
			"opes_http_status_429",
			"opes_http_status_502",
			"opes_http_status_503",
			"opes_http_status_504":
			return true
		}
	}
	return false
}

func shouldRetryExistingTopicRegistryUpdateV0(
	existing orquestadomainwork.DomainWorkJobRecordV0,
	request orquestadomainwork.DomainWorkJobRequestV0,
	updater OPESCausalTopicRegistryUpdaterPortV0,
) bool {
	if updater == nil || request.WorkKind != opesTopicRegistryUpdateWorkKindV0 {
		return false
	}
	return !stringsInSetV0(existing.Job.EvidenceRefs, opesTopicRegistryAppliedEvidenceV0) &&
		!stringsInSetV0(existing.Request.EvidenceRefs, opesTopicRegistryAppliedEvidenceV0)
}

func acceptedArtifactRequestsV0(
	record OPESCausalArtifactRecordV0,
) ([]orquestadomainwork.DomainWorkJobRequestV0, []orquestadomainwork.DomainWorkIssueV0) {
	var requests []orquestadomainwork.DomainWorkJobRequestV0
	var issues []orquestadomainwork.DomainWorkIssueV0
	if record.ArtifactType == orquestadomainwork.DomainDocumentPlanArtifactTypeV0 {
		plan, ok, planIssues := documentPlanFromArtifactRecordV0(record)
		if !ok {
			issues = append(issues, planIssues...)
		} else {
			expansion := orquestadocumentplanexpander.ExpandDomainDocumentPlanV0(
				orquestadocumentplanexpander.DomainDocumentPlanExpansionRequestV0{
					Plan:          plan,
					CorrelationID: firstNonEmptyV0(record.CorrelationID, record.IdempotencyKey),
					RequestedBy:   OPESCausalProducerDefaultRequestedByV0,
					InterfaceRefs: []string{"orquesta.domain_work.v0", "opes.rest.v0"},
					InputFields:   provenanceFieldsV0(record, "", orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0("draft_content_block")),
					ExternalRefs:  provenanceExternalRefsV0(record, ""),
					EvidenceRefs:  provenanceEvidenceRefsV0(record),
				},
			)
			if len(expansion.Issues) > 0 {
				issues = append(issues, expansion.Issues...)
			} else {
				requests = append(requests, expansion.Jobs...)
			}
			if missing := missingTemarioSequenceWorkKindsV0(plan); len(missing) > 0 {
				requests = append(requests, planSequenceReworkRequestV0(record, plan, missing))
			}
		}
	}
	if shouldRequestTopicRegistryUpdateV0(record) {
		requests = append(requests, topicRegistryUpdateRequestV0(record))
	}
	for _, followupRef := range followupRefsForRecordV0(record) {
		requests = append(requests, followupRequestV0(record, followupRef))
	}
	return requests, issues
}

func existingCausalJobV0(
	ctx context.Context,
	source orquestadomainwork.DomainWorkJobRecordSourcePortV0,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (orquestadomainwork.DomainWorkJobRecordV0, bool, error) {
	if source == nil {
		return orquestadomainwork.DomainWorkJobRecordV0{}, false, nil
	}
	records, err := source.ListDomainWorkJobRecordsV0(ctx, orquestadomainwork.DomainWorkJobRecordFilterV0{
		DomainRef:      request.DomainRef,
		IdempotencyKey: request.IdempotencyKey,
		Limit:          1,
	})
	if err != nil {
		return orquestadomainwork.DomainWorkJobRecordV0{}, false, err
	}
	if len(records) == 0 {
		return orquestadomainwork.DomainWorkJobRecordV0{}, false, nil
	}
	return records[0], true, nil
}

func issueV0(code string, field string) orquestadomainwork.DomainWorkIssueV0 {
	return orquestadomainwork.DomainWorkIssueV0{Code: strings.TrimSpace(code), Field: strings.TrimSpace(field)}
}

func missingTemarioSequenceWorkKindsV0(plan orquestadomainwork.DomainDocumentPlanV0) []string {
	if plan.WorkKind != orquestadomainwork.DomainWorkKindPlanSyllabusV0 {
		return nil
	}
	present := map[string]bool{}
	for _, section := range plan.Sections {
		present[section.WorkKind] = true
	}
	for _, visual := range plan.Visuals {
		present[visual.WorkKind] = true
	}
	for _, review := range plan.ReviewSteps {
		present[review.WorkKind] = true
	}
	var missing []string
	for _, workKind := range orquestaopesbridge.OPESFullTemarioJobTypeSequenceV0() {
		if workKind == orquestadomainwork.DomainWorkKindPlanSyllabusV0 {
			continue
		}
		if !present[workKind] {
			missing = append(missing, workKind)
		}
	}
	return missing
}
