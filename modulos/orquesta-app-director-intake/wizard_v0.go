package orquestaappdirectorintake

import (
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func AdvanceAppDirectorIntakeWizardV0(
	request AppDirectorIntakeWizardRequestV0,
) (AppDirectorIntakeWizardResultV0, error) {
	request = normalizeAppDirectorIntakeWizardRequestV0(request)
	draft, issues := applyAppDirectorIntakeWizardAnswersV0(request.Draft, request.Answers)
	request.Draft = draft
	request = normalizeAppDirectorIntakeWizardRequestV0(request)
	if len(issues) > 0 {
		return invalidAppDirectorIntakeWizardResultV0(request, issues), nil
	}
	if issue := validateAppDirectorIntakeWizardEnvelopeV0(request); issue != nil {
		return invalidAppDirectorIntakeWizardResultV0(request, []AppDirectorIntakeWizardIssueV0{*issue}), nil
	}
	if field := nextAppDirectorIntakeWizardFieldV0(request.Draft); field != "" {
		question, err := appDirectorIntakeWizardQuestionV0(request, field)
		if err != nil {
			return AppDirectorIntakeWizardResultV0{}, err
		}
		return AppDirectorIntakeWizardResultV0{
			SchemaVersion: AppDirectorIntakeWizardResultSchemaVersionV0,
			Status:        AppDirectorIntakeWizardStatusNeedsInputV0,
			Draft:         request.Draft,
			NextQuestion:  &question,
			EvidenceRefs:  appDirectorIntakeWizardEvidenceRefsV0(request),
		}, nil
	}
	if factoryIssues := orquestafactory.ValidateAppSpecRequestV0(request.Draft); len(factoryIssues) > 0 {
		return invalidAppDirectorIntakeWizardResultV0(
			request,
			appDirectorIntakeWizardIssuesFromFactoryV0(factoryIssues),
		), nil
	}
	now, err := time.Parse(time.RFC3339, request.OccurredAt)
	if err != nil {
		return invalidAppDirectorIntakeWizardResultV0(request, []AppDirectorIntakeWizardIssueV0{{
			Code:  ErrAppDirectorIntakeWizardOccurredAtInvalidV0,
			Field: "occurred_at",
		}}), nil
	}
	spec, factoryIssues := orquestafactory.SolicitarNuevaAppV0(request.Draft, now)
	if len(factoryIssues) > 0 {
		return invalidAppDirectorIntakeWizardResultV0(
			request,
			appDirectorIntakeWizardIssuesFromFactoryV0(factoryIssues),
		), nil
	}
	prepared, err := PrepareAppDirectorIntakeV0(PrepareAppDirectorIntakeRequestV0{
		RunRef:        request.RunRef,
		ProjectRef:    request.ProjectRef,
		OccurredAt:    request.OccurredAt,
		CorrelationID: request.CorrelationID,
		RequestedBy:   request.RequestedBy,
		AppSpec:       spec,
	})
	if err != nil {
		return AppDirectorIntakeWizardResultV0{}, err
	}
	return AppDirectorIntakeWizardResultV0{
		SchemaVersion: AppDirectorIntakeWizardResultSchemaVersionV0,
		Status:        AppDirectorIntakeWizardStatusReadyV0,
		Draft:         request.Draft,
		AppSpec:       &spec,
		Prepared:      &prepared,
		EvidenceRefs:  appDirectorIntakeWizardEvidenceRefsV0(request),
	}, nil
}

func validateAppDirectorIntakeWizardEnvelopeV0(
	request AppDirectorIntakeWizardRequestV0,
) *AppDirectorIntakeWizardIssueV0 {
	if request.RequestRef == "" {
		return &AppDirectorIntakeWizardIssueV0{
			Code:  ErrAppDirectorIntakeWizardRequestRefV0,
			Field: "request_ref",
		}
	}
	if request.OccurredAt == "" {
		return &AppDirectorIntakeWizardIssueV0{
			Code:  ErrAppDirectorIntakeWizardOccurredAtV0,
			Field: "occurred_at",
		}
	}
	if _, err := time.Parse(time.RFC3339, request.OccurredAt); err != nil {
		return &AppDirectorIntakeWizardIssueV0{
			Code:  ErrAppDirectorIntakeWizardOccurredAtInvalidV0,
			Field: "occurred_at",
		}
	}
	return nil
}

func invalidAppDirectorIntakeWizardResultV0(
	request AppDirectorIntakeWizardRequestV0,
	issues []AppDirectorIntakeWizardIssueV0,
) AppDirectorIntakeWizardResultV0 {
	return AppDirectorIntakeWizardResultV0{
		SchemaVersion: AppDirectorIntakeWizardResultSchemaVersionV0,
		Status:        AppDirectorIntakeWizardStatusInvalidV0,
		Draft:         request.Draft,
		EvidenceRefs:  appDirectorIntakeWizardEvidenceRefsV0(request),
		Issues:        append([]AppDirectorIntakeWizardIssueV0(nil), issues...),
	}
}

func appDirectorIntakeWizardIssuesFromFactoryV0(
	issues []orquestafactory.ValidationIssue,
) []AppDirectorIntakeWizardIssueV0 {
	out := make([]AppDirectorIntakeWizardIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, AppDirectorIntakeWizardIssueV0{
			Code:  issue.Code,
			Field: issue.Field,
		})
	}
	return out
}

func appDirectorIntakeWizardEvidenceRefsV0(
	request AppDirectorIntakeWizardRequestV0,
) []string {
	return compactDirectorIntakeStringsV0([]string{
		"evidence-ref-app-director-intake-wizard-v0",
		"evidence-ref-" + safeDirectorIntakeRefPartV0(request.RequestRef),
	})
}
