package orquestadirectoragent

func (v *directorAgentDecisionValidatorV0) validateCreateMicrotask(decision DirectorAgentDecisionV0) {
	if decision.CreateMicrotask == nil {
		v.add("director_agent_payload_requerido", "create_microtask")
		return
	}
	task := decision.CreateMicrotask.Task
	v.require("create_microtask.task.schema_version", task.SchemaVersion, DirectorAgentMicrotaskSchemaVersionV0)
	v.requireRef("create_microtask.task.task_id", task.TaskID)
	v.requireRef("create_microtask.task.run_id", task.RunID)
	v.requireRef("create_microtask.task.phase_id", task.PhaseID)
	v.requireText("create_microtask.task.title", task.Title)
	v.requireText("create_microtask.task.summary", task.Summary)
	v.requireTextList("create_microtask.task.write_set", task.WriteSet)
	v.requireTextList("create_microtask.task.acceptance_criteria", task.AcceptanceCriteria)
	if task.PhaseID == "programacion" {
		v.requireTextList("create_microtask.task.required_tests", task.RequiredTests)
	} else {
		v.requireOptionalTextList("create_microtask.task.required_tests", task.RequiredTests)
	}
	v.requireOptionalRefs("create_microtask.task.depends_on", task.DependsOn)
	v.validateFunctionRefs("create_microtask.task.function_contract_refs", task.FunctionContractRefs)
	if task.RunID != decision.RunID {
		v.add("director_agent_run_mismatch", "create_microtask.task.run_id")
	}
	if decision.PhaseID != DirectorAgentPlanningPhaseIDV0 &&
		decision.PhaseID != "programacion" {
		v.add("director_agent_phase_mismatch", "phase_id")
	}
}
