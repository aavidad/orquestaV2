package microprogramacionapp

import (
	"strings"
	"testing"
)

type fakeRepositorio struct {
	lastCreated *EspecificacionFuncion
	nextID      int64
	byID        map[int64]*EspecificacionFuncion
}

type fakeDespachador struct {
	lastSolicitud *SolicitudDespachoMicrotarea
	resultado     *MicrotareaDespachada
	err           error
}

func (f *fakeRepositorio) CrearEspecificacionFuncion(spec *EspecificacionFuncion) (int64, error) {
	f.lastCreated = spec
	if f.nextID == 0 {
		f.nextID = 1
	}
	return f.nextID, nil
}

func (f *fakeRepositorio) ObtenerEspecificacionFuncion(id int64) (*EspecificacionFuncion, error) {
	if f.byID == nil {
		return nil, nil
	}
	return f.byID[id], nil
}

func (f *fakeRepositorio) ListarEspecificacionesFuncion(filtro FiltroEspecificaciones) ([]*EspecificacionFuncion, error) {
	return nil, nil
}

func (f *fakeDespachador) DespacharMicrotarea(solicitud SolicitudDespachoMicrotarea) (*MicrotareaDespachada, error) {
	f.lastSolicitud = &solicitud
	if f.err != nil {
		return nil, f.err
	}
	if f.resultado != nil {
		return f.resultado, nil
	}
	return &MicrotareaDespachada{
		EspecificacionID: solicitud.EspecificacionID,
		ProyectoID:       solicitud.ProyectoID,
		AgenteDestino:    solicitud.AgenteDestino,
		RuntimeOrderID:   77,
		Microtarea:       solicitud.Microtarea,
	}, nil
}

func TestServicioCrearNormalizaYFuerzaArchivoEnWriteSet(t *testing.T) {
	repo := &fakeRepositorio{}
	service := NewService(repo)
	id, err := service.Crear(EntradaCrearEspecificacion{
		Titulo:            "  ",
		ArchivoObjetivo:   "./cmd/api.go",
		SimboloObjetivo:   "apiHandler",
		Descripcion:       " Implementar el handler ",
		WriteSet:          []string{"cmd/api.go", "cmd/api.go", " cmd/runtime.go "},
		TestsObligatorios: []string{"go test ./cmd -run TestAPI", "go test ./cmd -run TestAPI"},
		CreadoPor:         "",
	})
	if err != nil {
		t.Fatalf("Crear: %v", err)
	}
	if id != 1 {
		t.Fatalf("id inesperado: %d", id)
	}
	if repo.lastCreated == nil {
		t.Fatal("faltaba especificacion creada")
	}
	if repo.lastCreated.ArchivoObjetivo != "cmd/api.go" {
		t.Fatalf("archivo objetivo inesperado: %q", repo.lastCreated.ArchivoObjetivo)
	}
	if repo.lastCreated.Titulo != "cmd/api.go::apiHandler" {
		t.Fatalf("titulo inesperado: %q", repo.lastCreated.Titulo)
	}
	if len(repo.lastCreated.WriteSet) != 2 || repo.lastCreated.WriteSet[0] != "cmd/api.go" || repo.lastCreated.WriteSet[1] != "cmd/runtime.go" {
		t.Fatalf("write_set inesperado: %#v", repo.lastCreated.WriteSet)
	}
	if len(repo.lastCreated.TestsObligatorios) != 1 {
		t.Fatalf("tests obligatorios inesperados: %#v", repo.lastCreated.TestsObligatorios)
	}
	if repo.lastCreated.Estado != EstadoEspecificacionActiva {
		t.Fatalf("estado inesperado: %q", repo.lastCreated.Estado)
	}
	if repo.lastCreated.CreadoPor != "alberto" {
		t.Fatalf("creado_por inesperado: %q", repo.lastCreated.CreadoPor)
	}
}

func TestServicioCrearRechazaRutaAbsolutaYSinTests(t *testing.T) {
	repo := &fakeRepositorio{}
	service := NewService(repo)
	if _, err := service.Crear(EntradaCrearEspecificacion{
		ArchivoObjetivo:   "/tmp/api.go",
		SimboloObjetivo:   "apiHandler",
		Descripcion:       "desc",
		WriteSet:          []string{"/tmp/api.go"},
		TestsObligatorios: []string{"go test ./cmd"},
	}); err == nil {
		t.Fatal("debería rechazar archivo absoluto")
	}
	if _, err := service.Crear(EntradaCrearEspecificacion{
		ArchivoObjetivo: "cmd/api.go",
		SimboloObjetivo: "apiHandler",
		Descripcion:     "desc",
		WriteSet:        []string{"cmd/api.go"},
	}); err == nil {
		t.Fatal("debería exigir tests obligatorios")
	}
}

func TestServicioCrearRechazaDependenciasSolapadas(t *testing.T) {
	repo := &fakeRepositorio{}
	service := NewService(repo)
	_, err := service.Crear(EntradaCrearEspecificacion{
		ArchivoObjetivo:        "cmd/api.go",
		SimboloObjetivo:        "apiHandler",
		Descripcion:            "desc",
		WriteSet:               []string{"cmd/api.go"},
		TestsObligatorios:      []string{"go test ./cmd"},
		DependenciasPermitidas: []string{"orquesta/db"},
		DependenciasProhibidas: []string{" ORQUESTA/DB "},
	})
	if err == nil {
		t.Fatal("debería rechazar dependencias solapadas")
	}
}

func TestServicioEmitirConstruyeMicrotareaCerrada(t *testing.T) {
	repo := &fakeRepositorio{
		byID: map[int64]*EspecificacionFuncion{
			31: {
				ID:                     31,
				Titulo:                 "runtimeagente/driver.go::BuildSpec",
				ArchivoObjetivo:        "runtimeagente/driver.go",
				SimboloObjetivo:        "BuildSpec",
				Descripcion:            "Construir el spec del siguiente slice",
				Precondiciones:         []string{"Existe driver base"},
				Postcondiciones:        []string{"No rompe la firma pública"},
				DependenciasPermitidas: []string{"orquesta/runtimeagente"},
				DependenciasProhibidas: []string{"orquesta/db"},
				TestsObligatorios:      []string{"go test ./runtimeagente -run TestBuildSpec"},
				WriteSet:               []string{"runtimeagente/driver.go", "runtimeagente/driver_test.go"},
				FormatoSalida:          "patch+evidencia",
			},
		},
	}
	service := NewService(repo)
	salida, err := service.Emitir(31, EntradaEmitirMicrotarea{Contexto: "Trabaja solo sobre la tarea #505."})
	if err != nil {
		t.Fatalf("Emitir: %v", err)
	}
	if salida == nil || salida.EspecificacionID != 31 {
		t.Fatalf("salida inesperada: %+v", salida)
	}
	for _, token := range []string{
		"MICROTAREA CERRADA",
		"SIMBOLO: BuildSpec",
		"ARCHIVO: runtimeagente/driver.go",
		"WRITE_SET: runtimeagente/driver.go, runtimeagente/driver_test.go",
		"TESTS: go test ./runtimeagente -run TestBuildSpec",
		"CONTEXTO: Trabaja solo sobre la tarea #505.",
	} {
		if !strings.Contains(salida.Mensaje, token) {
			t.Fatalf("mensaje sin %q:\n%s", token, salida.Mensaje)
		}
	}
}

func TestServicioEmitirYDespacharUsaDespachadorCanonico(t *testing.T) {
	proyectoID := int64(99)
	repo := &fakeRepositorio{
		byID: map[int64]*EspecificacionFuncion{
			31: {
				ID:                31,
				ProyectoID:        &proyectoID,
				Titulo:            "runtimeagente/driver.go::BuildSpec",
				ArchivoObjetivo:   "runtimeagente/driver.go",
				SimboloObjetivo:   "BuildSpec",
				Descripcion:       "Construir el spec del siguiente slice",
				TestsObligatorios: []string{"go test ./runtimeagente -run TestBuildSpec"},
				WriteSet:          []string{"runtimeagente/driver.go", "runtimeagente/driver_test.go"},
				FormatoSalida:     "patch+evidencia",
			},
		},
	}
	despachador := &fakeDespachador{}
	service := NewService(repo)
	service.SetDespachador(despachador)

	resultado, err := service.EmitirYDespachar(31, EntradaDespacharMicrotarea{
		AgenteDestino: "Gemma1",
		Contexto:      "Trabaja solo sobre esta microtarea.",
	})
	if err != nil {
		t.Fatalf("EmitirYDespachar: %v", err)
	}
	if resultado == nil || resultado.RuntimeOrderID != 77 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if despachador.lastSolicitud == nil {
		t.Fatal("faltaba solicitud despachada")
	}
	if despachador.lastSolicitud.AgenteDestino != "Gemma1" {
		t.Fatalf("agente destino inesperado: %+v", despachador.lastSolicitud)
	}
	if despachador.lastSolicitud.ProyectoID == nil || *despachador.lastSolicitud.ProyectoID != proyectoID {
		t.Fatalf("proyecto de despacho inesperado: %+v", despachador.lastSolicitud)
	}
	if despachador.lastSolicitud.Microtarea == nil || !strings.Contains(despachador.lastSolicitud.Microtarea.Mensaje, "Trabaja solo sobre esta microtarea.") {
		t.Fatalf("microtarea sin contexto esperado: %+v", despachador.lastSolicitud.Microtarea)
	}
}

func TestServicioValidarEntregaAceptaContratoValido(t *testing.T) {
	repo := &fakeRepositorio{
		byID: map[int64]*EspecificacionFuncion{
			51: {
				ID:                     51,
				Titulo:                 "runtimesapp/service.go::ResolveControlHandle",
				ArchivoObjetivo:        "runtimesapp/service.go",
				SimboloObjetivo:        "ResolveControlHandle",
				Descripcion:            "Resolver el handle de control canónico",
				DependenciasPermitidas: []string{"orquesta/runtimesapp", "orquesta/sesionesapp"},
				DependenciasProhibidas: []string{"orquesta/db"},
				TestsObligatorios:      []string{"go test ./runtimesapp -run TestResolveControlHandle -count=1"},
				WriteSet:               []string{"runtimesapp/service.go", "runtimesapp/service_test.go"},
			},
		},
	}
	service := NewService(repo)
	resultado, err := service.ValidarEntrega(51, EntradaValidarEntrega{
		SimboloEntregado:   "ResolveControlHandle",
		WriteSetEntregado:  []string{"runtimesapp/service.go", "runtimesapp/service_test.go"},
		DependenciasUsadas: []string{"orquesta/runtimesapp"},
		TestsEjecutados:    []string{"go test ./runtimesapp -run TestResolveControlHandle -count=1"},
		Evidencia:          "go test ./runtimesapp -run TestResolveControlHandle -count=1 => ok",
	})
	if err != nil {
		t.Fatalf("ValidarEntrega: %v", err)
	}
	if resultado == nil || !resultado.Valida || len(resultado.Hallazgos) != 0 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
}

func TestServicioValidarEntregaToleraDependenciaCortaOCompleta(t *testing.T) {
	repo := &fakeRepositorio{
		byID: map[int64]*EspecificacionFuncion{
			53: {
				ID:                     53,
				Titulo:                 "runtimesapp/service.go::ResolveControlHandle",
				ArchivoObjetivo:        "runtimesapp/service.go",
				SimboloObjetivo:        "ResolveControlHandle",
				Descripcion:            "Resolver el handle de control canónico",
				DependenciasPermitidas: []string{"runtimesapp"},
				TestsObligatorios:      []string{"go test ./runtimesapp -run TestResolveControlHandle -count=1"},
				WriteSet:               []string{"runtimesapp/service.go"},
			},
		},
	}
	service := NewService(repo)
	resultado, err := service.ValidarEntrega(53, EntradaValidarEntrega{
		SimboloEntregado:   "ResolveControlHandle",
		WriteSetEntregado:  []string{"runtimesapp/service.go"},
		DependenciasUsadas: []string{"orquesta/runtimesapp"},
		TestsEjecutados:    []string{"go test ./runtimesapp -run TestResolveControlHandle -count=1"},
		Evidencia:          "ok",
	})
	if err != nil {
		t.Fatalf("ValidarEntrega: %v", err)
	}
	if resultado == nil || !resultado.Valida {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
}

func TestServicioValidarEntregaDetectaDeriva(t *testing.T) {
	repo := &fakeRepositorio{
		byID: map[int64]*EspecificacionFuncion{
			52: {
				ID:                     52,
				Titulo:                 "cmd/api.go::apiHandlerMicro",
				ArchivoObjetivo:        "cmd/api.go",
				SimboloObjetivo:        "apiHandlerMicro",
				Descripcion:            "Implementar solo el handler micro",
				DependenciasPermitidas: []string{"orquesta/runtimesapp"},
				DependenciasProhibidas: []string{"orquesta/db"},
				TestsObligatorios:      []string{"go test ./cmd -run TestAPIMicro -count=1"},
				WriteSet:               []string{"cmd/api.go", "cmd/api_test.go"},
			},
		},
	}
	service := NewService(repo)
	resultado, err := service.ValidarEntrega(52, EntradaValidarEntrega{
		SimboloEntregado:   "apiHandlerOtro",
		WriteSetEntregado:  []string{"cmd/api.go", "db/agentes.go"},
		DependenciasUsadas: []string{"orquesta/db", "orquesta/otra"},
		TestsEjecutados:    []string{"go test ./cmd -run TestOtra -count=1"},
		TestsFallidos:      []string{"go test ./cmd -run TestAPIMicro -count=1"},
	})
	if err != nil {
		t.Fatalf("ValidarEntrega: %v", err)
	}
	if resultado == nil || resultado.Valida {
		t.Fatalf("debería ser inválida: %+v", resultado)
	}
	wantCodes := []string{
		"simbolo_distinto",
		"evidencia_requerida",
		"fuera_de_write_set",
		"dependencia_prohibida",
		"dependencia_no_permitida",
		"test_obligatorio_no_ejecutado",
		"test_obligatorio_fallido",
	}
	for _, code := range wantCodes {
		if !containsHallazgoCode(resultado.Hallazgos, code) {
			t.Fatalf("faltaba hallazgo %q en %+v", code, resultado.Hallazgos)
		}
	}
}

func containsHallazgoCode(items []HallazgoValidacionEntrega, code string) bool {
	for _, item := range items {
		if item.Codigo == code {
			return true
		}
	}
	return false
}
