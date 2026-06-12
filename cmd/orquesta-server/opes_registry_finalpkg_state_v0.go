package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func readOPESRegistryFinalPkgRegistryV0(path string) (opesRegistryFinalPkgRegistryV0, error) {
	var registry opesRegistryFinalPkgRegistryV0
	if err := readOPESRegistryFinalPkgJSONFileV0(path, &registry); err != nil {
		return opesRegistryFinalPkgRegistryV0{}, err
	}
	return registry, nil
}

func readOPESRegistryFinalPkgAppChangeStateV0(path string) (opesRegistryFinalPkgAppChangeStateV0, error) {
	var state opesRegistryFinalPkgAppChangeStateV0
	if err := readOPESRegistryFinalPkgJSONFileV0(path, &state); err != nil {
		return opesRegistryFinalPkgAppChangeStateV0{}, err
	}
	return state, nil
}

func readOPESRegistryFinalPkgJSONFileV0(path string, target any) error {
	data, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("json_decode_error")
	}
	return nil
}

func findOPESRegistryFinalPkgTemplateRequestV0(
	state opesRegistryFinalPkgAppChangeStateV0,
	runRef string,
) (json.RawMessage, error) {
	runRef = strings.TrimSpace(runRef)
	for _, record := range state.Records {
		var ref struct {
			RunRef string `json:"run_ref"`
		}
		if err := json.Unmarshal(record.Request, &ref); err != nil {
			return nil, fmt.Errorf("app_change_state_decode_error")
		}
		if strings.TrimSpace(ref.RunRef) == runRef {
			return append(json.RawMessage(nil), record.Request...), nil
		}
	}
	return nil, fmt.Errorf("template_run_not_found")
}

func collectOPESRegistryFinalPkgExistingRunRefsV0(
	state opesRegistryFinalPkgAppChangeStateV0,
) (map[string]struct{}, error) {
	refs := map[string]struct{}{}
	for _, record := range state.Records {
		var ref struct {
			RunRef string `json:"run_ref"`
		}
		if err := json.Unmarshal(record.Request, &ref); err != nil {
			return nil, fmt.Errorf("app_change_state_decode_error")
		}
		if trimmed := strings.TrimSpace(ref.RunRef); trimmed != "" {
			refs[trimmed] = struct{}{}
		}
	}
	return refs, nil
}

func collectOPESRegistryFinalPkgActiveRunRefsV0(runsDir string) map[string]opesRegistryFinalPkgRunProjectionV0 {
	refs := map[string]opesRegistryFinalPkgRunProjectionV0{}
	entries, err := os.ReadDir(strings.TrimSpace(runsDir))
	if err != nil {
		return refs
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		projection, ok := readOPESRegistryFinalPkgRunProjectionV0(filepath.Join(runsDir, entry.Name()))
		if ok {
			refs[projection.runRef] = projection.opesRegistryFinalPkgRunProjectionV0
		}
	}
	return refs
}

type opesRegistryFinalPkgRunProjectionFileV0 struct {
	runRef string
	opesRegistryFinalPkgRunProjectionV0
}

func readOPESRegistryFinalPkgRunProjectionV0(path string) (opesRegistryFinalPkgRunProjectionFileV0, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return opesRegistryFinalPkgRunProjectionFileV0{}, false
	}
	var envelope struct {
		RunRef string `json:"run_ref"`
		Status string `json:"status"`
		Run    struct {
			RunID           string   `json:"run_id"`
			Status          string   `json:"status"`
			Deliveries      []string `json:"deliveries"`
			ClosedTasks     []string `json:"closed_tasks"`
			AcceptedReviews []string `json:"accepted_reviews"`
			Blockers        []string `json:"blockers"`
		} `json:"run"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return opesRegistryFinalPkgRunProjectionFileV0{}, false
	}
	ref := strings.TrimSpace(firstExternalBridgeValueV0(envelope.RunRef, envelope.Run.RunID))
	status := firstExternalBridgeValueV0(envelope.Run.Status, envelope.Status)
	if ref == "" || opesRegistryFinalPkgRunStatusTerminalV0(status) {
		return opesRegistryFinalPkgRunProjectionFileV0{}, false
	}
	return opesRegistryFinalPkgRunProjectionFileV0{
		runRef: ref,
		opesRegistryFinalPkgRunProjectionV0: opesRegistryFinalPkgRunProjectionV0{
			Status:          status,
			Deliveries:      compactOPESRegistryFinalPkgStringsV0(envelope.Run.Deliveries),
			ClosedTasks:     compactOPESRegistryFinalPkgStringsV0(envelope.Run.ClosedTasks),
			AcceptedReviews: compactOPESRegistryFinalPkgStringsV0(envelope.Run.AcceptedReviews),
			Blockers:        compactOPESRegistryFinalPkgStringsV0(envelope.Run.Blockers),
		},
	}, true
}

func opesRegistryFinalPkgRunStatusTerminalV0(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "cerrada", "closed", "cancelled", "canceled", "stopped":
		return true
	default:
		return false
	}
}

func inspectOPESRegistryFinalPkgRunRefsV0(
	config opesRegistryFinalPkgConfigV0,
	refs map[string]opesRegistryFinalPkgRunProjectionV0,
) ([]string, []opesRegistryFinalPkgCompletionDriftV0) {
	prefix, suffix := opesRegistryFinalPkgRunRefPartsV0(config)
	out := make([]string, 0, len(refs))
	completedNonTerminal := make([]opesRegistryFinalPkgCompletionDriftV0, 0)
	for ref, projection := range refs {
		if strings.HasPrefix(ref, prefix) && strings.HasSuffix(ref, suffix) {
			if !opesRegistryFinalPkgRunCountsAsInFlightV0(config, ref, projection) {
				topicID, _ := opesRegistryFinalPkgTopicIDFromRunRefV0(config, ref)
				completedNonTerminal = append(completedNonTerminal, opesRegistryFinalPkgCompletionDriftV0{
					RunRef:  ref,
					TopicID: topicID,
					Status:  strings.TrimSpace(projection.Status),
					Reason:  "package_complete_run_non_terminal",
				})
				continue
			}
			out = append(out, ref)
		}
	}
	sort.Strings(out)
	sort.Slice(completedNonTerminal, func(left int, right int) bool {
		return completedNonTerminal[left].RunRef < completedNonTerminal[right].RunRef
	})
	return out, completedNonTerminal
}
