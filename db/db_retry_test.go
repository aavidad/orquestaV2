package db

import (
	"testing"
	"time"
)

func TestPersistenciaBusyDebeReintentarRespetaBudget(t *testing.T) {
	prevBudget := persistenciaBusyBudget
	defer func() { persistenciaBusyBudget = prevBudget }()
	persistenciaBusyBudget = 250 * time.Millisecond

	inicio := time.Now()
	if !persistenciaBusyDebeReintentar(inicio, 0) {
		t.Fatalf("deberia permitir el primer reintento dentro del budget")
	}
	if persistenciaBusyDebeReintentar(inicio.Add(-300*time.Millisecond), 0) {
		t.Fatalf("no deberia seguir reintentando fuera del budget")
	}
}

func TestPersistenciaBusyDebeReintentarRespetaMaxIntentos(t *testing.T) {
	if persistenciaBusyDebeReintentar(time.Now(), persistenciaBusyMaxIntentos-1) {
		t.Fatalf("no deberia reintentar al agotar max intentos")
	}
}
