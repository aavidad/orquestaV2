package orquestastatefileoutbox

import (
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

type ledgerIssueV0 struct {
	Code    string
	Field   string
	Message string
}

func directorIssueV0(
	code string,
	field string,
	message string,
) orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0 {
	return orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{
		Code:    code,
		Field:   field,
		Message: message,
	}
}

func dispatchIssueV0(code string, field string, message string) orquestaoutboxdispatch.DispatchIssueV0 {
	return orquestaoutboxdispatch.DispatchIssueV0{
		Code:    code,
		Field:   field,
		Message: message,
	}
}

func directorIssuesFromLedgerV0(
	issues []ledgerIssueV0,
) []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0 {
	if len(issues) == 0 {
		return nil
	}
	out := make([]orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, directorIssueV0(issue.Code, issue.Field, issue.Message))
	}
	return out
}

func directorPersistenceIssueV0() []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0 {
	return []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{
		directorIssueV0(errPersistenceUnavailableV0, "outbox_ledger", "persistencia no disponible"),
	}
}

func dispatchPersistenceIssueV0() []orquestaoutboxdispatch.DispatchIssueV0 {
	return []orquestaoutboxdispatch.DispatchIssueV0{
		dispatchIssueV0(errPersistenceUnavailableV0, "outbox_ledger", "persistencia no disponible"),
	}
}
