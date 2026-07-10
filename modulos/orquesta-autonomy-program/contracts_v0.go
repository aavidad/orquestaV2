// Package orquestaautonomyprogram contains the durable, neutral parent-program
// contract. It deliberately has no runtime, provider, process, or filesystem
// dependency.
package orquestaautonomyprogram

import (
	"context"
	"fmt"
	"path"
	"reflect"
	"sort"
	"strings"
)

const AutonomyProgramSchemaVersionV0 = "orquesta_autonomy_program.v0"

type AutonomyProgramStatusV0 string

const (
	AutonomyProgramActiveV0       AutonomyProgramStatusV0 = "active"
	AutonomyProgramWaitExternalV0 AutonomyProgramStatusV0 = "wait_external"
	AutonomyProgramCompletedV0    AutonomyProgramStatusV0 = "completed"
	AutonomyProgramBlockedV0      AutonomyProgramStatusV0 = "blocked"
)

type AutonomyNodeStatusV0 string

const (
	AutonomyNodePendingV0      AutonomyNodeStatusV0 = "pending"
	AutonomyNodeLaunchedV0     AutonomyNodeStatusV0 = "launched"
	AutonomyNodeWaitExternalV0 AutonomyNodeStatusV0 = "wait_external"
	AutonomyNodeAcceptedV0     AutonomyNodeStatusV0 = "accepted"
	AutonomyNodeBlockedV0      AutonomyNodeStatusV0 = "blocked"
	AutonomyNodeReworkV0       AutonomyNodeStatusV0 = "rework"
)

type AutonomyProgramV0 struct {
	SchemaVersion string                  `json:"schema_version"`
	ProgramRef    string                  `json:"program_ref"`
	ProjectRef    string                  `json:"project_ref"`
	RootRef       string                  `json:"root_ref"`
	ParentGoalRef string                  `json:"parent_goal_ref,omitempty"`
	Status        AutonomyProgramStatusV0 `json:"status"`
	Nodes         []AutonomyProgramNodeV0 `json:"nodes"`
	OperatorTasks []OperatorTaskV0        `json:"operator_tasks,omitempty"`
}

// AutonomyProgramNodeV0 is a task/goal owned by a parent program. GoalRef
// identifies a child goal when one exists; it never turns the first child into
// the global run.
type AutonomyProgramNodeV0 struct {
	NodeRef              string                        `json:"node_ref"`
	GoalRef              string                        `json:"goal_ref,omitempty"`
	DependsOn            []string                      `json:"depends_on,omitempty"`
	ExternalDependsOn    []string                      `json:"external_depends_on,omitempty"`
	WriteSet             []string                      `json:"write_set"`
	RequiredTests        []string                      `json:"required_tests"`
	Status               AutonomyNodeStatusV0          `json:"status"`
	LaunchRef            string                        `json:"launch_ref,omitempty"`
	PriorLaunchRefs      []string                      `json:"prior_launch_refs,omitempty"`
	ClosureReceiptRef    string                        `json:"closure_receipt_ref,omitempty"`
	ClosureCausalRefs    []string                      `json:"closure_causal_refs,omitempty"`
	RequiredTestEvidence []RequiredTestProofV0         `json:"required_test_evidence,omitempty"`
	ExternalReceipts     []ExternalDependencyReceiptV0 `json:"external_receipts,omitempty"`
	ReworkCount          int                           `json:"rework_count,omitempty"`
}

type RequiredTestProofV0 struct {
	TestRef     string `json:"test_ref"`
	Status      string `json:"status"`
	EvidenceRef string `json:"evidence_ref"`
	CausalRef   string `json:"causal_ref"`
}

// OperatorTaskV0 records a typed external verification request. It is an open
// state, not a narrative terminal marker.
type OperatorTaskV0 struct {
	OperatorTaskRef string              `json:"operator_task_ref"`
	ProgramRef      string              `json:"program_ref"`
	ProjectRef      string              `json:"project_ref"`
	RootRef         string              `json:"root_ref"`
	NodeRef         string              `json:"node_ref"`
	Action          string              `json:"action"`
	CausalRefs      []string            `json:"causal_refs"`
	Status          string              `json:"status"`
	Receipts        []OperatorReceiptV0 `json:"receipts,omitempty"`
}

type OperatorReceiptV0 struct {
	ReceiptRef      string `json:"receipt_ref"`
	OperatorTaskRef string `json:"operator_task_ref"`
	ProgramRef      string `json:"program_ref"`
	ProjectRef      string `json:"project_ref"`
	RootRef         string `json:"root_ref"`
	NodeRef         string `json:"node_ref"`
	Decision        string `json:"decision"`
	CausalRef       string `json:"causal_ref"`
}

// ExternalDependencyReceiptV0 resolves one declared opaque dependency without
// mutating the DAG topology. The full scope is durable so a similarly named
// dependency from another program cannot release this node.
type ExternalDependencyReceiptV0 struct {
	ReceiptRef    string `json:"receipt_ref"`
	ProgramRef    string `json:"program_ref"`
	ProjectRef    string `json:"project_ref"`
	RootRef       string `json:"root_ref"`
	NodeRef       string `json:"node_ref"`
	DependencyRef string `json:"dependency_ref"`
	Decision      string `json:"decision"`
	CausalRef     string `json:"causal_ref"`
}

type AutonomyNodeClosureV0 struct {
	NodeRef              string                `json:"node_ref"`
	Decision             AutonomyNodeStatusV0  `json:"decision"`
	ReceiptRef           string                `json:"receipt_ref"`
	CausalRefs           []string              `json:"causal_refs"`
	RequiredTestEvidence []RequiredTestProofV0 `json:"required_test_evidence"`
}

type AutonomyProgramStorePortV0 interface {
	// Save creates the aggregate or accepts an identical retry. State changes
	// use CompareAndSwap so separate schedulers cannot overwrite each other.
	SaveAutonomyProgramV0(context.Context, AutonomyProgramV0) error
	LoadAutonomyProgramV0(context.Context, string, string, string) (AutonomyProgramV0, error)
	CompareAndSwapAutonomyProgramV0(context.Context, AutonomyProgramV0, AutonomyProgramV0) (bool, error)
}

// Actuator is deliberately separate from legacy runs/control. A composition
// may implement it using a runner or operator transport without leaking either
// into this neutral contract.
type AutonomyProgramActuatorPortV0 interface {
	// Implementations are idempotent by LaunchRef and ReceiptRef respectively;
	// recovery may redeliver the same durable action after an ambiguous crash.
	LaunchAutonomyProgramNodeV0(context.Context, AutonomyProgramLaunchV0) error
	ResumeAutonomyProgramNodeV0(context.Context, OperatorReceiptV0) error
}

func NewAutonomyProgramV0(value AutonomyProgramV0) (AutonomyProgramV0, error) {
	value = cloneAutonomyProgramV0(value)
	value.SchemaVersion = strings.TrimSpace(value.SchemaVersion)
	if value.SchemaVersion == "" {
		value.SchemaVersion = AutonomyProgramSchemaVersionV0
	}
	if value.SchemaVersion != AutonomyProgramSchemaVersionV0 {
		return AutonomyProgramV0{}, fmt.Errorf("schema_version invalida")
	}
	value.ProgramRef, value.ProjectRef, value.RootRef = compactRefV0(value.ProgramRef), compactRefV0(value.ProjectRef), compactRefV0(value.RootRef)
	value.ParentGoalRef = compactRefV0(value.ParentGoalRef)
	if value.ProgramRef == "" || value.ProjectRef == "" || value.RootRef == "" {
		return AutonomyProgramV0{}, fmt.Errorf("program_ref, project_ref y root_ref requeridos")
	}
	if len(value.Nodes) == 0 {
		return AutonomyProgramV0{}, fmt.Errorf("nodes requeridos")
	}
	seen := map[string]bool{}
	goalRefs := map[string]bool{}
	for index := range value.Nodes {
		if duplicateCompactRefsV0(value.Nodes[index].DependsOn) || duplicateCompactRefsV0(value.Nodes[index].ExternalDependsOn) {
			return AutonomyProgramV0{}, fmt.Errorf("depends_on duplicado para %s", compactRefV0(value.Nodes[index].NodeRef))
		}
		node, err := normalizeAutonomyProgramNodeV0(value.Nodes[index])
		if err != nil {
			return AutonomyProgramV0{}, err
		}
		if seen[node.NodeRef] {
			return AutonomyProgramV0{}, fmt.Errorf("node_ref duplicado: %s", node.NodeRef)
		}
		seen[node.NodeRef] = true
		if node.GoalRef != "" {
			if node.GoalRef == value.ParentGoalRef || goalRefs[node.GoalRef] {
				return AutonomyProgramV0{}, fmt.Errorf("goal_ref colisiona: %s", node.GoalRef)
			}
			goalRefs[node.GoalRef] = true
		}
		value.Nodes[index] = node
	}
	for _, node := range value.Nodes {
		for _, dep := range node.DependsOn {
			if !seen[dep] {
				return AutonomyProgramV0{}, fmt.Errorf("depends_on desconocido: %s", dep)
			}
			if dep == node.NodeRef {
				return AutonomyProgramV0{}, fmt.Errorf("depends_on circular: %s", dep)
			}
		}
		for _, dep := range node.ExternalDependsOn {
			if seen[dep] || containsRefV0(node.DependsOn, dep) {
				return AutonomyProgramV0{}, fmt.Errorf("depends_on interno/externo ambiguo: %s", dep)
			}
		}
	}
	operatorRefs := map[string]bool{}
	receiptRefs := map[string]bool{}
	launchRefs := map[string]bool{}
	closureRefs := map[string]bool{}
	for _, node := range value.Nodes {
		for _, launchRef := range append(append([]string(nil), node.PriorLaunchRefs...), node.LaunchRef) {
			if launchRef == "" {
				continue
			}
			if launchRefs[launchRef] {
				return AutonomyProgramV0{}, fmt.Errorf("launch_ref duplicado: %s", launchRef)
			}
			launchRefs[launchRef] = true
		}
		if node.ClosureReceiptRef != "" {
			if closureRefs[node.ClosureReceiptRef] {
				return AutonomyProgramV0{}, fmt.Errorf("closure_receipt_ref duplicado: %s", node.ClosureReceiptRef)
			}
			closureRefs[node.ClosureReceiptRef] = true
			receiptRefs[node.ClosureReceiptRef] = true
		}
		for _, receipt := range node.ExternalReceipts {
			if receiptRefs[receipt.ReceiptRef] {
				return AutonomyProgramV0{}, fmt.Errorf("receipt_ref duplicado: %s", receipt.ReceiptRef)
			}
			receiptRefs[receipt.ReceiptRef] = true
		}
	}
	for index := range value.OperatorTasks {
		task, err := normalizeOperatorTaskV0(value.OperatorTasks[index], value, seen)
		if err != nil {
			return AutonomyProgramV0{}, err
		}
		if operatorRefs[task.OperatorTaskRef] {
			return AutonomyProgramV0{}, fmt.Errorf("operator_task_ref duplicado: %s", task.OperatorTaskRef)
		}
		operatorRefs[task.OperatorTaskRef] = true
		for _, receipt := range task.Receipts {
			if receiptRefs[receipt.ReceiptRef] {
				return AutonomyProgramV0{}, fmt.Errorf("receipt_ref duplicado: %s", receipt.ReceiptRef)
			}
			receiptRefs[receipt.ReceiptRef] = true
		}
		value.OperatorTasks[index] = task
	}
	if err := validateOperatorWaitStatesV0(value); err != nil {
		return AutonomyProgramV0{}, err
	}
	if autonomyProgramHasCycleV0(value.Nodes) {
		return AutonomyProgramV0{}, fmt.Errorf("depends_on contiene ciclo")
	}
	value.Status = deriveAutonomyProgramStatusV0(value)
	return value, nil
}

func cloneAutonomyProgramV0(value AutonomyProgramV0) AutonomyProgramV0 {
	value.Nodes = append([]AutonomyProgramNodeV0(nil), value.Nodes...)
	for index := range value.Nodes {
		node := &value.Nodes[index]
		node.DependsOn = append([]string(nil), node.DependsOn...)
		node.ExternalDependsOn = append([]string(nil), node.ExternalDependsOn...)
		node.WriteSet = append([]string(nil), node.WriteSet...)
		node.RequiredTests = append([]string(nil), node.RequiredTests...)
		node.PriorLaunchRefs = append([]string(nil), node.PriorLaunchRefs...)
		node.ClosureCausalRefs = append([]string(nil), node.ClosureCausalRefs...)
		node.RequiredTestEvidence = append([]RequiredTestProofV0(nil), node.RequiredTestEvidence...)
		node.ExternalReceipts = append([]ExternalDependencyReceiptV0(nil), node.ExternalReceipts...)
	}
	value.OperatorTasks = append([]OperatorTaskV0(nil), value.OperatorTasks...)
	for index := range value.OperatorTasks {
		value.OperatorTasks[index].CausalRefs = append([]string(nil), value.OperatorTasks[index].CausalRefs...)
		value.OperatorTasks[index].Receipts = append([]OperatorReceiptV0(nil), value.OperatorTasks[index].Receipts...)
	}
	return value
}

func normalizeOperatorTaskV0(task OperatorTaskV0, program AutonomyProgramV0, nodes map[string]bool) (OperatorTaskV0, error) {
	task.OperatorTaskRef, task.ProgramRef, task.ProjectRef, task.RootRef, task.NodeRef, task.Action, task.Status = compactRefV0(task.OperatorTaskRef), compactRefV0(task.ProgramRef), compactRefV0(task.ProjectRef), compactRefV0(task.RootRef), compactRefV0(task.NodeRef), compactRefV0(task.Action), compactRefV0(task.Status)
	task.CausalRefs = compactRefsV0(task.CausalRefs)
	if task.OperatorTaskRef == "" || task.ProgramRef != program.ProgramRef || task.ProjectRef != program.ProjectRef || task.RootRef != program.RootRef || !nodes[task.NodeRef] || task.Action == "" || len(task.CausalRefs) == 0 {
		return OperatorTaskV0{}, fmt.Errorf("operator_task invalida")
	}
	if task.Status == "" {
		task.Status = "open"
	}
	if task.Status != "open" && task.Status != "resolved" {
		return OperatorTaskV0{}, fmt.Errorf("operator_task status invalido")
	}
	if task.Status == "open" && len(task.Receipts) != 0 {
		return OperatorTaskV0{}, fmt.Errorf("operator_task abierta con receipt")
	}
	if task.Status == "resolved" && len(task.Receipts) == 0 {
		return OperatorTaskV0{}, fmt.Errorf("operator_task resuelta sin receipt")
	}
	seenReceipts := map[string]bool{}
	for index := range task.Receipts {
		receipt, err := normalizeOperatorReceiptV0(task.Receipts[index], task)
		if err != nil || seenReceipts[receipt.ReceiptRef] {
			return OperatorTaskV0{}, fmt.Errorf("operator_task receipt invalido")
		}
		seenReceipts[receipt.ReceiptRef] = true
		task.Receipts[index] = receipt
	}
	return task, nil
}

func normalizeAutonomyProgramNodeV0(node AutonomyProgramNodeV0) (AutonomyProgramNodeV0, error) {
	node.NodeRef, node.GoalRef, node.LaunchRef, node.ClosureReceiptRef = compactRefV0(node.NodeRef), compactRefV0(node.GoalRef), compactRefV0(node.LaunchRef), compactRefV0(node.ClosureReceiptRef)
	if node.NodeRef == "" {
		return AutonomyProgramNodeV0{}, fmt.Errorf("node_ref requerido")
	}
	node.DependsOn, node.ExternalDependsOn, node.RequiredTests = compactRefsV0(node.DependsOn), compactRefsV0(node.ExternalDependsOn), compactRefsV0(node.RequiredTests)
	writeSet, err := normalizeWriteSetV0(node.WriteSet)
	if err != nil {
		return AutonomyProgramNodeV0{}, fmt.Errorf("write_set inseguro para %s: %w", node.NodeRef, err)
	}
	node.WriteSet = writeSet
	if len(node.WriteSet) == 0 {
		return AutonomyProgramNodeV0{}, fmt.Errorf("write_set requerido para %s", node.NodeRef)
	}
	if len(node.RequiredTests) == 0 {
		return AutonomyProgramNodeV0{}, fmt.Errorf("required_tests requerido para %s", node.NodeRef)
	}
	if node.Status == "" {
		node.Status = AutonomyNodePendingV0
	}
	switch node.Status {
	case AutonomyNodePendingV0, AutonomyNodeLaunchedV0, AutonomyNodeWaitExternalV0, AutonomyNodeAcceptedV0, AutonomyNodeBlockedV0, AutonomyNodeReworkV0:
	default:
		return AutonomyProgramNodeV0{}, fmt.Errorf("status invalido para %s", node.NodeRef)
	}
	if node.ReworkCount < 0 || node.ReworkCount > 1 {
		return AutonomyProgramNodeV0{}, fmt.Errorf("rework_count invalido para %s", node.NodeRef)
	}
	if duplicateCompactRefsV0(node.PriorLaunchRefs) {
		return AutonomyProgramNodeV0{}, fmt.Errorf("prior_launch_refs duplicados para %s", node.NodeRef)
	}
	node.PriorLaunchRefs, node.ClosureCausalRefs = compactRefsV0(node.PriorLaunchRefs), compactRefsV0(node.ClosureCausalRefs)
	proofs, err := normalizeRequiredTestProofsV0(node.RequiredTests, node.RequiredTestEvidence, node.Status == AutonomyNodeAcceptedV0)
	if err != nil && len(node.RequiredTestEvidence) > 0 {
		return AutonomyProgramNodeV0{}, fmt.Errorf("required_tests invalidos para %s: %w", node.NodeRef, err)
	}
	node.RequiredTestEvidence = proofs
	dependencyReceipts := map[string]bool{}
	for index := range node.ExternalReceipts {
		receipt, receiptErr := normalizeExternalDependencyReceiptV0(node.ExternalReceipts[index], node)
		if receiptErr != nil {
			return AutonomyProgramNodeV0{}, receiptErr
		}
		if dependencyReceipts[receipt.DependencyRef] {
			return AutonomyProgramNodeV0{}, fmt.Errorf("external dependency receipt duplicado: %s", receipt.DependencyRef)
		}
		dependencyReceipts[receipt.DependencyRef] = true
		node.ExternalReceipts[index] = receipt
	}
	if err := validateAutonomyNodeStateV0(node); err != nil {
		return AutonomyProgramNodeV0{}, err
	}
	return node, nil
}

func compactRefV0(value string) string       { return strings.TrimSpace(value) }
func compactRefsV0(values []string) []string { return compactValuesV0(values, false) }
func compactValuesV0(values []string, cleanPath bool) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if cleanPath {
			value = strings.Trim(strings.ReplaceAll(value, "\\", "/"), "/")
		}
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func normalizeWriteSetV0(values []string) ([]string, error) {
	out, seen := make([]string, 0, len(values)), map[string]bool{}
	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		value := strings.ReplaceAll(raw, "\\", "/")
		if strings.HasPrefix(value, "~") || path.IsAbs(value) || hasURISchemeV0(value) || strings.ContainsRune(value, 0) || windowsDrivePathV0(value) {
			return nil, fmt.Errorf("ruta absoluta o especial: %s", raw)
		}
		for _, segment := range strings.Split(value, "/") {
			if segment == ".." {
				return nil, fmt.Errorf("traversal no permitido: %s", raw)
			}
		}
		value = path.Clean(value)
		if value == "." || value == ".." || strings.HasPrefix(value, "../") {
			return nil, fmt.Errorf("raiz o traversal no permitido: %s", raw)
		}
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out, nil
}

func windowsDrivePathV0(value string) bool {
	return len(value) >= 2 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':'
}

// hasURISchemeV0 rejects every URI form, including file:/tmp and opaque URIs
// without an authority component. A write-set is a repository-relative path.
func hasURISchemeV0(value string) bool {
	separator := strings.IndexByte(value, ':')
	if separator <= 0 || !((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) {
		return false
	}
	for _, character := range value[1:separator] {
		if !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '+' || character == '-' || character == '.') {
			return false
		}
	}
	return true
}

func autonomyProgramHasCycleV0(nodes []AutonomyProgramNodeV0) bool {
	byRef := map[string]AutonomyProgramNodeV0{}
	for _, node := range nodes {
		byRef[node.NodeRef] = node
	}
	visiting, done := map[string]bool{}, map[string]bool{}
	var visit func(string) bool
	visit = func(ref string) bool {
		if visiting[ref] {
			return true
		}
		if done[ref] {
			return false
		}
		visiting[ref] = true
		for _, dep := range byRef[ref].DependsOn {
			if visit(dep) {
				return true
			}
		}
		delete(visiting, ref)
		done[ref] = true
		return false
	}
	for _, node := range nodes {
		if visit(node.NodeRef) {
			return true
		}
	}
	return false
}

func normalizeRequiredTestProofsV0(required []string, proofs []RequiredTestProofV0, requirePassed bool) ([]RequiredTestProofV0, error) {
	requiredSet, found := map[string]bool{}, map[string]bool{}
	for _, testRef := range required {
		requiredSet[testRef] = true
	}
	out := make([]RequiredTestProofV0, 0, len(proofs))
	for _, proof := range proofs {
		proof.TestRef, proof.Status, proof.EvidenceRef, proof.CausalRef = compactRefV0(proof.TestRef), compactRefV0(proof.Status), compactRefV0(proof.EvidenceRef), compactRefV0(proof.CausalRef)
		if !requiredSet[proof.TestRef] || found[proof.TestRef] || proof.EvidenceRef == "" || proof.CausalRef == "" || (proof.Status != "passed" && proof.Status != "failed") || (requirePassed && proof.Status != "passed") {
			return nil, fmt.Errorf("atestacion invalida: %s", proof.TestRef)
		}
		found[proof.TestRef] = true
		out = append(out, proof)
	}
	for _, testRef := range required {
		if !found[testRef] {
			return nil, fmt.Errorf("atestacion ausente: %s", testRef)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TestRef < out[j].TestRef })
	return out, nil
}

func deriveAutonomyProgramStatusV0(program AutonomyProgramV0) AutonomyProgramStatusV0 {
	allClosed, waiting := true, false
	for _, node := range program.Nodes {
		switch node.Status {
		case AutonomyNodeAcceptedV0:
		case AutonomyNodeBlockedV0:
			return AutonomyProgramBlockedV0
		case AutonomyNodeWaitExternalV0:
			allClosed = false
			waiting = true
		default:
			allClosed = false
		}
		if !externalDependenciesAcceptedV0(node) {
			waiting = true
		}
	}
	if allClosed {
		return AutonomyProgramCompletedV0
	}
	if waiting {
		return AutonomyProgramWaitExternalV0
	}
	return AutonomyProgramActiveV0
}

func duplicateCompactRefsV0(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		value = compactRefV0(value)
		if value == "" {
			continue
		}
		if seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}

func containsRefV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func intersectsRefsV0(left, right []string) bool {
	for _, value := range left {
		if containsRefV0(right, value) {
			return true
		}
	}
	return false
}

func validateAutonomyNodeStateV0(node AutonomyProgramNodeV0) error {
	hasClosure := node.ClosureReceiptRef != "" || len(node.ClosureCausalRefs) != 0 || len(node.RequiredTestEvidence) != 0
	switch node.Status {
	case AutonomyNodePendingV0:
		if node.LaunchRef != "" || len(node.PriorLaunchRefs) != 0 || hasClosure || node.ReworkCount != 0 {
			return fmt.Errorf("pending con estado causal previo: %s", node.NodeRef)
		}
	case AutonomyNodeLaunchedV0, AutonomyNodeWaitExternalV0:
		if node.LaunchRef == "" {
			return fmt.Errorf("node activo sin launch_ref: %s", node.NodeRef)
		}
		if node.ReworkCount == 0 && hasClosure {
			return fmt.Errorf("node activo con cierre espurio: %s", node.NodeRef)
		}
		if node.ReworkCount == 1 && (node.ClosureReceiptRef == "" || len(node.ClosureCausalRefs) == 0 || len(node.RequiredTestEvidence) == 0 || !intersectsRefsV0(node.ClosureCausalRefs, node.PriorLaunchRefs)) {
			return fmt.Errorf("rework activo sin historia causal: %s", node.NodeRef)
		}
	case AutonomyNodeAcceptedV0, AutonomyNodeBlockedV0, AutonomyNodeReworkV0:
		if node.LaunchRef == "" || node.ClosureReceiptRef == "" || !containsRefV0(node.ClosureCausalRefs, node.LaunchRef) {
			return fmt.Errorf("cierre sin launch/receipt causal: %s", node.NodeRef)
		}
		if _, err := normalizeRequiredTestProofsV0(node.RequiredTests, node.RequiredTestEvidence, node.Status == AutonomyNodeAcceptedV0); err != nil {
			return fmt.Errorf("cierre sin tests atestados: %s", node.NodeRef)
		}
		for _, proof := range node.RequiredTestEvidence {
			if !containsRefV0(node.ClosureCausalRefs, proof.CausalRef) {
				return fmt.Errorf("test fuera de causalidad de cierre: %s", node.NodeRef)
			}
		}
		if node.Status == AutonomyNodeReworkV0 && node.ReworkCount != 1 {
			return fmt.Errorf("rework sin contador causal: %s", node.NodeRef)
		}
	}
	return nil
}

func normalizeOperatorReceiptV0(receipt OperatorReceiptV0, task OperatorTaskV0) (OperatorReceiptV0, error) {
	receipt.ReceiptRef, receipt.OperatorTaskRef, receipt.ProgramRef, receipt.ProjectRef, receipt.RootRef, receipt.NodeRef, receipt.Decision, receipt.CausalRef = compactRefV0(receipt.ReceiptRef), compactRefV0(receipt.OperatorTaskRef), compactRefV0(receipt.ProgramRef), compactRefV0(receipt.ProjectRef), compactRefV0(receipt.RootRef), compactRefV0(receipt.NodeRef), compactRefV0(receipt.Decision), compactRefV0(receipt.CausalRef)
	if receipt.ReceiptRef == "" || receipt.OperatorTaskRef != task.OperatorTaskRef || receipt.ProgramRef != task.ProgramRef || receipt.ProjectRef != task.ProjectRef || receipt.RootRef != task.RootRef || receipt.NodeRef != task.NodeRef || receipt.Decision == "" || receipt.CausalRef != task.OperatorTaskRef {
		return OperatorReceiptV0{}, fmt.Errorf("operator_receipt invalido")
	}
	return receipt, nil
}

func normalizeExternalDependencyReceiptV0(receipt ExternalDependencyReceiptV0, node AutonomyProgramNodeV0) (ExternalDependencyReceiptV0, error) {
	receipt.ReceiptRef, receipt.ProgramRef, receipt.ProjectRef, receipt.RootRef, receipt.NodeRef, receipt.DependencyRef, receipt.Decision, receipt.CausalRef = compactRefV0(receipt.ReceiptRef), compactRefV0(receipt.ProgramRef), compactRefV0(receipt.ProjectRef), compactRefV0(receipt.RootRef), compactRefV0(receipt.NodeRef), compactRefV0(receipt.DependencyRef), compactRefV0(receipt.Decision), compactRefV0(receipt.CausalRef)
	if receipt.ReceiptRef == "" || receipt.NodeRef != node.NodeRef || !containsRefV0(node.ExternalDependsOn, receipt.DependencyRef) || receipt.Decision != "satisfied" || receipt.CausalRef != receipt.DependencyRef {
		return ExternalDependencyReceiptV0{}, fmt.Errorf("external_dependency_receipt invalido para %s", node.NodeRef)
	}
	return receipt, nil
}

func externalDependenciesAcceptedV0(node AutonomyProgramNodeV0) bool {
	found := map[string]bool{}
	for _, receipt := range node.ExternalReceipts {
		if receipt.Decision == "satisfied" {
			found[receipt.DependencyRef] = true
		}
	}
	for _, dependency := range node.ExternalDependsOn {
		if !found[dependency] {
			return false
		}
	}
	return true
}

func validateOperatorWaitStatesV0(program AutonomyProgramV0) error {
	openByNode := map[string]int{}
	byNode := autonomyProgramNodeIndexV0(program.Nodes)
	for _, task := range program.OperatorTasks {
		if task.Status == "open" {
			openByNode[task.NodeRef]++
		}
		node := byNode[task.NodeRef]
		launchRefs := append(append([]string(nil), node.PriorLaunchRefs...), node.LaunchRef)
		if task.Status == "open" && !containsRefV0(task.CausalRefs, node.LaunchRef) {
			return fmt.Errorf("operator_task abierta fuera del launch actual: %s", task.OperatorTaskRef)
		}
		if task.Status == "resolved" && !intersectsRefsV0(task.CausalRefs, launchRefs) {
			return fmt.Errorf("operator_task resuelta sin launch causal: %s", task.OperatorTaskRef)
		}
	}
	for _, node := range program.Nodes {
		if openByNode[node.NodeRef] > 1 || (node.Status == AutonomyNodeWaitExternalV0) != (openByNode[node.NodeRef] == 1) {
			return fmt.Errorf("wait_external/operator_task inconsistente: %s", node.NodeRef)
		}
		for index := range node.ExternalReceipts {
			receipt := &node.ExternalReceipts[index]
			if receipt.ProgramRef != program.ProgramRef || receipt.ProjectRef != program.ProjectRef || receipt.RootRef != program.RootRef {
				return fmt.Errorf("external_dependency_receipt fuera de scope: %s", receipt.ReceiptRef)
			}
		}
	}
	return nil
}

// SameAutonomyProgramTopologyV0 compares the immutable DAG portion.
func SameAutonomyProgramTopologyV0(left, right AutonomyProgramV0) bool {
	left, leftErr := NewAutonomyProgramV0(left)
	right, rightErr := NewAutonomyProgramV0(right)
	if leftErr != nil || rightErr != nil || left.SchemaVersion != right.SchemaVersion || left.ProgramRef != right.ProgramRef || left.ProjectRef != right.ProjectRef || left.RootRef != right.RootRef || left.ParentGoalRef != right.ParentGoalRef || len(left.Nodes) != len(right.Nodes) {
		return false
	}
	for index := range left.Nodes {
		a, b := left.Nodes[index], right.Nodes[index]
		if a.NodeRef != b.NodeRef || a.GoalRef != b.GoalRef || !reflect.DeepEqual(a.DependsOn, b.DependsOn) || !reflect.DeepEqual(a.ExternalDependsOn, b.ExternalDependsOn) || !reflect.DeepEqual(a.WriteSet, b.WriteSet) || !reflect.DeepEqual(a.RequiredTests, b.RequiredTests) {
			return false
		}
	}
	return true
}
