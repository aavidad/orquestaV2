package application

import (
	"context"
	"sync"
	"testing"

	"orquesta/internal/goal"
)

func TestConcurrentIdenticalControlCASLoserReturnsExactReplay(t *testing.T) {
	system := newControlTestSystem(t, nil)
	winnerDone := make(chan struct{})
	gate := &controlReplayRaceState{
		StateRepository: system.repository,
		initial:         make(map[string]bool),
		bothInitial:     make(chan struct{}),
		winnerDone:      winnerDone,
	}
	system.orchestrator.state = gate
	request := system.request(
		t, "control:concurrent-identical", ControlPause, ControlTargetGoal,
		goal.WorkItemRef{}, goal.ExecutionRef{},
	)
	type outcome struct {
		role   string
		result ControlResult
		err    error
	}
	results := make(chan outcome, 2)
	go func() {
		ctx := context.WithValue(context.Background(), controlRaceRoleKey{}, "winner")
		result, err := system.orchestrator.Control(ctx, system.access, request)
		close(winnerDone)
		results <- outcome{role: "winner", result: result, err: err}
	}()
	go func() {
		ctx := context.WithValue(context.Background(), controlRaceRoleKey{}, "loser")
		result, err := system.orchestrator.Control(ctx, system.access, request)
		results <- outcome{role: "loser", result: result, err: err}
	}()

	byRole := make(map[string]outcome, 2)
	for range 2 {
		result := <-results
		byRole[result.role] = result
	}
	winner, loser := byRole["winner"], byRole["loser"]
	if winner.err != nil || !winner.result.Created {
		t.Fatalf("winner: result=%+v err=%v", winner.result, winner.err)
	}
	if loser.err != nil || loser.result.Created || loser.result.Control.Ref != winner.result.Control.Ref {
		t.Fatalf("loser replay: result=%+v winner=%+v err=%v", loser.result, winner.result, loser.err)
	}
	record := system.record(t)
	if len(record.Controls) != 1 || record.Controls[0].Ref != winner.result.Control.Ref {
		t.Fatalf("controls=%+v", record.Controls)
	}
	system.repository.mu.Lock()
	controlEvents := 0
	for _, event := range system.repository.events {
		if event.Kind == "control.pause" {
			controlEvents++
		}
	}
	system.repository.mu.Unlock()
	if controlEvents != 1 {
		t.Fatalf("control events=%d, want 1", controlEvents)
	}
}

type controlRaceRoleKey struct{}

type controlReplayRaceState struct {
	StateRepository
	mu          sync.Mutex
	initial     map[string]bool
	initialSeen int
	bothInitial chan struct{}
	winnerDone  <-chan struct{}
}

func (state *controlReplayRaceState) ControlReplay(
	ctx context.Context,
	request ControlReplayRequest,
) (ControlRecord, bool, error) {
	role, _ := ctx.Value(controlRaceRoleKey{}).(string)
	state.mu.Lock()
	if !state.initial[role] {
		state.initial[role] = true
		state.initialSeen++
		if state.initialSeen == 2 {
			close(state.bothInitial)
		}
		both := state.bothInitial
		state.mu.Unlock()
		<-both
		return ControlRecord{}, false, nil
	}
	state.mu.Unlock()
	return state.StateRepository.ControlReplay(ctx, request)
}

func (state *controlReplayRaceState) GetGoal(
	ctx context.Context,
	ref goal.GoalRef,
) (GoalRecord, error) {
	if role, _ := ctx.Value(controlRaceRoleKey{}).(string); role == "loser" {
		<-state.winnerDone
	}
	return state.StateRepository.GetGoal(ctx, ref)
}
