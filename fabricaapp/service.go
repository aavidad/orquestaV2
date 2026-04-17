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
	Nombre      string
	Descripcion string

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
		Descripcion:  "Documentar arquitectura hexagonal, módulos, uso, despliegue, i18n y operación. Incluir atribución visible a Orquesta de Alberto Avidad Fernández en README, manuales y piezas documentales generadas.",
		Modulo:       "docs",
		Prioridad:    PrioridadMedia,
		Dependencias: []string{"qa_smoke"},
	}, "documentador", "documentacion", "Documentación operativa y técnica",
		"README y manuales actualizados",
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
	spec.Tipo = strings.ToLower(strings.TrimSpace(spec.Tipo))
	spec.Idiomas = normalizeIdiomas(spec.Idiomas)

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

	return spec, nil
}

func normalizeIdiomas(idiomas []string) []string {
	out := make([]string, 0, len(idiomas))
	for _, idioma := range idiomas {
		idioma = strings.ToLower(strings.TrimSpace(idioma))
		if idioma == "" || slices.Contains(out, idioma) {
			continue
		}
		out = append(out, idioma)
	}
	return out
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
	return spec.API
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
	return withSOP(BlueprintTask{
		Key:              "frontend_base",
		Titulo:           fmt.Sprintf("Implementar frontend base de %s", spec.Nombre),
		Descripcion:      "Crear shell visual, navegación principal, estados base y estructura inicial de interfaz. El frontend se implementa como adaptador de UI en la arquitectura hexagonal, sin lógica de negocio.",
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
	}
	if spec.Descripcion != "" {
		parts = append(parts, "Contexto: "+spec.Descripcion)
	}
	soList := composeSoList(spec)
	if len(soList) > 0 {
		parts = append(parts, fmt.Sprintf("SO objetivo: %s.", strings.Join(soList, ", ")))
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
	if hasCompliance(spec) {
		parts = append(parts, "Revisar normativas aplicables y su impacto técnico.")
	}
	if spec.PlatDesktop || spec.PlatMobile {
		parts = append(parts, "Evaluar frameworks multiplataforma y sus restricciones.")
	}
	return strings.Join(parts, " ")
}

func composeArchitectureDescription(spec AppSpec) string {
	parts := []string{
		"Diseñar arquitectura hexagonal (puertos y adaptadores): núcleo de dominio aislado de infraestructura, adaptadores de entrada (API, CLI, UI) y salida (BD, servicios externos).",
		"Definir módulos con contratos explícitos entre sí. Ningún módulo depende de la implementación concreta de otro.",
	}
	if requiresResidentService(spec) {
		parts = append(parts, "Si la app necesita servicio residente, el núcleo siempre vivo debe ser mínimo: continuidad operativa, hot state imprescindible y coordinación ligera. La supervisión rica, snapshots pesados, reconciliaciones profundas y lecturas no críticas deben salir a workers separados, caminos on-demand o procesamiento orientado a eventos.")
	}
	if spec.Database {
		parts = append(parts, "Incluir puerto de persistencia con adaptador(es) de BD.")
	}
	if spec.Auth {
		parts = append(parts, "Incluir puerto de autenticación/autorización.")
	}
	if spec.I18n {
		parts = append(parts, "Diseñar con i18n desde el inicio.")
	}
	soList := composeSoList(spec)
	if len(soList) > 1 {
		parts = append(parts, fmt.Sprintf("Estrategia multiplataforma para: %s.", strings.Join(soList, ", ")))
	}
	if spec.Monitoring {
		parts = append(parts, "Incluir puertos de observabilidad (logs, métricas, trazas).")
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

func composePersistenceDescription(spec AppSpec) string {
	parts := []string{
		"Definir esquema, migraciones, acceso a datos y contratos de almacenamiento.",
		"La BD se accede exclusivamente a través del adaptador de persistencia (puerto hexagonal). Ninguna capa de aplicación o dominio importa el driver directamente.",
	}
	if spec.ComplianceRGPD {
		parts = append(parts, "Cifrar datos personales en reposo. Implementar retención y derecho al olvido.")
	}
	return strings.Join(parts, " ")
}

func composeAuthDescription(spec AppSpec) string {
	parts := []string{
		"Implementar autenticación y autorización como adaptador de seguridad en la arquitectura hexagonal.",
		"Definir roles, permisos, validación de acceso y puntos de integración de seguridad.",
	}
	if spec.ComplianceRGPD {
		parts = append(parts, "Mínimo privilegio en acceso a datos personales. Log de accesos.")
	}
	if spec.ComplianceENS {
		parts = append(parts, "Gestión de identidades conforme a ENS.")
	}
	return strings.Join(parts, " ")
}

func composeI18nDescription(spec AppSpec) string {
	if len(spec.Idiomas) == 0 {
		return "Aplicar i18n desde el inicio, evitando literales fijas y dejando bundles preparados para varios idiomas."
	}
	return fmt.Sprintf("Aplicar i18n desde el inicio y dejar bundles operativos para: %s.", strings.Join(spec.Idiomas, ", "))
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
	if spec.Docker {
		parts = append(parts, "Incluir build y push de imagen Docker.")
	}
	if spec.Kubernetes {
		parts = append(parts, "Incluir deploy automático a Kubernetes.")
	}
	if spec.ComplianceENS {
		parts = append(parts, "Incluir análisis de vulnerabilidades (SAST/DAST) en el pipeline.")
	}
	return strings.Join(parts, " ")
}

func composeMonitoringDescription(spec AppSpec) string {
	parts := []string{
		"Implementar los tres pilares de observabilidad: logs estructurados (JSON), métricas (Prometheus/OpenMetrics) y trazas distribuidas (OpenTelemetry).",
		"Configurar dashboard básico de salud y alertas críticas.",
	}
	if spec.ComplianceENS {
		parts = append(parts, "Los logs de seguridad deben conservarse según política ENS.")
	}
	if spec.ComplianceRGPD {
		parts = append(parts, "No registrar datos personales en logs sin anonimización.")
	}
	return strings.Join(parts, " ")
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
