# Tareas locales: orquesta-core

Cada tarea debe ser pequena y cerrada.

```text
ID: CORE-012
Objetivo: Reconciliar T198 para descriptors MCP que referencian FunctionContract y contratos core.
Write-set:
  - docs/tareas.md
  - docs/decisiones.md
  - docs/pruebas.md
  - README.md
Simbolo foco: FunctionContractV0; contratos core publicados.
Contrato: mcp.resource.descriptor_source.v0 como consumidor externo de core.
Validacion: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-observability ./modulos/orquesta-governance ./modulos/orquesta-core ./cmd/orquesta-server.
Bloqueos: Core no importa MCP, HTTP, CLI ni transporte; la reconciliacion queda en docs y backlog.
Estado: completada_documental 2026-05-27
Revalidacion: `agent-ref-task-autoprogramming-c3678e9bc306-g01` confirma cierre
stale documental sin codigo nuevo.
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

## Arranque mini-proyecto core: registro provisional de proyecto/backlog

```text
ID: CORE-000
Objetivo: Documentar el contrato local provisional para recibir AppSpecV0 validada y BacklogInicialPropuestoV0 desde factory.
Write-set:
  - docs/contratos.md
  - docs/tareas.md
  - docs/decisiones.md
  - docs/pruebas.md
Simbolo foco: RegistrarProyectoDesdeAppSpec v0
Contrato: RegistrarProyectoDesdeAppSpec v0
Validacion: Revision documental contra AGENTS.md, README.md, modulos/CONTRATOS.md y contratos/contratos_iniciales.md.
Bloqueos: Ninguno para propuesta local; promocion a contrato global requiere director.
Estado: hecho
```

```text
ID: CORE-001
Objetivo: Crear fixtures contractuales minimos para command ok, app_spec no validada, backlog vacio y version no soportada.
Write-set:
  - docs/fixtures/registrar_proyecto_appspec_v0/*.json
  - docs/tareas.md
  - docs/pruebas.md
  - docs/decisiones.md
Simbolo foco: RegistrarProyectoDesdeAppSpecCommandV0
Contrato: RegistrarProyectoDesdeAppSpec v0
Validacion: Fixtures serializables y legibles contra DTOs publicos de factory: command_ok_minimo, error_app_spec_no_validada, error_backlog_vacio, error_version_incompatible y error_idempotency_key_vacia.
Bloqueos: Ninguno; la forma contractual se contrasto con AppSpecV0, BacklogInicialPropuestoV0 y RegistrarProyectoDesdeAppSpecCommandV0 publicos.
Estado: completada
```

```text
ID: CORE-002
Objetivo: Definir el mapeo puro de BacklogInicialPropuestoV0 a ItemBacklogCoreV0 y fases iniciales de ProyectoPlanBorradorV0.
Write-set:
  - docs/contratos.md
  - docs/decisiones.md
  - docs/pruebas.md
Simbolo foco: ProyectoPlanBorradorV0
Contrato: RegistrarProyectoDesdeAppSpec v0
Validacion: Implementacion existente en mapFasesV0, mapMicrotareasV0 y construirProyectoPlanBorradorV0; tests Go cubren mapeo de fases, microtareas, contratos, backlog_normalizado y evento de borrador sin DB/runtime.
Bloqueos: Ninguno para ProyectoPlanBorradorV0; ProyectoPlan global definitivo sigue fuera de este corte.
Estado: completada
```

```text
ID: CORE-003
Objetivo: Preparar pruebas de contrato ejecutables cuando exista implementacion del caso de uso puro.
Write-set:
  - docs/pruebas.md
Simbolo foco: pruebas RegistrarProyectoDesdeAppSpec v0
Contrato: RegistrarProyectoDesdeAppSpec v0
Validacion: registrar_proyecto_appspec_v0_test.go ejecuta caso OK, app_spec_no_validada, backlog_vacio, backlog_incompatible_con_app_spec, idempotency_key_requerida, contrato_factory_no_soportado e idempotencia determinista.
Bloqueos: Ninguno; existe arbol Go y runner con go test -count=1 ./modulos/orquesta-core.
Estado: completada
```

```text
ID: CORE-004
Objetivo: Elevar al director la promocion del puerto local a contrato compartido si factory necesita consumirlo de forma estable.
Write-set:
  - docs/tareas.md
  - docs/decisiones.md
  - ../../CONTRATOS.md
Simbolo foco: CONSULTA AL DIRECTOR
Contrato: RegistrarProyectoDesdeAppSpec v0
Validacion: Consulta contiene modulo origen, modulo afectado, bloqueo, pregunta, opcion recomendada e impacto.
Bloqueos: ninguno; decision del director recibida.
Estado: completada; resumen minimo promovido a ../../CONTRATOS.md
```

## CONSULTA AL DIRECTOR

```text
Modulo origen: orquesta-core
Modulo afectado: orquesta-factory
Bloqueo: El mapa global ya menciona "Registrar proyecto y backlog" con propietario orquesta-core, pero el nombre y campos del puerto compartido no estan formalizados en modulos/CONTRATOS.md.
Pregunta concreta: Confirmar si el puerto provisional local debe promocionarse como contrato global `RegistrarProyectoDesdeAppSpec v0`.
Opcion recomendada: Mantener el detalle en orquesta-core/docs/contratos.md y registrar en modulos/CONTRATOS.md solo nombre, propietario, consumidor factory, versiones aceptadas y regla de evolucion.
Impacto: Permite que factory entregue AppSpecV0 validada y BacklogInicialPropuestoV0 sin acoplarse a DB/runtime ni a entidades internas de core.
Decision del director: Promocionar `RegistrarProyectoDesdeAppSpec v0` a contrato global minimo. El detalle queda en orquesta-core/docs/contratos.md; modulos/CONTRATOS.md solo registra propietario, consumidores, DTOs, errores, invariantes y regla de evolucion. Consulta cerrada.
```

## Backlog desde DB v1

```text
ID: CORE-005
Objetivo: Formalizar `FunctionContract v0` a partir de `especificaciones_funcion` y reglas de microprogramacion de DB v1.
Write-set:
  - docs/contratos.md
  - docs/tareas.md
  - docs/pruebas.md
  - ../../CONTRATOS.md
Simbolo foco: FunctionContractV0
Contrato: FunctionContract v0
Validacion: contrato incluye objetivo, archivo/simbolo foco, write-set, dependencias permitidas/prohibidas, pre/postcondiciones, tests obligatorios y formato de entrega.
Bloqueos: ninguno vivo; la DB v1 fue solo fuente forense historica y no prerequisito ejecutable de automejora.
Estado: completada; detalle canonico en docs/contratos.md, pruebas en docs/pruebas.md, decision en docs/decisiones.md y resumen minimo promovido a ../../CONTRATOS.md desde este documento.
```

## CORE-005 evidencia

```text
Fuente forense en cuarentena: ref relativo opaco `backups/legacy-sqlite-20260422/orquesta.db` cuando exista localmente.
Estado documental: `forense`/`historico`/`quarantine`; no es prerequisito vivo ni fuente operativa para agentes.
Tabla revisada: especificaciones_funcion
Campos reutilizados como forma contractual: titulo, archivo_objetivo, simbolo_objetivo, descripcion, precondiciones_json, postcondiciones_json, dependencias_permitidas_json, dependencias_prohibidas_json, tests_obligatorios_json, write_set_json, formato_salida, estado, version.
Filas importadas como canon: ninguna.
Ids importados como canon: ninguno.
Formatos observados: patch+evidencia, patch_unificado, ficheros+evidencia.
Decision del director: el registro global entre mini-proyectos es modulos/CONTRATOS.md; desde docs locales se referencia como ../../CONTRATOS.md.
```

## Implementacion pura RegistrarProyectoDesdeAppSpec v0

```text
ID: CORE-006
Objetivo: Implementar `RegistrarProyectoDesdeAppSpecV0` como caso de uso puro que consume `AppSpecV0` validada y `BacklogInicialPropuestoV0` compatible desde factory.
Write-set:
  - registrar_proyecto_appspec_v0.go
  - registrar_proyecto_appspec_v0_test.go
  - docs/contratos.md
  - docs/tareas.md
  - docs/pruebas.md
  - docs/decisiones.md
Simbolo foco: RegistrarProyectoDesdeAppSpecV0
Contrato: RegistrarProyectoDesdeAppSpec v0
Validacion: `go test -count=1 ./modulos/orquesta-core ./modulos/orquesta-factory` y `git diff --check`.
Bloqueos: Ninguno para v0 puro; persistencia, runtime, agentes, capacidad y observability quedan fuera.
Estado: completada
```

## Puertos de salida promovidos globalmente

```text
ID: CORE-007
Objetivo: Coordinar desde core los puertos de salida locales para PersistenceRepository v0, RuntimeLaunchRequest v0, OrquestaEvent v0 y GovernanceCatalog v0 sin implementar adaptadores reales.
Write-set:
  - puertos_salida_v0.go
  - puertos_salida_v0_test.go
  - docs/contratos.md
  - docs/tareas.md
  - docs/pruebas.md
  - docs/decisiones.md
Simbolo foco: PersistenceRepositoryPortV0, RuntimeLauncherPortV0, OrquestaEventPublisherPortV0, GovernanceCatalogPortV0
Contrato: Puertos de salida v0 de orquesta-core contra contratos globales promovidos.
Validacion: `go test -count=1 ./modulos/orquesta-core` y `git diff --check -- modulos/orquesta-core`.
Bloqueos: Ninguno para interfaces/DTOs/mappers puros; adaptadores reales quedan para persistence, runtime, observability y governance.
Estado: completada
```

## Consulta CLI sobre FunctionContractV0

```text
ID: CORE-008
Objetivo: Responder localmente a la consulta CLI sobre operaciones publicas candidatas para FunctionContractV0, limitandolas a solo lectura.
Write-set:
  - docs/contratos.md
  - docs/tareas.md
  - docs/pruebas.md
  - docs/decisiones.md
Simbolo foco: ListarFunctionContractsV0, VerFunctionContractV0, RegistrarFunctionContractV0
Contrato: FunctionContract v0
Validacion: Documentacion local revisada y `git diff --check -- .` desde orquesta-core.
Bloqueos: `registrar FunctionContractV0` queda bloqueado hasta cerrar OrchestrationRun, CommandHandler y Outbox; la promocion global de listar/ver requiere decision del director si CLI los implementa.
Estado: completada
```

## Saneamiento zona roja puertos de salida

```text
ID: CORE-009
Objetivo: Sanear la zona roja preservada dividiendo `puertos_salida_v0.go` por familias de puertos/adaptadores sin cambiar nombres publicos, interfaces, errores, imports publicos ni comportamiento.
Write-set:
  - puertos_salida_v0.go
  - puertos_salida_events_v0.go
  - puertos_salida_function_contract_v0.go
  - puertos_salida_governance_v0.go
  - puertos_salida_persistence_v0.go
  - puertos_salida_runtime_v0.go
  - docs/tareas.md
  - docs/pruebas.md
  - docs/decisiones.md
Simbolo foco: PersistenceRepositoryPortV0, OrquestaEventPublisherPortV0, RuntimeLauncherPortV0, GovernanceCatalogPortV0, FunctionContractV0
Contrato: Puertos de salida v0 de orquesta-core contra contratos globales promovidos.
Validacion: `gofmt`, `go test -count=1 ./modulos/orquesta-core`, `git diff --check -- modulos/orquesta-core` y `wc -l puertos_salida_v0.go puertos_salida_*_v0.go`.
Bloqueos: Ninguno; split mecanico sin adaptadores reales, DB, runtime, deploy ni proveedores.
Estado: completada
```

## Saneamiento RegistrarProyectoDesdeAppSpec v0

```text
ID: CORE-010
Objetivo: Sanear el ultimo fichero Go por encima de 300 lineas, `registrar_proyecto_appspec_v0.go`, dividiendolo mecanicamente por responsabilidades sin cambiar contrato publico, nombres exportados, errores, invariantes ni comportamiento.
Write-set:
  - registrar_proyecto_appspec_v0.go
  - registrar_proyecto_appspec_ensamblado_v0.go
  - registrar_proyecto_appspec_helpers_v0.go
  - registrar_proyecto_appspec_validacion_v0.go
  - docs/tareas.md
  - docs/pruebas.md
  - docs/decisiones.md
Simbolo foco: RegistrarProyectoDesdeAppSpecV0
Contrato: RegistrarProyectoDesdeAppSpec v0
Validacion: `gofmt`, `go test -count=1 ./modulos/orquesta-core`, `git diff --check -- modulos/orquesta-core` y `wc -l registrar_proyecto_appspec_v0.go registrar_proyecto_appspec_*_v0.go`.
Bloqueos: Ninguno; split mecanico sin cambios de contrato, errores, invariantes ni comportamiento.
Estado: completada
```

## Frontera hexagonal core -> factory

```text
ID: CORE-011
Objetivo: Romper el acoplamiento transitorio `orquesta-core -> orquesta-factory -> net/http` sin meter validacion de AppSpec ni transporte en core.
Write-set aplicado:
  - modulos/orquesta-core
  - modulos/orquesta-director
  - architecture_boundaries_test.go
Simbolo foco: RegistrarProyectoDesdeAppSpecV0, AppSpecV0, BacklogInicialPropuestoV0
Contrato: RegistrarProyectoDesdeAppSpec v0 consume DTOs neutrales propios del core; `orquesta-director` adapta desde `orquesta-factory` sin que core dependa de factory ni de su HTTP.
Validacion:
  - `go list -f '{{join .Imports "\n"}}' orquesta/modulos/orquesta-core` no debe incluir `orquesta/modulos/orquesta-factory`.
  - `go list -deps -f '{{.ImportPath}}' orquesta/modulos/orquesta-core` no debe incluir `orquesta/modulos/orquesta-factory`, `net/http`, adaptadores producto ni persistencia concreta.
  - `go test -count=1 . -run TestNeutralCoreDoesNotDependTransitivelyOnAdapters`.
Implementacion:
  - `RegistrarProyectoDesdeAppSpecCommandV0` usa `RegistrarAppSpecV0` y `RegistrarBacklogInicialV0`, DTOs minimos del core.
  - `orquesta-director` convierte `orquestafactory.AppSpecV0` y `BacklogInicialPropuestoV0` a esos DTOs al hacer bootstrap.
  - La prueba raiz cubre imports directos y deps transitivas de `orquesta-core`.
Bloqueos:
  - Queda fuera de este corte revisar si otros paquetes neutrales historicos arrastran `orquesta-factory`; este cierre protege `orquesta-core`.
Estado: completada
```
