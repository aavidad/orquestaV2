# Tareas locales: orquesta-factory

Cada tarea debe ser pequena y cerrada.

## Backlog inicial propuesto: AppSpec v0

```text
ID: FTY-000
Objetivo: Registrar la propuesta documental minima de AppSpec v0.
Write-set: docs/contratos.md, docs/tareas.md, docs/decisiones.md
Simbolo foco: AppSpecV0
Contrato: SolicitarNuevaApp v0, AppSpecRequestV0, AppSpecV0, BacklogInicialPropuestoV0
Validacion: revisar diff documental y confirmar ausencia de runtime/DB.
Bloqueos: ninguno.
Estado: completada documental; la propuesta minima quedo absorbida por FTY-001..FTY-007 y consolidada en contratos, decisiones, pruebas y adaptador REST v0.
```

```text
ID: FTY-001
Objetivo: Extraer un schema validable para AppSpecRequestV0 y AppSpecV0.
Write-set: docs/schemas/app_spec_request_v0.schema.json, docs/schemas/app_spec_v0.schema.json
Simbolo foco: AppSpecV0
Contrato: AppSpecRequestV0, AppSpecV0
Validacion: validar fixtures minimos validos e invalidos contra el schema.
Bloqueos: ninguno.
Estado: completada; schemas Draft 2020-12 creados y validados con jq/jsonschema el 2026-05-04
```

```text
ID: FTY-002
Objetivo: Definir fixtures de contrato para requests validas e invalidas.
Write-set: docs/fixtures/app_spec_v0/request_minima_valida.json, docs/fixtures/app_spec_v0/request_i18n_invalida.json, docs/fixtures/app_spec_v0/request_db_directa_invalida.json
Simbolo foco: AppSpecRequestV0
Contrato: SolicitarNuevaApp v0
Validacion: cada fixture declara resultado esperado y error publico cuando aplique.
Bloqueos: FTY-001.
Estado: completada; fixtures creados con expected.valid y expected.public_error, validados el 2026-05-04
```

```text
ID: FTY-003
Objetivo: Implementar validador puro de AppSpecRequestV0 sin adaptadores.
Write-set: appspec_request_v0.go, appspec_request_v0_test.go, docs/tareas.md, docs/pruebas.md
Simbolo foco: ValidateAppSpecRequestV0
Contrato: AppSpecRequestV0
Validacion: unit tests de campos obligatorios, locale BCP 47, i18n por defecto, hexagonal por defecto y rechazo de DB directa/runtime.
Bloqueos: FTY-001, FTY-002.
Estado: completada; validador puro y tests de fixtures implementados.
```

```text
ID: FTY-004
Objetivo: Implementar caso de uso SolicitarNuevaApp que normaliza request y devuelve AppSpecV0.
Write-set: appspec_request_v0.go, appspec_usecase_v0.go, appspec_usecase_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: SolicitarNuevaApp
Contrato: SolicitarNuevaApp v0, AppSpecV0
Validacion: contract tests con request minima valida y errores publicos.
Bloqueos: FTY-003.
Estado: completada; caso de uso puro implementado sin DB, runtime, filesystem, web ni MCP.
```

```text
ID: FTY-005
Objetivo: Generar BacklogInicialPropuestoV0 desde una AppSpecV0 valida.
Write-set: backlog_v0.go, backlog_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: GenerarBacklogInicialPropuestoV0
Contrato: BacklogInicialPropuestoV0
Validacion: cada microtarea generada incluye objetivo, write-set previsto, contrato, validacion y bloqueo si afecta otro modulo.
Bloqueos: FTY-004.
Estado: completada; generador puro implementado sin DB, runtime, filesystem, web, MCP ni LLM.
```

```text
ID: FTY-006
Objetivo: Preparar contrato de adaptador web sin acoplar orquesta-web al core interno.
Write-set: docs/tareas.md, docs/decisiones.md, ../CONTRATOS.md
Simbolo foco: SolicitarNuevaApp
Contrato: SolicitarNuevaApp v0
Validacion: orquesta-web consume solo DTOs publicos y errores publicos.
Bloqueos: ninguno; decision del director recibida.
Estado: completada; resumen minimo promovido a ../CONTRATOS.md
```

```text
ID: FTY-007
Objetivo: Implementar adaptador inbound REST v0 para SolicitarNuevaApp v0.
Write-set: appspec_http_v0.go, appspec_http_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: NewAppSpecHTTPHandlerV0
Contrato: SolicitarNuevaApp v0, AppSpecRequestV0, AppSpecV0, BacklogInicialPropuestoV0
Validacion: httptest de POST /api/v0/apps/spec valido, errores 400, metodo no permitido, backlog canonico; go test -count=1 ./modulos/orquesta-factory y git diff --check.
Bloqueos: ninguno.
Estado: completada; adaptador REST v0 implementado sin servidor real, router externo, DB, runtime, filesystem, MCP ni UI.
```

```text
ID: FTY-008
Objetivo: Sanear appspec_usecase_v0.go dividiendo el caso de uso por responsabilidades sin cambiar contratos ni comportamiento.
Write-set: appspec_usecase_v0.go, appspec_dto_v0.go, appspec_assemble_v0.go, appspec_normalize_v0.go, appspec_defaults_v0.go, appspec_identity_v0.go, appspec_collections_v0.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: SolicitarNuevaAppV0
Contrato: SolicitarNuevaApp v0, AppSpecRequestV0, AppSpecV0, BacklogInicialPropuestoV0
Validacion: gofmt; go test -count=1 ./modulos/orquesta-factory; git diff --check -- modulos/orquesta-factory; wc -l de ficheros Go tocados.
Bloqueos: ninguno.
Estado: completada; usecase reducido a shell de 15 lineas y responsabilidades separadas en ficheros de 140 lineas o menos.
```

## Plantilla

```text
ID:
Objetivo:
Write-set:
Simbolo foco:
Contrato:
Validacion:
Bloqueos:
Estado:
```
