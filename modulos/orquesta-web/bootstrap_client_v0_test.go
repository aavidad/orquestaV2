package orquestaweb

import (
	"context"
	"errors"
	"testing"

	orquestadirector "orquesta/modulos/orquesta-director"
)

func TestLocalBootstrapProyectoDesdeAppSpecClientV0DelegaEnPuertoDirector(t *testing.T) {
	called := 0
	client := LocalBootstrapProyectoDesdeAppSpecClientV0{
		Bootstrap: func(cmd orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0) (orquestadirector.BootstrapProyectoDesdeAppSpecResultV0, error) {
			called++
			if cmd.IdempotencyKey != " idem-web-1 " || cmd.RequestID != "req-web-1" {
				t.Fatalf("cmd no preservado: %+v", cmd)
			}
			return bootstrapResultForWebV0(), nil
		},
	}

	vm, err := client.BootstrapProyectoDesdeAppSpec(context.Background(), orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0{
		IdempotencyKey: " idem-web-1 ",
		RequestID:      "req-web-1",
	})
	if err != nil {
		t.Fatalf("BootstrapProyectoDesdeAppSpec: %v", err)
	}

	if called != 1 {
		t.Fatalf("called=%d", called)
	}
	if vm.Estado != WebBootstrapProyectoEstadoRegistrado || vm.ProjectRef != "projectref_123" {
		t.Fatalf("vm=%+v", vm)
	}
}

func TestLocalBootstrapProyectoDesdeAppSpecClientV0ErrorPublicoAViewModel(t *testing.T) {
	client := LocalBootstrapProyectoDesdeAppSpecClientV0{
		Bootstrap: func(orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0) (orquestadirector.BootstrapProyectoDesdeAppSpecResultV0, error) {
			return orquestadirector.BootstrapProyectoDesdeAppSpecResultV0{}, orquestadirector.BootstrapProyectoDesdeAppSpecErrorV0{
				Code:          orquestadirector.ErrDirectorBootstrapInvalidoV0,
				Message:       "idempotency_key requerida",
				Field:         "idempotency_key",
				CorrelationID: "corr-web-1",
			}
		},
	}

	vm, err := client.BootstrapProyectoDesdeAppSpec(context.Background(), orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0{})
	if err != nil {
		t.Fatalf("error publico no debe ser error de cliente: %v", err)
	}
	if vm.Estado != WebBootstrapProyectoEstadoInvalido ||
		len(vm.Errores) != 1 ||
		vm.Errores[0].Code != orquestadirector.ErrDirectorBootstrapInvalidoV0 ||
		vm.Errores[0].Field != "idempotency_key" {
		t.Fatalf("vm error=%+v", vm)
	}
}

func TestLocalBootstrapProyectoDesdeAppSpecClientV0ErrorNoPublicoEstable(t *testing.T) {
	client := LocalBootstrapProyectoDesdeAppSpecClientV0{
		Bootstrap: func(orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0) (orquestadirector.BootstrapProyectoDesdeAppSpecResultV0, error) {
			return orquestadirector.BootstrapProyectoDesdeAppSpecResultV0{}, errors.New("detalle privado")
		},
	}

	_, err := client.BootstrapProyectoDesdeAppSpec(context.Background(), orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0{})
	if !IsWebBootstrapProyectoClientErrorCodeV0(err, WebBootstrapProyectoErrClienteV0) {
		t.Fatalf("error=%v", err)
	}
}
