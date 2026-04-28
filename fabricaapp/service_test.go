package fabricaapp

import (
	"strings"
	"testing"
)

func TestGenerateWebAPIBacklog(t *testing.T) {
	svc := NewService()

	result, err := svc.Generate(AppSpec{
		Nombre:      "Expedientes",
		Descripcion: "Gestion integral de expedientes",
		Tipo:        "web_api",
		Database:    true,
		Auth:        true,
		Docker:      true,
		I18n:        true,
		Idiomas:     []string{"es", "en", "fr"},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(result.Tasks) < 8 {
		t.Fatalf("se esperaban suficientes tareas base, got=%d", len(result.Tasks))
	}

	assertHasTask(t, result, "briefing")
	assertHasTask(t, result, "investigacion")
	assertHasTask(t, result, "servicio_residente_minimo")
	assertHasTask(t, result, "arquitectura")
	assertHasTask(t, result, "baseline_profesional")
	assertHasTask(t, result, "persistencia")
	assertHasTask(t, result, "api_base")
	assertHasTask(t, result, "frontend_base")
	assertHasTask(t, result, "ui_temas_branding")
	assertHasTask(t, result, "auth")
	assertHasTask(t, result, "i18n")
	assertHasTask(t, result, "qa_smoke")
	assertHasTask(t, result, "documentacion")
	assertHasTask(t, result, "docker")
	assertHasTask(t, result, "entrega_final")

	i18n := mustTask(t, result, "i18n")
	if len(i18n.Dependencias) == 0 {
		t.Fatalf("la tarea de i18n debe depender de tareas previas")
	}
	arquitectura := mustTask(t, result, "arquitectura")
	if len(arquitectura.Dependencias) != 2 {
		t.Fatalf("arquitectura deberia depender de briefing e investigacion: %+v", arquitectura)
	}
	if got := mustTask(t, result, "entrega_final"); len(got.Dependencias) < 2 {
		t.Fatalf("la entrega debe depender de qa/docs y opcionalmente docker: %+v", got)
	}
	if got := mustTask(t, result, "documentacion").Descripcion; !strings.Contains(got, "Orquesta de Alberto Avidad Fernández") {
		t.Fatalf("descripcion docs sin atribucion: %s", got)
	}
	if got := mustTask(t, result, "documentacion").Descripcion; !strings.Contains(got, "es, en, fr") {
		t.Fatalf("descripcion docs sin idiomas elegidos: %s", got)
	}
	if got := mustTask(t, result, "entrega_final").Descripcion; !strings.Contains(got, "Orquesta de Alberto Avidad Fernández") {
		t.Fatalf("descripcion entrega sin atribucion: %s", got)
	}
	arquitectura = mustTask(t, result, "arquitectura")
	if arquitectura.RolSugerido != "arquitecto" || arquitectura.Fase != "arquitectura" || arquitectura.Entregable == "" || len(arquitectura.CriteriosCierre) < 3 {
		t.Fatalf("SOP de arquitectura incompleto: %+v", arquitectura)
	}
	frontend := mustTask(t, result, "frontend_base")
	if frontend.RolSugerido != "frontend" || frontend.Entregable == "" {
		t.Fatalf("SOP de frontend incompleto: %+v", frontend)
	}
}

func TestGenerateCapturesProfessionalChoicesInDescriptions(t *testing.T) {
	svc := NewService()

	result, err := svc.Generate(AppSpec{
		Nombre:            "Intranet",
		Tipo:              "web_api",
		Auth:              true,
		Database:          true,
		Arquitectura:      "clean",
		AuthMode:          "sso",
		IdentityProvider:  "active_directory",
		RBAC:              true,
		DatabaseEngine:    "postgres",
		ServicioResidente: true,
		Queue:             true,
		Backups:           true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if got := mustTask(t, result, "arquitectura").Descripcion; !strings.Contains(got, "clean architecture") || !strings.Contains(got, "Active Directory") {
		t.Fatalf("descripcion arquitectura inesperada: %s", got)
	}
	if got := mustTask(t, result, "auth").Descripcion; !strings.Contains(got, "roles y permisos") || !strings.Contains(got, "Active Directory") {
		t.Fatalf("descripcion auth inesperada: %s", got)
	}
	if got := mustTask(t, result, "baseline_profesional").Descripcion; !strings.Contains(got, "configuración y secretos") || !strings.Contains(got, "migraciones") {
		t.Fatalf("descripcion baseline inesperada: %s", got)
	}
}

func TestGenerateCLIBacklogIsMinimal(t *testing.T) {
	svc := NewService()

	result, err := svc.Generate(AppSpec{
		Nombre: "cli-util",
		Tipo:   "cli",
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if hasTask(result.Tasks, "frontend_base") {
		t.Fatalf("una app CLI no deberia incluir frontend")
	}
	if hasTask(result.Tasks, "api_base") {
		t.Fatalf("una app CLI no deberia incluir api por defecto")
	}
	if hasTask(result.Tasks, "servicio_residente_minimo") {
		t.Fatalf("una app CLI no deberia forzar un servicio residente")
	}
	if hasTask(result.Tasks, "ui_temas_branding") {
		t.Fatalf("una app CLI no deberia incluir theming UI")
	}
	assertHasTask(t, result, "briefing")
	assertHasTask(t, result, "investigacion")
	assertHasTask(t, result, "arquitectura")
	assertHasTask(t, result, "integracion")
	assertHasTask(t, result, "qa_smoke")
	assertHasTask(t, result, "documentacion")
	assertHasTask(t, result, "entrega_final")
}

func TestGenerateRequiresNameAndType(t *testing.T) {
	svc := NewService()
	if _, err := svc.Generate(AppSpec{Tipo: "web"}); err == nil {
		t.Fatalf("se esperaba error por nombre vacio")
	}
	if _, err := svc.Generate(AppSpec{Nombre: "SinTipo"}); err == nil {
		t.Fatalf("se esperaba error por tipo vacio")
	}
	result, err := svc.Generate(AppSpec{Nombre: "Raro", Tipo: "desktop"})
	if err != nil {
		t.Fatalf("desktop deberia estar soportado: %v", err)
	}
	assertHasTask(t, result, "desktop_base")
	if got := mustTask(t, result, "desktop_base").Descripcion; !strings.Contains(got, "aplicación de escritorio") {
		t.Fatalf("descripcion desktop inesperada: %s", got)
	}
}

func TestGenerateNormalizesI18nIdiomas(t *testing.T) {
	svc := NewService()
	result, err := svc.Generate(AppSpec{
		Nombre:  "Portal",
		Tipo:    "web",
		I18n:    true,
		Idiomas: []string{" ES ", "en", "es", ""},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	i18n := mustTask(t, result, "i18n")
	if want := "es, en"; !strings.Contains(i18n.Descripcion, want) {
		t.Fatalf("descripcion i18n inesperada: %s", i18n.Descripcion)
	}
}

func TestGenerateLanguageExpansion(t *testing.T) {
	svc := NewService()
	result, err := svc.GenerateLanguageExpansion("Portal", []string{" EN ", "fr", "en"})
	if err != nil {
		t.Fatalf("GenerateLanguageExpansion: %v", err)
	}
	assertHasTask(t, result, "i18n_expand_en")
	assertHasTask(t, result, "documentacion_expand_en")
	assertHasTask(t, result, "qa_i18n_expand_en")
	assertHasTask(t, result, "i18n_expand_fr")
	if got := mustTask(t, result, "documentacion_expand_en").Descripcion; !strings.Contains(got, "usuarios finales") || !strings.Contains(got, "sistemas/operación") {
		t.Fatalf("descripcion docs expansion inesperada: %s", got)
	}
	if got := mustTask(t, result, "qa_i18n_expand_en").Dependencias; len(got) != 1 || got[0] != "documentacion_expand_en" {
		t.Fatalf("dependencias QA expansion inesperadas: %+v", got)
	}
}

func TestRecommendDatabaseEngine(t *testing.T) {
	t.Run("none for stateless cli", func(t *testing.T) {
		got := RecommendDatabaseEngine(AppSpec{
			Nombre: "lint-runner",
			Tipo:   "cli",
		})
		if got != "none" {
			t.Fatalf("recommend db for stateless cli = %q, want none", got)
		}
	})

	t.Run("sqlite for local first app", func(t *testing.T) {
		got := RecommendDatabaseEngine(AppSpec{
			Nombre:      "field-app",
			Tipo:        "mobile",
			Database:    true,
			OfflineMode: true,
			FileUploads: true,
		})
		if got != "sqlite" {
			t.Fatalf("recommend db for local-first app = %q, want sqlite", got)
		}
	})

	t.Run("postgres for multiuser server app", func(t *testing.T) {
		got := RecommendDatabaseEngine(AppSpec{
			Nombre:           "portal-interno",
			Tipo:             "web_api",
			Auth:             true,
			AuthMode:         "sso",
			IdentityProvider: "active_directory",
			RBAC:             true,
			Reporting:        true,
			BackgroundJobs:   true,
		})
		if got != "postgres" {
			t.Fatalf("recommend db for multiuser server app = %q, want postgres", got)
		}
	})
}

func TestRecommendAuthMode(t *testing.T) {
	t.Run("none when auth disabled", func(t *testing.T) {
		if got := RecommendAuthMode(AppSpec{Tipo: "web_api"}); got != "none" {
			t.Fatalf("recommend auth for disabled auth = %q, want none", got)
		}
	})
	t.Run("sso when external identity provider", func(t *testing.T) {
		if got := RecommendAuthMode(AppSpec{Tipo: "web_api", Auth: true, IdentityProvider: "active_directory"}); got != "sso" {
			t.Fatalf("recommend auth for external idp = %q, want sso", got)
		}
	})
	t.Run("jwt for api only", func(t *testing.T) {
		if got := RecommendAuthMode(AppSpec{Tipo: "api", Auth: true}); got != "jwt" {
			t.Fatalf("recommend auth for api only = %q, want jwt", got)
		}
	})
}

func TestRecommendDeploymentAndArtifact(t *testing.T) {
	t.Run("serverless and static site for simple web", func(t *testing.T) {
		spec := AppSpec{Tipo: "web", Frontend: true}
		if got := RecommendDeploymentTarget(spec); got != "serverless" {
			t.Fatalf("recommend deployment for simple web = %q, want serverless", got)
		}
		if got := RecommendArtifactType(spec); got != "static_site" {
			t.Fatalf("recommend artifact for simple web = %q, want static_site", got)
		}
	})
	t.Run("vm and desktop installer for desktop app", func(t *testing.T) {
		spec := AppSpec{Tipo: "desktop"}
		if got := RecommendDeploymentTarget(spec); got != "vm" {
			t.Fatalf("recommend deployment for desktop = %q, want vm", got)
		}
		if got := RecommendArtifactType(spec); got != "desktop_installer" {
			t.Fatalf("recommend artifact for desktop = %q, want desktop_installer", got)
		}
	})
	t.Run("docker image for server app", func(t *testing.T) {
		spec := AppSpec{Tipo: "web_api", API: true, Frontend: true, DeploymentTarget: "docker"}
		if got := RecommendDeploymentTarget(spec); got != "docker" {
			t.Fatalf("recommend deployment for web_api = %q, want docker", got)
		}
		if got := RecommendArtifactType(spec); got != "docker_image" {
			t.Fatalf("recommend artifact for web_api = %q, want docker_image", got)
		}
	})
}

func TestRecommendFrontendStack(t *testing.T) {
	t.Run("react for rich product ui", func(t *testing.T) {
		spec := AppSpec{Tipo: "web_api", Frontend: true, Reporting: true, BrandingProfiles: true}
		if got := RecommendFrontendStack(spec); got != "react" {
			t.Fatalf("recommend frontend for rich ui = %q, want react", got)
		}
	})
	t.Run("server rendered for simple web", func(t *testing.T) {
		spec := AppSpec{Tipo: "web", Frontend: true}
		if got := RecommendFrontendStack(spec); got != "server_rendered" {
			t.Fatalf("recommend frontend for simple web = %q, want server_rendered", got)
		}
	})
}

func TestGenerateUsesRecommendedDatabaseEngineWhenNotExplicit(t *testing.T) {
	svc := NewService()

	result, err := svc.Generate(AppSpec{
		Nombre:           "Portal interno",
		Tipo:             "web_api",
		Database:         true,
		Auth:             true,
		AuthMode:         "sso",
		IdentityProvider: "active_directory",
		RBAC:             true,
		Reporting:        true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if got := mustTask(t, result, "persistencia").Descripcion; !strings.Contains(got, "PostgreSQL") {
		t.Fatalf("persistencia sin recomendacion postgres: %s", got)
	}
	if got := mustTask(t, result, "auth").Descripcion; !strings.Contains(got, "SSO corporativo") {
		t.Fatalf("auth sin recomendacion sso: %s", got)
	}

	result, err = svc.Generate(AppSpec{
		Nombre:      "Desktop inventario",
		Tipo:        "desktop",
		Database:    true,
		OfflineMode: true,
	})
	if err != nil {
		t.Fatalf("generate desktop: %v", err)
	}
	if got := mustTask(t, result, "persistencia").Descripcion; !strings.Contains(got, "SQLite") {
		t.Fatalf("persistencia sin recomendacion sqlite: %s", got)
	}
}

func TestGenerateFrontendStackAndThemingAreReflected(t *testing.T) {
	svc := NewService()
	result, err := svc.Generate(AppSpec{
		Nombre:           "Portal ciudadano",
		Tipo:             "web",
		Frontend:         true,
		FrontendStack:    "vue",
		ThemeSupport:     true,
		BrandingProfiles: true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if got := mustTask(t, result, "frontend_base").Descripcion; !strings.Contains(got, "Vue") {
		t.Fatalf("frontend_base sin stack web: %s", got)
	}
	if got := mustTask(t, result, "frontend_base").Descripcion; !strings.Contains(got, "`web/`") {
		t.Fatalf("frontend_base sin superficie web explicita: %s", got)
	}
	if got := mustTask(t, result, "ui_temas_branding").Descripcion; !strings.Contains(got, "branding") || !strings.Contains(got, "Vue") {
		t.Fatalf("ui_temas_branding sin theming/branding: %s", got)
	}
}

func assertHasTask(t *testing.T, result GenerationResult, key string) {
	t.Helper()
	if !hasTask(result.Tasks, key) {
		t.Fatalf("falta la tarea %s en %+v", key, result.Tasks)
	}
}

func hasTask(tasks []BlueprintTask, key string) bool {
	for _, item := range tasks {
		if item.Key == key {
			return true
		}
	}
	return false
}

func mustTask(t *testing.T, result GenerationResult, key string) BlueprintTask {
	t.Helper()
	for _, item := range result.Tasks {
		if item.Key == key {
			return item
		}
	}
	t.Fatalf("no existe la tarea %s", key)
	return BlueprintTask{}
}
