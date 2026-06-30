package orquestaweb

import (
	"strings"
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestWebNuevaAppDirectorErrorViewModelV0NoExponeMensajeTecnico(t *testing.T) {
	form := WebNuevaAppFormV0{
		RequestID: "req-director-error",
		Locale:    "es",
		Nombre:    "Agenda",
		Objetivo:  "Coordinar reservas",
		TipoApp:   "web",
	}
	result := WebArrancarDirectorAppResultV0{
		Errores: []WebNuevaAppIssueV0{{
			Code:    orquestafactory.ErrAppSpecInvalida,
			Field:   "nombre",
			Message: "required field at /home/alberto/.config/token",
		}},
	}

	vm := NewWebNuevaAppDirectorErrorViewModelV0(form, result)

	if len(vm.ErroresPublicos) != 1 {
		t.Fatalf("errores_publicos=%+v", vm.ErroresPublicos)
	}
	err := vm.ErroresPublicos[0]
	if err.Code != orquestafactory.ErrAppSpecInvalida || err.Field != "nombre" || err.Message != "Completa este campo." {
		t.Fatalf("error publico inesperado: %+v", err)
	}
	for _, forbidden := range []string{"required field", "/home/alberto", ".config/token"} {
		if strings.Contains(err.Message, forbidden) {
			t.Fatalf("mensaje tecnico filtrado %q en %+v", forbidden, err)
		}
	}
}

func TestWebNuevaAppGoalPreviewErrorViewModelV0NoExponeMensajeTecnico(t *testing.T) {
	form := WebNuevaAppFormV0{
		RequestID: "req-goal-preview-error",
		Locale:    "es",
		Nombre:    "Agenda",
		Objetivo:  "Coordinar reservas",
		TipoApp:   "web",
	}
	result := WebPreviewDirectorAppResultV0{
		Errores: []WebNuevaAppIssueV0{{
			Code:    orquestafactory.ErrAppSpecInvalida,
			Field:   "objetivo",
			Message: "required field at /home/alberto/.config/token",
		}},
	}

	vm := NewWebNuevaAppGoalPreviewErrorViewModelV0(form, result)

	if len(vm.ErroresPublicos) != 1 {
		t.Fatalf("errores_publicos=%+v", vm.ErroresPublicos)
	}
	err := vm.ErroresPublicos[0]
	if err.Code != orquestafactory.ErrAppSpecInvalida || err.Field != "objetivo" || err.Message != "Completa este campo." {
		t.Fatalf("error publico inesperado: %+v", err)
	}
	for _, forbidden := range []string{"required field", "/home/alberto", ".config/token"} {
		if strings.Contains(err.Message, forbidden) {
			t.Fatalf("mensaje tecnico filtrado %q en %+v", forbidden, err)
		}
	}
}
