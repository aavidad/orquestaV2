package orquestamcp

import (
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestNewMCPNuevaAppErrorResultV0PreservesConnectorPolicyIssue(t *testing.T) {
	result := NewMCPNuevaAppErrorResultV0("req-connector", "corr-connector", []orquestafactory.ValidationIssue{{
		Code:    orquestafactory.ErrConectorRequeridoNoDisponible,
		Field:   "integraciones.0.nombre",
		Message: "la integracion debe declarar una capacidad",
	}})

	if len(result.Errores) != 1 {
		t.Fatalf("errores=%+v", result.Errores)
	}
	err := result.Errores[0]
	if err.Code != orquestafactory.ErrConectorRequeridoNoDisponible ||
		err.Field != "integraciones.0.nombre" ||
		err.Message != "la integracion debe declarar una capacidad" {
		t.Fatalf("connector policy issue reinterpreted: %+v", err)
	}
}
