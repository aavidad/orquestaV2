package main

import "strings"

func guardianLocalDiagnosticsV0(config guardianConfigV0, result guardianResultV0) []guardianLocalDiagnosticRefV0 {
	paths := []struct {
		kind        string
		path        string
		evidenceRef string
	}{
		{"manifest", result.ManifestPath, guardianManifestRefV0(config, result)},
		{"repair_packet", result.RepairPacketPath, guardianRepairPacketRefV0(config, result)},
		{"repair_launch_packet", result.RepairLaunchPath, guardianRepairLaunchRefV0(config, result)},
	}
	for _, command := range result.Commands {
		paths = append(paths, struct {
			kind        string
			path        string
			evidenceRef string
		}{"command_output", command.OutputPath, guardianCommandOutputRefV0(config, command)})
	}
	diagnostics := make([]guardianLocalDiagnosticRefV0, 0, len(paths))
	for _, item := range paths {
		if strings.TrimSpace(item.path) == "" {
			continue
		}
		ref := guardianLocalDiagnosticPublicRefV0(config, result, item.kind, item.evidenceRef)
		if ref == "" {
			continue
		}
		diagnostics = append(diagnostics, guardianLocalDiagnosticRefV0{
			Ref:             ref,
			Kind:            item.kind,
			Classification:  "local_diagnostic_path",
			RedactionLevel:  guardianPublicRedactionLevelV0,
			AccessPolicyRef: "guardian-local-diagnostic-access-policy-ref-v0",
		})
	}
	return diagnostics
}

func guardianRedactOperationalTextV0(config guardianConfigV0, value string) string {
	redacted := strings.TrimSpace(value)
	for _, raw := range []string{
		config.ProjectDir,
		config.StateDir,
		config.CurrentBin,
		config.CandidateBin,
		config.LastGoodBin,
		config.ManifestPath,
		config.RepairPacketPath,
		config.CandidateStateDir,
		config.CandidateRuntimeDir,
	} {
		if strings.TrimSpace(raw) != "" {
			redacted = strings.ReplaceAll(redacted, raw, "[local_path_redacted]")
		}
	}
	return redacted
}
