package orquestaruntime

import (
	"fmt"
	"path/filepath"
	"strings"
)

func validateProcessRuntimeLaunchRequestV0(req ProcessRuntimeLaunchRequestV0) error {
	if strings.TrimSpace(req.CommandPath) == "" {
		return processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "command_path")
	}
	if !filepath.IsAbs(req.CommandPath) {
		return processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "command_path")
	}
	if processRuntimeOperationalPathUnsafeV0(req.CommandPath) || looksLikeSecret(req.CommandPath) {
		return processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "command_path")
	}
	if processRuntimeCommandIsShellV0(req.CommandPath) {
		return processRuntimeErrorV0(ProcessRuntimeShellProhibidaV0, "command_path")
	}
	if strings.TrimSpace(req.WorkingDir) == "" {
		return processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "working_dir")
	}
	if !filepath.IsAbs(req.WorkingDir) {
		return processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "working_dir")
	}
	if processRuntimeOperationalPathUnsafeV0(req.WorkingDir) {
		return processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "working_dir")
	}
	if req.Env == nil {
		return processRuntimeErrorV0(ProcessRuntimeEnvProhibidoV0, "env")
	}
	for i, arg := range req.Args {
		if processRuntimeUnsafeValueV0(arg) {
			return processRuntimeErrorV0(
				ProcessRuntimeConfigInvalidaV0,
				fmt.Sprintf("args[%d]", i),
			)
		}
	}
	for i, item := range req.Env {
		if !processRuntimeEnvEntryAllowedV0(item) {
			return processRuntimeErrorV0(
				ProcessRuntimeEnvProhibidoV0,
				fmt.Sprintf("env[%d]", i),
			)
		}
	}
	return nil
}
