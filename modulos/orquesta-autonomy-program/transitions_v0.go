package orquestaautonomyprogram

import "fmt"

func PutAutonomyProgramNodeWaitExternalV0(program AutonomyProgramV0, task OperatorTaskV0) (AutonomyProgramV0, error) {
	program, err := NewAutonomyProgramV0(program)
	if err != nil {
		return AutonomyProgramV0{}, err
	}
	if task.ProgramRef != program.ProgramRef || task.ProjectRef != program.ProjectRef || task.RootRef != program.RootRef || task.NodeRef == "" || task.OperatorTaskRef == "" || task.Action == "" || len(compactRefsV0(task.CausalRefs)) == 0 {
		return AutonomyProgramV0{}, fmt.Errorf("operator_task causal fuera de scope o incompleta")
	}
	for index := range program.Nodes {
		if program.Nodes[index].NodeRef == task.NodeRef {
			for _, existing := range program.OperatorTasks {
				if existing.OperatorTaskRef == task.OperatorTaskRef {
					if program.Nodes[index].Status == AutonomyNodeWaitExternalV0 && existing.Status == "open" {
						return program, nil
					}
					return AutonomyProgramV0{}, fmt.Errorf("operator_task_ref ya usado: %s", task.OperatorTaskRef)
				}
				if existing.NodeRef == task.NodeRef && existing.Status == "open" {
					return AutonomyProgramV0{}, fmt.Errorf("node ya espera operator_task: %s", task.NodeRef)
				}
			}
			if program.Nodes[index].Status != AutonomyNodeLaunchedV0 {
				return AutonomyProgramV0{}, fmt.Errorf("node no lanzado: %s", task.NodeRef)
			}
			if !containsRefV0(compactRefsV0(task.CausalRefs), program.Nodes[index].LaunchRef) {
				return AutonomyProgramV0{}, fmt.Errorf("operator_task no deriva del launch actual: %s", task.NodeRef)
			}
			program.Nodes[index].Status = AutonomyNodeWaitExternalV0
			task.Status = "open"
			task.Receipts = nil
			program.OperatorTasks = append(program.OperatorTasks, task)
			return NewAutonomyProgramV0(program)
		}
	}
	return AutonomyProgramV0{}, fmt.Errorf("node_ref desconocido: %s", task.NodeRef)
}

// ResumeAutonomyProgramNodeV0 consumes a causal operator receipt and makes the
// exact waiting node to the already durable launch. It does not create a new
// launch attempt. A receipt from another project/root cannot resume this
// program; persist this transition before invoking Resume on the actuator.
func ResumeAutonomyProgramNodeV0(program AutonomyProgramV0, receipt OperatorReceiptV0) (AutonomyProgramV0, error) {
	program, err := NewAutonomyProgramV0(program)
	if err != nil {
		return AutonomyProgramV0{}, err
	}
	if receipt.ProgramRef != program.ProgramRef || receipt.ProjectRef != program.ProjectRef || receipt.RootRef != program.RootRef || receipt.NodeRef == "" || receipt.ReceiptRef == "" || receipt.OperatorTaskRef == "" || receipt.CausalRef == "" || compactRefV0(receipt.Decision) == "" {
		return AutonomyProgramV0{}, fmt.Errorf("operator_receipt causal fuera de scope o incompleto")
	}
	operatorIndex := -1
	for index, task := range program.OperatorTasks {
		for _, existingReceipt := range task.Receipts {
			if existingReceipt.ReceiptRef == receipt.ReceiptRef {
				if reflectOperatorReceiptV0(existingReceipt, receipt) {
					return program, nil
				}
				return AutonomyProgramV0{}, fmt.Errorf("operator_receipt_ref ya usado")
			}
		}
		if task.OperatorTaskRef == receipt.OperatorTaskRef && task.NodeRef == receipt.NodeRef && task.Status == "open" {
			operatorIndex = index
			break
		}
	}
	if operatorIndex < 0 {
		return AutonomyProgramV0{}, fmt.Errorf("operator_receipt sin operator_task abierto")
	}
	normalizedReceipt, err := normalizeOperatorReceiptV0(receipt, program.OperatorTasks[operatorIndex])
	if err != nil {
		return AutonomyProgramV0{}, err
	}
	for index := range program.Nodes {
		if program.Nodes[index].NodeRef == receipt.NodeRef {
			if program.Nodes[index].Status != AutonomyNodeWaitExternalV0 {
				return AutonomyProgramV0{}, fmt.Errorf("node no espera operador: %s", receipt.NodeRef)
			}
			program.Nodes[index].Status = AutonomyNodeLaunchedV0
			program.OperatorTasks[operatorIndex].Status = "resolved"
			program.OperatorTasks[operatorIndex].Receipts = append(program.OperatorTasks[operatorIndex].Receipts, normalizedReceipt)
			return NewAutonomyProgramV0(program)
		}
	}
	return AutonomyProgramV0{}, fmt.Errorf("node_ref desconocido: %s", receipt.NodeRef)
}

func reflectOperatorReceiptV0(left, right OperatorReceiptV0) bool {
	return compactRefV0(left.ReceiptRef) == compactRefV0(right.ReceiptRef) && compactRefV0(left.OperatorTaskRef) == compactRefV0(right.OperatorTaskRef) && compactRefV0(left.ProgramRef) == compactRefV0(right.ProgramRef) && compactRefV0(left.ProjectRef) == compactRefV0(right.ProjectRef) && compactRefV0(left.RootRef) == compactRefV0(right.RootRef) && compactRefV0(left.NodeRef) == compactRefV0(right.NodeRef) && compactRefV0(left.Decision) == compactRefV0(right.Decision) && compactRefV0(left.CausalRef) == compactRefV0(right.CausalRef)
}

// ResolveAutonomyProgramExternalDependencyV0 records a scoped receipt for one
// opaque dependency. The dependency remains immutable in the DAG and becomes
// ready only through this durable evidence.
func ResolveAutonomyProgramExternalDependencyV0(program AutonomyProgramV0, receipt ExternalDependencyReceiptV0) (AutonomyProgramV0, error) {
	program, err := NewAutonomyProgramV0(program)
	if err != nil {
		return AutonomyProgramV0{}, err
	}
	if receipt.ProgramRef != program.ProgramRef || receipt.ProjectRef != program.ProjectRef || receipt.RootRef != program.RootRef {
		return AutonomyProgramV0{}, fmt.Errorf("external_dependency_receipt fuera de scope")
	}
	for nodeIndex := range program.Nodes {
		node := &program.Nodes[nodeIndex]
		for _, existing := range node.ExternalReceipts {
			if existing.ReceiptRef == compactRefV0(receipt.ReceiptRef) || existing.DependencyRef == compactRefV0(receipt.DependencyRef) {
				if existing == receipt {
					return program, nil
				}
				return AutonomyProgramV0{}, fmt.Errorf("external_dependency_receipt ya consumido")
			}
		}
		if node.NodeRef != compactRefV0(receipt.NodeRef) {
			continue
		}
		if node.Status != AutonomyNodePendingV0 {
			return AutonomyProgramV0{}, fmt.Errorf("node no espera dependencia externa: %s", node.NodeRef)
		}
		normalized, normalizeErr := normalizeExternalDependencyReceiptV0(receipt, *node)
		if normalizeErr != nil {
			return AutonomyProgramV0{}, normalizeErr
		}
		node.ExternalReceipts = append(node.ExternalReceipts, normalized)
		return NewAutonomyProgramV0(program)
	}
	return AutonomyProgramV0{}, fmt.Errorf("node_ref desconocido: %s", receipt.NodeRef)
}

func CloseAutonomyProgramNodeV0(program AutonomyProgramV0, closure AutonomyNodeClosureV0) (AutonomyProgramV0, error) {
	program, err := NewAutonomyProgramV0(program)
	if err != nil {
		return AutonomyProgramV0{}, err
	}
	if closure.NodeRef == "" || closure.ReceiptRef == "" || len(compactRefsV0(closure.CausalRefs)) == 0 {
		return AutonomyProgramV0{}, fmt.Errorf("closure causal incompleto")
	}
	if closure.Decision != AutonomyNodeAcceptedV0 && closure.Decision != AutonomyNodeBlockedV0 && closure.Decision != AutonomyNodeReworkV0 {
		return AutonomyProgramV0{}, fmt.Errorf("decision de cierre invalida")
	}
	for index := range program.Nodes {
		if program.Nodes[index].NodeRef != closure.NodeRef {
			continue
		}
		node := &program.Nodes[index]
		if node.ClosureReceiptRef == closure.ReceiptRef && node.Status == closure.Decision {
			return program, nil
		}
		for _, other := range program.Nodes {
			if other.NodeRef != node.NodeRef && other.ClosureReceiptRef == compactRefV0(closure.ReceiptRef) {
				return AutonomyProgramV0{}, fmt.Errorf("closure receipt ya usado")
			}
		}
		for _, task := range program.OperatorTasks {
			for _, receipt := range task.Receipts {
				if receipt.ReceiptRef == compactRefV0(closure.ReceiptRef) {
					return AutonomyProgramV0{}, fmt.Errorf("closure receipt ya usado")
				}
			}
		}
		if node.Status != AutonomyNodeLaunchedV0 {
			return AutonomyProgramV0{}, fmt.Errorf("node no esta cerrado/revisable: %s", closure.NodeRef)
		}
		closure.CausalRefs = compactRefsV0(closure.CausalRefs)
		if !containsRefV0(closure.CausalRefs, node.LaunchRef) {
			return AutonomyProgramV0{}, fmt.Errorf("closure no deriva del launch actual: %s", closure.NodeRef)
		}
		proofs, proofErr := normalizeRequiredTestProofsV0(node.RequiredTests, closure.RequiredTestEvidence, closure.Decision == AutonomyNodeAcceptedV0)
		if proofErr != nil {
			return AutonomyProgramV0{}, fmt.Errorf("required_tests sin atestacion causal: %s", closure.NodeRef)
		}
		for _, proof := range proofs {
			if !containsRefV0(closure.CausalRefs, proof.CausalRef) {
				return AutonomyProgramV0{}, fmt.Errorf("required_test fuera de causalidad: %s", closure.NodeRef)
			}
		}
		if closure.Decision == AutonomyNodeReworkV0 {
			if node.ReworkCount >= 1 {
				return AutonomyProgramV0{}, fmt.Errorf("rework ya consumido: %s", closure.NodeRef)
			}
			node.ReworkCount++
			node.Status = AutonomyNodeReworkV0
		} else {
			node.Status = closure.Decision
		}
		node.ClosureReceiptRef = closure.ReceiptRef
		node.ClosureCausalRefs = closure.CausalRefs
		node.RequiredTestEvidence = proofs
		return NewAutonomyProgramV0(program)
	}
	return AutonomyProgramV0{}, fmt.Errorf("node_ref desconocido: %s", closure.NodeRef)
}
