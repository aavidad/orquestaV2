package orquestaoperatormcp

import (
	"errors"
	"testing"
)

func TestOperatorMCPPublicErrorCatalogV0CubreContratoLocal(t *testing.T) {
	for _, code := range operatorToolPublicErrorsV0() {
		if !OperatorMCPPublicErrorCodeKnownV0(code) {
			t.Fatalf("codigo operador fuera de catalogo: %s", code)
		}
		entry, ok := OperatorMCPPublicErrorDescriptorV0(code)
		if !ok || entry.I18nKey == "" || entry.Severity == "" {
			t.Fatalf("metadata operador incompleta: %+v", entry)
		}
	}
}

func TestPublicOperatorMCPErrorCodeV0NoPropagaErrorNoCatalogado(t *testing.T) {
	if code, ok := PublicOperatorMCPErrorCodeV0(NewOperatorMCPPublicErrorV0(ErrOperatorMCPPortUnavailableV0)); !ok || code != ErrOperatorMCPPortUnavailableV0 {
		t.Fatalf("error publico catalogado: code=%q ok=%v", code, ok)
	}
	if code, ok := PublicOperatorMCPErrorCodeV0(errors.New("sql password=/tmp/private")); ok || code != "" {
		t.Fatalf("error no catalogado filtrado: code=%q ok=%v", code, ok)
	}
}
