package sqlite

import (
	"database/sql"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
)

func councilExecutionRole(execution application.ExecutionRecord) (council.Role, bool) {
	switch execution.Purpose {
	case application.ExecutionPurposeCouncilProposer:
		return council.RoleProposer, true
	case application.ExecutionPurposeCouncilCritic:
		return council.RoleCritic, true
	case application.ExecutionPurposeCouncilArbiter:
		return council.RoleArbiter, true
	default:
		return "", false
	}
}

func councilRoundFor(
	rounds []application.CouncilRoundRecord, digest application.CouncilSubjectDigest,
) (application.CouncilRoundRecord, bool) {
	for _, round := range rounds {
		if round.SubjectDigest == digest {
			return round, true
		}
	}
	return application.CouncilRoundRecord{}, false
}

func councilSkipFor(
	skips []application.CouncilSkipRecord, digest application.CouncilSubjectDigest,
) *application.CouncilSkipRecord {
	for i := range skips {
		if skips[i].SubjectDigest == digest {
			return &skips[i]
		}
	}
	return nil
}

func councilFactsFor(
	facts []council.ContributionFact, digest application.CouncilSubjectDigest,
) []council.ContributionFact {
	var result []council.ContributionFact
	for _, fact := range facts {
		if fact.SubjectDigest == string(digest) {
			result = append(result, fact)
		}
	}
	return result
}

func councilFactsForRole(
	facts []council.ContributionFact, digest application.CouncilSubjectDigest, role council.Role,
) []council.ContributionFact {
	var result []council.ContributionFact
	for _, fact := range councilFactsFor(facts, digest) {
		if fact.Role == role {
			result = append(result, fact)
		}
	}
	return result
}

func executionFor(
	executions []application.ExecutionRecord, ref goal.ExecutionRef,
) (application.ExecutionRecord, bool) {
	for _, execution := range executions {
		if execution.Ref == ref {
			return execution, true
		}
	}
	return application.ExecutionRecord{}, false
}

func storedCouncilResolution(
	resolution *application.CouncilResolution,
) (string, any, any, any, any) {
	if resolution == nil {
		return "", nil, nil, nil, nil
	}
	return string(resolution.SubjectDigest), nullableString(resolution.DecisionRef),
		nullableString(string(resolution.DecisionDigest)), nullableString(resolution.SkipRef),
		nullableString(string(resolution.SkipDigest))
}

func restoreCouncilResolution(
	subject string,
	decisionRef, decisionDigest, skipRef, skipDigest sql.NullString,
) (*application.CouncilResolution, error) {
	if subject == "" && !decisionRef.Valid && !decisionDigest.Valid && !skipRef.Valid && !skipDigest.Valid {
		return nil, nil
	}
	resolution := &application.CouncilResolution{SubjectDigest: application.CouncilSubjectDigest(subject)}
	if decisionRef.Valid {
		resolution.DecisionRef = decisionRef.String
	}
	if decisionDigest.Valid {
		resolution.DecisionDigest = application.CouncilSubjectDigest(decisionDigest.String)
	}
	if skipRef.Valid {
		resolution.SkipRef = skipRef.String
	}
	if skipDigest.Valid {
		resolution.SkipDigest = application.CouncilSubjectDigest(skipDigest.String)
	}
	if resolution.Validate() != nil {
		return nil, invalid(errors.New("sqlite.council_resolution_invalid"))
	}
	return resolution, nil
}
