package fabricaapp

import (
	"fmt"
	"slices"
	"strings"

	"orquesta/db"
)

type AppSpec struct {
	Nombre      string
	Descripcion string
	Tipo        string
	Frontend    bool
	API         bool
	Auth        bool
	Database    bool
	Docker      bool
	I18n        bool
	Idiomas     []string
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
	Prioridad        db.PrioridadTarea
	Dependencias     []string
	ContratoDefinido bool
}

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
	builder.add(withSOP(BlueprintTask{
		Key:              "briefing",
		Titulo:           fmt.Sprintf("Definir briefing funcional de %s", normalized.Nombre),
		Descripcion:      composeBriefingDescription(normalized),
		Modulo:           "producto",
		Prioridad:        db.PrioridadAlta,
		ContratoDefinido: true,
	}, "product_manager", "descubrimiento", "Brief funcional validado", "Objetivo de producto claro", "Alcance y restricciones iniciales documentados", "Criterios de exito entendibles por el resto del equipo"))
	builder.add(withSOP(BlueprintTask{
		Key:              "investigacion",
		Titulo:           fmt.Sprintf("Revisar catalogo y referencias de %s", normalized.Nombre),
		Descripcion:      composeResearchDescription(normalized),
		Modulo:           "analisis",
		Prioridad:        db.PrioridadAlta,
		ContratoDefinido: true,
	}, "analista", "descubrimiento", "Informe de referencias y restricciones", "Catalogo compartido revisado", "Referencias externas resumidas", "Riesgos tempranos identificados"))
	builder.add(withSOP(BlueprintTask{
		Key:              "arquitectura",
		Titulo:           fmt.Sprintf("Cerrar arquitectura base de %s", normalized.Nombre),
		Descripcion:      composeArchitectureDescription(normalized),
		Modulo:           "arquitectura",
		Prioridad:        db.PrioridadAlta,
		Dependencias:     []string{"briefing", "investigacion"},
		ContratoDefinido: true,
	}, "arquitecto", "arquitectura", "Diseño base aprobado", "Capas y contratos definidos", "Estrategia de reparto entre agentes asentada", "Riesgos arquitectonicos con mitigacion"))

	switch normalized.Tipo {
	case "web":
		builder.add(frontendTask(normalized, []string{"arquitectura"}))
	case "api":
		builder.add(apiTask(normalized, []string{"arquitectura"}))
	case "web_api":
		builder.add(frontendTask(normalized, []string{"arquitectura"}))
		builder.add(apiTask(normalized, []string{"arquitectura"}))
	case "cli":
		builder.add(cliTask(normalized, []string{"arquitectura"}))
	}

	if normalized.Database {
		builder.add(withSOP(BlueprintTask{
			Key:              "persistencia",
			Titulo:           fmt.Sprintf("Implementar persistencia principal de %s", normalized.Nombre),
			Descripcion:      "Definir esquema, acceso a datos, migraciones y contratos de almacenamiento necesarios para la app.",
			Modulo:           "persistencia",
			Prioridad:        db.PrioridadAlta,
			Dependencias:     []string{"arquitectura"},
			ContratoDefinido: true,
		}, "backend", "implementacion", "Persistencia funcional y versionada", "Esquema y migraciones reproducibles", "Acceso a datos con contratos claros", "Cobertura minima de persistencia"))
	}
	if normalized.Auth {
		deps := []string{"arquitectura"}
		if normalized.API {
			deps = append(deps, "api_base")
		}
		if normalized.Frontend {
			deps = append(deps, "frontend_base")
		}
		if normalized.Tipo == "cli" {
			deps = append(deps, "cli_base")
		}
		builder.add(withSOP(BlueprintTask{
			Key:              "auth",
			Titulo:           fmt.Sprintf("Cerrar autenticacion y autorizacion de %s", normalized.Nombre),
			Descripcion:      "Implementar login, roles, validacion de acceso y puntos de integracion de seguridad requeridos por la app.",
			Modulo:           "seguridad",
			Prioridad:        db.PrioridadAlta,
			Dependencias:     uniqueKeys(deps...),
			ContratoDefinido: true,
		}, "seguridad", "implementacion", "Flujo de autenticacion y autorizacion operativo", "Casos de login y permiso definidos", "Protecciones aplicadas en puntos sensibles", "Cobertura minima de errores y permisos"))
	}
	if normalized.I18n {
		builder.add(withSOP(BlueprintTask{
			Key:              "i18n",
			Titulo:           fmt.Sprintf("Aplicar i18n base en %s", normalized.Nombre),
			Descripcion:      composeI18nDescription(normalized),
			Modulo:           "i18n",
			Prioridad:        db.PrioridadAlta,
			Dependencias:     []string{"arquitectura"},
			ContratoDefinido: true,
		}, "i18n", "implementacion", "Superficie inicial multilenguaje", "Bundles iniciales cargados", "Idiomas base operativos", "Sin literales nuevos en el flujo principal"))
	}

	builder.add(integrationTask(normalized, builder.featureDependencyKeys()))

	if normalized.Docker {
		builder.add(withSOP(BlueprintTask{
			Key:          "docker",
			Titulo:       fmt.Sprintf("Preparar despliegue Docker de %s", normalized.Nombre),
			Descripcion:  "Crear imagen, compose/manifiestos, variables de entorno y healthchecks para la entrega operativa.",
			Modulo:       "deploy",
			Prioridad:    db.PrioridadMedia,
			Dependencias: []string{"integracion"},
		}, "devops", "despliegue", "Artefactos de despliegue reproducibles", "Imagen o manifiestos ejecutables", "Variables y healthchecks definidos", "Arranque verificado"))
	}

	qaDeps := []string{"integracion"}
	if normalized.Docker {
		qaDeps = append(qaDeps, "docker")
	}
	builder.add(withSOP(BlueprintTask{
		Key:          "qa_smoke",
		Titulo:       fmt.Sprintf("Validar smoke tests y QA de %s", normalized.Nombre),
		Descripcion:  "Ejecutar pruebas criticas, revisar regresiones y dejar evidencia de que la app esta lista para entrega tecnica.",
		Modulo:       "qa",
		Prioridad:    db.PrioridadAlta,
		Dependencias: qaDeps,
	}, "qa", "validacion", "Evidencia minima de calidad", "Smoke tests ejecutados", "Regresiones criticas revisadas", "Riesgos residuales anotados"))
	builder.add(withSOP(BlueprintTask{
		Key:          "documentacion",
		Titulo:       fmt.Sprintf("Cerrar documentacion operativa de %s", normalized.Nombre),
		Descripcion:  "Documentar arquitectura, uso, despliegue, i18n y operacion para que el proyecto pueda mantenerse y entregarse. Incluir atribucion visible a Orquesta de Alberto Avidad Fernandez en README, manuales y piezas documentales generadas.",
		Modulo:       "docs",
		Prioridad:    db.PrioridadMedia,
		Dependencias: []string{"qa_smoke"},
	}, "documentador", "documentacion", "Documentacion operativa y tecnica", "README y manuales actualizados", "Operacion y despliegue explicados", "Atribucion visible a Orquesta"))
	finalDeps := []string{"qa_smoke", "documentacion"}
	if normalized.Docker {
		finalDeps = append(finalDeps, "docker")
	}
	builder.add(withSOP(BlueprintTask{
		Key:          "entrega_final",
		Titulo:       fmt.Sprintf("Cerrar entrega final de %s", normalized.Nombre),
		Descripcion:  "Verificar que no quedan flecos abiertos y que la app completa puede darse por terminada y entregable. Confirmar metadatos y atribucion a Orquesta de Alberto Avidad Fernandez en artefactos y documentacion final.",
		Modulo:       "release",
		Prioridad:    db.PrioridadAlta,
		Dependencias: uniqueKeys(finalDeps...),
	}, "release_manager", "cierre", "Release candidata a entrega", "No quedan flecos criticos abiertos", "Artefactos finales validados", "Criterio de completitud explicitado"))

	return GenerationResult{Tasks: builder.tasks}, nil
}

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
	featureKeys := make([]string, 0, len(b.tasks))
	for _, task := range b.tasks {
		switch task.Key {
		case "briefing", "arquitectura", "integracion", "docker", "qa_smoke", "documentacion", "entrega_final":
			continue
		default:
			featureKeys = append(featureKeys, task.Key)
		}
	}
	return uniqueKeys(featureKeys...)
}

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
	case "api":
		spec.API = true
	case "web_api":
		spec.Frontend = true
		spec.API = true
	case "cli":
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

func withSOP(task BlueprintTask, rol, fase, entregable string, criterios ...string) BlueprintTask {
	task.RolSugerido = strings.TrimSpace(rol)
	task.Fase = strings.TrimSpace(fase)
	task.Entregable = strings.TrimSpace(entregable)
	task.CriteriosCierre = uniqueKeys(criterios...)
	return task
}

func composeBriefingDescription(spec AppSpec) string {
	desc := []string{
		fmt.Sprintf("Alinear alcance, personas usuarias, flujos principales y criterios de entrega de la app %s.", spec.Nombre),
		fmt.Sprintf("Tipo objetivo: %s.", spec.Tipo),
	}
	if spec.Descripcion != "" {
		desc = append(desc, "Contexto: "+spec.Descripcion)
	}
	return strings.Join(desc, " ")
}

func composeResearchDescription(spec AppSpec) string {
	partes := []string{
		"Revisar estudios de apps comparables, catalogo compartido de librerias y restricciones tecnicas antes de programar.",
		"Extraer decisiones reutilizables, dependencias candidatas y riesgos tempranos para " + spec.Nombre + ".",
	}
	if spec.I18n {
		partes = append(partes, "Confirmar implicaciones de i18n desde el inicio.")
	}
	return strings.Join(partes, " ")
}

func composeArchitectureDescription(spec AppSpec) string {
	parts := []string{
		"Definir modulos, contratos de entrada/salida, integraciones, criterios de reparto entre agentes y estrategia de despliegue.",
	}
	if spec.Database {
		parts = append(parts, "Debe incluir persistencia.")
	}
	if spec.Auth {
		parts = append(parts, "Debe incluir seguridad y autenticacion.")
	}
	if spec.I18n {
		parts = append(parts, "Debe nacer con i18n.")
	}
	return strings.Join(parts, " ")
}

func composeI18nDescription(spec AppSpec) string {
	if len(spec.Idiomas) == 0 {
		return "Aplicar i18n desde el inicio, evitando literales fijas y dejando bundles preparados para varios idiomas."
	}
	return fmt.Sprintf("Aplicar i18n desde el inicio y dejar bundles operativos para: %s.", strings.Join(spec.Idiomas, ", "))
}

func frontendTask(spec AppSpec, deps []string) BlueprintTask {
	return withSOP(BlueprintTask{
		Key:              "frontend_base",
		Titulo:           fmt.Sprintf("Implementar frontend base de %s", spec.Nombre),
		Descripcion:      "Crear shell visual, navegacion principal, estados base y estructura inicial de interfaz para la app.",
		Modulo:           "frontend",
		Prioridad:        db.PrioridadAlta,
		Dependencias:     uniqueKeys(deps...),
		ContratoDefinido: true,
	}, "frontend", "implementacion", "UI principal funcional", "Shell visual y navegacion base cerrados", "Flujo principal navegable", "Integracion base preparada")
}

func apiTask(spec AppSpec, deps []string) BlueprintTask {
	return withSOP(BlueprintTask{
		Key:              "api_base",
		Titulo:           fmt.Sprintf("Implementar API base de %s", spec.Nombre),
		Descripcion:      "Definir endpoints, contratos de entrada/salida, validacion y capa de aplicacion necesarias para la app.",
		Modulo:           "backend",
		Prioridad:        db.PrioridadAlta,
		Dependencias:     uniqueKeys(deps...),
		ContratoDefinido: true,
	}, "backend", "implementacion", "API principal operativa", "Endpoints base cerrados", "Contratos de entrada/salida definidos", "Validacion y errores principales cubiertos")
}

func cliTask(spec AppSpec, deps []string) BlueprintTask {
	return withSOP(BlueprintTask{
		Key:              "cli_base",
		Titulo:           fmt.Sprintf("Implementar CLI base de %s", spec.Nombre),
		Descripcion:      "Crear comandos principales, contratos de salida y flujo operativo inicial de la aplicacion CLI.",
		Modulo:           "cli",
		Prioridad:        db.PrioridadAlta,
		Dependencias:     uniqueKeys(deps...),
		ContratoDefinido: true,
	}, "backend", "implementacion", "CLI principal operativa", "Comandos base disponibles", "Ayuda y salida principal coherentes", "Flujo principal probado")
}

func integrationTask(spec AppSpec, deps []string) BlueprintTask {
	description := "Integrar modulos principales y cerrar el flujo funcional de extremo a extremo."
	switch spec.Tipo {
	case "web":
		description = "Integrar frontend, servicios y flujos principales de la aplicacion web."
	case "api":
		description = "Integrar endpoints, persistencia y casos de uso de la API."
	case "web_api":
		description = "Integrar frontend, API, persistencia y flujos de extremo a extremo de la app."
	case "cli":
		description = "Integrar comandos, persistencia y flujos operativos completos de la CLI."
	}
	return withSOP(BlueprintTask{
		Key:          "integracion",
		Titulo:       fmt.Sprintf("Integrar flujo principal de %s", spec.Nombre),
		Descripcion:  description,
		Modulo:       "integracion",
		Prioridad:    db.PrioridadAlta,
		Dependencias: uniqueKeys(deps...),
	}, "integrador", "integracion", "Sistema integrado demostrable", "Las piezas principales conversan entre si", "Flujo end-to-end verificable", "Incidencias de integracion acotadas")
}
