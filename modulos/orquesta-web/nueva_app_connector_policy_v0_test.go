package orquestaweb

import (
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestWebNuevaAppErrorViewModelV0PreservesConnectorPolicyIssue(t *testing.T) {
	vm := NewWebNuevaAppErrorViewModelV0("req-connector", "es", []orquestafactory.ValidationIssue{{
		Code:    orquestafactory.ErrConectorRequeridoNoDisponible,
		Field:   "integraciones.0.nombre",
		Message: "la integracion debe declarar una capacidad",
	}})

	if len(vm.ErroresPublicos) != 1 {
		t.Fatalf("errores_publicos=%+v", vm.ErroresPublicos)
	}
	err := vm.ErroresPublicos[0]
	if err.Code != orquestafactory.ErrConectorRequeridoNoDisponible ||
		err.Field != "integraciones.0.nombre" ||
		err.Message != "la integracion debe declarar una capacidad" {
		t.Fatalf("connector policy issue reinterpreted: %+v", err)
	}
}
