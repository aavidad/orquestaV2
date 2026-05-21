package orquestadirectoragent

import "strings"

func normalizeDirectorAgentPlanTeamCommandV0(
	payload DirectorAgentPlanTeamCommandV0,
) DirectorAgentPlanTeamCommandV0 {
	return DirectorAgentPlanTeamCommandV0{
		Plan: normalizeDirectorAgentAutonomousPlanTeamV0(payload.Plan),
	}
}

func normalizeDirectorAgentAutonomousPlanTeamV0(
	plan DirectorAgentAutonomousPlanTeamV0,
) DirectorAgentAutonomousPlanTeamV0 {
	return DirectorAgentAutonomousPlanTeamV0{
		SchemaVersion: strings.TrimSpace(plan.SchemaVersion),
		PlanRef:       strings.TrimSpace(plan.PlanRef),
		RunID:         strings.TrimSpace(plan.RunID),
		PhaseID:       strings.TrimSpace(plan.PhaseID),
		GoalRef:       strings.TrimSpace(plan.GoalRef),
		Summary:       strings.TrimSpace(plan.Summary),
		Team:          normalizeDirectorAgentTeamMembersV0(plan.Team),
		WorkUnits:     normalizeDirectorAgentWorkUnitsV0(plan.WorkUnits),
		EvidenceRefs:  compactDirectorAgentStringsV0(plan.EvidenceRefs),
	}
}

func normalizeDirectorAgentTeamMembersV0(
	members []DirectorAgentTeamMemberV0,
) []DirectorAgentTeamMemberV0 {
	result := make([]DirectorAgentTeamMemberV0, 0, len(members))
	seen := map[string]struct{}{}
	for _, member := range members {
		compact := DirectorAgentTeamMemberV0{
			MemberRef:          strings.TrimSpace(member.MemberRef),
			Role:               strings.TrimSpace(member.Role),
			Capacity:           strings.TrimSpace(member.Capacity),
			ResponsibilityRefs: compactDirectorAgentStringsV0(member.ResponsibilityRefs),
		}
		if compact.MemberRef == "" {
			continue
		}
		if _, ok := seen[compact.MemberRef]; ok {
			continue
		}
		seen[compact.MemberRef] = struct{}{}
		result = append(result, compact)
	}
	return result
}

func normalizeDirectorAgentWorkUnitsV0(
	units []DirectorAgentAutonomousWorkUnitV0,
) []DirectorAgentAutonomousWorkUnitV0 {
	result := make([]DirectorAgentAutonomousWorkUnitV0, 0, len(units))
	seen := map[string]struct{}{}
	for _, unit := range units {
		compact := DirectorAgentAutonomousWorkUnitV0{
			WorkUnitRef:          strings.TrimSpace(unit.WorkUnitRef),
			PhaseID:              strings.TrimSpace(unit.PhaseID),
			WorkProfileKind:      strings.TrimSpace(unit.WorkProfileKind),
			Title:                strings.TrimSpace(unit.Title),
			Summary:              strings.TrimSpace(unit.Summary),
			AssignedMemberRef:    strings.TrimSpace(unit.AssignedMemberRef),
			WriteSet:             compactDirectorAgentStringsV0(unit.WriteSet),
			AcceptanceCriteria:   compactDirectorAgentStringsV0(unit.AcceptanceCriteria),
			FunctionContractRefs: normalizeDirectorAgentFunctionRefsV0(unit.FunctionContractRefs),
			DependsOn:            compactDirectorAgentStringsV0(unit.DependsOn),
		}
		if compact.WorkUnitRef == "" {
			continue
		}
		if _, ok := seen[compact.WorkUnitRef]; ok {
			continue
		}
		seen[compact.WorkUnitRef] = struct{}{}
		result = append(result, compact)
	}
	return result
}
