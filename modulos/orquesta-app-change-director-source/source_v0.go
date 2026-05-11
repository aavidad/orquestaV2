package orquestaappchangedirectorsource

import (
	"context"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
)

type AppChangeDirectorDecisionSourceV0 struct {
	Store orquestaappchange.AppChangeRecordSourcePortV0
}

var _ orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0 = AppChangeDirectorDecisionSourceV0{}

func (source AppChangeDirectorDecisionSourceV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if source.Store == nil {
		return nil, AppChangeDirectorSourceIssueV0{Field: "store"}
	}
	runRef := strings.TrimSpace(request.Run.RunID)
	if runRef == "" {
		return nil, AppChangeDirectorSourceIssueV0{Field: "run_ref"}
	}
	records, err := source.Store.ListAppChangeRecordsV0(
		ctx,
		orquestaappchange.AppChangeRecordFilterV0{RunRef: runRef},
	)
	if err != nil {
		return nil, err
	}
	return source.decisionsFromRecordsV0(request, records)
}

func (source AppChangeDirectorDecisionSourceV0) decisionsFromRecordsV0(
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
	records []orquestaappchange.AppChangeRecordV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	out := make([]orquestadirectoragent.DirectorAgentDecisionV0, 0, len(records)*8)
	for _, record := range records {
		refs := appChangeRefsV0(record.Request.ChangeRef)
		if !appChangeReadyForAutoPlanV0(record.Request) ||
			!appChangeQuestionReadyV0(request.Run, refs.QuestionRef) {
			continue
		}
		decisions := buildAppChangeDecisionsV0(request.Run, record)
		if err := validateAppChangeDirectorDecisionsV0(decisions); err != nil {
			return nil, err
		}
		out = append(out, decisions...)
	}
	return out, nil
}

func validateAppChangeDirectorDecisionsV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) error {
	for _, decision := range decisions {
		if issues := orquestadirectoragent.ValidateDirectorAgentDecisionV0(decision); len(issues) > 0 {
			return AppChangeDirectorSourceIssueV0{Field: "decision." + issues[0].Field}
		}
	}
	return nil
}
