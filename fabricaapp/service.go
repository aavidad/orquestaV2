package fabricaapp

import (
	"fmt"
	"slices"
	"strings"
)

// AppSpec define la especificación completa de una app a fabricar.
//
// Campos fijos (siempre activos, no configurables por el usuario):
//   - Hexagonal:    arquitectura hexagonal (puertos y adaptadores)
//   - Modularidad:  separación en módulos independientes
//   - Testing:      tests unitarios e integración desde el inicio
//   - Linting:      configuración de linter y formateo
//
// Campos configurables por el usuario:
//   - Plataformas, SO, compliance, infraestructura, etc.
type AppSpec struct {
	Nombre           string
	Descripcion      string
	ObjetivoNegocio  string
	UsuariosObjetivo string
	Restricciones    string

	// Tipo de interfaz principal
	// Valores: web, api, web_api, cli, desktop, mobile, embedded
	Tipo string

	// Componentes de interfaz
	Frontend bool // UI web
	API      bool // API HTTP/REST/GraphQL
	Auth     bool // Autenticación y autorización
	Database bool // Persistencia
	Docker   bool // Contenedores
	I18n     bool // Internacionalización
	Idiomas  []string

	// Plataformas de despliegue (puede ser múltiple)
	PlatWeb      bool // Web (navegador)
	PlatDesktop  bool // App de escritorio (Electron, Tauri, Qt, GTK…)
	PlatMobile   bool // App móvil (Android/iOS)
	PlatCLI      bool // Línea de comandos
	PlatEmbedded bool // Sistemas embebidos / IoT

	// Sistemas operativos objetivo
	SOLinux   bool
	SOWindows bool
	SOmacOS   bool
	SOAndroid bool
	SOiOS     bool

	// Compliance legal y normativo (fijos conceptualmente, elegibles por proyecto)
	ComplianceRGPD          bool // Reglamento General de Protección de Datos (UE 2016/679)
	ComplianceENS           bool // Esquema Nacional de Seguridad (RD 311/2022)
	ComplianceLSSI          bool // Ley Servicios Sociedad de la Información
	ComplianceWCAG          bool // Accesibilidad web WCAG 2.1 AA (RD 1112/2018)
	ComplianceFacturaElec   bool // Facturación electrónica (Ley Crea y Crece)
	ComplianceReutilizacion bool // Reutilización información sector público (Ley 37/2007)

	// Infraestructura y operación
	CI         bool // Pipeline CI/CD (GitHub Actions, GitLab CI…)
	Kubernetes bool // Orquestación de contenedores
	Terraform  bool // Infraestructura como código
	Monitoring bool // Observabilidad (métricas, trazas, logs)

	// Decisiones guiadas de arquitectura y producto
	Arquitectura          string // hexagonal, clean, layered, event_driven
	APIStyle              string // rest, graphql, grpc, async, mixed
	FrontendStack         string // react, vue, sveltekit, nextjs, astro, htmx, server_rendered, angular, other
	DatabaseEngine        string // postgres, mysql, sqlite, mongodb, redis, other
	AuthMode              string // session, jwt, oauth, sso, api_key
	IdentityProvider      string // active_directory, ldap, entra_id, keycloak, auth0, google, other
	TestingLevel          string // base, strict, tdd
	ObservabilityLevel    string // basic, standard, strict
	DeploymentTarget      string // docker, kubernetes, serverless, vm, edge
	ArtifactType          string // executable, docker_image, library, desktop_installer, mobile_bundle, firmware, static_site
	BackgroundJobs        bool
	Notifications         bool
	MultiTenant           bool
	RBAC                  bool
	ThemeSupport          bool
	BrandingProfiles      bool
	OfflineMode           bool
	ImportExport          bool
	Webhooks              bool
	FileUploads           bool
	Reporting             bool
	ServicioResidente     bool
	Cache                 bool
	Queue                 bool
	Scheduler             bool
	ObjectStorage         bool
	Search                bool
	RateLimiting          bool
	FeatureFlags          bool
	AuditTrail            bool
	Backups               bool
	DisasterRecovery      bool
	IntegracionesExternas []string

	// Calidad y arquitectura (siempre activos internamente)
	// No se exponen como campos porque son obligatorios en toda app generada.
	// Se generan tareas para ellos siempre.
}

type BlueprintTask struct {
	Key              string
	Titulo           string
	Descripcion      string
	Modulo           string
	RolSugerido      string
	Fase             string
	Entregable       string
	CriteriosCierre  []string
	Prioridad        PrioridadTarea
	Dependencias     []string
	ContratoDefinido bool
}

type PrioridadTarea string

const (
	PrioridadAlta  PrioridadTarea = "alta"
	PrioridadMedia PrioridadTarea = "media"
	PrioridadBaja  PrioridadTarea = "baja"
)

type GenerationResult struct {
	Tasks []BlueprintTask
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// RecommendDatabaseEngine devuelve el motor recomendado para el primer corte del proyecto.
// No impone la decisión final; solo sugiere un default sensato para el wizard.
func RecommendDatabaseEngine(spec AppSpec) string {
	spec = normalizeSpecForDatabaseRecommendation(spec)

	durable := needsDurablePersistence(spec)
	serverGrade := needsServerGradeDatabase(spec)
	embeddedPreferred := prefersEmbeddedDatabase(spec)

	switch {
	case !durable && !serverGrade && !embeddedPreferred:
		return "none"
	case embeddedPreferred:
		return "sqlite"
	case serverGrade:
		return "postgres"
	case spec.Database || durable:
		if spec.OfflineMode || spec.PlatDesktop || spec.PlatMobile || spec.PlatCLI || spec.PlatEmbedded {
			return "sqlite"
		}
		return "postgres"
	default:
		return "none"
	}
}

// RecommendFrontendStack devuelve el stack web recomendado cuando la app tiene UI web.
func RecommendFrontendStack(spec AppSpec) string {
	spec = normalizeSpecForDatabaseRecommendation(spec)

	if !spec.Frontend && spec.Tipo != "web" && spec.Tipo != "web_api" {
		return "server_rendered"
	}
	if spec.OfflineMode || spec.Reporting || spec.FeatureFlags || spec.MultiTenant || spec.FileUploads {
		return "react"
	}
	if spec.Tipo == "web" && !spec.API && !spec.ServicioResidente && !spec.BackgroundJobs {
		return "server_rendered"
	}
	return "react"
}

// RecommendAuthMode devuelve el modo de autenticación recomendado.
func RecommendAuthMode(spec AppSpec) string {
	spec = normalizeSpecForDatabaseRecommendation(spec)

	switch {
	case !spec.Auth:
		return "none"
	case spec.IdentityProvider != "none":
		return "sso"
	case spec.Tipo == "mobile":
		return "oauth"
	case spec.Tipo == "cli":
		return "api_key"
	case spec.Tipo == "api" && !spec.Frontend:
		return "jwt"
	default:
		return "session"
	}
}

// RecommendDeploymentTarget devuelve el target de despliegue recomendado.
func RecommendDeploymentTarget(spec AppSpec) string {
	spec = normalizeSpecForDatabaseRecommendation(spec)

	switch {
	case spec.Kubernetes:
		return "kubernetes"
	case spec.Tipo == "embedded" || spec.PlatEmbedded:
		return "edge"
	case spec.Tipo == "desktop" || spec.Tipo == "mobile" || spec.Tipo == "cli" || spec.PlatDesktop || spec.PlatMobile || spec.PlatCLI:
		return "vm"
	case spec.Tipo == "web" && !spec.API && !spec.ServicioResidente && !spec.BackgroundJobs && !spec.Queue && !spec.Scheduler:
		return "serverless"
	default:
		return "docker"
	}
}

// RecommendArtifactType devuelve el artefacto principal recomendado.
func RecommendArtifactType(spec AppSpec) string {
	spec = normalizeSpecForDatabaseRecommendation(spec)

	switch {
	case spec.Tipo == "embedded" || spec.PlatEmbedded:
		return "firmware"
	case spec.Tipo == "mobile" || spec.PlatMobile:
		return "mobile_bundle"
	case spec.Tipo == "desktop" || spec.PlatDesktop:
		return "desktop_installer"
	case spec.Tipo == "web" && !spec.API && !spec.ServicioResidente && !spec.BackgroundJobs && !spec.Queue && !spec.Scheduler:
		return "static_site"
	case spec.ArtifactType == "library":
		return "library"
	case spec.DeploymentTarget == "kubernetes" || spec.Kubernetes || spec.DeploymentTarget == "docker":
		return "docker_image"
	default:
		return "executable"
	}
}

func (s *Service) Generate(spec AppSpec) (GenerationResult, error) {
	normalized, err := normalizeSpec(spec)
	if err != nil {
		return GenerationResult{}, err
	}

	builder := blueprintBuilder{spec: normalized}

	// ── Fase 0: Descubrimiento ────────────────────────────────────────────────

	builder.add(withSOP(BlueprintTask{
		Key:              "briefing",
		Titulo:           fmt.Sprintf("Definir briefing funcional de %s", normalized.Nombre),
		Descripcion:      composeBriefingDescription(normalized),
		Modulo:           "producto",
		Prioridad:        PrioridadAlta,
		ContratoDefinido: true,
	}, "product_manager", "descubrimiento", "Brief funcional validado",
		"Objetivo de producto claro",
		"Alcance y restricciones iniciales documentados",
		"Plataformas y SO objetivo confirmados",
		"Criterios de éxito entendibles por el equipo"))

	builder.add(withSOP(BlueprintTask{
		Key:              "investigacion",
		Titulo:           fmt.Sprintf("Revisar catálogo y referencias de %s", normalized.Nombre),
		Descripcion:      composeResearchDescription(normalized),
		Modulo:           "analisis",
		Prioridad:        PrioridadAlta,
		ContratoDefinido: true,
	}, "analista", "descubrimiento", "Informe de referencias y restricciones",
		"Catálogo compartido revisado",
		"Referencias externas resumidas",
		"Riesgos tempranos identificados"))

	if requiresResidentService(normalized) {
		builder.add(withSOP(BlueprintTask{
			Key:              "servicio_residente_minimo",
			Titulo:           fmt.Sprintf("Diseñar núcleo residente mínimo de %s", normalized.Nombre),
			Descripcion:      composeResidentServiceDescription(normalized),
			Modulo:           "arquitectura",
			Prioridad:        PrioridadAlta,
			Dependencias:     []string{"briefing", "investigacion"},
			ContratoDefinido: true,
		}, "arquitecto", "arquitectura", "Modelo de servicio residente mínimo aprobado",
			"Núcleo residente acotado y justificado",
			"Trabajo pesado separado en workers, eventos o caminos on-demand",
			"Estado caliente y persistencia durable distinguidos",
			"Frecuencias de observación y presupuestos operativos documentadas"))
	}

	// ── Fase 1: Arquitectura (siempre hexagonal + modular) ───────────────────

	builder.add(withSOP(BlueprintTask{
		Key:              "arquitectura",
		Titulo:           fmt.Sprintf("Cerrar arquitectura hexagonal y modular de %s", normalized.Nombre),
		Descripcion:      composeArchitectureDescription(normalized),
		Modulo:           "arquitectura",
		Prioridad:        PrioridadAlta,
		Dependencias:     []string{"briefing", "investigacion"},
		ContratoDefinido: true,
	}, "arquitecto", "arquitectura", "Diseño hexagonal y modular aprobado",
		"Puertos y adaptadores definidos",
		"Módulos con contratos claros entre sí",
		"Estrategia de despliegue por plataforma documentada",
		"Riesgos arquitectónicos con mitigación"))

	// Calidad de código: linting y convenciones (fijo siempre)
	builder.add(withSOP(BlueprintTask{
		Key:              "tooling_calidad",
		Titulo:           fmt.Sprintf("Configurar tooling de calidad para %s", normalized.Nombre),
		Descripcion:      "Configurar linter, formatter, pre-commit hooks y convenciones de código. Garantizar que todo el equipo parte del mismo estándar de calidad desde el primer commit.",
		Modulo:           "tooling",
		Prioridad:        PrioridadAlta,
		Dependencias:     []string{"arquitectura"},
		ContratoDefinido: true,
	}, "devops", "arquitectura", "Tooling de calidad operativo",
		"Linter y formatter configurados y documentados",
		"Pre-commit hooks activos",
		"Convenciones de código acordadas y escritas"))

	// Testing base (fijo siempre)
	builder.add(withSOP(BlueprintTask{
		Key:              "testing_base",
		Titulo:           fmt.Sprintf("Establecer estrategia y base de tests de %s", normalized.Nombre),
		Descripcion:      "Definir pirámide de tests (unitarios, integración, e2e), framework de test, convenciones de nombrado y cobertura mínima aceptable. Los tests son ciudadanos de primera clase, no un añadido posterior.",
		Modulo:           "testing",
		Prioridad:        PrioridadAlta,
		Dependencias:     []string{"arquitectura"},
		ContratoDefinido: true,
	}, "qa", "arquitectura", "Estrategia de tests definida y base operativa",
		"Framework de test instalado y con primer test verde",
		"Cobertura mínima acordada y documentada",
		"Tests de integración con BD/servicios externos planificados"))

	builder.add(withSOP(BlueprintTask{
		Key:              "baseline_profesional",
		Titulo:           fmt.Sprintf("Cerrar baseline de codificación profesional de %s", normalized.Nombre),
		Descripcion:      composeProfessionalBaselineDescription(normalized),
		Modulo:           "ingenieria",
		Prioridad:        PrioridadAlta,
		Dependencias:     []string{"arquitectura", "tooling_calidad", "testing_base"},
		ContratoDefinido: true,
	}, "tech_lead", "arquitectura", "Baseline profesional acordada y operativa",
		"Configuración, secretos y entornos definidos sin hardcodes",
		"Política de errores, validación y logging estructurado documentada",
		"Desarrollo local y CI reproducibles",
		"Seguridad baseline y dependencias críticas revisadas"))

	// ── Fase 2: Compliance (si aplica) ───────────────────────────────────────

	if hasCompliance(normalized) {
		builder.add(withSOP(BlueprintTask{
			Key:              "compliance",
			Titulo:           fmt.Sprintf("Diseñar plan de compliance legal y normativo de %s", normalized.Nombre),
			Descripcion:      composeComplianceDescription(normalized),
			Modulo:           "compliance",
			Prioridad:        PrioridadAlta,
			Dependencias:     []string{"briefing", "investigacion"},
			ContratoDefinido: true,
		}, "juridico", "arquitectura", "Plan de compliance documentado y revisado",
			"Normativas aplicables identificadas y priorizadas",
			"Impacto en arquitectura y datos evaluado",
			"Responsable de protección de datos asignado si aplica RGPD"))
	}

	// ── Fase 3: Implementación por tipo/plataforma ────────────────────────────

	archDeps := []string{"arquitectura", "tooling_calidad", "testing_base"}
	if requiresResidentService(normalized) {
		archDeps = append(archDeps, "servicio_residente_minimo")
	}
	if hasCompliance(normalized) {
		archDeps = append(archDeps, "compliance")
	}

	switch normalized.Tipo {
	case "web":
		builder.add(frontendTask(normalized, archDeps))
	case "api":
		builder.add(apiTask(normalized, archDeps))
	case "web_api":
		builder.add(frontendTask(normalized, archDeps))
		builder.add(apiTask(normalized, archDeps))
	case "cli":
		builder.add(cliTask(normalized, archDeps))
	case "desktop":
		builder.add(desktopTask(normalized, archDeps))
	case "mobile":
		builder.add(mobileTask(normalized, archDeps))
	case "embedded":
		builder.add(embeddedTask(normalized, archDeps))
	}

	if hasThemeableUI(normalized) {
		builder.add(withSOP(BlueprintTask{
			Key:              "ui_temas_branding",
			Titulo:           fmt.Sprintf("Cerrar sistema de temas y branding de %s", normalized.Nombre),
			Descripcion:      composeUIThemesDescription(normalized),
			Modulo:           "frontend",
			Prioridad:        PrioridadAlta,
			Dependencias:     uniqueKeys(append(archDeps, featureDepsForAuth(normalized)...)...),
			ContratoDefinido: true,
		}, "frontend", "implementacion", "Sistema de temas desacoplado y reutilizable",
			"Tokens semánticos de color, tipografía y espaciado definidos",
			"Componentes sin estilos hardcodeados por negocio",
			"Cambio de tema o branding sin tocar lógica de aplicación",
			"Documentación de theming y branding disponible"))
	}

	if normalized.Database {
		builder.add(withSOP(BlueprintTask{
			Key:              "persistencia",
			Titulo:           fmt.Sprintf("Implementar persistencia principal de %s", normalized.Nombre),
			Descripcion:      composePersistenceDescription(normalized),
			Modulo:           "persistencia",
			Prioridad:        PrioridadAlta,
			Dependencias:     archDeps,
			ContratoDefinido: true,
		}, "backend", "implementacion", "Persistencia funcional y versionada",
			"Esquema y migraciones reproducibles",
			"Adaptador de BD detrás de puerto hexagonal",
			"Cobertura mínima de persistencia"))
	}

	if normalized.Auth {
		deps := uniqueKeys(append(archDeps, featureDepsForAuth(normalized)...)...)
		builder.add(withSOP(BlueprintTask{
			Key:              "auth",
			Titulo:           fmt.Sprintf("Cerrar autenticación y autorización de %s", normalized.Nombre),
			Descripcion:      composeAuthDescription(normalized),
			Modulo:           "seguridad",
			Prioridad:        PrioridadAlta,
			Dependencias:     deps,
			ContratoDefinido: true,
		}, "seguridad", "implementacion", "Flujo de autenticación y autorización operativo",
			"Casos de login y permiso definidos",
			"Protecciones aplicadas en puntos sensibles",
			"Cobertura mínima de errores y permisos",
			"Tokens/sesiones con expiración y revocación"))
	}

	if normalized.I18n {
		builder.add(withSOP(BlueprintTask{
			Key:              "i18n",
			Titulo:           fmt.Sprintf("Aplicar i18n base en %s", normalized.Nombre),
			Descripcion:      composeI18nDescription(normalized),
			Modulo:           "i18n",
			Prioridad:        PrioridadAlta,
			Dependencias:     archDeps,
			ContratoDefinido: true,
		}, "i18n", "implementacion", "Superficie inicial multilenguaje",
			"Bundles iniciales cargados",
			"Idiomas base operativos",
			"Sin literales nuevos en el flujo principal"))
	}

	if normalized.ComplianceRGPD {
		builder.add(withSOP(BlueprintTask{
			Key:              "rgpd_impl",
			Titulo:           fmt.Sprintf("Implementar medidas RGPD en %s", normalized.Nombre),
			Descripcion:      "Implementar: consentimiento informado, registro de tratamientos, derecho al olvido, portabilidad de datos, cifrado en reposo y tránsito, política de retención, avisos legales y cookie banner si aplica web. Revisar con el plan de compliance.",
			Modulo:           "compliance",
			Prioridad:        PrioridadAlta,
			Dependencias:     uniqueKeys(append(archDeps, "compliance")...),
			ContratoDefinido: true,
		}, "juridico", "implementacion", "Medidas RGPD implementadas y auditables",
			"Registro de tratamientos creado",
			"Consentimiento informado operativo",
			"Cifrado de datos personales activo",
			"Aviso legal y política de privacidad redactados"))
	}

	if normalized.ComplianceENS {
		builder.add(withSOP(BlueprintTask{
			Key:              "ens_impl",
			Titulo:           fmt.Sprintf("Adecuación ENS de %s", normalized.Nombre),
			Descripcion:      "Implementar controles del Esquema Nacional de Seguridad según categoría del sistema (básica/media/alta): gestión de accesos, trazabilidad, continuidad, inventario de activos, análisis de riesgos y declaración de aplicabilidad.",
			Modulo:           "compliance",
			Prioridad:        PrioridadAlta,
			Dependencias:     uniqueKeys(append(archDeps, "compliance")...),
			ContratoDefinido: true,
		}, "seguridad", "implementacion", "Controles ENS implementados",
			"Categoría del sistema determinada",
			"Análisis de riesgos documentado",
			"Declaración de aplicabilidad redactada"))
	}

	if normalized.ComplianceWCAG {
		builder.add(withSOP(BlueprintTask{
			Key:              "accesibilidad",
			Titulo:           fmt.Sprintf("Implementar accesibilidad WCAG 2.1 AA en %s", normalized.Nombre),
			Descripcion:      "Garantizar conformidad con WCAG 2.1 nivel AA (RD 1112/2018): contraste de color, navegación por teclado, textos alternativos, ARIA, formularios accesibles, página de accesibilidad y declaración oficial.",
			Modulo:           "accesibilidad",
			Prioridad:        PrioridadAlta,
			Dependencias:     uniqueKeys(append(archDeps, "compliance")...),
			ContratoDefinido: true,
		}, "frontend", "implementacion", "Conformidad WCAG 2.1 AA verificada",
			"Auditoría automática sin errores críticos",
			"Navegación por teclado operativa",
			"Declaración de accesibilidad publicada"))
	}

	if normalized.ComplianceFacturaElec {
		builder.add(withSOP(BlueprintTask{
			Key:              "factura_elec",
			Titulo:           fmt.Sprintf("Integrar facturación electrónica en %s", normalized.Nombre),
			Descripcion:      "Implementar emisión de facturas electrónicas según Ley Crea y Crece (Ley 18/2022): formato Facturae o UBL, firma electrónica, envío a plataformas habilitadas (FACe, FACeB2B), conservación y trazabilidad.",
			Modulo:           "facturacion",
			Prioridad:        PrioridadAlta,
			Dependencias:     uniqueKeys(append(archDeps, "compliance")...),
			ContratoDefinido: true,
		}, "backend", "implementacion", "Facturación electrónica operativa",
			"Formato Facturae/UBL generado correctamente",
			"Firma electrónica integrada",
			"Envío a plataforma pública verificado"))
	}

	if normalized.ComplianceLSSI {
		builder.add(withSOP(BlueprintTask{
			Key:              "lssi_impl",
			Titulo:           fmt.Sprintf("Cumplimiento LSSI en %s", normalized.Nombre),
			Descripcion:      "Implementar requisitos de la Ley de Servicios de la Sociedad de la Información: aviso legal, identificación del prestador, cookies, comunicaciones comerciales y condiciones de contratación electrónica.",
			Modulo:           "compliance",
			Prioridad:        PrioridadMedia,
			Dependencias:     uniqueKeys(append(archDeps, "compliance")...),
			ContratoDefinido: false,
		}, "juridico", "implementacion", "Textos legales LSSI publicados",
			"Aviso legal y política de cookies operativos",
			"Identificación del prestador visible"))
	}

	if normalized.ComplianceReutilizacion {
		builder.add(withSOP(BlueprintTask{
			Key:              "reutilizacion",
			Titulo:           fmt.Sprintf("Política de reutilización y licencias en %s", normalized.Nombre),
			Descripcion:      "Definir licencia del software (GPL v3 para software público, o la acordada), publicar en repositorio accesible, aplicar metadatos de reutilización según Ley 37/2007 y NTI de reutilización.",
			Modulo:           "compliance",
			Prioridad:        PrioridadMedia,
			Dependencias:     archDeps,
			ContratoDefinido: false,
		}, "juridico", "documentacion", "Política de reutilización publicada",
			"Licencia definida y aplicada a todos los ficheros",
			"Repositorio público con metadatos correctos"))
	}

	// ── Fase 4: Infraestructura ───────────────────────────────────────────────

	if normalized.CI {
		builder.add(withSOP(BlueprintTask{
			Key:              "ci_cd",
			Titulo:           fmt.Sprintf("Configurar pipeline CI/CD de %s", normalized.Nombre),
			Descripcion:      composeCICDDescription(normalized),
			Modulo:           "devops",
			Prioridad:        PrioridadAlta,
			Dependencias:     []string{"testing_base", "tooling_calidad"},
			ContratoDefinido: true,
		}, "devops", "implementacion", "Pipeline CI/CD operativo",
			"Build automático en cada PR",
			"Tests ejecutados en CI",
			"Deploy automático a entorno de pruebas"))
	}

	if normalized.Kubernetes {
		builder.add(withSOP(BlueprintTask{
			Key:              "kubernetes",
			Titulo:           fmt.Sprintf("Configurar despliegue Kubernetes de %s", normalized.Nombre),
			Descripcion:      "Crear manifiestos Kubernetes: Deployment, Service, Ingress, ConfigMap, Secret, HPA, PodDisruptionBudget, NetworkPolicy. Configurar namespaces, RBAC y health checks.",
			Modulo:           "devops",
			Prioridad:        PrioridadMedia,
			Dependencias:     []string{"docker"},
			ContratoDefinido: false,
		}, "devops", "despliegue", "Manifiestos Kubernetes operativos",
			"App desplegada en cluster de pruebas",
			"HPA y health checks funcionando",
			"NetworkPolicy y RBAC configurados"))
	}

	if normalized.Terraform {
		builder.add(withSOP(BlueprintTask{
			Key:              "terraform",
			Titulo:           fmt.Sprintf("Infraestructura como código (Terraform) para %s", normalized.Nombre),
			Descripcion:      "Definir infraestructura en Terraform: redes, servidores, BD, almacenamiento, DNS, certificados. Configurar estado remoto y pipeline de apply.",
			Modulo:           "devops",
			Prioridad:        PrioridadMedia,
			Dependencias:     []string{"arquitectura"},
			ContratoDefinido: false,
		}, "devops", "despliegue", "Infraestructura reproducible en Terraform",
			"Estado remoto configurado",
			"Apply en entorno de pruebas verificado",
			"Variables sensibles en vault/secrets"))
	}

	if normalized.Monitoring {
		builder.add(withSOP(BlueprintTask{
			Key:              "observabilidad",
			Titulo:           fmt.Sprintf("Implementar observabilidad en %s", normalized.Nombre),
			Descripcion:      composeMonitoringDescription(normalized),
			Modulo:           "observabilidad",
			Prioridad:        PrioridadMedia,
			Dependencias:     archDeps,
			ContratoDefinido: false,
		}, "devops", "implementacion", "Observabilidad operativa",
			"Logs estructurados emitidos",
			"Métricas de negocio y sistema expuestas",
			"Trazas distribuidas activas",
			"Dashboard básico de salud configurado"))
	}

	// ── Fase 5: Integración y cierre ─────────────────────────────────────────

	builder.add(integrationTask(normalized, builder.featureDependencyKeys()))

	if normalized.Docker {
		dockerDeps := []string{"integracion"}
		if normalized.Kubernetes {
			dockerDeps = append(dockerDeps, "kubernetes")
		}
		builder.add(withSOP(BlueprintTask{
			Key:          "docker",
			Titulo:       fmt.Sprintf("Preparar despliegue Docker de %s", normalized.Nombre),
			Descripcion:  "Crear imagen multi-stage, compose/manifiestos, variables de entorno, healthchecks y README de operación para la entrega operativa.",
			Modulo:       "deploy",
			Prioridad:    PrioridadMedia,
			Dependencias: dockerDeps,
		}, "devops", "despliegue", "Artefactos de despliegue reproducibles",
			"Imagen multi-stage construida y publicada",
			"Variables y healthchecks definidos",
			"Arranque verificado en entorno limpio"))
	}

	qaDeps := []string{"integracion"}
	if normalized.Docker {
		qaDeps = append(qaDeps, "docker")
	}
	if normalized.CI {
		qaDeps = append(qaDeps, "ci_cd")
	}

	builder.add(withSOP(BlueprintTask{
		Key:          "qa_smoke",
		Titulo:       fmt.Sprintf("Validar smoke tests y QA de %s", normalized.Nombre),
		Descripcion:  "Ejecutar pruebas críticas, revisar regresiones, verificar cobertura mínima y dejar evidencia de que la app está lista para entrega técnica.",
		Modulo:       "qa",
		Prioridad:    PrioridadAlta,
		Dependencias: uniqueKeys(qaDeps...),
	}, "qa", "validacion", "Evidencia mínima de calidad",
		"Smoke tests ejecutados",
		"Regresiones críticas revisadas",
		"Cobertura mínima alcanzada",
		"Riesgos residuales anotados"))

	builder.add(withSOP(BlueprintTask{
		Key:          "documentacion",
		Titulo:       fmt.Sprintf("Cerrar documentación operativa de %s", normalized.Nombre),
		Descripcion:  composeDocumentationDescription(normalized),
		Modulo:       "docs",
		Prioridad:    PrioridadMedia,
		Dependencias: []string{"qa_smoke"},
	}, "documentador", "documentacion", "Documentación para usuarios, técnicos y sistemas",
		"README y manuales actualizados",
		"Guía de usuario básica disponible",
		"Guía técnica para desarrollo disponible",
		"Guía de sistemas y operación disponible",
		"Arquitectura hexagonal documentada con diagrama",
		"Operación y despliegue explicados",
		"Atribución visible a Orquesta"))

	finalDeps := []string{"qa_smoke", "documentacion"}
	if normalized.Docker {
		finalDeps = append(finalDeps, "docker")
	}
	if normalized.CI {
		finalDeps = append(finalDeps, "ci_cd")
	}

	builder.add(withSOP(BlueprintTask{
		Key:          "entrega_final",
		Titulo:       fmt.Sprintf("Cerrar entrega final de %s", normalized.Nombre),
		Descripcion:  "Verificar que no quedan flecos abiertos y que la app completa puede darse por terminada. Confirmar metadatos y atribución a Orquesta de Alberto Avidad Fernández en artefactos y documentación final.",
		Modulo:       "release",
		Prioridad:    PrioridadAlta,
		Dependencias: uniqueKeys(finalDeps...),
	}, "release_manager", "cierre", "Release candidata a entrega",
		"No quedan flecos críticos abiertos",
		"Artefactos finales validados",
		"Criterio de completitud explicitado"))

	return GenerationResult{Tasks: builder.tasks}, nil
}

func (s *Service) GenerateLanguageExpansion(nombre string, idiomas []string) (GenerationResult, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return GenerationResult{}, fmt.Errorf("nombre obligatorio")
	}
	idiomas = normalizeIdiomas(idiomas)
	if len(idiomas) == 0 {
		return GenerationResult{}, fmt.Errorf("idiomas obligatorios")
	}

	builder := blueprintBuilder{}
	for _, idioma := range idiomas {
		langKey := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(idioma)), "-", "_")
		baseKey := "i18n_expand_" + langKey
		docsKey := "documentacion_expand_" + langKey
		qaKey := "qa_i18n_expand_" + langKey
		builder.add(withSOP(BlueprintTask{
			Key:              baseKey,
			Titulo:           fmt.Sprintf("Ampliar %s a %s", nombre, strings.ToUpper(idioma)),
			Descripcion:      composeI18nExpansionDescription(nombre, idioma),
			Modulo:           "i18n",
			Prioridad:        PrioridadAlta,
			ContratoDefinido: true,
		}, "i18n", "implementacion", "Bundles y cadenas nuevas operativas",
			"Bundles base del idioma creados y conectados",
			"UI, API y mensajes operativos sin literales fijas en la superficie nueva",
			"Formato de fechas, números y textos críticos validado para el idioma nuevo",
		))
		builder.add(withSOP(BlueprintTask{
			Key:              docsKey,
			Titulo:           fmt.Sprintf("Traducir manuales de %s a %s", nombre, strings.ToUpper(idioma)),
			Descripcion:      composeLanguageDocumentationExpansionDescription(nombre, idioma),
			Modulo:           "documentacion",
			Prioridad:        PrioridadMedia,
			Dependencias:     []string{baseKey},
			ContratoDefinido: true,
		}, "documentador", "documentacion", "Manuales del nuevo idioma publicados",
			"Manual de usuario actualizado en el idioma nuevo",
			"Manual tecnico/desarrollo actualizado en el idioma nuevo",
			"Manual de sistemas/operacion actualizado en el idioma nuevo",
		))
		builder.add(withSOP(BlueprintTask{
			Key:              qaKey,
			Titulo:           fmt.Sprintf("Validar QA multilenguaje para %s en %s", nombre, strings.ToUpper(idioma)),
			Descripcion:      composeLanguageQAExpansionDescription(nombre, idioma),
			Modulo:           "qa",
			Prioridad:        PrioridadMedia,
			Dependencias:     []string{docsKey},
			ContratoDefinido: true,
		}, "qa", "validacion", "Evidencia funcional del nuevo idioma",
			"Navegacion principal validada en el nuevo idioma",
			"Mensajes de error, confirmacion y ayuda revisados",
			"Documentacion y producto coherentes para el idioma nuevo",
		))
	}
	return GenerationResult{Tasks: builder.tasks}, nil
}

// ── Builder ───────────────────────────────────────────────────────────────────

type blueprintBuilder struct {
	spec  AppSpec
	tasks []BlueprintTask
}

func (b *blueprintBuilder) add(task BlueprintTask) {
	task.Key = strings.TrimSpace(task.Key)
	task.Titulo = strings.TrimSpace(task.Titulo)
	task.Descripcion = strings.TrimSpace(task.Descripcion)
	task.Modulo = strings.TrimSpace(task.Modulo)
	task.RolSugerido = strings.TrimSpace(task.RolSugerido)
	task.Fase = strings.TrimSpace(task.Fase)
	task.Entregable = strings.TrimSpace(task.Entregable)
	task.Dependencias = uniqueKeys(task.Dependencias...)
	task.CriteriosCierre = uniqueKeys(task.CriteriosCierre...)
	b.tasks = append(b.tasks, task)
}

func (b *blueprintBuilder) featureDependencyKeys() []string {
	skip := map[string]bool{
		"briefing": true, "arquitectura": true, "tooling_calidad": true,
		"testing_base": true, "integracion": true, "docker": true,
		"qa_smoke": true, "documentacion": true, "entrega_final": true,
		"compliance": true, "ci_cd": true,
	}
	out := make([]string, 0, len(b.tasks))
	for _, t := range b.tasks {
		if !skip[t.Key] {
			out = append(out, t.Key)
		}
	}
	return uniqueKeys(out...)
}

// ── Normalización ─────────────────────────────────────────────────────────────

func normalizeSpec(spec AppSpec) (AppSpec, error) {
	spec.Nombre = strings.TrimSpace(spec.Nombre)
	spec.Descripcion = strings.TrimSpace(spec.Descripcion)
	spec.ObjetivoNegocio = strings.TrimSpace(spec.ObjetivoNegocio)
	spec.UsuariosObjetivo = strings.TrimSpace(spec.UsuariosObjetivo)
	spec.Restricciones = strings.TrimSpace(spec.Restricciones)
	spec.Tipo = strings.ToLower(strings.TrimSpace(spec.Tipo))
	spec.Idiomas = normalizeIdiomas(spec.Idiomas)
	spec.Arquitectura = normalizeArchitecture(spec.Arquitectura)
	spec.APIStyle = normalizeAPIStyle(spec.APIStyle)
	spec.FrontendStack = normalizeFrontendStack(spec.FrontendStack)
	spec.DatabaseEngine = normalizeDatabaseEngine(spec.DatabaseEngine)
	spec.AuthMode = normalizeAuthMode(spec.AuthMode)
	spec.IdentityProvider = normalizeIdentityProvider(spec.IdentityProvider)
	spec.TestingLevel = normalizeTestingLevel(spec.TestingLevel)
	spec.ObservabilityLevel = normalizeObservabilityLevel(spec.ObservabilityLevel)
	spec.DeploymentTarget = normalizeDeploymentTarget(spec.DeploymentTarget)
	spec.ArtifactType = normalizeArtifactType(spec.ArtifactType)
	spec.IntegracionesExternas = normalizeCSVList(spec.IntegracionesExternas)

	if spec.Nombre == "" {
		return AppSpec{}, fmt.Errorf("nombre obligatorio")
	}
	if spec.Tipo == "" {
		return AppSpec{}, fmt.Errorf("tipo obligatorio")
	}

	switch spec.Tipo {
	case "web":
		spec.Frontend = true
		spec.PlatWeb = true
	case "api":
		spec.API = true
		spec.PlatWeb = true
	case "web_api":
		spec.Frontend = true
		spec.API = true
		spec.PlatWeb = true
	case "cli":
		spec.PlatCLI = true
	case "desktop":
		spec.PlatDesktop = true
	case "mobile":
		spec.PlatMobile = true
	case "embedded":
		spec.PlatEmbedded = true
	default:
		return AppSpec{}, fmt.Errorf("tipo de app no soportado: %s", spec.Tipo)
	}

	if !spec.Database {
		spec.DatabaseEngine = "none"
	} else if spec.DatabaseEngine == "none" {
		spec.DatabaseEngine = RecommendDatabaseEngine(spec)
	}
	if hasThemeableUI(spec) && spec.FrontendStack == "server_rendered" && (spec.Tipo == "web_api" || spec.Frontend) {
		spec.FrontendStack = RecommendFrontendStack(spec)
	}
	if !spec.Auth {
		spec.AuthMode = "none"
	} else if spec.AuthMode == "none" {
		spec.AuthMode = RecommendAuthMode(spec)
	}
	if hasThemeableUI(spec) {
		spec.ThemeSupport = true
	}

	return spec, nil
}

func normalizeSpecForDatabaseRecommendation(spec AppSpec) AppSpec {
	spec.Tipo = strings.ToLower(strings.TrimSpace(spec.Tipo))
	spec.AuthMode = normalizeAuthMode(spec.AuthMode)
	spec.IdentityProvider = normalizeIdentityProvider(spec.IdentityProvider)
	spec.DeploymentTarget = normalizeDeploymentTarget(spec.DeploymentTarget)
	spec.ArtifactType = normalizeArtifactType(spec.ArtifactType)

	switch spec.Tipo {
	case "web":
		spec.Frontend = true
		spec.PlatWeb = true
	case "api":
		spec.API = true
		spec.PlatWeb = true
	case "web_api":
		spec.Frontend = true
		spec.API = true
		spec.PlatWeb = true
	case "cli":
		spec.PlatCLI = true
	case "desktop":
		spec.PlatDesktop = true
	case "mobile":
		spec.PlatMobile = true
	case "embedded":
		spec.PlatEmbedded = true
	}

	return spec
}

func normalizeIdiomas(idiomas []string) []string {
	return normalizeCSVList(idiomas)
}

func normalizeCSVList(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" || slices.Contains(out, item) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func normalizeArchitecture(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "hexagonal":
		return "hexagonal"
	case "clean", "layered", "event_driven":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "hexagonal"
	}
}

func normalizeAPIStyle(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "rest":
		return "rest"
	case "graphql", "grpc", "async", "mixed":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "rest"
	}
}

func normalizeFrontendStack(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "server_rendered":
		return "server_rendered"
	case "react", "vue", "sveltekit", "nextjs", "astro", "htmx", "angular", "other":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "other"
	}
}

func normalizeDatabaseEngine(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "none":
		return "none"
	case "postgres", "mysql", "sqlite", "mongodb", "redis", "other":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "other"
	}
}

func normalizeAuthMode(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "none":
		return "none"
	case "session", "jwt", "oauth", "sso", "api_key":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "none"
	}
}

func normalizeTestingLevel(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "base":
		return "base"
	case "strict", "tdd":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "base"
	}
}

func normalizeIdentityProvider(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "none":
		return "none"
	case "active_directory", "ldap", "entra_id", "keycloak", "auth0", "google", "other":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "none"
	}
}

func normalizeObservabilityLevel(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "standard":
		return "standard"
	case "basic", "strict":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "standard"
	}
}

func normalizeDeploymentTarget(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "docker":
		return "docker"
	case "kubernetes", "serverless", "vm", "edge":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "docker"
	}
}

func normalizeArtifactType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "executable":
		return "executable"
	case "docker_image", "library", "desktop_installer", "mobile_bundle", "firmware", "static_site":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "executable"
	}
}

func needsDurablePersistence(spec AppSpec) bool {
	return spec.Database ||
		spec.Auth ||
		spec.AuthMode != "none" ||
		spec.IdentityProvider != "none" ||
		spec.FileUploads ||
		spec.ImportExport ||
		spec.Reporting ||
		spec.Search ||
		spec.AuditTrail ||
		spec.MultiTenant ||
		spec.BackgroundJobs ||
		spec.Queue ||
		spec.Scheduler ||
		spec.ServicioResidente ||
		spec.Notifications ||
		spec.Webhooks ||
		spec.ObjectStorage ||
		spec.Backups ||
		spec.DisasterRecovery
}

func needsServerGradeDatabase(spec AppSpec) bool {
	return spec.API ||
		spec.MultiTenant ||
		spec.RBAC ||
		spec.Reporting ||
		spec.BackgroundJobs ||
		spec.Queue ||
		spec.Scheduler ||
		spec.ServicioResidente ||
		spec.Notifications ||
		spec.Webhooks ||
		spec.AuditTrail ||
		spec.Backups ||
		spec.DisasterRecovery ||
		spec.AuthMode == "oauth" ||
		spec.AuthMode == "sso" ||
		spec.AuthMode == "api_key" ||
		spec.IdentityProvider != "none" ||
		spec.DeploymentTarget == "kubernetes" ||
		spec.Kubernetes ||
		spec.ArtifactType == "docker_image"
}

func prefersEmbeddedDatabase(spec AppSpec) bool {
	return needsDurablePersistence(spec) &&
		(spec.OfflineMode ||
			spec.Tipo == "desktop" ||
			spec.Tipo == "mobile" ||
			spec.Tipo == "cli" ||
			spec.Tipo == "embedded" ||
			spec.PlatDesktop ||
			spec.PlatMobile ||
			spec.PlatCLI ||
			spec.PlatEmbedded) &&
		!needsServerGradeDatabase(spec)
}

func uniqueKeys(keys ...string) []string {
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" || slices.Contains(out, key) {
			continue
		}
		out = append(out, key)
	}
	return out
}

func hasCompliance(spec AppSpec) bool {
	return spec.ComplianceRGPD || spec.ComplianceENS || spec.ComplianceLSSI ||
		spec.ComplianceWCAG || spec.ComplianceFacturaElec || spec.ComplianceReutilizacion
}

func requiresResidentService(spec AppSpec) bool {
	return spec.API || spec.ServicioResidente || spec.BackgroundJobs || spec.Queue || spec.Scheduler
}

func hasThemeableUI(spec AppSpec) bool {
	return spec.Frontend || spec.Tipo == "web" || spec.Tipo == "web_api" || spec.Tipo == "desktop" || spec.Tipo == "mobile"
}

// ── Helpers de tareas específicas ─────────────────────────────────────────────

func withSOP(task BlueprintTask, rol, fase, entregable string, criterios ...string) BlueprintTask {
	task.RolSugerido = strings.TrimSpace(rol)
	task.Fase = strings.TrimSpace(fase)
	task.Entregable = strings.TrimSpace(entregable)
	task.CriteriosCierre = uniqueKeys(criterios...)
	return task
}

func featureDepsForAuth(spec AppSpec) []string {
	var deps []string
	if spec.API {
		deps = append(deps, "api_base")
	}
	if spec.Frontend {
		deps = append(deps, "frontend_base")
	}
	if spec.Tipo == "cli" {
		deps = append(deps, "cli_base")
	}
	if spec.Tipo == "desktop" {
		deps = append(deps, "desktop_base")
	}
	if spec.Tipo == "mobile" {
		deps = append(deps, "mobile_base")
	}
	return deps
}

func frontendTask(spec AppSpec, deps []string) BlueprintTask {
	description := "Crear shell visual, navegación principal, estados base y estructura inicial de interfaz. El frontend se implementa como adaptador de UI en la arquitectura hexagonal, sin lógica de negocio."
	if stack := describeFrontendStack(spec.FrontendStack); stack != "" {
		description += " Stack preferido: " + stack + "."
	}
	if root := describeUISurfaceRoot(spec); root != "" {
		description += " La estructura debe dejar una superficie UI explícita en " + root + "."
	}
	return withSOP(BlueprintTask{
		Key:              "frontend_base",
		Titulo:           fmt.Sprintf("Implementar frontend base de %s", spec.Nombre),
		Descripcion:      description,
		Modulo:           "frontend",
		Prioridad:        PrioridadAlta,
		Dependencias:     uniqueKeys(deps...),
		ContratoDefinido: true,
	}, "frontend", "implementacion", "UI principal funcional",
		"Shell visual y navegación base cerrados",
		"Flujo principal navegable",
		"Sin lógica de negocio en la capa UI")
}

func apiTask(spec AppSpec, deps []string) BlueprintTask {
	return withSOP(BlueprintTask{
		Key:              "api_base",
		Titulo:           fmt.Sprintf("Implementar API base de %s", spec.Nombre),
		Descripcion:      "Definir endpoints, contratos de entrada/salida, validación y capa de aplicación. La API es un adaptador de entrada (puerto HTTP) en la arquitectura hexagonal.",
		Modulo:           "backend",
		Prioridad:        PrioridadAlta,
		Dependencias:     uniqueKeys(deps...),
		ContratoDefinido: true,
	}, "backend", "implementacion", "API principal operativa",
		"Endpoints base cerrados",
		"Contratos de entrada/salida definidos",
		"Validación y errores principales cubiertos",
		"Sin acceso directo a BD desde handlers")
}

func cliTask(spec AppSpec, deps []string) BlueprintTask {
	return withSOP(BlueprintTask{
		Key:              "cli_base",
		Titulo:           fmt.Sprintf("Implementar CLI base de %s", spec.Nombre),
		Descripcion:      "Crear comandos principales, contratos de salida y flujo operativo inicial. La CLI es un adaptador de entrada en la arquitectura hexagonal.",
		Modulo:           "cli",
		Prioridad:        PrioridadAlta,
		Dependencias:     uniqueKeys(deps...),
		ContratoDefinido: true,
	}, "backend", "implementacion", "CLI principal operativa",
		"Comandos base disponibles",
		"Ayuda y salida principal coherentes",
		"Flujo principal probado")
}

func desktopTask(spec AppSpec, deps []string) BlueprintTask {
	soList := composeSoList(spec)
	desc := fmt.Sprintf("Implementar aplicación de escritorio para %s.", strings.Join(soList, ", "))
	desc += " Elegir framework (Tauri, Electron, Qt, GTK, etc.) según SO objetivo. La lógica de negocio reside en el núcleo hexagonal; la UI de escritorio es un adaptador."
	return withSOP(BlueprintTask{
		Key:              "desktop_base",
		Titulo:           fmt.Sprintf("Implementar app de escritorio base de %s", spec.Nombre),
		Descripcion:      desc,
		Modulo:           "desktop",
		Prioridad:        PrioridadAlta,
		Dependencias:     uniqueKeys(deps...),
		ContratoDefinido: true,
	}, "frontend", "implementacion", "App de escritorio funcional en SO objetivo",
		"Instalador o binario generado para cada SO objetivo",
		"Flujo principal navegable en escritorio",
		"Firma de ejecutables configurada si aplica")
}

func mobileTask(spec AppSpec, deps []string) BlueprintTask {
	platforms := []string{}
	if spec.SOAndroid {
		platforms = append(platforms, "Android")
	}
	if spec.SOiOS {
		platforms = append(platforms, "iOS")
	}
	if len(platforms) == 0 {
		platforms = []string{"Android", "iOS"}
	}
	desc := fmt.Sprintf("Implementar app móvil para %s.", strings.Join(platforms, " y "))
	desc += " Elegir stack (React Native, Flutter, nativo) según requisitos. La lógica de negocio reside en el núcleo hexagonal."
	return withSOP(BlueprintTask{
		Key:              "mobile_base",
		Titulo:           fmt.Sprintf("Implementar app móvil base de %s", spec.Nombre),
		Descripcion:      desc,
		Modulo:           "mobile",
		Prioridad:        PrioridadAlta,
		Dependencias:     uniqueKeys(deps...),
		ContratoDefinido: true,
	}, "frontend", "implementacion", "App móvil funcional en plataformas objetivo",
		"Build generado para cada plataforma objetivo",
		"Flujo principal operativo en dispositivo o emulador",
		"Permisos mínimos necesarios declarados")
}

func embeddedTask(spec AppSpec, deps []string) BlueprintTask {
	return withSOP(BlueprintTask{
		Key:              "embedded_base",
		Titulo:           fmt.Sprintf("Implementar firmware/software embebido base de %s", spec.Nombre),
		Descripcion:      "Implementar el núcleo del sistema embebido: HAL (Hardware Abstraction Layer), drivers básicos, bucle principal y protocolo de comunicación. Arquitectura en capas con núcleo de negocio aislado del hardware.",
		Modulo:           "embedded",
		Prioridad:        PrioridadAlta,
		Dependencias:     uniqueKeys(deps...),
		ContratoDefinido: true,
	}, "backend", "implementacion", "Firmware base operativo en hardware objetivo",
		"Compilación reproducible documentada",
		"HAL con tests unitarios en host",
		"Protocolo de comunicación especificado")
}

func integrationTask(spec AppSpec, deps []string) BlueprintTask {
	description := "Integrar módulos principales y cerrar el flujo funcional de extremo a extremo."
	switch spec.Tipo {
	case "web":
		description = "Integrar frontend, servicios y flujos principales de la aplicación web."
	case "api":
		description = "Integrar endpoints, persistencia y casos de uso de la API."
	case "web_api":
		description = "Integrar frontend, API, persistencia y flujos de extremo a extremo de la app."
	case "cli":
		description = "Integrar comandos, persistencia y flujos operativos completos de la CLI."
	case "desktop":
		description = "Integrar UI de escritorio con núcleo de negocio y persistencia."
	case "mobile":
		description = "Integrar UI móvil con núcleo de negocio, APIs y persistencia."
	case "embedded":
		description = "Integrar firmware con hardware objetivo y validar en entorno real o simulado."
	}
	return withSOP(BlueprintTask{
		Key:          "integracion",
		Titulo:       fmt.Sprintf("Integrar flujo principal de %s", spec.Nombre),
		Descripcion:  description,
		Modulo:       "integracion",
		Prioridad:    PrioridadAlta,
		Dependencias: uniqueKeys(deps...),
	}, "integrador", "integracion", "Sistema integrado demostrable",
		"Las piezas principales conversan entre sí",
		"Flujo end-to-end verificable",
		"Incidencias de integración acotadas")
}

// ── Composición de descripciones ──────────────────────────────────────────────

func composeBriefingDescription(spec AppSpec) string {
	parts := []string{
		fmt.Sprintf("Alinear alcance, personas usuarias, flujos principales y criterios de entrega de la app %s.", spec.Nombre),
		fmt.Sprintf("Tipo objetivo: %s.", spec.Tipo),
		fmt.Sprintf("Arquitectura objetivo: %s.", describeArchitecture(spec.Arquitectura)),
	}
	if spec.Descripcion != "" {
		parts = append(parts, "Contexto: "+spec.Descripcion)
	}
	if hasThemeableUI(spec) {
		parts = append(parts, "La app debe soportar theming y branding desacoplado como capacidad de producto, no como CSS puntual.")
	}
	if spec.ObjetivoNegocio != "" {
		parts = append(parts, "Objetivo de negocio: "+spec.ObjetivoNegocio)
	}
	if spec.UsuariosObjetivo != "" {
		parts = append(parts, "Usuarios objetivo: "+spec.UsuariosObjetivo)
	}
	if spec.Restricciones != "" {
		parts = append(parts, "Restricciones iniciales: "+spec.Restricciones)
	}
	if spec.API {
		parts = append(parts, fmt.Sprintf("Estilo de interfaz API esperado: %s.", describeAPIStyle(spec.APIStyle)))
	}
	if spec.Database {
		parts = append(parts, fmt.Sprintf("Persistencia prevista: %s.", describeDatabaseEngine(spec.DatabaseEngine)))
	}
	if spec.Auth {
		parts = append(parts, fmt.Sprintf("Modelo de acceso preferido: %s.", describeAuthMode(spec.AuthMode)))
		if spec.RBAC {
			parts = append(parts, "La autorización debe resolverse con RBAC.")
		}
		if idp := describeIdentityProvider(spec.IdentityProvider); idp != "" {
			parts = append(parts, fmt.Sprintf("Proveedor de identidad previsto: %s.", idp))
		}
	}
	soList := composeSoList(spec)
	if len(soList) > 0 {
		parts = append(parts, fmt.Sprintf("SO objetivo: %s.", strings.Join(soList, ", ")))
	}
	if spec.BackgroundJobs {
		parts = append(parts, "La app debe contemplar procesos asíncronos o jobs en segundo plano.")
	}
	if spec.OfflineMode {
		parts = append(parts, "La app debe contemplar modo offline y estrategia de sincronización.")
	}
	if spec.ServicioResidente {
		parts = append(parts, "La app requiere un núcleo o servicio residente explícito.")
	}
	if spec.Queue {
		parts = append(parts, "La app debe incorporar colas o mensajería asíncrona.")
	}
	if spec.Scheduler {
		parts = append(parts, "La app necesita tareas programadas o cron de negocio.")
	}
	if spec.Cache {
		parts = append(parts, "La app debe contemplar una capa de caché.")
	}
	if spec.ObjectStorage {
		parts = append(parts, "La app debe gestionar ficheros o blobs en object storage.")
	}
	if spec.Search {
		parts = append(parts, "La app necesita búsqueda o indexación dedicada.")
	}
	if spec.ImportExport {
		parts = append(parts, "La app debe importar y exportar datos de forma controlada.")
	}
	if spec.Webhooks {
		parts = append(parts, "La app debe exponer o consumir webhooks/eventos con terceros.")
	}
	if spec.FileUploads {
		parts = append(parts, "La app debe gestionar subida y ciclo de vida de ficheros.")
	}
	if spec.Reporting {
		parts = append(parts, "La app debe ofrecer reporting, cuadros o analítica funcional.")
	}
	if spec.Notifications {
		parts = append(parts, "La app debe contemplar notificaciones salientes o entrantes.")
	}
	if spec.MultiTenant {
		parts = append(parts, "La app debe diseñarse con separación multi-tenant.")
	}
	if len(spec.IntegracionesExternas) > 0 {
		parts = append(parts, fmt.Sprintf("Integraciones externas declaradas: %s.", strings.Join(spec.IntegracionesExternas, ", ")))
	}
	if spec.ArtifactType != "" {
		parts = append(parts, fmt.Sprintf("Artefacto principal de entrega: %s.", describeArtifactType(spec.ArtifactType)))
	}
	if hasCompliance(spec) {
		parts = append(parts, "Incluir requisitos legales y normativos en el brief.")
	}
	return strings.Join(parts, " ")
}

func composeResearchDescription(spec AppSpec) string {
	parts := []string{
		"Revisar estudios de apps comparables, catálogo compartido de librerías y restricciones técnicas antes de programar.",
		fmt.Sprintf("Extraer decisiones reutilizables, dependencias candidatas y riesgos tempranos para %s.", spec.Nombre),
	}
	if spec.I18n {
		parts = append(parts, "Confirmar implicaciones de i18n desde el inicio.")
	}
	if spec.Database {
		parts = append(parts, fmt.Sprintf("Comparar alternativas de persistencia y validar por qué %s encaja mejor.", describeDatabaseEngine(spec.DatabaseEngine)))
	}
	if spec.API {
		parts = append(parts, fmt.Sprintf("Validar el encaje de %s frente a otras opciones de integración.", describeAPIStyle(spec.APIStyle)))
	}
	if spec.Auth {
		parts = append(parts, fmt.Sprintf("Evaluar riesgos y tradeoffs de %s para autenticación y autorización.", describeAuthMode(spec.AuthMode)))
		if idp := describeIdentityProvider(spec.IdentityProvider); idp != "" {
			parts = append(parts, fmt.Sprintf("Revisar integración con %s y sus requisitos de provisioning, grupos y claims.", idp))
		}
	}
	if hasCompliance(spec) {
		parts = append(parts, "Revisar normativas aplicables y su impacto técnico.")
	}
	if spec.PlatDesktop || spec.PlatMobile {
		parts = append(parts, "Evaluar frameworks multiplataforma y sus restricciones.")
	}
	if len(spec.IntegracionesExternas) > 0 {
		parts = append(parts, fmt.Sprintf("Revisar SDKs, cuotas y contratos de las integraciones declaradas: %s.", strings.Join(spec.IntegracionesExternas, ", ")))
	}
	if spec.Cache || spec.Queue || spec.ObjectStorage || spec.Search || spec.Webhooks || spec.FileUploads {
		parts = append(parts, "Revisar también servicios de infraestructura gestionada, costes y estrategia local/dev frente a producción.")
	}
	return strings.Join(parts, " ")
}

func composeArchitectureDescription(spec AppSpec) string {
	parts := []string{
		fmt.Sprintf("Diseñar arquitectura %s: núcleo de dominio aislado de infraestructura, adaptadores de entrada (API, CLI, UI) y salida (BD, servicios externos).", describeArchitecture(spec.Arquitectura)),
		"Definir módulos con contratos explícitos entre sí. Ningún módulo depende de la implementación concreta de otro.",
	}
	if requiresResidentService(spec) {
		parts = append(parts, "Si la app necesita servicio residente, el núcleo siempre vivo debe ser mínimo: continuidad operativa, hot state imprescindible y coordinación ligera. La supervisión rica, snapshots pesados, reconciliaciones profundas y lecturas no críticas deben salir a workers separados, caminos on-demand o procesamiento orientado a eventos.")
	}
	if spec.Database {
		parts = append(parts, fmt.Sprintf("Incluir puerto de persistencia con adaptador(es) para %s.", describeDatabaseEngine(spec.DatabaseEngine)))
	}
	if spec.Auth {
		parts = append(parts, fmt.Sprintf("Incluir puerto de autenticación/autorización alineado con %s.", describeAuthMode(spec.AuthMode)))
		if spec.RBAC {
			parts = append(parts, "Modelar RBAC explícito con roles, permisos y puntos de decisión de acceso.")
		}
		if idp := describeIdentityProvider(spec.IdentityProvider); idp != "" {
			parts = append(parts, "Separar la integración con "+idp+" detrás de un adaptador de identidad.")
		}
	}
	if spec.I18n {
		parts = append(parts, "Diseñar con i18n desde el inicio.")
	}
	soList := composeSoList(spec)
	if len(soList) > 1 {
		parts = append(parts, fmt.Sprintf("Estrategia multiplataforma para: %s.", strings.Join(soList, ", ")))
	}
	if spec.Monitoring {
		parts = append(parts, fmt.Sprintf("Incluir puertos de observabilidad con nivel %s (logs, métricas, trazas).", describeObservabilityLevel(spec.ObservabilityLevel)))
	}
	if spec.BackgroundJobs {
		parts = append(parts, "Separar explícitamente jobs, colas o tareas programadas del request path principal.")
	}
	if spec.OfflineMode {
		parts = append(parts, "Diseñar estrategia de sincronización, versionado y resolución de conflictos para trabajo offline.")
	}
	if spec.Cache {
		parts = append(parts, "Diseñar puerto/adaptador de caché y reglas de invalidación.")
	}
	if spec.Queue {
		parts = append(parts, "Diseñar puerto/adaptador para colas, reintentos y dead-letter si aplica.")
	}
	if spec.ObjectStorage {
		parts = append(parts, "Diseñar puerto/adaptador para almacenamiento de objetos y lifecycle de ficheros.")
	}
	if spec.Search {
		parts = append(parts, "Diseñar puerto/adaptador para indexación y consulta de búsqueda.")
	}
	if spec.RateLimiting {
		parts = append(parts, "Definir estrategia de rate limiting y cuotas por actor, tenant o API key.")
	}
	if spec.FeatureFlags {
		parts = append(parts, "Prever feature flags para despliegue progresivo y rollback funcional.")
	}
	if spec.AuditTrail {
		parts = append(parts, "Definir eventos auditables y trazabilidad funcional de acciones sensibles.")
	}
	if spec.Webhooks {
		parts = append(parts, "Diseñar contratos de webhook firmados, idempotencia y reintentos.")
	}
	if spec.FileUploads {
		parts = append(parts, "Diseñar flujo de subida, validación, antivirus y lifecycle de ficheros.")
	}
	if spec.Reporting {
		parts = append(parts, "Separar cargas OLTP y reporting cuando el volumen lo exija.")
	}
	if spec.MultiTenant {
		parts = append(parts, "Definir estrategia de aislamiento multi-tenant en dominio, persistencia y observabilidad.")
	}
	if spec.Notifications {
		parts = append(parts, "Modelar adaptadores para email, push, webhooks o mensajería si forman parte del producto.")
	}
	return strings.Join(parts, " ")
}

func composeResidentServiceDescription(spec AppSpec) string {
	parts := []string{
		fmt.Sprintf("La app %s necesita servicio residente, pero no debe convertir todo su trabajo interno en un daemon caliente.", spec.Nombre),
		"Definir un núcleo residente mínimo: entradas de servicio, coordinación ligera, continuidad operativa y solo el estado caliente imprescindible en memoria.",
		"Separar del núcleo todo lo que pueda ejecutarse por evento, por worker específico o bajo demanda: snapshots pesados, auditorías, reconciliaciones profundas, observabilidad rica y scans globales.",
		"Distinguir de forma explícita el estado caliente efímero del estado durable. Lo efímero puede vivir en memoria; lo contractual o recuperable tras reinicio debe seguir siendo persistente.",
		"Fijar frecuencias de observación razonables: presupuesto al inicio de cada sesión de trabajo y después cada 1-2 minutos; snapshots de estado y revisiones de procesos no más rápidas de lo necesario.",
	}
	return strings.Join(parts, " ")
}

func composeProfessionalBaselineDescription(spec AppSpec) string {
	parts := []string{
		"Fijar el baseline profesional obligatorio del repositorio desde el primer commit.",
		"Externalizar configuración y secretos por entorno, con bootstrap reproducible para desarrollo, CI y despliegue.",
		"Definir política de errores, validación de entrada, contratos de salida y logging estructurado con contexto suficiente para operar.",
		"Documentar convenciones de ramas, commits, revisión, ownership y runbook mínimo de desarrollo.",
		"Dejar automatizados formatter, linter, tests mínimos y verificación de dependencias críticas.",
	}
	if spec.Database {
		parts = append(parts, "Asegurar migraciones, seed mínimo y estrategia de rollback de cambios persistentes.")
	}
	if spec.Monitoring || spec.ObservabilityLevel != "basic" {
		parts = append(parts, "Incluir correlación entre logs, métricas y trazas desde la base.")
	}
	if spec.Auth || spec.ComplianceENS || spec.ComplianceRGPD {
		parts = append(parts, "Aplicar baseline de seguridad: mínimos privilegios, manejo correcto de secretos y revisión de dependencias y superficies expuestas.")
	}
	return strings.Join(parts, " ")
}

func composeDocumentationDescription(spec AppSpec) string {
	parts := []string{
		"Documentar arquitectura hexagonal, módulos, uso, despliegue, i18n y operación.",
		"La documentación mínima obligatoria se produce para tres públicos: usuarios finales, técnicos/desarrollo y sistemas/operación.",
		"Incluir atribución visible a Orquesta de Alberto Avidad Fernández en README, manuales y piezas documentales generadas.",
	}
	if spec.I18n && len(spec.Idiomas) > 0 {
		parts = append(parts, "Los manuales deben generarse en los idiomas elegidos: "+strings.Join(spec.Idiomas, ", ")+".")
	}
	return strings.Join(parts, " ")
}

func composePersistenceDescription(spec AppSpec) string {
	parts := []string{
		"Definir esquema, migraciones, acceso a datos y contratos de almacenamiento.",
		"La BD se accede exclusivamente a través del adaptador de persistencia (puerto hexagonal). Ninguna capa de aplicación o dominio importa el driver directamente.",
	}
	if engine := describeDatabaseEngine(spec.DatabaseEngine); engine != "" && spec.DatabaseEngine != "none" {
		parts = append(parts, "Motor objetivo: "+engine+".")
	}
	if spec.ComplianceRGPD {
		parts = append(parts, "Cifrar datos personales en reposo. Implementar retención y derecho al olvido.")
	}
	if spec.MultiTenant {
		parts = append(parts, "Asegurar partición o scoping tenant-aware en esquema, queries e índices.")
	}
	if spec.Backups {
		parts = append(parts, "Definir política de backups y restauración verificable.")
	}
	return strings.Join(parts, " ")
}

func composeAuthDescription(spec AppSpec) string {
	parts := []string{
		"Implementar autenticación y autorización como adaptador de seguridad en la arquitectura hexagonal.",
		"Definir roles, permisos, validación de acceso y puntos de integración de seguridad.",
	}
	if spec.AuthMode != "none" {
		parts = append(parts, "Modo objetivo: "+describeAuthMode(spec.AuthMode)+".")
	}
	if spec.RBAC {
		parts = append(parts, "La matriz de roles y permisos debe quedar explícita y versionada.")
	}
	if idp := describeIdentityProvider(spec.IdentityProvider); idp != "" {
		parts = append(parts, "Proveedor de identidad a integrar: "+idp+".")
	}
	if spec.ComplianceRGPD {
		parts = append(parts, "Mínimo privilegio en acceso a datos personales. Log de accesos.")
	}
	if spec.ComplianceENS {
		parts = append(parts, "Gestión de identidades conforme a ENS.")
	}
	if spec.RateLimiting {
		parts = append(parts, "Aplicar límites de uso y controles anti abuso en accesos sensibles.")
	}
	return strings.Join(parts, " ")
}

func composeUIThemesDescription(spec AppSpec) string {
	parts := []string{
		"Definir theming y branding desacoplados de la lógica de negocio.",
		"Usar tokens semánticos de color, tipografía, espaciado e iconografía en vez de valores duros repartidos por la UI.",
		"El cambio de tema debe poder hacerse creando o cargando un tema, sin tocar casos de uso, servicios ni componentes de negocio.",
	}
	if stack := describeFrontendStack(spec.FrontendStack); stack != "" {
		parts = append(parts, "Stack de UI previsto: "+stack+".")
	}
	if root := describeUISurfaceRoot(spec); root != "" {
		parts = append(parts, "La superficie de UI debe vivir de forma explícita en "+root+" y no dispersa por el repositorio.")
	}
	if spec.MultiTenant || spec.BrandingProfiles {
		parts = append(parts, "Preparar branding por tenant o perfiles white-label cuando aplique.")
	} else {
		parts = append(parts, "Preparar al menos tema base, oscuro y un branding alternativo demostrable.")
	}
	return strings.Join(parts, " ")
}

func describeUISurfaceRoot(spec AppSpec) string {
	if !hasThemeableUI(spec) {
		return ""
	}
	if spec.FrontendStack == "server_rendered" {
		return "`adaptadores/web/` para handlers, templates y assets, con temas aislados en una subcarpeta propia"
	}
	if spec.Tipo == "desktop" || spec.Tipo == "mobile" {
		return "`ui/` como superficie visual principal, con temas aislados bajo esa raíz"
	}
	return "`web/` como raíz del frontend, con temas/branding desacoplados dentro de esa superficie"
}

func composeI18nDescription(spec AppSpec) string {
	if len(spec.Idiomas) == 0 {
		return "Aplicar i18n desde el inicio, evitando literales fijas y dejando bundles preparados para varios idiomas."
	}
	return fmt.Sprintf("Aplicar i18n desde el inicio y dejar bundles operativos para: %s.", strings.Join(spec.Idiomas, ", "))
}

func composeI18nExpansionDescription(nombre, idioma string) string {
	return fmt.Sprintf(
		"Ampliar el soporte multilenguaje de %s para el idioma %s sin reabrir la arquitectura completa. "+
			"Crear o completar bundles, catálogos, textos de UI, mensajes de API, notificaciones y superficies operativas necesarias. "+
			"El idioma nuevo debe quedar integrado en la estrategia de i18n existente, sin literales duras ni bifurcaciones de lógica.",
		nombre, strings.ToUpper(strings.TrimSpace(idioma)))
}

func composeLanguageDocumentationExpansionDescription(nombre, idioma string) string {
	return fmt.Sprintf(
		"Actualizar la documentación de %s en el idioma %s para los tres públicos obligatorios: usuarios finales, técnicos/desarrollo y sistemas/operación. "+
			"No basta con traducir pantallas; los manuales y runbooks del proyecto deben quedar disponibles también en el idioma nuevo.",
		nombre, strings.ToUpper(strings.TrimSpace(idioma)))
}

func composeLanguageQAExpansionDescription(nombre, idioma string) string {
	return fmt.Sprintf(
		"Ejecutar una validación específica del idioma %s en %s cubriendo UI, mensajes, flujos críticos y documentación publicada. "+
			"La salida debe dejar evidencia de que el idioma nuevo funciona de forma consistente y no rompe accesibilidad, navegación ni soporte operativo.",
		strings.ToUpper(strings.TrimSpace(idioma)), nombre)
}

func composeComplianceDescription(spec AppSpec) string {
	normas := []string{}
	if spec.ComplianceRGPD {
		normas = append(normas, "RGPD (UE 2016/679)")
	}
	if spec.ComplianceENS {
		normas = append(normas, "ENS (RD 311/2022)")
	}
	if spec.ComplianceLSSI {
		normas = append(normas, "LSSI (Ley 34/2002)")
	}
	if spec.ComplianceWCAG {
		normas = append(normas, "WCAG 2.1 AA (RD 1112/2018)")
	}
	if spec.ComplianceFacturaElec {
		normas = append(normas, "Facturación electrónica (Ley 18/2022)")
	}
	if spec.ComplianceReutilizacion {
		normas = append(normas, "Reutilización sector público (Ley 37/2007)")
	}
	return fmt.Sprintf(
		"Analizar el impacto de las normativas aplicables en la arquitectura, datos y procesos de %s: %s. "+
			"Producir un plan de adecuación con responsables, plazos y controles por normativa.",
		spec.Nombre, strings.Join(normas, ", "))
}

func composeCICDDescription(spec AppSpec) string {
	parts := []string{
		"Configurar pipeline CI/CD: build, tests, linting, análisis estático y despliegue automático a entorno de pruebas.",
	}
	parts = append(parts, "Nivel de testing esperado: "+describeTestingLevel(spec.TestingLevel)+".")
	if spec.Docker {
		parts = append(parts, "Incluir build y push de imagen Docker.")
	}
	if spec.Kubernetes {
		parts = append(parts, "Incluir deploy automático a Kubernetes.")
	}
	if spec.DeploymentTarget != "" {
		parts = append(parts, "Objetivo principal de despliegue: "+describeDeploymentTarget(spec.DeploymentTarget)+".")
	}
	if spec.ArtifactType != "" {
		parts = append(parts, "Artefacto principal a producir: "+describeArtifactType(spec.ArtifactType)+".")
	}
	if spec.Backups {
		parts = append(parts, "Automatizar política de backups, verificación y restore drill.")
	}
	if spec.DisasterRecovery {
		parts = append(parts, "Definir runbooks y pruebas mínimas de disaster recovery.")
	}
	if spec.ComplianceENS {
		parts = append(parts, "Incluir análisis de vulnerabilidades (SAST/DAST) en el pipeline.")
	}
	return strings.Join(parts, " ")
}

func composeMonitoringDescription(spec AppSpec) string {
	parts := []string{
		fmt.Sprintf("Implementar observabilidad con nivel %s: logs estructurados (JSON), métricas (Prometheus/OpenMetrics) y trazas distribuidas (OpenTelemetry).", describeObservabilityLevel(spec.ObservabilityLevel)),
		"Configurar dashboard básico de salud y alertas críticas.",
	}
	if spec.ComplianceENS {
		parts = append(parts, "Los logs de seguridad deben conservarse según política ENS.")
	}
	if spec.ComplianceRGPD {
		parts = append(parts, "No registrar datos personales en logs sin anonimización.")
	}
	if spec.AuditTrail {
		parts = append(parts, "Mantener audit trail funcional separado de logs técnicos cuando aplique.")
	}
	return strings.Join(parts, " ")
}

func describeArchitecture(v string) string {
	switch v {
	case "clean":
		return "clean architecture con límites explícitos"
	case "layered":
		return "arquitectura por capas con adaptadores aislados"
	case "event_driven":
		return "arquitectura orientada a eventos con puertos y adaptadores"
	default:
		return "arquitectura hexagonal (puertos y adaptadores)"
	}
}

func describeAPIStyle(v string) string {
	switch v {
	case "graphql":
		return "GraphQL"
	case "grpc":
		return "gRPC"
	case "async":
		return "mensajería/eventos asíncronos"
	case "mixed":
		return "combinación de REST y otros contratos"
	default:
		return "REST/HTTP"
	}
}

func describeFrontendStack(v string) string {
	switch v {
	case "react":
		return "React"
	case "vue":
		return "Vue"
	case "sveltekit":
		return "SvelteKit"
	case "nextjs":
		return "Next.js"
	case "astro":
		return "Astro"
	case "htmx":
		return "HTMX"
	case "server_rendered":
		return "server-rendered"
	case "angular":
		return "Angular"
	case "other":
		return "otro stack web"
	default:
		return ""
	}
}

func describeDatabaseEngine(v string) string {
	switch v {
	case "postgres":
		return "PostgreSQL"
	case "mysql":
		return "MySQL/MariaDB"
	case "sqlite":
		return "SQLite"
	case "mongodb":
		return "MongoDB"
	case "redis":
		return "Redis o almacenamiento clave/valor"
	case "other":
		return "motor no relacional o específico del dominio"
	default:
		return ""
	}
}

func describeAuthMode(v string) string {
	switch v {
	case "session":
		return "sesiones servidoras"
	case "jwt":
		return "tokens JWT"
	case "oauth":
		return "OAuth/OIDC"
	case "sso":
		return "SSO corporativo"
	case "api_key":
		return "API keys"
	default:
		return "sin autenticación específica"
	}
}

func describeTestingLevel(v string) string {
	switch v {
	case "strict":
		return "cobertura fuerte con unit, integración y smoke"
	case "tdd":
		return "TDD o disciplina de tests-first"
	default:
		return "base con unit y smoke mínimos"
	}
}

func describeIdentityProvider(v string) string {
	switch v {
	case "active_directory":
		return "Active Directory"
	case "ldap":
		return "LDAP corporativo"
	case "entra_id":
		return "Microsoft Entra ID"
	case "keycloak":
		return "Keycloak"
	case "auth0":
		return "Auth0"
	case "google":
		return "Google Identity"
	default:
		return ""
	}
}

func describeObservabilityLevel(v string) string {
	switch v {
	case "basic":
		return "básico"
	case "strict":
		return "estricto"
	default:
		return "estándar"
	}
}

func describeDeploymentTarget(v string) string {
	switch v {
	case "kubernetes":
		return "Kubernetes"
	case "serverless":
		return "plataforma serverless"
	case "vm":
		return "máquina virtual o bare metal"
	case "edge":
		return "entorno edge"
	default:
		return "contenedores Docker"
	}
}

func describeArtifactType(v string) string {
	switch v {
	case "docker_image":
		return "imagen Docker"
	case "library":
		return "librería o SDK"
	case "desktop_installer":
		return "instalador de escritorio"
	case "mobile_bundle":
		return "bundle o paquete móvil"
	case "firmware":
		return "firmware"
	case "static_site":
		return "sitio estático"
	default:
		return "ejecutable o servicio desplegable"
	}
}

func composeSoList(spec AppSpec) []string {
	var so []string
	if spec.SOLinux {
		so = append(so, "Linux")
	}
	if spec.SOWindows {
		so = append(so, "Windows")
	}
	if spec.SOmacOS {
		so = append(so, "macOS")
	}
	if spec.SOAndroid {
		so = append(so, "Android")
	}
	if spec.SOiOS {
		so = append(so, "iOS")
	}
	return so
}
