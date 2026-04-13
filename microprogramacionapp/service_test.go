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

type fakeEscritorArchivos struct {
	raiz     string
	archivos []ArchivoEntrega
}

func (f *fakeEscritorArchivos) Escribir(raizProyecto string, archivos []ArchivoEntrega) ([]string, error) {
	f.raiz = raizProyecto
	f.archivos = append([]ArchivoEntrega(nil), archivos...)
	rutas := make([]string, 0, len(archivos))
	for _, archivo := range archivos {
		rutas = append(rutas, archivo.RutaRelativa)
	}
	return rutas, nil
}

type fakeDespachador struct {
	lastSolicitud *SolicitudDespachoMicrotarea
	resultado     *MicrotareaDespachada
	err           error
}

type fakeRecolectorEntregaGit struct {
	entradaAgente      string
	entradaProyectoID  *int64
	entradaProyectoSlug string
	resultado          *EntregaGitCapturada
	err                error
}

func (f *fakeRecolectorEntregaGit) CapturarEntregaGit(agente string, proyectoID *int64, proyectoSlug string) (*EntregaGitCapturada, error) {
	f.entradaAgente = agente
	f.entradaProyectoID = proyectoID
	f.entradaProyectoSlug = proyectoSlug
	if f.err != nil {
		return nil, f.err
	}
	return f.resultado, nil
}

type fakeIntegradorGit struct {
	entrada SolicitudMergeMicroprogramacion
	id      int64
	err     error
}

func (f *fakeIntegradorGit) RegistrarSolicitudMerge(entrada SolicitudMergeMicroprogramacion) (int64, error) {
	f.entrada = entrada
	if f.err != nil {
		return 0, f.err
	}
	if f.id == 0 {
		f.id = 81
	}
	return f.id, nil
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

func TestServicioEmitirConstruyeMicrotareaGitWorktree(t *testing.T) {
	repo := &fakeRepositorio{
		byID: map[int64]*EspecificacionFuncion{
			32: {
				ID:                32,
				Titulo:            "microprogramacionapp/service.go::RegistrarEntregaGit",
				ArchivoObjetivo:   "microprogramacionapp/service.go",
				SimboloObjetivo:   "RegistrarEntregaGit",
				Descripcion:       "Registrar la entrega canonica por git",
				TestsObligatorios: []string{"go test ./microprogramacionapp -run TestServicioRegistrarEntregaGitValidaWriteSetYRegistraMerge -count=1"},
				WriteSet:          []string{"microprogramacionapp/service.go", "microprogramacionapp/service_test.go"},
				FormatoSalida:     "git_worktree+evidencia",
			},
		},
	}
	service := NewService(repo)
	salida, err := service.Emitir(32, EntradaEmitirMicrotarea{})
	if err != nil {
		t.Fatalf("Emitir: %v", err)
	}
	for _, token := range []string{
		"ENTREGA_GIT: trabaja dentro de tu worktree activa",
		"RESPUESTA_ESPERADA: resume breve, tests ejecutados y estado del diff/branch",
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

func TestServicioMaterializarEntregaValidaWriteSetYArchivoObjetivo(t *testing.T) {
	repo := &fakeRepositorio{
		byID: map[int64]*EspecificacionFuncion{
			71: {
				ID:              71,
				ArchivoObjetivo: "microprogramacionapp/service.go",
				SimboloObjetivo: "ResolveControlHandle",
				WriteSet:        []string{"microprogramacionapp/service.go", "microprogramacionapp/service_test.go"},
			},
		},
	}
	escritor := &fakeEscritorArchivos{}
	service := NewService(repo)
	service.SetEscritorArchivos(escritor)

	resultado, err := service.MaterializarEntrega(71, EntradaMaterializarEntrega{
		RaizProyecto: "/tmp/demo",
		Evidencia:    "transcript#123",
		Archivos: []ArchivoEntrega{
			{RutaRelativa: "microprogramacionapp/service.go", Contenido: "package microprogramacionapp\n\nfunc ResolveControlHandle() {}\n"},
			{RutaRelativa: "microprogramacionapp/service_test.go", Contenido: "package microprogramacionapp\n"},
		},
	})
	if err != nil {
		t.Fatalf("MaterializarEntrega: %v", err)
	}
	if resultado == nil || len(resultado.ArchivosMaterializados) != 2 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if escritor.raiz != "/tmp/demo" || len(escritor.archivos) != 2 {
		t.Fatalf("escritor inesperado: raiz=%q archivos=%+v", escritor.raiz, escritor.archivos)
	}
}

func TestServicioMaterializarEntregaRechazaFueraDeWriteSet(t *testing.T) {
	repo := &fakeRepositorio{
		byID: map[int64]*EspecificacionFuncion{
			72: {
				ID:              72,
				ArchivoObjetivo: "microprogramacionapp/service.go",
				WriteSet:        []string{"microprogramacionapp/service.go"},
			},
		},
	}
	service := NewService(repo)
	service.SetEscritorArchivos(&fakeEscritorArchivos{})

	_, err := service.MaterializarEntrega(72, EntradaMaterializarEntrega{
		RaizProyecto: "/tmp/demo",
		Evidencia:    "transcript#124",
		Archivos: []ArchivoEntrega{
			{RutaRelativa: "cmd/api.go", Contenido: "package cmd\n"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "fuera del write_set") {
		t.Fatalf("se esperaba rechazo por write_set, got=%v", err)
	}
}

func TestServicioMaterializarEntregaRechazaGoInvalido(t *testing.T) {
	repo := &fakeRepositorio{
		byID: map[int64]*EspecificacionFuncion{
			73: {
				ID:              73,
				ArchivoObjetivo: "microprogramacionapp/service.go",
				SimboloObjetivo: "ResolveControlHandle",
				WriteSet:        []string{"microprogramacionapp/service.go"},
			},
		},
	}
	escritor := &fakeEscritorArchivos{}
	service := NewService(repo)
	service.SetEscritorArchivos(escritor)

	_, err := service.MaterializarEntrega(73, EntradaMaterializarEntrega{
		RaizProyecto: "/tmp/demo",
		Evidencia:    "transcript#125",
		Archivos: []ArchivoEntrega{
			{RutaRelativa: "microprogramacionapp/service.go", Contenido: "package microprogramacionapp\n\nfunc ResolveControlHandle( {}\n"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "contenido go invalido") {
		t.Fatalf("se esperaba rechazo por sintaxis Go, got=%v", err)
	}
	if len(escritor.archivos) != 0 {
		t.Fatalf("no deberia escribir archivos invalidos: %+v", escritor.archivos)
	}
}

func TestServicioMaterializarEntregaRechazaSiFaltaSimboloObjetivo(t *testing.T) {
	repo := &fakeRepositorio{
		byID: map[int64]*EspecificacionFuncion{
			74: {
				ID:              74,
				ArchivoObjetivo: "microprogramacionapp/service.go",
				SimboloObjetivo: "ResolveControlHandle",
				WriteSet:        []string{"microprogramacionapp/service.go"},
			},
		},
	}
	escritor := &fakeEscritorArchivos{}
	service := NewService(repo)
	service.SetEscritorArchivos(escritor)

	_, err := service.MaterializarEntrega(74, EntradaMaterializarEntrega{
		RaizProyecto: "/tmp/demo",
		Evidencia:    "transcript#126",
		Archivos: []ArchivoEntrega{
			{RutaRelativa: "microprogramacionapp/service.go", Contenido: "package microprogramacionapp\n\ntype Otro struct{}\n"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "no declara el simbolo") {
		t.Fatalf("se esperaba rechazo por simbolo ausente, got=%v", err)
	}
	if len(escritor.archivos) != 0 {
		t.Fatalf("no deberia escribir archivos sin simbolo objetivo: %+v", escritor.archivos)
	}
}

func TestServicioRegistrarEntregaGitValidaWriteSetYRegistraMerge(t *testing.T) {
	proyectoID := int64(22)
	repo := &fakeRepositorio{
		byID: map[int64]*EspecificacionFuncion{
			91: {
				ID:                91,
				ProyectoID:        &proyectoID,
				ArchivoObjetivo:   "microprogramacionapp/service.go",
				SimboloObjetivo:   "RegistrarEntregaGit",
				WriteSet:          []string{"microprogramacionapp/service.go", "microprogramacionapp/service_test.go"},
				TestsObligatorios: []string{"go test ./microprogramacionapp -run TestServicioRegistrarEntregaGitValidaWriteSetYRegistraMerge -count=1"},
			},
		},
	}
	recolector := &fakeRecolectorEntregaGit{
		resultado: &EntregaGitCapturada{
			ProyectoSlug:       "orquestador",
			WorktreeID:         14,
			RutaWorktree:       "/tmp/wt-gemma1",
			Branch:             "orq/orquestador/gemma1/t91",
			BaseRef:            "origin/main",
			HeadCommit:         "abc123",
			ArchivosModificados: []string{"microprogramacionapp/service.go"},
			Diff:               "diff --git a/microprogramacionapp/service.go b/microprogramacionapp/service.go",
		},
	}
	integrador := &fakeIntegradorGit{}
	service := NewService(repo)
	service.SetRecolectorEntregaGit(recolector)
	service.SetIntegradorGit(integrador)

	resultado, err := service.RegistrarEntregaGit(91, EntradaRegistrarEntregaGit{
		Agente:       "Gemma1",
		ProyectoID:   &proyectoID,
		ProyectoSlug: "orquestador",
		SolicitadoPor: "orquesta",
		Evidencia:    "go test ./microprogramacionapp ... => ok",
	})
	if err != nil {
		t.Fatalf("RegistrarEntregaGit: %v", err)
	}
	if resultado == nil || resultado.GitMergeID != 81 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if integrador.entrada.TargetBranch != "main" {
		t.Fatalf("target branch inesperada: %+v", integrador.entrada)
	}
	if integrador.entrada.SourceBranch != "orq/orquestador/gemma1/t91" {
		t.Fatalf("source branch inesperada: %+v", integrador.entrada)
	}
	if !strings.Contains(integrador.entrada.MetadataJSON, `"especificacion_id":91`) {
		t.Fatalf("metadata sin especificacion: %s", integrador.entrada.MetadataJSON)
	}
}

func TestServicioRegistrarEntregaGitRechazaFueraDeWriteSet(t *testing.T) {
	repo := &fakeRepositorio{
		byID: map[int64]*EspecificacionFuncion{
			92: {
				ID:              92,
				ArchivoObjetivo: "microprogramacionapp/service.go",
				SimboloObjetivo: "RegistrarEntregaGit",
				WriteSet:        []string{"microprogramacionapp/service.go"},
			},
		},
	}
	recolector := &fakeRecolectorEntregaGit{
		resultado: &EntregaGitCapturada{
			ProyectoSlug:       "orquestador",
			WorktreeID:         14,
			RutaWorktree:       "/tmp/wt-gemma1",
			Branch:             "orq/orquestador/gemma1/t92",
			BaseRef:            "main",
			HeadCommit:         "abc123",
			ArchivosModificados: []string{"cmd/api.go"},
		},
	}
	service := NewService(repo)
	service.SetRecolectorEntregaGit(recolector)
	service.SetIntegradorGit(&fakeIntegradorGit{})

	if _, err := service.RegistrarEntregaGit(92, EntradaRegistrarEntregaGit{Agente: "Gemma1", ProyectoSlug: "orquestador"}); err == nil {
		t.Fatal("debería rechazar entrega git fuera de write_set")
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
