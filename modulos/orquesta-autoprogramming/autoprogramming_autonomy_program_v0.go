package orquestaautoprogramming

import (
	"fmt"
	"strings"

	orquestaautonomyprogram "orquesta/modulos/orquesta-autonomy-program"
)

// BuildAutoprogrammingAutonomyProgramV0 compiles the already accepted work into
// one durable parent aggregate. It does not create a run or select a backend.
func BuildAutoprogrammingAutonomyProgramV0(work AutoprogrammingProgrammableWorkV0) (orquestaautonomyprogram.AutonomyProgramV0, []AutoprogrammingRequestIssueV0) {
	program := orquestaautonomyprogram.AutonomyProgramV0{
		SchemaVersion: orquestaautonomyprogram.AutonomyProgramSchemaVersionV0,
		ProgramRef:    "autonomy-program:" + strings.TrimSpace(work.RequestRef),
		ProjectRef:    strings.TrimSpace(work.ProjectRef),
		RootRef:       "autonomy-root:" + strings.TrimSpace(work.RequestRef),
		Status:        orquestaautonomyprogram.AutonomyProgramActiveV0,
	}
	nodeRefs := make(map[string]bool, len(work.Groups))
	for index, group := range work.Groups {
		nodeRef := "autonomy-node:" + strings.TrimSpace(work.RequestRef) + ":" + fmt.Sprintf("%02d", index+1)
		if strings.TrimSpace(group.Task.TaskID) != "" {
			nodeRef = group.Task.TaskID
		}
		nodeRefs[nodeRef] = true
	}
	for index, group := range work.Groups {
		nodeRef := "autonomy-node:" + strings.TrimSpace(work.RequestRef) + ":" + fmt.Sprintf("%02d", index+1)
		if strings.TrimSpace(group.Task.TaskID) != "" {
			nodeRef = group.Task.TaskID
		}
		goalRef := ""
		if index < len(work.GoalSpecs) {
			goalRef = work.GoalSpecs[index].GoalRef
		}
		dependsOn, externalDependsOn := autoprogrammingAutonomyProgramDependenciesV0(group.DependsOn, nodeRefs)
		program.Nodes = append(program.Nodes, orquestaautonomyprogram.AutonomyProgramNodeV0{
			NodeRef:           nodeRef,
			GoalRef:           goalRef,
			DependsOn:         dependsOn,
			ExternalDependsOn: externalDependsOn,
			WriteSet:          append([]string(nil), group.WriteSet...),
			RequiredTests:     append([]string(nil), group.RequiredTests...),
			Status:            orquestaautonomyprogram.AutonomyNodePendingV0,
		})
	}
	normalized, err := orquestaautonomyprogram.NewAutonomyProgramV0(program)
	if err == nil {
		return normalized, nil
	}
	return orquestaautonomyprogram.AutonomyProgramV0{}, []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0("autonomy_program_invalid", "autonomy_program", err.Error())}
}

func autoprogrammingAutonomyProgramDependenciesV0(dependencies []string, nodeRefs map[string]bool) ([]string, []string) {
	internal, external := make([]string, 0, len(dependencies)), make([]string, 0)
	for _, dependency := range dependencies {
		dependency = strings.TrimSpace(dependency)
		if dependency == "" {
			continue
		}
		if nodeRefs[dependency] {
			internal = append(internal, dependency)
		} else {
			external = append(external, dependency)
		}
	}
	return internal, external
}
