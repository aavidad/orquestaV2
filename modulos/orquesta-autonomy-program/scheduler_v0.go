package orquestaautonomyprogram

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
)

var ErrAutonomyProgramCASConflictV0 = errors.New("autonomy_program_cas_conflict")

type AutonomyProgramLaunchV0 struct {
	ProgramRef string   `json:"program_ref"`
	ProjectRef string   `json:"project_ref"`
	RootRef    string   `json:"root_ref"`
	NodeRef    string   `json:"node_ref"`
	LaunchRef  string   `json:"launch_ref"`
	WriteSet   []string `json:"write_set"`
}

type AutonomyProgramFrontierV0 struct {
	Program  AutonomyProgramV0         `json:"program"`
	Launches []AutonomyProgramLaunchV0 `json:"launches"`
}

// PrepareAutonomyProgramFrontierV0 marks only the dependency-ready, disjoint
// frontier as launched. Persist the returned program before invoking an
// actuator; replays then return no duplicate launch for the same node.
func PrepareAutonomyProgramFrontierV0(program AutonomyProgramV0) (AutonomyProgramFrontierV0, error) {
	program, err := NewAutonomyProgramV0(program)
	if err != nil {
		return AutonomyProgramFrontierV0{}, err
	}
	if program.Status == AutonomyProgramCompletedV0 || program.Status == AutonomyProgramBlockedV0 {
		return AutonomyProgramFrontierV0{Program: program, Launches: []AutonomyProgramLaunchV0{}}, nil
	}
	byRef := autonomyProgramNodeIndexV0(program.Nodes)
	active := make([]AutonomyProgramNodeV0, 0)
	for _, node := range program.Nodes {
		if node.Status == AutonomyNodeLaunchedV0 || node.Status == AutonomyNodeWaitExternalV0 {
			active = append(active, node)
		}
	}
	candidates := make([]AutonomyProgramNodeV0, 0)
	for _, node := range program.Nodes {
		if (node.Status == AutonomyNodePendingV0 || node.Status == AutonomyNodeReworkV0) && externalDependenciesAcceptedV0(node) && dependenciesAcceptedV0(node, byRef) && !overlapsAnySelectedV0(node, active) {
			candidates = append(candidates, node)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].NodeRef < candidates[j].NodeRef })
	selected := make([]AutonomyProgramNodeV0, 0, len(candidates))
	for _, node := range candidates {
		if overlapsAnySelectedV0(node, selected) {
			continue
		}
		selected = append(selected, node)
	}
	launches := make([]AutonomyProgramLaunchV0, 0, len(selected))
	for _, selectedNode := range selected {
		launchRef := autonomyProgramLaunchRefV0(program, selectedNode)
		for index := range program.Nodes {
			if program.Nodes[index].NodeRef == selectedNode.NodeRef {
				if program.Nodes[index].LaunchRef != "" {
					program.Nodes[index].PriorLaunchRefs = append(program.Nodes[index].PriorLaunchRefs, program.Nodes[index].LaunchRef)
				}
				program.Nodes[index].Status = AutonomyNodeLaunchedV0
				program.Nodes[index].LaunchRef = launchRef
			}
		}
		launches = append(launches, AutonomyProgramLaunchV0{ProgramRef: program.ProgramRef, ProjectRef: program.ProjectRef, RootRef: program.RootRef, NodeRef: selectedNode.NodeRef, LaunchRef: launchRef, WriteSet: append([]string(nil), selectedNode.WriteSet...)})
	}
	program.Status = deriveAutonomyProgramStatusV0(program)
	return AutonomyProgramFrontierV0{Program: program, Launches: launches}, nil
}

func autonomyProgramLaunchRefV0(program AutonomyProgramV0, node AutonomyProgramNodeV0) string {
	payload := program.ProjectRef + "\x00" + program.RootRef + "\x00" + program.ProgramRef + "\x00" + node.NodeRef + "\x00" + fmt.Sprintf("%d", node.ReworkCount)
	sum := sha256.Sum256([]byte(payload))
	return "autonomy-launch:" + hex.EncodeToString(sum[:])
}

// ClaimAutonomyProgramFrontierV0 is the persist-before-act boundary. It only
// returns launches after the exact loaded aggregate won the store CAS. A
// competing scheduler either observes the durable claim or receives a CAS
// conflict and therefore has no launches to actuate.
func ClaimAutonomyProgramFrontierV0(ctx context.Context, store AutonomyProgramStorePortV0, projectRef, rootRef, programRef string) (AutonomyProgramFrontierV0, error) {
	if store == nil {
		return AutonomyProgramFrontierV0{}, fmt.Errorf("autonomy_program store requerido")
	}
	current, err := store.LoadAutonomyProgramV0(ctx, projectRef, rootRef, programRef)
	if err != nil {
		return AutonomyProgramFrontierV0{}, err
	}
	frontier, err := PrepareAutonomyProgramFrontierV0(current)
	if err != nil || len(frontier.Launches) == 0 {
		return frontier, err
	}
	swapped, err := store.CompareAndSwapAutonomyProgramV0(ctx, current, frontier.Program)
	if err != nil {
		return AutonomyProgramFrontierV0{}, err
	}
	if !swapped {
		return AutonomyProgramFrontierV0{}, ErrAutonomyProgramCASConflictV0
	}
	return frontier, nil
}

func autonomyProgramNodeIndexV0(nodes []AutonomyProgramNodeV0) map[string]AutonomyProgramNodeV0 {
	out := make(map[string]AutonomyProgramNodeV0, len(nodes))
	for _, node := range nodes {
		out[node.NodeRef] = node
	}
	return out
}
func dependenciesAcceptedV0(node AutonomyProgramNodeV0, byRef map[string]AutonomyProgramNodeV0) bool {
	for _, dep := range node.DependsOn {
		if byRef[dep].Status != AutonomyNodeAcceptedV0 {
			return false
		}
	}
	return true
}
func overlapsAnySelectedV0(node AutonomyProgramNodeV0, selected []AutonomyProgramNodeV0) bool {
	for _, prior := range selected {
		if autonomyWriteSetsOverlapV0(node.WriteSet, prior.WriteSet) {
			return true
		}
	}
	return false
}
func autonomyWriteSetsOverlapV0(left, right []string) bool {
	for _, a := range left {
		for _, b := range right {
			if a == b || hasPathPrefixV0(a, b) || hasPathPrefixV0(b, a) {
				return true
			}
		}
	}
	return false
}
func hasPathPrefixV0(child, parent string) bool {
	return len(child) > len(parent) && len(parent) > 0 && child[:len(parent)] == parent && child[len(parent)] == '/'
}
