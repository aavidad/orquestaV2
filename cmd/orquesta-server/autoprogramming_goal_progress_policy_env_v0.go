package main

import orquestamcp "orquesta/modulos/orquesta-mcp"

func serverAutoprogrammingGoalProgressPolicyFromEnvV0() orquestamcp.MCPAutoprogrammingGoalProgressPolicyV0 {
	return orquestamcp.NormalizeMCPAutoprogrammingGoalProgressPolicyV0(
		orquestamcp.MCPAutoprogrammingGoalProgressPolicyV0{
			CheckpointOnlyHighConsumptionTokens: int64(intEnvOrDefaultV0(
				envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0,
				0,
			)),
			CheckpointOnlyMaxWaitSeconds: int64(intEnvOrDefaultV0(
				envAutoprogrammingCheckpointOnlyMaxWaitSecondsV0,
				0,
			)),
			NoCheckpointWarningMaxWaitSeconds: int64(intEnvOrDefaultV0(
				envAutoprogrammingNoCheckpointWarningMaxWaitSecondsV0,
				0,
			)),
		},
	)
}
