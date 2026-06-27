package main

import (
	"context"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaopesdirector "orquesta/modulos/orquesta-opes-director"
	orquestaopestopicregistry "orquesta/modulos/orquesta-opes-topic-registry"
)

type serverOPESCausalProducerV0 struct {
	artifacts            orquestaopesdirector.OPESCausalArtifactRecordSourcePortV0
	jobCreator           orquestadomainwork.DomainWorkJobCreatorPortV0
	jobRecords           orquestadomainwork.DomainWorkJobRecordSourcePortV0
	topicRegistryUpdater orquestaopesdirector.OPESCausalTopicRegistryUpdaterPortV0
}

func newServerOPESCausalProducerV0(
	stack *orquestaappcodexstack.StackV0,
) *serverOPESCausalProducerV0 {
	if stack == nil || stack.DomainWork == nil || !stack.DomainDelivery.Enabled || stack.DomainDelivery.Ledger == nil {
		return nil
	}
	reader, ok := stack.DomainDelivery.Ledger.(orquestaappcodexstack.DomainWorkArtifactSubmissionRecordReaderPortV0)
	if !ok || reader == nil {
		return nil
	}
	return &serverOPESCausalProducerV0{
		artifacts:            serverOPESCausalArtifactSourceV0{reader: reader},
		jobCreator:           serverDomainWorkMCPJobCreatorV0{executor: stack.DomainWork},
		jobRecords:           serverDomainWorkJobRecordSourceV0(stack.DomainWork),
		topicRegistryUpdater: serverOPESTopicRegistryUpdaterFromEnvV0(),
	}
}

func (producer *serverOPESCausalProducerV0) ProduceV0(
	ctx context.Context,
	command orquestaopesdirector.OPESCausalProducerRequestV0,
) (orquestaopesdirector.OPESCausalProducerResultV0, error) {
	if producer == nil {
		return orquestaopesdirector.OPESCausalProducerResultV0{
			SchemaVersion: orquestaopesdirector.OPESCausalProducerResultSchemaV0,
			Status:        orquestaopesdirector.OPESCausalProducerStatusCompletedV0,
		}, nil
	}
	return orquestaopesdirector.ProduceOPESCausalJobsV0(
		ctx,
		command,
		orquestaopesdirector.OPESCausalProducerPortsV0{
			ArtifactSource:       producer.artifacts,
			JobCreator:           producer.jobCreator,
			JobRecords:           producer.jobRecords,
			TopicRegistryUpdater: producer.topicRegistryUpdater,
		},
	)
}

type serverOPESTopicRegistryUpdaterV0 struct {
	config opesTopicRegistryConfigV0
	runner orquestaopestopicregistry.TopicRegistryCommandRunnerPortV0
}

func serverOPESTopicRegistryUpdaterFromEnvV0() orquestaopesdirector.OPESCausalTopicRegistryUpdaterPortV0 {
	config := opesTopicRegistryConfigFromEnvV0()
	if !config.Enabled {
		return nil
	}
	return serverOPESTopicRegistryUpdaterV0{
		config: config,
		runner: orquestaopestopicregistry.ExecTopicRegistryCommandRunnerV0{},
	}
}

func (updater serverOPESTopicRegistryUpdaterV0) ApplyOPESCausalTopicRegistryUpdateV0(
	ctx context.Context,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (orquestaopestopicregistry.TopicRegistryUpdateResultV0, error) {
	topicRequest := orquestaopestopicregistry.TopicRegistryUpdateRequestFromDomainWorkJobV0(
		request,
		updater.config.ToolPath,
		updater.config.AgentID,
	)
	topicRequest.Force = updater.config.Force
	return orquestaopestopicregistry.ApplyTopicRegistryUpdateV0(ctx, topicRequest, updater.runner)
}

type serverOPESCausalArtifactSourceV0 struct {
	reader orquestaappcodexstack.DomainWorkArtifactSubmissionRecordReaderPortV0
}

func (source serverOPESCausalArtifactSourceV0) ListOPESCausalArtifactRecordsV0(
	ctx context.Context,
	filter orquestaopesdirector.OPESCausalArtifactRecordFilterV0,
) ([]orquestaopesdirector.OPESCausalArtifactRecordV0, error) {
	records, err := source.reader.ListDomainWorkArtifactSubmissionsV0(
		ctx,
		orquestaappcodexstack.DomainWorkArtifactSubmissionRecordFilterV0{},
	)
	if err != nil {
		return nil, err
	}
	domainRef := strings.TrimSpace(filter.DomainRef)
	correlationID := strings.TrimSpace(filter.CorrelationID)
	out := make([]orquestaopesdirector.OPESCausalArtifactRecordV0, 0, len(records))
	for _, record := range records {
		if domainRef != "" && strings.TrimSpace(record.DomainRef) != domainRef {
			continue
		}
		if correlationID != "" && strings.TrimSpace(record.CorrelationID) != correlationID {
			continue
		}
		status := strings.TrimSpace(record.Status)
		if status != orquestaappcodexstack.DomainWorkArtifactSubmissionStatusAcceptedV0 &&
			status != orquestaappcodexstack.DomainWorkArtifactSubmissionStatusRejectedV0 {
			continue
		}
		out = append(out, opesCausalArtifactRecordFromSubmissionRecordV0(record))
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func opesCausalArtifactRecordFromSubmissionRecordV0(
	record orquestaappcodexstack.DomainWorkArtifactSubmissionRecordV0,
) orquestaopesdirector.OPESCausalArtifactRecordV0 {
	return orquestaopesdirector.OPESCausalArtifactRecordV0{
		IdempotencyKey: record.IdempotencyKey,
		Status:         record.Status,
		RunRef:         record.RunRef,
		TaskRef:        record.TaskRef,
		DeliveryRef:    record.DeliveryRef,
		CorrelationID:  record.CorrelationID,
		DomainRef:      record.DomainRef,
		JobRef:         record.JobRef,
		ArtifactRef:    record.ArtifactRef,
		ArtifactType:   record.ArtifactType,
		Summary:        record.Summary,
		PayloadFields:  append([]orquestadomainwork.DomainWorkFieldV0(nil), record.PayloadFields...),
		PayloadRefs:    append([]string(nil), record.PayloadRefs...),
		ExternalRefs:   append([]orquestadomainwork.DomainWorkExternalRefV0(nil), record.ExternalRefs...),
		CompleteJob:    record.CompleteJob,
		ReceiptRef:     record.ReceiptRef,
		EvidenceRefs:   append([]string(nil), record.EvidenceRefs...),
		IssueRefs:      append([]string(nil), record.IssueRefs...),
		RecordedAt:     record.RecordedAt,
	}
}

type serverDomainWorkMCPJobCreatorV0 struct {
	executor orquestamcp.MCPDomainWorkExecutorPortV0
}

func (creator serverDomainWorkMCPJobCreatorV0) CreateDomainWorkJobV0(
	ctx context.Context,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (orquestadomainwork.DomainWorkJobV0, error) {
	if creator.executor == nil {
		return invalidServerDomainWorkJobV0(request, []orquestadomainwork.DomainWorkIssueV0{{
			Code:  orquestamcp.MCPDomainWorkCreatorUnavailableV0,
			Field: "domain_work",
		}}), nil
	}
	result, err := creator.executor.Execute(ctx, orquestamcp.MCPDomainWorkToolInputV0{
		RequestID:     request.RequestID,
		CorrelationID: request.CorrelationID,
		Action:        orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest:    request,
	})
	if err != nil {
		return orquestadomainwork.DomainWorkJobV0{}, err
	}
	if result.Job != nil {
		return *result.Job, nil
	}
	return invalidServerDomainWorkJobV0(request, serverDomainWorkIssuesFromMCPV0(result.Errores)), nil
}

func serverDomainWorkJobRecordSourceV0(
	executor orquestamcp.MCPDomainWorkExecutorPortV0,
) orquestadomainwork.DomainWorkJobRecordSourcePortV0 {
	switch concrete := executor.(type) {
	case orquestamcp.MCPDomainWorkToolExecutorV0:
		if records, ok := concrete.JobCreator.(orquestadomainwork.DomainWorkJobRecordSourcePortV0); ok {
			return records
		}
	case *orquestamcp.MCPDomainWorkToolExecutorV0:
		if concrete != nil {
			if records, ok := concrete.JobCreator.(orquestadomainwork.DomainWorkJobRecordSourcePortV0); ok {
				return records
			}
		}
	}
	return nil
}

func invalidServerDomainWorkJobV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
	issues []orquestadomainwork.DomainWorkIssueV0,
) orquestadomainwork.DomainWorkJobV0 {
	request = orquestadomainwork.NormalizeDomainWorkJobRequestV0(request)
	return orquestadomainwork.DomainWorkJobV0{
		SchemaVersion:  orquestadomainwork.DomainWorkJobSchemaV0,
		Status:         orquestadomainwork.DomainWorkStatusInvalidV0,
		DomainRef:      request.DomainRef,
		WorkKind:       request.WorkKind,
		CorrelationID:  request.CorrelationID,
		IdempotencyKey: request.IdempotencyKey,
		ExternalRefs:   append([]orquestadomainwork.DomainWorkExternalRefV0(nil), request.ExternalRefs...),
		EvidenceRefs:   append([]string(nil), request.EvidenceRefs...),
		Issues:         issues,
	}
}

func serverDomainWorkIssuesFromMCPV0(
	issues []orquestamcp.MCPValidationIssueV0,
) []orquestadomainwork.DomainWorkIssueV0 {
	out := make([]orquestadomainwork.DomainWorkIssueV0, 0, len(issues))
	for _, issue := range issues {
		code := strings.TrimSpace(issue.Code)
		if code == "" {
			code = orquestamcp.MCPDomainWorkDefaultErrorMessageV0
		}
		out = append(out, orquestadomainwork.DomainWorkIssueV0{
			Code:  code,
			Field: strings.TrimSpace(issue.Field),
		})
	}
	if len(out) == 0 {
		out = append(out, orquestadomainwork.DomainWorkIssueV0{
			Code:  orquestamcp.MCPDomainWorkDefaultErrorMessageV0,
			Field: "domain_work",
		})
	}
	return out
}
