package main

import (
	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func externalWorkRunRuntimeCompatibilityGuardConfigFromServerV0(
	config orquestaserver.ConfigV0,
) orquestaappcodexstack.ExternalWorkRunRuntimeCompatibilityGuardConfigV0 {
	return orquestaappcodexstack.ExternalWorkRunRuntimeCompatibilityGuardConfigV0{
		RuntimeIdentity: orquestaappcodexstack.ExternalWorkRunRuntimeIdentityV0{
			BinarySHA256: config.RuntimeIdentity.BinarySHA256,
			BuildRef:     config.RuntimeIdentity.BuildRef,
			CommitRef:    config.RuntimeIdentity.CommitRef,
			EvidenceRefs: config.RuntimeIdentity.EvidenceRefs,
		},
	}
}
