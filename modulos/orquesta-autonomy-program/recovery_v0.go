package orquestaautonomyprogram

// AutonomyProgramRecoveryV0 reconstructs durable actions after a crash between
// persistence and actuation. Adapters must deduplicate LaunchRef/ReceiptRef:
// external exactly-once effects cannot be manufactured by this pure contract.
type AutonomyProgramRecoveryV0 struct {
	Launches []AutonomyProgramLaunchV0 `json:"launches,omitempty"`
	Resumes  []OperatorReceiptV0       `json:"resumes,omitempty"`
}

func RecoverAutonomyProgramActionsV0(program AutonomyProgramV0) (AutonomyProgramRecoveryV0, error) {
	program, err := NewAutonomyProgramV0(program)
	if err != nil {
		return AutonomyProgramRecoveryV0{}, err
	}
	recovery := AutonomyProgramRecoveryV0{}
	if program.Status == AutonomyProgramCompletedV0 || program.Status == AutonomyProgramBlockedV0 {
		return recovery, nil
	}
	for _, node := range program.Nodes {
		if node.Status != AutonomyNodeLaunchedV0 {
			continue
		}
		var latestResume *OperatorReceiptV0
		hasTaskForLaunch := false
		for _, task := range program.OperatorTasks {
			if task.NodeRef != node.NodeRef || !containsRefV0(task.CausalRefs, node.LaunchRef) {
				continue
			}
			hasTaskForLaunch = true
			if task.Status == "resolved" && len(task.Receipts) > 0 {
				receipt := task.Receipts[len(task.Receipts)-1]
				latestResume = &receipt
			}
		}
		if latestResume != nil {
			recovery.Resumes = append(recovery.Resumes, *latestResume)
			continue
		}
		if !hasTaskForLaunch {
			recovery.Launches = append(recovery.Launches, AutonomyProgramLaunchV0{ProgramRef: program.ProgramRef, ProjectRef: program.ProjectRef, RootRef: program.RootRef, NodeRef: node.NodeRef, LaunchRef: node.LaunchRef, WriteSet: append([]string(nil), node.WriteSet...)})
		}
	}
	return recovery, nil
}
