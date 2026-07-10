package orquestaautonomyprogram

import (
	"fmt"
	"reflect"
)

// ValidateAutonomyProgramSuccessorV0 rejects durable rollback even when a
// caller owns a current CAS snapshot. State may only advance through the
// transitions exposed by this package.
func ValidateAutonomyProgramSuccessorV0(current, next AutonomyProgramV0) error {
	current, err := NewAutonomyProgramV0(current)
	if err != nil {
		return err
	}
	next, err = NewAutonomyProgramV0(next)
	if err != nil {
		return err
	}
	if !SameAutonomyProgramTopologyV0(current, next) {
		return fmt.Errorf("autonomy_program topology changed")
	}
	if len(next.OperatorTasks) < len(current.OperatorTasks) {
		return fmt.Errorf("operator_tasks rollback")
	}
	for index := range current.Nodes {
		if err := validateAutonomyNodeSuccessorV0(current.Nodes[index], next.Nodes[index]); err != nil {
			return err
		}
	}
	for index := range current.OperatorTasks {
		if err := validateOperatorTaskSuccessorV0(current.OperatorTasks[index], next.OperatorTasks[index]); err != nil {
			return err
		}
	}
	for index := len(current.OperatorTasks); index < len(next.OperatorTasks); index++ {
		if next.OperatorTasks[index].Status != "open" {
			return fmt.Errorf("operator_task nueva no abierta")
		}
	}
	return nil
}

func validateAutonomyNodeSuccessorV0(current, next AutonomyProgramNodeV0) error {
	if current.NodeRef != next.NodeRef || !slicePrefixEqualV0(current.ExternalReceipts, next.ExternalReceipts) {
		return fmt.Errorf("node state rollback: %s", current.NodeRef)
	}
	if reflect.DeepEqual(current, next) {
		return nil
	}
	switch {
	case current.Status == AutonomyNodePendingV0 && next.Status == AutonomyNodePendingV0:
		copyNext := next
		copyNext.ExternalReceipts = current.ExternalReceipts
		if !reflect.DeepEqual(current, copyNext) {
			return fmt.Errorf("pending mutation invalid: %s", current.NodeRef)
		}
	case current.Status == AutonomyNodePendingV0 && next.Status == AutonomyNodeLaunchedV0:
		copyNext := next
		copyNext.Status, copyNext.LaunchRef, copyNext.PriorLaunchRefs = current.Status, current.LaunchRef, current.PriorLaunchRefs
		if !reflect.DeepEqual(current, copyNext) {
			return fmt.Errorf("initial launch mutation invalid: %s", current.NodeRef)
		}
	case current.Status == AutonomyNodeReworkV0 && next.Status == AutonomyNodeLaunchedV0:
		copyNext := next
		copyNext.Status, copyNext.LaunchRef, copyNext.PriorLaunchRefs = current.Status, current.LaunchRef, current.PriorLaunchRefs
		if !reflect.DeepEqual(current, copyNext) || !sliceEqualStringsV0(next.PriorLaunchRefs, append(append([]string(nil), current.PriorLaunchRefs...), current.LaunchRef)) {
			return fmt.Errorf("rework launch mutation invalid: %s", current.NodeRef)
		}
	case current.Status == AutonomyNodeLaunchedV0 && next.Status == AutonomyNodeWaitExternalV0,
		current.Status == AutonomyNodeWaitExternalV0 && next.Status == AutonomyNodeLaunchedV0:
		copyNext := next
		copyNext.Status = current.Status
		if !reflect.DeepEqual(current, copyNext) {
			return fmt.Errorf("wait transition invalid: %s", current.NodeRef)
		}
	case (current.Status == AutonomyNodeLaunchedV0) && (next.Status == AutonomyNodeAcceptedV0 || next.Status == AutonomyNodeBlockedV0 || next.Status == AutonomyNodeReworkV0):
		copyNext := next
		copyNext.Status, copyNext.ClosureReceiptRef, copyNext.ClosureCausalRefs, copyNext.RequiredTestEvidence, copyNext.ReworkCount = current.Status, current.ClosureReceiptRef, current.ClosureCausalRefs, current.RequiredTestEvidence, current.ReworkCount
		if !reflect.DeepEqual(current, copyNext) {
			return fmt.Errorf("closure mutation invalid: %s", current.NodeRef)
		}
		if next.Status == AutonomyNodeReworkV0 && next.ReworkCount != current.ReworkCount+1 {
			return fmt.Errorf("rework count invalid: %s", current.NodeRef)
		}
		if next.Status != AutonomyNodeReworkV0 && next.ReworkCount != current.ReworkCount {
			return fmt.Errorf("closure changed rework count: %s", current.NodeRef)
		}
	default:
		return fmt.Errorf("node transition invalid: %s %s->%s", current.NodeRef, current.Status, next.Status)
	}
	return nil
}

func validateOperatorTaskSuccessorV0(current, next OperatorTaskV0) error {
	if current.OperatorTaskRef != next.OperatorTaskRef {
		return fmt.Errorf("operator_task order changed")
	}
	if reflect.DeepEqual(current, next) {
		return nil
	}
	copyNext := next
	copyNext.Status, copyNext.Receipts = current.Status, current.Receipts
	if !reflect.DeepEqual(current, copyNext) || current.Status != "open" || next.Status != "resolved" || len(next.Receipts) <= len(current.Receipts) || !slicePrefixEqualV0(current.Receipts, next.Receipts) {
		return fmt.Errorf("operator_task transition invalid: %s", current.OperatorTaskRef)
	}
	return nil
}

func slicePrefixEqualV0[T any](prefix, values []T) bool {
	return len(prefix) <= len(values) && reflect.DeepEqual(prefix, values[:len(prefix)])
}

func sliceEqualStringsV0(left, right []string) bool { return reflect.DeepEqual(left, right) }
