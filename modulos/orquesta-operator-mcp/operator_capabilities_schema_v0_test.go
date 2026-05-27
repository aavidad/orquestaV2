package orquestaoperatormcp

import (
	"reflect"
	"strings"
	"testing"
)

func TestOperatorMCPCapabilitiesV0AlineanShapesConDTOs(t *testing.T) {
	capabilities := NewOperatorMCPCapabilitiesV0()
	dtoByTool := map[string]any{
		OperatorMCPStatusToolNameV0:    OperatorStatusQueryV0{},
		OperatorMCPBurstToolNameV0:     OperatorSupervisedBurstRequestV0{},
		OperatorMCPOutboxToolNameV0:    OperatorPendingOutboxQueryV0{},
		OperatorMCPDirectedQueryToolV0: OperatorDirectedQueryV0{},
	}
	for _, tool := range capabilities.Tools {
		dto, ok := dtoByTool[tool.Name]
		if !ok {
			t.Fatalf("tool operador sin DTO canonico: %s", tool.Name)
		}
		fields := jsonFieldsOperatorMCPTestV0(dto)
		assertSameStringsOperatorMCPTestV0(t, tool.Name, tool.InputRefs, fields)
		for _, required := range tool.RequiredInputRefs {
			if !containsStringOperatorMCPTestV0(fields, required) {
				t.Fatalf("%s required stale: %s", tool.Name, required)
			}
		}
		if !containsStringOperatorMCPTestV0(tool.PublicErrors, ErrOperatorMCPPortUnavailableV0) ||
			strings.TrimSpace(tool.InputShape) == "" ||
			strings.TrimSpace(tool.OutputShape) == "" {
			t.Fatalf("descriptor incompleto: %+v", tool)
		}
	}
}

func jsonFieldsOperatorMCPTestV0(dto any) []string {
	typ := reflect.TypeOf(dto)
	out := []string{}
	for idx := 0; idx < typ.NumField(); idx++ {
		field := typ.Field(idx)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			out = append(out, name)
		}
	}
	return out
}

func assertSameStringsOperatorMCPTestV0(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s fields=%v want=%v", label, got, want)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("%s fields=%v want=%v", label, got, want)
		}
	}
}

func containsStringOperatorMCPTestV0(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
