package orquestanativesmoketool

import (
	"reflect"
	"testing"
)

func TestToolNameV0EsEstable(t *testing.T) {
	if ToolNameV0 != "orquesta.native.smoke.v0" {
		t.Fatalf("ToolNameV0=%q", ToolNameV0)
	}
}

func TestExecuteV0NormalizaValor(t *testing.T) {
	result := ExecuteV0(InputV0{Value: "  Hola\tMUNDO  "})
	if result.ToolName != ToolNameV0 {
		t.Fatalf("ToolName=%q", result.ToolName)
	}
	if result.NormalizedValue != "hola mundo" {
		t.Fatalf("NormalizedValue=%q", result.NormalizedValue)
	}
}

func TestExecuteV0EsDeterminista(t *testing.T) {
	input := InputV0{Value: " Valor  ESTABLE "}
	first := ExecuteV0(input)
	second := ExecuteV0(input)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
}
