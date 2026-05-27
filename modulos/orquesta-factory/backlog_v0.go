package orquestafactory

import (
	"fmt"
	"strings"
)

const BacklogInicialPropuestoSchemaV0 = "backlog_inicial_propuesto.v0"

type BacklogInicialPropuestoV0 struct {
	SchemaVersion       string                  `json:"schema_version"`
	SpecID              string                  `json:"spec_id"`
	Estado              string                  `json:"estado"`
	Freshness           BacklogFreshnessV0      `json:"freshness"`
	DirectorHandoff     BacklogDirectorHandoffV0 `json:"director_handoff"`
	Fases               []FaseInicialV0         `json:"fases"`
	Microtareas         []MicrotareaPropuestaV0 `json:"microtareas"`
	ContratosRequeridos []string                `json:"contratos_requeridos"`
	Riesgos             []string                `json:"riesgos"`
	PreguntasAbiertas   []string                `json:"preguntas_abiertas"`
}

type FaseInicialV0 struct {
	ID       string `json:"id"`
	Nombre   string `json:"nombre"`
	Objetivo string `json:"objetivo"`
	Orden    int    `json:"orden"`
}

type MicrotareaPropuestaV0 struct {
	ID               string   `json:"id"`
	Fase             string   `json:"fase"`
	ModuloSugerido   string   `json:"modulo_sugerido"`
	Objetivo         string   `json:"objetivo"`
	WriteSetPrevisto []string `json:"write_set_previsto"`
	Contrato         string   `json:"contrato"`
	Validacion       string   `json:"validacion"`
	Bloqueos         []string `json:"bloqueos"`
}

func GenerarBacklogInicialPropuestoV0(spec AppSpecV0) (BacklogInicialPropuestoV0, []ValidationIssue) {
	if issues := validateBacklogSpecV0(spec); len(issues) > 0 {
		return BacklogInicialPropuestoV0{}, issues
	}
	builder := backlogBuilderV0{spec: spec}
	return BacklogInicialPropuestoV0{
		SchemaVersion:       BacklogInicialPropuestoSchemaV0,
		SpecID:              strings.TrimSpace(spec.SpecID),
		Estado:              BacklogInicialEstadoPreviewNoEjecutableV0,
		Freshness:           buildBacklogFreshnessV0(spec),
		DirectorHandoff:     buildBacklogDirectorHandoffV0(spec),
		Fases:               builder.fases(),
		Microtareas:         builder.microtareas(),
		ContratosRequeridos: builder.contratosRequeridos(),
		Riesgos:             builder.riesgos(),
		PreguntasAbiertas:   emptyStringsV0(compactUniqueV0(spec.Scope.PreguntasAbiertas)),
	}, nil
}

type backlogBuilderV0 struct {
	spec AppSpecV0
}

func (b backlogBuilderV0) fases() []FaseInicialV0 {
	return []FaseInicialV0{
		{ID: "discovery", Nombre: "Descubrimiento", Objetivo: "Cerrar alcance, supuestos y preguntas abiertas antes de programar.", Orden: 10},
		{ID: "arquitectura", Nombre: "Arquitectura", Objetivo: "Definir puertos, contratos y fronteras hexagonales.", Orden: 20},
		{ID: "i18n_docs", Nombre: "I18n y documentacion", Objetivo: "Preparar catalogos i18n y documentacion inicial con idioma declarado.", Orden: 30},
		{ID: "implementacion", Nombre: "Implementacion", Objetivo: "Construir slices verticales pequenos con FunctionContract.", Orden: 40},
		{ID: "validacion", Nombre: "Validacion", Objetivo: "Cerrar pruebas, revision, seguridad y evidencias.", Orden: 50},
	}
}

func (b backlogBuilderV0) microtareas() []MicrotareaPropuestaV0 {
	tasks := []MicrotareaPropuestaV0{
		b.discoveryTask(),
		b.architectureTask(),
		b.i18nDocsTask(),
	}
	if b.spec.Data.PersistenceRequired {
		tasks = append(tasks, b.persistenceTask())
	}
	for _, connector := range b.spec.Connectors.Required {
		if strings.TrimSpace(connector.Nombre) == "" || strings.TrimSpace(connector.Nombre) == "persistence" {
			continue
		}
		tasks = append(tasks, b.requiredConnectorTask(connector))
	}
	if b.spec.Deploy.Target != "" && b.spec.Deploy.Target != "sin_preferencia" {
		tasks = append(tasks, b.deployTask())
	}
	tasks = append(tasks, b.firstSliceTask(), b.validationTask())
	return assignMicrotaskIDsV0(tasks)
}

func (b backlogBuilderV0) discoveryTask() MicrotareaPropuestaV0 {
	return MicrotareaPropuestaV0{
		ID:               "BLG-001",
		Fase:             "discovery",
		ModuloSugerido:   "producto",
		Objetivo:         "Cerrar alcance inicial de " + b.spec.App.Nombre + " y registrar supuestos/preguntas.",
		WriteSetPrevisto: []string{"docs/app_spec.md", "docs/decisiones.md"},
		Contrato:         "AppSpecV0",
		Validacion:       "AppSpec revisada, preguntas abiertas registradas y sin decisiones inventadas.",
		Bloqueos:         emptyStringsV0(compactUniqueV0(b.spec.Scope.PreguntasAbiertas)),
	}
}

func (b backlogBuilderV0) architectureTask() MicrotareaPropuestaV0 {
	return MicrotareaPropuestaV0{
		ID:               "BLG-002",
		Fase:             "arquitectura",
		ModuloSugerido:   "core",
		Objetivo:         "Definir puertos de entrada/salida, entidades y conectores iniciales con arquitectura hexagonal.",
		WriteSetPrevisto: []string{"docs/arquitectura.md", "docs/contratos.md"},
		Contrato:         "FunctionContract v0",
		Validacion:       "Contratos documentados sin DB, runtime, filesystem ni proveedor como dependencia del core.",
		Bloqueos:         []string{"FunctionContract v0"},
	}
}

func (b backlogBuilderV0) i18nDocsTask() MicrotareaPropuestaV0 {
	return MicrotareaPropuestaV0{
		ID:               "BLG-003",
		Fase:             "i18n_docs",
		ModuloSugerido:   "i18n-docs",
		Objetivo:         "Generar catalogos i18n y documentacion inicial para locales declarados.",
		WriteSetPrevisto: []string{"i18n/", "docs/manual_usuario.md", "docs/manual_desarrollador.md", "docs/manual_sistemas.md"},
		Contrato:         "GenerarI18nDocsIniciales v0",
		Validacion:       "Todo texto visible sale de catalogos o plantillas con locale declarado.",
		Bloqueos:         []string{"GenerarI18nDocsIniciales v0"},
	}
}

func (b backlogBuilderV0) persistenceTask() MicrotareaPropuestaV0 {
	return MicrotareaPropuestaV0{
		Fase:             "arquitectura",
		ModuloSugerido:   "persistence",
		Objetivo:         "Definir persistencia como conector para la necesidad funcional declarada.",
		WriteSetPrevisto: []string{"docs/persistencia.md", "adapters/persistence/", "tests/persistence_contract_test.go"},
		Contrato:         "PersistenceRepository v0",
		Validacion:       "Persistencia expresada por puerto y contrato, sin tablas ni proveedor en el core.",
		Bloqueos:         []string{"PersistenceRepository v0"},
	}
}

func (b backlogBuilderV0) requiredConnectorTask(connector ConnectorSpecV0) MicrotareaPropuestaV0 {
	name := strings.TrimSpace(connector.Nombre)
	contract := firstNonEmptyV0(connector.Contrato, "ConnectorContract v0")
	return MicrotareaPropuestaV0{
		Fase:             "arquitectura",
		ModuloSugerido:   "connectors",
		Objetivo:         "Definir conector requerido " + name + " como puerto versionado para " + strings.TrimSpace(connector.Proposito) + ".",
		WriteSetPrevisto: []string{"docs/connectors/" + slugV0(name) + ".md", "adapters/" + slugV0(name) + "/", "tests/" + slugV0(name) + "_contract_test.go"},
		Contrato:         contract,
		Validacion:       "Contrato del conector documentado, fixture de contrato en verde y sin dependencias directas desde el core.",
		Bloqueos:         []string{contract},
	}
}

func (b backlogBuilderV0) deployTask() MicrotareaPropuestaV0 {
	return MicrotareaPropuestaV0{
		Fase:             "arquitectura",
		ModuloSugerido:   "deploy",
		Objetivo:         "Preparar DeploymentPlan v0 para target de deploy " + b.spec.Deploy.Target + " y validarlo por dry-run.",
		WriteSetPrevisto: []string{"contracts/deployment_plan_v0.json", "docs/deploy.md", "tests/deployment_plan_dry_run_test.go"},
		Contrato:         "DeploymentPlan v0",
		Validacion:       "Dry-run de DeploymentPlan v0 devuelve refs de plan/evidencia sin ejecutar Docker, Kubernetes, cloud ni secretos.",
		Bloqueos:         []string{"DeploymentPlan v0 dry-run"},
	}
}

func (b backlogBuilderV0) firstSliceTask() MicrotareaPropuestaV0 {
	blockers := []string{"BLG-001", "BLG-002", "BLG-003"}
	if b.spec.Data.PersistenceRequired {
		blockers = append(blockers, "PersistenceRepository v0")
	}
	for _, connector := range b.spec.Connectors.Required {
		if strings.TrimSpace(connector.Nombre) == "" || strings.TrimSpace(connector.Nombre) == "persistence" {
			continue
		}
		blockers = append(blockers, firstNonEmptyV0(connector.Contrato, "ConnectorContract v0"))
	}
	if b.spec.Deploy.Target != "" && b.spec.Deploy.Target != "sin_preferencia" {
		blockers = append(blockers, "DeploymentPlan v0")
	}
	return MicrotareaPropuestaV0{
		Fase:             "implementacion",
		ModuloSugerido:   "core",
		Objetivo:         "Implementar el primer caso de uso vertical minimo de " + b.spec.App.Nombre + ".",
		WriteSetPrevisto: []string{"core/", "adapters/inbound/", "tests/"},
		Contrato:         "FunctionContract v0",
		Validacion:       "Tests del slice en verde, write-set respetado y entrega sin cambios fuera de contrato.",
		Bloqueos:         blockers,
	}
}

func (b backlogBuilderV0) validationTask() MicrotareaPropuestaV0 {
	return MicrotareaPropuestaV0{
		Fase:             "validacion",
		ModuloSugerido:   "quality",
		Objetivo:         "Ejecutar revision final de pruebas, seguridad, i18n, documentacion y conectores.",
		WriteSetPrevisto: []string{"docs/pruebas.md", "docs/revision_final.md"},
		Contrato:         "FunctionContract v0",
		Validacion:       "Evidencia de tests/revision registrada y riesgos abiertos explicitados.",
		Bloqueos:         []string{"Primer slice vertical minimo completado"},
	}
}

func (b backlogBuilderV0) contratosRequeridos() []string {
	contracts := []string{"AppSpecV0", "FunctionContract v0", "GenerarI18nDocsIniciales v0", "CapacityDecision v0"}
	contracts = append(contracts, b.spec.Architecture.ContratosEsperados...)
	if b.spec.Data.PersistenceRequired {
		contracts = append(contracts, "PersistenceRepository v0")
	}
	for _, connector := range b.spec.Connectors.Required {
		if contract := strings.TrimSpace(connector.Contrato); contract != "" {
			contracts = append(contracts, contract)
			continue
		}
		if strings.TrimSpace(connector.Nombre) != "" && strings.TrimSpace(connector.Nombre) != "persistence" {
			contracts = append(contracts, "ConnectorContract v0")
		}
	}
	if b.spec.Deploy.Target != "" && b.spec.Deploy.Target != "sin_preferencia" {
		contracts = append(contracts, "DeploymentPlan v0")
	}
	return compactUniqueV0(contracts)
}

func (b backlogBuilderV0) riesgos() []string {
	var risks []string
	if !b.spec.I18N.Enabled {
		risks = append(risks, "i18n desactivado requiere excepcion revisada antes de generar UI o docs.")
	}
	if b.spec.Data.PersistenceRequired {
		risks = append(risks, "PersistenceRepository v0 debe cerrarse antes de implementar almacenamiento.")
	}
	if len(b.spec.Connectors.Required) > 0 {
		risks = append(risks, "Los conectores requeridos deben validarse como puertos antes de programar adaptadores.")
	}
	if b.spec.Deploy.Target != "" && b.spec.Deploy.Target != "sin_preferencia" {
		risks = append(risks, "DeploymentPlan v0 debe preparar entorno, healthcheck y rollback.")
	}
	if len(b.spec.Scope.PreguntasAbiertas) > 0 {
		risks = append(risks, "Existen preguntas abiertas que pueden cambiar alcance o contratos.")
	}
	return emptyStringsV0(compactUniqueV0(risks))
}

func validateBacklogSpecV0(spec AppSpecV0) []ValidationIssue {
	var issues []ValidationIssue
	if spec.SchemaVersion != AppSpecSchemaV0 {
		issues = append(issues, issue(ErrAppSpecInvalida, "schema_version", "AppSpecV0 requerido"))
	}
	if strings.TrimSpace(spec.SpecID) == "" {
		issues = append(issues, issue(ErrAppSpecInvalida, "spec_id", "spec_id requerido"))
	}
	if spec.Validation.Estado != "valida" {
		issues = append(issues, issue(ErrAppSpecInvalida, "validation.estado", "AppSpecV0 valida requerida"))
	}
	if strings.TrimSpace(spec.App.Nombre) == "" {
		issues = append(issues, issue(ErrAppSpecInvalida, "app.nombre", "nombre de app requerido"))
	}
	if strings.TrimSpace(spec.App.Objetivo) == "" {
		issues = append(issues, issue(ErrAppSpecInvalida, "app.objetivo", "objetivo de app requerido"))
	}
	return issues
}

func assignMicrotaskIDsV0(tasks []MicrotareaPropuestaV0) []MicrotareaPropuestaV0 {
	for index := range tasks {
		tasks[index].ID = microtaskIDV0(index + 1)
		tasks[index].WriteSetPrevisto = emptyStringsV0(compactUniqueV0(tasks[index].WriteSetPrevisto))
		tasks[index].Bloqueos = emptyStringsV0(compactUniqueV0(tasks[index].Bloqueos))
	}
	return tasks
}

func microtaskIDV0(n int) string {
	return fmt.Sprintf("BLG-%03d", n)
}
