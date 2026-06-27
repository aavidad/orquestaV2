package orquestamcp

import (
	"context"
	"encoding/json"

	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
)

func mcpExternalWorkDryRunTransportHandlerV0(
	executor MCPTransportExternalWorkDryRunExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPExternalWorkDryRunToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if executor == nil {
			result := BuildExternalWorkDryRunV0(input, orquestaExternalWorkDryRunDefaultConfigV0())
			return json.Marshal(result)
		}
		result, err := executor.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}

func orquestaExternalWorkDryRunDefaultConfigV0() orquestaexternalworkrun.StartExternalWorkRunConfigV0 {
	return orquestaexternalworkrun.StartExternalWorkRunConfigV0{}
}
