package main

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	orquestacontext "orquesta/modulos/orquesta-context"
)

const defaultCodeContextToolOrphanMinAgeV0 = 120 * time.Second

type serverCodeContextToolProcessV0 struct {
	PID            int
	PPID           int
	PGID           int
	CPUPercent     int
	ElapsedSeconds int
	Command        string
}

type serverCodeContextToolProcessGuardResultV0 struct {
	Observed       int
	Owned          int
	Orphans        int
	OrphansStopped int
	Errors         int
	Evidence       []string
}

type serverCodeContextToolProcessListerV0 interface {
	ListCodeContextToolProcessesV0(context.Context) ([]serverCodeContextToolProcessV0, error)
}

type serverCodeContextToolProcessStopperV0 interface {
	StopCodeContextToolProcessV0(context.Context, int) ([]string, error)
}

type serverCodeContextToolOwnerMarkerListerV0 interface {
	ListCodeContextToolOwnerMarkersV0() ([]serverCodeContextToolOwnerMarkerV0, error)
}

type serverCodeContextToolProcessGuardV0 struct {
	Lister       serverCodeContextToolProcessListerV0
	Stopper      serverCodeContextToolProcessStopperV0
	OwnerMarkers serverCodeContextToolOwnerMarkerListerV0
	StopOrphans  bool
	OrphanMinAge time.Duration
}

func (guard serverCodeContextToolProcessGuardV0) RunOnceV0(
	ctx context.Context,
) (serverCodeContextToolProcessGuardResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if guard.Lister == nil {
		return serverCodeContextToolProcessGuardResultV0{}, nil
	}
	if err := ctx.Err(); err != nil {
		return serverCodeContextToolProcessGuardResultV0{}, err
	}
	processes, err := guard.Lister.ListCodeContextToolProcessesV0(ctx)
	if err != nil {
		return serverCodeContextToolProcessGuardResultV0{}, err
	}
	ownedPIDs, markerEvidence, err := guard.ownedCodeContextToolPIDsV0()
	if err != nil {
		return serverCodeContextToolProcessGuardResultV0{}, err
	}
	result := serverCodeContextToolProcessGuardResultV0{Evidence: markerEvidence}
	minAge := guard.OrphanMinAge
	if minAge <= 0 {
		minAge = defaultCodeContextToolOrphanMinAgeV0
	}
	for _, process := range processes {
		if !isServerCodebaseMemoryMCPProcessV0(process) {
			continue
		}
		result.Observed++
		if ownedPIDs[process.PID] {
			result.Owned++
			result.Evidence = append(result.Evidence, "evidence-ref-code-context-codebase-memory-owner-marker-protected")
			continue
		}
		if time.Duration(process.ElapsedSeconds)*time.Second < minAge {
			result.Evidence = append(result.Evidence, "evidence-ref-code-context-codebase-memory-young-process-observed")
			continue
		}
		result.Orphans++
		result.Evidence = append(result.Evidence, "evidence-ref-code-context-codebase-memory-orphan-detected")
		if !guard.StopOrphans {
			continue
		}
		stopper := guard.Stopper
		if stopper == nil {
			stopper = serverSignalCodeContextToolProcessStopperV0{}
		}
		evidence, stopErr := stopper.StopCodeContextToolProcessV0(ctx, process.PID)
		if stopErr != nil {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			result.Errors++
			result.Evidence = append(result.Evidence, "evidence-ref-code-context-codebase-memory-orphan-stop-error-"+compactExternalBridgeErrorCodeV0(stopErr))
			continue
		}
		result.OrphansStopped++
		result.Evidence = append(result.Evidence, evidence...)
	}
	result.Evidence = compactEnvlessStringsV0(result.Evidence)
	return result, nil
}

func (guard serverCodeContextToolProcessGuardV0) ownedCodeContextToolPIDsV0() (map[int]bool, []string, error) {
	owned := map[int]bool{}
	if guard.OwnerMarkers == nil {
		return owned, nil, nil
	}
	markers, err := guard.OwnerMarkers.ListCodeContextToolOwnerMarkersV0()
	if err != nil {
		return owned, nil, err
	}
	evidence := []string{}
	for _, marker := range markers {
		if marker.ProviderKind != orquestacontext.CodeContextProviderKindCodebaseMCPV0 || marker.PID <= 0 {
			continue
		}
		owned[marker.PID] = true
		evidence = append(evidence, "evidence-ref-code-context-codebase-memory-owned-pid-listed")
	}
	return owned, compactEnvlessStringsV0(evidence), nil
}

type serverPSCodeContextToolProcessListerV0 struct {
	Command string
}

func (lister serverPSCodeContextToolProcessListerV0) ListCodeContextToolProcessesV0(
	ctx context.Context,
) ([]serverCodeContextToolProcessV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	command := strings.TrimSpace(lister.Command)
	if command == "" {
		command = "ps"
	}
	output, err := exec.CommandContext(ctx, command, "-eo", "pid=,ppid=,pgid=,stat=,pcpu=,etimes=,args=").Output()
	if err != nil {
		return nil, err
	}
	return parseServerPSCodeContextToolProcessesV0(string(output)), nil
}

type serverSignalCodeContextToolProcessStopperV0 struct {
	Signal func(int, syscall.Signal) error
}

func (stopper serverSignalCodeContextToolProcessStopperV0) StopCodeContextToolProcessV0(
	ctx context.Context,
	pid int,
) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if pid <= 0 {
		return nil, errors.New("code_context_tool_process_pid_invalid")
	}
	signal := stopper.Signal
	if signal == nil {
		signal = serverSignalProcessV0
	}
	if err := signal(pid, syscall.SIGTERM); err != nil {
		return nil, err
	}
	return []string{"evidence-ref-code-context-codebase-memory-orphan-term-signal-sent"}, nil
}

func parseServerPSCodeContextToolProcessesV0(raw string) []serverCodeContextToolProcessV0 {
	out := []serverCodeContextToolProcessV0{}
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}
		pid, pidOK := parseNonNegativeServerPSIntV0(fields[0])
		ppid, ppidOK := parseNonNegativeServerPSIntV0(fields[1])
		pgid, pgidOK := parseNonNegativeServerPSIntV0(fields[2])
		cpu, cpuOK := parseServerPSPercentIntV0(fields[4])
		elapsed, elapsedOK := parseNonNegativeServerPSIntV0(fields[5])
		if !pidOK || !ppidOK || !pgidOK || !cpuOK || !elapsedOK {
			continue
		}
		out = append(out, serverCodeContextToolProcessV0{
			PID:            pid,
			PPID:           ppid,
			PGID:           pgid,
			CPUPercent:     cpu,
			ElapsedSeconds: elapsed,
			Command:        strings.Join(fields[6:], " "),
		})
	}
	return out
}

func isServerCodebaseMemoryMCPProcessV0(process serverCodeContextToolProcessV0) bool {
	command := strings.TrimSpace(process.Command)
	if command == "" || process.PID <= 0 {
		return false
	}
	first := strings.Fields(command)
	if len(first) == 0 {
		return false
	}
	base := filepath.Base(first[0])
	return base == "codebase-memory-mcp" || strings.Contains(command, "/codebase-memory-mcp ")
}

func parseNonNegativeServerPSIntV0(raw string) (int, bool) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0, false
	}
	return value, true
}

func parseServerPSPercentIntV0(raw string) (int, bool) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || value < 0 {
		return 0, false
	}
	return int(value + 0.5), true
}
