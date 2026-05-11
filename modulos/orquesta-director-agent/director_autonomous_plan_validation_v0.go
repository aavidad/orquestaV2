package orquestadirectoragent

func (v *directorAgentDecisionValidatorV0) validatePlanTeam(decision DirectorAgentDecisionV0) {
	if decision.ProposePlanTeam == nil {
		v.add("director_agent_payload_requerido", "propose_autonomous_plan_team")
		return
	}
	plan := decision.ProposePlanTeam.Plan
	v.require("propose_autonomous_plan_team.plan.schema_version",
		plan.SchemaVersion, DirectorAgentAutonomousPlanTeamSchemaVersionV0)
	v.requireRef("propose_autonomous_plan_team.plan.plan_ref", plan.PlanRef)
	v.requireRef("propose_autonomous_plan_team.plan.run_id", plan.RunID)
	v.requireRef("propose_autonomous_plan_team.plan.phase_id", plan.PhaseID)
	v.requireRef("propose_autonomous_plan_team.plan.goal_ref", plan.GoalRef)
	v.requireText("propose_autonomous_plan_team.plan.summary", plan.Summary)
	v.requireEvidence("propose_autonomous_plan_team.plan.evidence_refs", plan.EvidenceRefs)
	v.validateTeamMembers("propose_autonomous_plan_team.plan.team", plan.Team)
	v.validateAutonomousWorkUnits("propose_autonomous_plan_team.plan.work_units", plan.WorkUnits)
	v.validateWorkUnitAssignments("propose_autonomous_plan_team.plan", plan)
	if plan.RunID != decision.RunID {
		v.add("director_agent_run_mismatch", "propose_autonomous_plan_team.plan.run_id")
	}
	if plan.PhaseID != decision.PhaseID {
		v.add("director_agent_phase_mismatch", "propose_autonomous_plan_team.plan.phase_id")
	}
	if decision.PhaseID != DirectorAgentPlanningPhaseIDV0 {
		v.add("director_agent_phase_mismatch", "phase_id")
	}
}

func (v *directorAgentDecisionValidatorV0) validateWorkUnitAssignments(
	field string,
	plan DirectorAgentAutonomousPlanTeamV0,
) {
	members := map[string]struct{}{}
	for _, member := range plan.Team {
		members[member.MemberRef] = struct{}{}
	}
	for _, unit := range plan.WorkUnits {
		if _, ok := members[unit.AssignedMemberRef]; !ok {
			v.add("director_agent_ref_invalida", field+".work_units.assigned_member_ref")
		}
	}
}

func (v *directorAgentDecisionValidatorV0) validateTeamMembers(
	field string,
	members []DirectorAgentTeamMemberV0,
) {
	if len(members) == 0 || len(members) > maxDirectorAgentEvidenceRefsV0 {
		v.add("director_agent_lista_invalida", field)
		return
	}
	seen := map[string]struct{}{}
	for _, member := range members {
		v.requireRef(field+".member_ref", member.MemberRef)
		v.requireText(field+".role", member.Role)
		v.requireCapacity(field+".capacity", member.Capacity)
		v.requireEvidence(field+".responsibility_refs", member.ResponsibilityRefs)
		if _, ok := seen[member.MemberRef]; ok {
			v.add("director_agent_ref_duplicada", field+".member_ref")
		}
		seen[member.MemberRef] = struct{}{}
	}
}

func (v *directorAgentDecisionValidatorV0) validateAutonomousWorkUnits(
	field string,
	units []DirectorAgentAutonomousWorkUnitV0,
) {
	if len(units) == 0 || len(units) > maxDirectorAgentEvidenceRefsV0 {
		v.add("director_agent_lista_invalida", field)
		return
	}
	seen := map[string]struct{}{}
	for _, unit := range units {
		v.requireRef(field+".work_unit_ref", unit.WorkUnitRef)
		v.requireRef(field+".phase_id", unit.PhaseID)
		v.requireText(field+".title", unit.Title)
		v.requireText(field+".summary", unit.Summary)
		v.requireRef(field+".assigned_member_ref", unit.AssignedMemberRef)
		v.requireTextList(field+".write_set", unit.WriteSet)
		v.requireTextList(field+".acceptance_criteria", unit.AcceptanceCriteria)
		v.validateFunctionRefs(field+".function_contract_refs", unit.FunctionContractRefs)
		v.requireEvidence(field+".depends_on", unit.DependsOn)
		if _, ok := seen[unit.WorkUnitRef]; ok {
			v.add("director_agent_ref_duplicada", field+".work_unit_ref")
		}
		seen[unit.WorkUnitRef] = struct{}{}
	}
}
