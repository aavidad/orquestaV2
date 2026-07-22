package sqlite

import (
	"context"
	"database/sql"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func readRequiredTestSpecs(ctx context.Context, source queryer, goalValue, workItemValue string) ([]goal.RequiredTestSpecSnapshot, error) {
	persisted, err := sqliteTableHasColumn(ctx, source, "work_item_required_tests", "ref")
	if err != nil || !persisted {
		return nil, mapDatabaseError(err)
	}
	rows, err := source.QueryContext(ctx, `SELECT required.ref,required.tool_ref,required.working_directory,argument.value FROM work_item_required_tests required LEFT JOIN work_item_required_test_arguments argument ON argument.goal_ref=required.goal_ref AND argument.work_item_ref=required.work_item_ref AND argument.required_test_ref=required.ref WHERE required.goal_ref=? AND required.work_item_ref=? ORDER BY required.position,argument.position`, goalValue, workItemValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []goal.RequiredTestSpecSnapshot
	for rows.Next() {
		var spec goal.RequiredTestSpecSnapshot
		var argument sql.NullString
		if err := rows.Scan(&spec.Ref, &spec.ToolRef, &spec.WorkingDirectory, &argument); err != nil {
			return nil, mapDatabaseError(err)
		}
		if len(result) == 0 || result[len(result)-1].Ref != spec.Ref {
			result = append(result, spec)
		}
		if argument.Valid {
			result[len(result)-1].Arguments = append(result[len(result)-1].Arguments, argument.String)
		}
	}
	return result, mapDatabaseError(rows.Err())
}

func readAttestationOutcomes(ctx context.Context, source queryer, attestationRef string) ([]ports.RequiredTestOutcome, error) {
	rows, err := source.QueryContext(ctx, `SELECT required_test_ref,exit_code,output_digest FROM attestation_test_outcomes WHERE attestation_ref=? ORDER BY position`, attestationRef)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []ports.RequiredTestOutcome
	for rows.Next() {
		var outcome ports.RequiredTestOutcome
		var ref string
		if err := rows.Scan(&ref, &outcome.ExitCode, &outcome.OutputDigest); err != nil {
			return nil, mapDatabaseError(err)
		}
		if outcome.RequiredTestRef, err = goal.NewRequiredTestRef(ref); err != nil {
			return nil, invalid(err)
		}
		result = append(result, outcome)
	}
	return result, mapDatabaseError(rows.Err())
}
