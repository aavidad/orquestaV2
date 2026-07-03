package orquestamcp

import (
	"encoding/json"
	"strings"
)

const mcpTransportToolShapeInlineMaxV0 = 160

type mcpTransportToolEnvelopeJSONV0 struct {
	Name            string                        `json:"name"`
	Version         string                        `json:"version"`
	ResourceURI     string                        `json:"resource_uri"`
	InputShape      string                        `json:"input_shape"`
	OutputShape     string                        `json:"output_shape"`
	Mode            string                        `json:"mode"`
	OutputBudget    MCPTransportOutputBudgetV0    `json:"output_budget"`
	ExecutionBudget MCPTransportExecutionBudgetV0 `json:"execution_budget"`
}

func (tool MCPTransportToolEnvelopeV0) MarshalJSON() ([]byte, error) {
	return json.Marshal(mcpTransportToolEnvelopeJSONV0{
		Name:            tool.Name,
		Version:         tool.Version,
		ResourceURI:     tool.ResourceURI,
		InputShape:      mcpTransportToolShapeForJSONV0(tool.Name, tool.ResourceURI, tool.InputShape, "input"),
		OutputShape:     mcpTransportToolShapeForJSONV0(tool.Name, tool.ResourceURI, tool.OutputShape, "output"),
		Mode:            tool.Mode,
		OutputBudget:    tool.OutputBudget,
		ExecutionBudget: tool.ExecutionBudget,
	})
}

func mcpTransportToolShapeForJSONV0(toolName string, resourceURI string, shape string, kind string) string {
	trimmed := strings.TrimSpace(shape)
	if len(trimmed) <= mcpTransportToolShapeInlineMaxV0 {
		return trimmed
	}
	ref := strings.TrimSpace(resourceURI)
	if ref == "" {
		ref = strings.TrimSpace(toolName)
	}
	if ref == "" {
		ref = "orquesta://contracts/unknown"
	}
	return "shape_ref:" + ref + "#" + kind
}
