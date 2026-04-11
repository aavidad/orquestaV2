package db

import (
	"path/filepath"
	"strings"
	"testing"

	"orquesta/runtimeagente"
)

func TestMergeResumePayloadPerfilEjecucionPreservaYSobrescribe(t *testing.T) {
	prev := MergeResumePayloadPerfilEjecucion("", "implementacion", "gemma4:26b", "medium")
	perfil, modelo, razonamiento := ResumePayloadPerfilEjecucion(prev)
	if perfil != "implementacion" || modelo != "gemma4:26b" || razonamiento != "medium" {
		t.Fatalf("perfil persistido inesperado: perfil=%q modelo=%q razonamiento=%q", perfil, modelo, razonamiento)
	}

	siguiente := MergeResumePayloadPerfilEjecucion(prev, "", "qwen2.5-coder:7b", "")
	perfil, modelo, razonamiento = ResumePayloadPerfilEjecucion(siguiente)
	if perfil != "implementacion" || modelo != "qwen2.5-coder:7b" || razonamiento != "medium" {
		t.Fatalf("perfil fusionado inesperado: perfil=%q modelo=%q razonamiento=%q", perfil, modelo, razonamiento)
	}
}

func TestPayloadJSONDesdePlanYResumePersistePerfilEjecucion(t *testing.T) {
	got := payloadJSONDesdePlanYResume(&runtimeagente.LaunchPlan{
		PerfilTarea:  "implementacion",
		Modelo:       "gemma4:26b",
		Razonamiento: "medium",
	}, runtimeagente.ResumeContext{
		ResumePayloadJSON: `{"persist":"ok"}`,
	})
	perfil, modelo, razonamiento := ResumePayloadPerfilEjecucion(got)
	if perfil != "implementacion" || modelo != "gemma4:26b" || razonamiento != "medium" {
		t.Fatalf("payload sin perfil de ejecución persistido: %s", got)
	}
	if ParseResumePayloadEnvelope(got)["persist"] != "ok" {
		t.Fatalf("payload no debería perder claves previas: %s", got)
	}
}

func TestPrepararStartRuntimeOrderReutilizaPerfilPersistidoEnResume(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	conector, err := GetConector("ollama-cli")
	if err != nil || conector == nil {
		t.Fatalf("get conector ollama-cli: %+v err=%v", conector, err)
	}
	resumePayload := MergeResumePayloadPerfilEjecucion("", "implementacion", "gemma4:26b", "medium")
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Gemma1",
		ConectorID:        &conector.ID,
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "ollama-cli",
		ResumePayloadJSON: resumePayload,
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	_, _, _, _, resume, _, plan, err := prepararStartRuntimeOrder("Gemma1", "orquestador", 0, "", "", "", "")
	if err != nil {
		t.Fatalf("prepararStartRuntimeOrder: %v", err)
	}
	if plan == nil {
		t.Fatal("faltaba plan de arranque")
	}
	if plan.Modelo != "gemma4:26b" || plan.Razonamiento != "medium" || plan.PerfilTarea != "implementacion" {
		t.Fatalf("plan sin perfil persistido: %+v", plan)
	}
	perfil, modelo, razonamiento := ResumePayloadPerfilEjecucion(resume.ResumePayloadJSON)
	if perfil != "implementacion" || modelo != "gemma4:26b" || razonamiento != "medium" {
		t.Fatalf("resume sin perfil persistido: %s", resume.ResumePayloadJSON)
	}
}

func TestPrepararStartRuntimeOrderPrefiereConectorPoolLocalCompartidoOllama(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := GuardarPool(&PoolCapacidad{
		Slug:           "ollama-gemma4",
		Proveedor:      "Ollama",
		Runtime:        "ollama",
		Plan:           "local",
		CapacidadTotal: 1,
		MetadataJSON:   `{"conector_canonico":"ollama_pool_local","conector_compatibilidad":"ollama-cli","slots_maximos":1}`,
		Activo:         true,
	}); err != nil {
		t.Fatalf("guardar pool: %v", err)
	}
	if _, err := GuardarPoolModelo("ollama-gemma4", &PoolModelo{
		ModelSlug:     "gemma4:26b",
		Activo:        true,
		Prioridad:     10,
		CosteRelativo: 1,
	}); err != nil {
		t.Fatalf("guardar modelo pool: %v", err)
	}
	if _, err := GuardarPoliticaModelo(&PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "implementacion",
		PerfilTarea:     "implementacion",
		PoolSlug:        "ollama-gemma4",
		ModelSlug:       "gemma4:26b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("guardar politica: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Gemma1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "ollama-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	_, _, conector, _, _, _, plan, err := prepararStartRuntimeOrder("Gemma1", "orquestador", 0, "", "", "", "implementacion")
	if err != nil {
		t.Fatalf("prepararStartRuntimeOrder: %v", err)
	}
	if conector == nil || conector.Slug != "ollama_pool_local" {
		t.Fatalf("conector inesperado: %+v", conector)
	}
	if plan == nil || plan.Transporte != "api" {
		t.Fatalf("plan inesperado: %+v", plan)
	}
}

func TestPrepararStartRuntimeOrderPoolLocalConservaResumenContinuidadPersistido(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := GuardarPool(&PoolCapacidad{
		Slug:           "ollama-gemma4",
		Proveedor:      "Ollama",
		Runtime:        "ollama",
		Plan:           "local",
		CapacidadTotal: 1,
		MetadataJSON:   `{"conector_canonico":"ollama_pool_local","conector_compatibilidad":"ollama-cli","slots_maximos":1}`,
		Activo:         true,
	}); err != nil {
		t.Fatalf("guardar pool: %v", err)
	}
	if _, err := GuardarPoolModelo("ollama-gemma4", &PoolModelo{
		ModelSlug:     "gemma4:26b",
		Activo:        true,
		Prioridad:     10,
		CosteRelativo: 1,
	}); err != nil {
		t.Fatalf("guardar modelo pool: %v", err)
	}
	if _, err := GuardarPoliticaModelo(&PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "implementacion",
		PerfilTarea:     "implementacion",
		PoolSlug:        "ollama-gemma4",
		ModelSlug:       "gemma4:26b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("guardar politica: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	conector, err := GetConector("ollama_pool_local")
	if err != nil || conector == nil {
		t.Fatalf("get conector ollama_pool_local: %+v err=%v", conector, err)
	}
	resumePayload := MergeResumePayloadPerfilEjecucion("", "implementacion", "gemma4:26b", "high")
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Gemma1",
		ConectorID:         &conector.ID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "ollama_pool_local",
		ResumePayloadJSON:  resumePayload,
		ResumenContinuidad: "seguir con firma pendiente y helper ya extraido",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	_, _, _, _, resume, _, _, err := prepararStartRuntimeOrder("Gemma1", "orquestador", 0, "", "", "", "implementacion")
	if err != nil {
		t.Fatalf("prepararStartRuntimeOrder: %v", err)
	}
	if got := resume.ResumenContinuidad; !strings.Contains(got, "seguir con firma pendiente y helper ya extraido") {
		t.Fatalf("resumen continuidad inesperado: %+v", resume)
	}
}
