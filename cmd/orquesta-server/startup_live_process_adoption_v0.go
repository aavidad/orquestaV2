package main

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type startupProcessAdopterV0 interface {
	AdoptProcessV0(context.Context, orquestaruntime.ProcessRuntimeSnapshotV0) (orquestaruntime.ProcessRuntimeSnapshotV0, error)
}

type startupLiveProcessAdoptionSummaryV0 struct {
	Scanned  int
	Adopted  int
	Stopped  int
	Missing  int
	Inferred int
}

func (check serverStartupCheckV0) adoptStartupLiveProcessesV0(
	ctx context.Context,
) (startupLiveProcessAdoptionSummaryV0, error) {
	adopter, ok := check.Stack.Codex.Runtime.(startupProcessAdopterV0)
	if !ok || adopter == nil {
		return startupLiveProcessAdoptionSummaryV0{}, nil
	}
	registry, ok := check.Stack.Stores.ProcessRegistry.(orquestacionnucleoapp.AgentProcessRegistryListPortV0)
	if !ok || registry == nil {
		return startupLiveProcessAdoptionSummaryV0{}, nil
	}
	records, err := registry.ListAgentProcessesV0(ctx, orquestacionnucleoapp.AgentProcessRegistryListFilterV0{})
	if err != nil {
		return startupLiveProcessAdoptionSummaryV0{}, err
	}
	pidsByAckPath := map[string]int{}
	summary := startupLiveProcessAdoptionSummaryV0{Scanned: len(records)}
	for _, record := range records {
		pid := record.PID
		if pid <= 0 {
			if len(pidsByAckPath) == 0 {
				pidsByAckPath = startupCodexPIDsByLastMessagePathV0()
			}
			inferred := check.startupInferPIDForAgentRecordV0(ctx, record, pidsByAckPath)
			if inferred > 0 {
				pid = inferred
				summary.Inferred++
			}
		}
		if pid <= 0 {
			summary.Missing++
			continue
		}
		snapshot, err := adopter.AdoptProcessV0(ctx, orquestaruntime.ProcessRuntimeSnapshotV0{
			SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
			ProcessRef:    record.ProcessRef,
			SessionRef:    record.SessionRef,
			LaunchRef:     record.LaunchRef,
			PID:           pid,
			Status:        orquestaruntime.ProcessRuntimeRunningV0,
		})
		if err != nil {
			summary.Missing++
			continue
		}
		if snapshot.Status == orquestaruntime.ProcessRuntimeRunningV0 {
			summary.Adopted++
			continue
		}
		summary.Stopped++
	}
	return summary, nil
}

func (check serverStartupCheckV0) startupInferPIDForAgentRecordV0(
	ctx context.Context,
	record orquestacionnucleoapp.AgentProcessRegistryRecordV0,
	pidsByLastMessagePath map[string]int,
) int {
	if check.Stack.Stores.ReceiptStore == nil {
		return 0
	}
	descriptors, err := check.Stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
			RunID:         record.RunID,
			StartedAgents: []string{record.AgentRequestID},
		},
	)
	if err != nil || len(descriptors) == 0 {
		return 0
	}
	for _, descriptor := range descriptors {
		lastMessagePath := filepath.Join(filepath.Dir(strings.TrimSpace(descriptor.AckPath)), "codex_last_message.txt")
		if pid := pidsByLastMessagePath[lastMessagePath]; pid > 0 {
			return pid
		}
	}
	return 0
}

func startupAdoptionMessageV0(
	base string,
	summary startupLiveProcessAdoptionSummaryV0,
) string {
	if summary.Scanned == 0 {
		return base
	}
	return base +
		"; agentes_adoptados=" + strconv.Itoa(summary.Adopted) +
		" inferidos=" + strconv.Itoa(summary.Inferred) +
		" sin_pid=" + strconv.Itoa(summary.Missing)
}

func startupAdoptionEvidenceRefsV0(
	base []string,
	summary startupLiveProcessAdoptionSummaryV0,
) []string {
	refs := append([]string(nil), base...)
	if summary.Scanned > 0 {
		refs = append(refs, "evidence-ref-orquesta-startup-live-process-adoption")
	}
	if summary.Inferred > 0 {
		refs = append(refs, "evidence-ref-orquesta-startup-live-pid-inferred")
	}
	return compactStartupStringsV0(refs)
}

type startupProcCodexCandidateV0 struct {
	PID             int
	PPID            int
	LastMessagePath string
}

func startupCodexPIDsByLastMessagePathV0() map[string]int {
	candidates := startupCodexProcessCandidatesV0()
	candidatePIDs := map[int]bool{}
	for _, candidate := range candidates {
		candidatePIDs[candidate.PID] = true
	}
	out := map[string]int{}
	for _, candidate := range candidates {
		if candidate.PID <= 0 || candidate.LastMessagePath == "" {
			continue
		}
		if candidatePIDs[candidate.PPID] {
			continue
		}
		if _, exists := out[candidate.LastMessagePath]; !exists {
			out[candidate.LastMessagePath] = candidate.PID
		}
	}
	return out
}

func startupCodexProcessCandidatesV0() []startupProcCodexCandidateV0 {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var out []startupProcCodexCandidateV0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		cmdline, ok := startupProcCmdlineV0(pid)
		if !ok {
			continue
		}
		lastMessagePath := startupCodexLastMessagePathFromCmdlineV0(cmdline)
		if lastMessagePath == "" {
			continue
		}
		out = append(out, startupProcCodexCandidateV0{
			PID:             pid,
			PPID:            startupProcPPIDV0(pid),
			LastMessagePath: lastMessagePath,
		})
	}
	return out
}

func startupProcCmdlineV0(pid int) ([]string, bool) {
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cmdline"))
	if err != nil || len(raw) == 0 {
		return nil, false
	}
	parts := strings.Split(string(raw), "\x00")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out, len(out) > 0
}

func startupCodexLastMessagePathFromCmdlineV0(args []string) string {
	for index, arg := range args {
		if arg == "--output-last-message" && index+1 < len(args) {
			return strings.TrimSpace(args[index+1])
		}
		if strings.HasPrefix(arg, "--output-last-message=") {
			return strings.TrimSpace(strings.TrimPrefix(arg, "--output-last-message="))
		}
	}
	return ""
}

func startupProcPPIDV0(pid int) int {
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0
	}
	text := string(raw)
	end := strings.LastIndex(text, ")")
	if end < 0 || end+2 >= len(text) {
		return 0
	}
	fields := strings.Fields(text[end+2:])
	if len(fields) < 2 {
		return 0
	}
	ppid, _ := strconv.Atoi(fields[1])
	return ppid
}
