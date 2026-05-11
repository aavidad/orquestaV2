package orquestadirectorcycle

import (
	"errors"

	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
)

func resultWithCycleStepRunnerErrorV0(
	result DirectorCycleStepResultV0,
	input DirectorCycleStepInputV0,
	runnerResult orquestadirectorrunner.DirectorCycleResultV0,
	err error,
) (DirectorCycleStepResultV0, error) {
	issues := []DirectorCycleStepIssueV0{
		cycleStepIssueV0(ErrDirectorCycleStepRunnerV0, "runner", "runner fallo"),
	}
	issues = append(issues, cycleStepIssuesFromRunnerV0(runnerResult.Issues)...)
	var runnerErr orquestadirectorrunner.DirectorCycleErrorV0
	if errors.As(err, &runnerErr) {
		issues = append(issues, cycleStepIssuesFromRunnerV0(runnerErr.Issues)...)
	}
	result.Issues = compactCycleStepIssuesV0(append(result.Issues, issues...))
	return result, cycleStepErrorV0(
		input,
		ErrDirectorCycleStepRunnerV0,
		"runner fallo",
		"runner",
		true,
		result.Issues,
	)
}

func cycleStepIssuesFromRunnerV0(
	issues []orquestadirectorrunner.DirectorCycleIssueV0,
) []DirectorCycleStepIssueV0 {
	if len(issues) == 0 {
		return nil
	}
	out := make([]DirectorCycleStepIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, cycleStepIssueV0(issue.Code, "runner."+issue.Field, issue.Message))
	}
	return out
}

func compactCycleStepIssuesV0(
	issues []DirectorCycleStepIssueV0,
) []DirectorCycleStepIssueV0 {
	seen := map[DirectorCycleStepIssueV0]bool{}
	out := make([]DirectorCycleStepIssueV0, 0, len(issues))
	for _, issue := range issues {
		if issue.Code == "" || seen[issue] {
			continue
		}
		seen[issue] = true
		out = append(out, issue)
	}
	return out
}
