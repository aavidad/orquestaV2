package orquestacionnucleoapp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type LiveProcessCapacityGatePortV0 interface {
	ReserveLiveProcessCapacityV0(
		context.Context,
		LiveProcessCapacityReservationRequestV0,
	) (LiveProcessCapacityReservationV0, error)
	ReleaseLiveProcessCapacityV0(context.Context, LiveProcessCapacityReservationV0) error
}

type LiveProcessCapacityReservationRequestV0 struct {
	RunRef      string
	TargetPort  string
	MessageType string
	Requested   int
}

type LiveProcessCapacityReservationV0 struct {
	ReservationRef string
	Granted        int
	Requested      int
	Live           int
	Reserved       int
	Limit          int
	EvidenceRefs   []string
}

type LiveProcessCapacityGateV0 struct {
	Registry       AgentProcessRegistryListPortV0
	SnapshotSource ProcessRuntimeIdentitySnapshotPortV0
	Limit          int
	ScopeRunRef    string
	EvidenceRefs   []string

	mu           sync.Mutex
	sequence     int
	reservations map[string]int
}

func (gate *LiveProcessCapacityGateV0) ReserveLiveProcessCapacityV0(
	ctx context.Context,
	request LiveProcessCapacityReservationRequestV0,
) (LiveProcessCapacityReservationV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return LiveProcessCapacityReservationV0{}, err
	}
	if err := gate.validateV0(); err != nil {
		return LiveProcessCapacityReservationV0{}, err
	}
	request = normalizeLiveProcessCapacityReservationRequestV0(request)
	if request.Requested <= 0 {
		request.Requested = 1
	}
	gate.mu.Lock()
	defer gate.mu.Unlock()
	live, err := gate.countLiveProcessesLockedV0(ctx)
	if err != nil {
		return LiveProcessCapacityReservationV0{}, err
	}
	reserved := gate.reservedLockedV0()
	available := gate.Limit - live - reserved
	if available < 0 {
		available = 0
	}
	granted := request.Requested
	if granted > available {
		granted = available
	}
	reservation := LiveProcessCapacityReservationV0{
		ReservationRef: gate.nextReservationRefLockedV0(request),
		Granted:        granted,
		Requested:      request.Requested,
		Live:           live,
		Reserved:       reserved,
		Limit:          gate.Limit,
		EvidenceRefs:   compactStringsV0(append([]string{"evidence-ref-live-process-capacity"}, gate.EvidenceRefs...)),
	}
	if granted > 0 {
		gate.ensureReservationsLockedV0()
		gate.reservations[reservation.ReservationRef] = granted
	}
	return reservation, nil
}

func (gate *LiveProcessCapacityGateV0) ReleaseLiveProcessCapacityV0(
	ctx context.Context,
	reservation LiveProcessCapacityReservationV0,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if gate == nil {
		return nil
	}
	gate.mu.Lock()
	defer gate.mu.Unlock()
	if gate.reservations == nil {
		return nil
	}
	delete(gate.reservations, strings.TrimSpace(reservation.ReservationRef))
	return nil
}

func (gate *LiveProcessCapacityGateV0) validateV0() error {
	switch {
	case gate == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "live_process_capacity_gate", "capacity gate requerido")
	case gate.Registry == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "agent_process_registry_list", "agent_process_registry_list requerido")
	case gate.SnapshotSource == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "process_snapshot_source", "process_snapshot_source requerido")
	case gate.Limit <= 0:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "live_process_limit", "live_process_limit requerido")
	default:
		return nil
	}
}

func (gate *LiveProcessCapacityGateV0) countLiveProcessesLockedV0(ctx context.Context) (int, error) {
	filter := AgentProcessRegistryListFilterV0{RunID: strings.TrimSpace(gate.ScopeRunRef)}
	records, err := gate.Registry.ListAgentProcessesV0(ctx, filter)
	if err != nil {
		return 0, err
	}
	live := 0
	seen := map[string]bool{}
	for _, record := range records {
		processRef := strings.TrimSpace(record.ProcessRef)
		if processRef == "" || seen[processRef] {
			continue
		}
		seen[processRef] = true
		snapshot, err := gate.SnapshotSource.SnapshotV0(processRef)
		if err != nil {
			continue
		}
		if snapshot.Status == orquestaruntime.ProcessRuntimeRunningV0 {
			live++
		}
	}
	return live, nil
}

func (gate *LiveProcessCapacityGateV0) reservedLockedV0() int {
	total := 0
	for _, count := range gate.reservations {
		if count > 0 {
			total += count
		}
	}
	return total
}

func (gate *LiveProcessCapacityGateV0) nextReservationRefLockedV0(
	request LiveProcessCapacityReservationRequestV0,
) string {
	gate.sequence++
	base := strings.TrimSpace(request.RunRef)
	if base == "" {
		base = "global"
	}
	return fmt.Sprintf("live-capacity-reservation-%s-%06d", liveProcessCapacitySafeRefPartV0(base), gate.sequence)
}

func (gate *LiveProcessCapacityGateV0) ensureReservationsLockedV0() {
	if gate.reservations == nil {
		gate.reservations = map[string]int{}
	}
}

func normalizeLiveProcessCapacityReservationRequestV0(
	request LiveProcessCapacityReservationRequestV0,
) LiveProcessCapacityReservationRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.TargetPort = strings.TrimSpace(request.TargetPort)
	request.MessageType = strings.TrimSpace(request.MessageType)
	return request
}

func liveProcessCapacitySafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "global"
	}
	return value
}
