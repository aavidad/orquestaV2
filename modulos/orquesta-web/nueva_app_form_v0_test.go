package orquestaweb

import (
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestWebNuevaAppFormV0ToAppSpecRequestV0DefaultsSchemaSourceI18N(t *testing.T) {
	form := WebNuevaAppFormV0{
		RequestID: " req-1 ",
		Locale:    " es ",
		Nombre:    " Agenda ",
		Objetivo:  " Coordinar ensayos ",
		TipoApp:   " web ",
	}

	req := form.ToAppSpecRequestV0()

	if req.SchemaVersion != orquestafactory.AppSpecRequestSchemaV0 {
		t.Fatalf("schema_version=%q", req.SchemaVersion)
	}
	if req.Source != WebNuevaAppSourceV0 {
		t.Fatalf("source=%q", req.Source)
	}
	if req.RequestID != "req-1" || req.Locale != "es" || req.Nombre != "Agenda" {
		t.Fatalf("normalizacion superficial inesperada: %+v", req)
	}
	if req.I18N.DefaultLocale != "es" {
		t.Fatalf("default_locale=%q", req.I18N.DefaultLocale)
	}
	if req.PreferenciasTecnicas.Arquitectura != "hexagonal" {
		t.Fatalf("arquitectura=%q", req.PreferenciasTecnicas.Arquitectura)
	}
	if req.RequestKind != orquestafactory.RequestKindCrearAppCompletaV0 || req.ExecutionMode != orquestafactory.ExecutionModeNormalV0 {
		t.Fatalf("request policy: kind=%q mode=%q", req.RequestKind, req.ExecutionMode)
	}
}

func TestWebNuevaAppFormV0ToAppSpecRequestV0PreservaCamposRicos(t *testing.T) {
	no := false
	yes := true
	form := WebNuevaAppFormV0{
		RequestID:     "req-rica",
		Locale:        "es",
		RequestKind:   "documentar_app",
		ExecutionMode: "debug",
		Nombre:        "Portal",
		Objetivo:      "Publicar contenidos",
		Descripcion:   "CMS interno",
		TipoApp:       "web",
		ProjectSource: WebNuevaAppProjectSourceFormV0{
			Kind:       " github ",
			GitURL:     " https://example.test/repo.git ",
			Branch:     " main ",
			ProjectRef: " project-ref-123 ",
		},
		UsuariosObjetivo: []string{" editores ", "operaciones"},
		Plataformas:      []string{"web", "mobile"},
		Integraciones: []WebNuevaAppConnectorFormV0{{
			Tipo:          "api",
			Nombre:        "crm",
			Proposito:     "sincronizar clientes",
			Direccion:     "bidireccional",
			Auth:          "oauth",
			DataScope:     "clientes",
			Criticidad:    "alta",
			Requerido:     true,
			Restricciones: []string{"oauth"},
		}},
		PreferenciasTecnicas: WebNuevaAppPreferenciasFormV0{
			Lenguaje:      "go",
			Framework:     "std",
			Arquitectura:  "hexagonal",
			Restricciones: []string{"sin proveedor cerrado"},
			Preferencias:  []string{"tests unitarios"},
		},
		Datos: WebNuevaAppDatosFormV0{
			DBRequired:         true,
			NecesidadFuncional: "guardar borradores",
			TiposDatos:         []string{"contenido", "usuarios"},
			TiposDetallados: []WebNuevaAppDataTypeFormV0{{
				Nombre:       "contenidos",
				Proposito:    "publicar borradores",
				Sensibilidad: "interna",
				Retencion:    "12 meses",
				Volumen:      "medio",
			}},
			Storage: []WebNuevaAppDataStorageFormV0{{
				Tipo:      "relacional",
				Proposito: "consultas transaccionales",
				Requerido: true,
			}},
			Fuentes: []WebNuevaAppDataSourceFormV0{{
				Nombre:     "CMS legado",
				Tipo:       "api",
				Proposito:  "importar contenidos existentes",
				Owner:      "comunicacion",
				Frecuencia: "diaria",
			}},
			Operacion: WebNuevaAppDataOperationFormV0{
				Criticidad:     "alta",
				Disponibilidad: "horario laboral",
				RPO:            "24h",
				RTO:            "4h",
				Auditoria:      true,
				Restricciones:  []string{"traza de cambios"},
			},
			Sensibilidad: "media",
			Retencion:    "12 meses",
		},
		Deploy: WebNuevaAppDeployFormV0{
			Target:        "contenedor",
			Restricciones: []string{"sin cloud propietaria"},
		},
		Calidad: WebNuevaAppCalidadFormV0{
			Pruebas:               "alta",
			Accesibilidad:         "wcag_aa",
			AccesibilidadOpciones: []string{"normal", "wcag_aa"},
			Compliance:            []string{"gdpr"},
			Observabilidad:        &no,
		},
		Documentacion: WebNuevaAppDocumentacionFormV0{
			Usuario:     &yes,
			Desarrollo:  &yes,
			Sistemas:    &no,
			Profundidad: "profunda",
			Locales:     []string{"es", "en"},
		},
		I18N: WebNuevaAppI18NFormV0{
			Enabled:       &yes,
			DefaultLocale: "es",
			Locales:       []string{"en"},
		},
		Agentes: WebNuevaAppAgentesFormV0{
			RevisionHumana: &yes,
			Autonomia:      "media",
			Preferencias:   []string{"pedir revision"},
		},
		Restricciones: []string{"sin secretos"},
	}

	req := form.ToAppSpecRequestV0()

	if len(req.Integraciones) != 1 ||
		req.Integraciones[0].Nombre != "crm" ||
		req.Integraciones[0].Direccion != "bidireccional" ||
		req.Integraciones[0].Auth != "oauth" ||
		req.Integraciones[0].DataScope != "clientes" ||
		req.Integraciones[0].Criticidad != "alta" ||
		!req.Integraciones[0].Requerido {
		t.Fatalf("integraciones no preservadas: %+v", req.Integraciones)
	}
	if !req.Datos.DBRequired || req.Datos.NecesidadFuncional != "guardar borradores" || req.Datos.TiposDatos[1] != "usuarios" {
		t.Fatalf("datos no preservados: %+v", req.Datos)
	}
	if len(req.Datos.TiposDetallados) != 1 ||
		req.Datos.TiposDetallados[0].Nombre != "contenidos" ||
		len(req.Datos.Storage) != 1 ||
		req.Datos.Storage[0].Tipo != "relacional" ||
		!req.Datos.Storage[0].Requerido {
		t.Fatalf("datos expertos no preservados: %+v", req.Datos)
	}
	if len(req.Datos.Fuentes) != 1 ||
		req.Datos.Fuentes[0].Nombre != "CMS legado" ||
		req.Datos.Fuentes[0].Owner != "comunicacion" ||
		req.Datos.Operacion.Criticidad != "alta" ||
		req.Datos.Operacion.RTO != "4h" ||
		!req.Datos.Operacion.Auditoria {
		t.Fatalf("datos fuentes/operacion no preservados: %+v", req.Datos)
	}
	if req.Deploy.Target != "contenedor" || req.Calidad.Observabilidad == nil || *req.Calidad.Observabilidad {
		t.Fatalf("deploy/calidad no preservados: deploy=%+v calidad=%+v", req.Deploy, req.Calidad)
	}
	if len(req.Calidad.AccesibilidadOpciones) != 2 || req.Calidad.AccesibilidadOpciones[0] != "normal" {
		t.Fatalf("opciones de accesibilidad no preservadas: %+v", req.Calidad)
	}
	if req.Documentacion.Sistemas == nil || *req.Documentacion.Sistemas || req.Documentacion.Profundidad != "profunda" {
		t.Fatalf("documentacion no preservada: %+v", req.Documentacion)
	}
	if req.RequestKind != "documentar_app" || req.ExecutionMode != "debug" {
		t.Fatalf("request policy no preservada: kind=%q mode=%q", req.RequestKind, req.ExecutionMode)
	}
	if req.I18N.Enabled == nil || !*req.I18N.Enabled || req.Agentes.RevisionHumana == nil || !*req.Agentes.RevisionHumana {
		t.Fatalf("i18n/agentes no preservados: i18n=%+v agentes=%+v", req.I18N, req.Agentes)
	}
	if req.ProjectSource.Kind != "github" ||
		req.ProjectSource.GitURL != "https://example.test/repo.git" ||
		req.ProjectSource.Branch != "main" ||
		req.ProjectSource.ProjectRef != "project-ref-123" {
		t.Fatalf("project_source no preservado hacia factory: %+v", req.ProjectSource)
	}
}

func TestWebNuevaAppFormV0ToAppSpecRequestV0NoValidaReglasDeNegocio(t *testing.T) {
	form := WebNuevaAppFormV0{
		RequestID: "req-invalid",
		Locale:    "es",
		Nombre:    "",
		Objetivo:  "",
		TipoApp:   "mainframe",
		Deploy:    WebNuevaAppDeployFormV0{Target: "target-no-soportado"},
	}

	req := form.ToAppSpecRequestV0()

	if req.Nombre != "" || req.Objetivo != "" || req.TipoApp != "mainframe" || req.Deploy.Target != "target-no-soportado" {
		t.Fatalf("el mapper altero campos que debe dejar a factory: %+v", req)
	}
}
