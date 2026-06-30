package main

import (
	"os"
	"path/filepath"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	envOPESProjectWorkDirV0           = "ORQUESTA_OPES_PROJECT_WORKDIR"
	defaultOPESProjectWorkDirV0       = "/home/alberto/Trabajo/OPES"
	opesProjectWorkDirGuardEvidenceV0 = "evidence-ref-opes-project-workdir-required"
	opesLocalExternalWriteSetPrefixV0 = "external/opes"
)

func externalWorkRunProjectWorkDirGuardConfigFromEnvV0(
	serverConfig orquestaserver.ConfigV0,
) orquestaappcodexstack.ExternalWorkRunProjectWorkDirGuardConfigV0 {
	requiredOPESProjectDir := strings.TrimSpace(os.Getenv(envOPESProjectWorkDirV0))
	if requiredOPESProjectDir == "" {
		requiredOPESProjectDir = defaultOPESProjectWorkDirV0
	}
	requiredOPESProjectDir = filepath.Clean(requiredOPESProjectDir)
	return orquestaappcodexstack.ExternalWorkRunProjectWorkDirGuardConfigV0{
		ProjectWorkDir: filepath.Clean(strings.TrimSpace(serverConfig.ProjectWorkDir)),
		Rules: []orquestaappcodexstack.ExternalWorkRunProjectWorkDirGuardRuleV0{
			{
				ProjectRef:                   "opes",
				RequiredProjectWorkDir:       requiredOPESProjectDir,
				EvidenceRef:                  opesProjectWorkDirGuardEvidenceV0,
				AllowedLocalWriteSetPrefixes: []string{opesLocalExternalWriteSetPrefixV0},
			},
			{
				ProjectRef:                   "project-ref-opes",
				RequiredProjectWorkDir:       requiredOPESProjectDir,
				EvidenceRef:                  opesProjectWorkDirGuardEvidenceV0,
				AllowedLocalWriteSetPrefixes: []string{opesLocalExternalWriteSetPrefixV0},
			},
		},
	}
}
