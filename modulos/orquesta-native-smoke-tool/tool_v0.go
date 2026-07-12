// Package orquestanativesmoketool provides a small, pure native smoke tool.
package orquestanativesmoketool

import "strings"

// ToolNameV0 is the stable public name for the native smoke tool.
const ToolNameV0 = "orquesta.native.smoke.v0"

// InputV0 is the input accepted by ExecuteV0.
type InputV0 struct {
	Value string
}

// ResultV0 is the deterministic result returned by ExecuteV0.
type ResultV0 struct {
	ToolName        string
	NormalizedValue string
}

// ExecuteV0 normalizes input without performing I/O or using external state.
func ExecuteV0(input InputV0) ResultV0 {
	return ResultV0{
		ToolName:        ToolNameV0,
		NormalizedValue: strings.ToLower(strings.Join(strings.Fields(input.Value), " ")),
	}
}
