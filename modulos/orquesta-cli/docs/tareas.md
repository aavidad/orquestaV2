# Tareas locales: orquesta-cli

Cada tarea debe ser pequena y cerrada.

## Backlog inicial

```text
ID: CLI-000
Objetivo: Arrancar el miniproyecto con backlog documental, contratos locales, decisiones e inventario V1.
Write-set: README.md, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: documentacion inicial orquesta-cli
Contrato: CONTRATOS.md global + contratos locales documentales
Validacion: git diff --check -- .
Bloqueos: ninguno
Estado: completada documental
```

```text
ID: CLI-001
Objetivo: Definir la politica de invocacion y salida estable de CLI.
Write-set: cli_contracts_v0.go, cli_contracts_v0_test.go, docs/tareas.md, docs/pruebas.md
Simbolo foco: CliInvocationContextV0, CliOutputEnvelopeV0
Contrato: contratos locales orquesta-cli
Validacion: TestNormalizeCliInvocationContextV0GeneraRequestYCorrelationID y TestCliOutputEnvelopeV0JSONEstable cubren defaults, request_id/correlation_id y golden JSON para exito, error de validacion y error de transporte. Render text/tsv queda fuera del slice sin binario.
Bloqueos: ninguno
Estado: completada ejecutable
```

```text
ID: CLI-002
Objetivo: Preparar el comando fino `app spec solicitar` sobre `SolicitarNuevaApp v0`.
Write-set: solicitar_nueva_app_client_v0.go, solicitar_nueva_app_client_v0_test.go, docs/tareas.md, docs/pruebas.md
Simbolo foco: SolicitarNuevaAppCliClientV0
Contrato: SolicitarNuevaApp v0
Validacion: tests con httptest verifican POST JSON, X-Correlation-ID, respuesta canonica app_spec/backlog, 400 publico, 500 sin body privado, timeout, respuesta invalida, alias remotos solo como compatibilidad y server_url sin credenciales.
Bloqueos: ninguno; tipos canonicos importables desde orquesta-factory resueltos.
Estado: completada ejecutable
```

```text
ID: CLI-003
Objetivo: Documentar y luego implementar adaptador de transporte configurable sin logica de negocio.
Write-set: transport_rest_v0.go, transport_rest_v0_test.go, solicitar_nueva_app_client_v0.go, operational_status_client_v0.go, docs/contratos.md, docs/pruebas.md, docs/tareas.md, docs/decisiones.md
Simbolo foco: cliRESTClientConfigV0, prepareCLIRESTRequestV0
Contrato: CliInvocationContextV0 + contrato publico consumido
Validacion: capa comun reutilizada por `SolicitarNuevaAppCliClientV0` y `OperationalStatusCliClientV0`; tests cubren normalizacion de `server_url`, timeout por defecto, headers canonicos, reutilizacion de cliente inyectado y validaciones previas existentes de timeouts, status no 2xx, invalid JSON y `server_url` sin credenciales.
Bloqueos: ninguno
Estado: completada ejecutable
```

```text
ID: CLI-004
Objetivo: Preparar comandos `contratos funcion listar|ver|registrar` como adaptadores de `FunctionContract v0`.
Write-set: function_contract_client_v0.go, function_contract_client_helpers_v0.go, function_contract_client_v0_test.go, function_contract_client_errors_v0_test.go, README.md, docs/contratos.md, docs/pruebas.md, docs/tareas.md, docs/decisiones.md
Simbolo foco: FunctionContractCliClientV0
Contrato: FunctionContract v0
Validacion: tests con httptest verifican `POST /api/v0/core/function-contracts/list`, `POST /api/v0/core/function-contracts/view`, `X-Correlation-ID`, request_id/correlation_id tecnicos, respuesta canonica read-only, 400 publico, timeout, respuesta invalida, rechazo de `server_url` con credenciales y `registrar` bloqueado con `registrar_function_contract_bloqueado`.
Bloqueos: ninguno para listar/ver read-only; `registrar FunctionContractV0` sigue bloqueado por OrchestrationRun/CommandHandler/Outbox.
Estado: completada ejecutable
```

```text
ID: CLI-005
Objetivo: Preparar lectura secundaria de `GovernanceCatalog v0`.
Write-set: governance_catalog_client_v0.go, governance_catalog_client_v0_test.go, docs/contratos.md, docs/pruebas.md, docs/tareas.md, docs/decisiones.md
Simbolo foco: GovernanceCatalogCliReaderV0
Contrato: GovernanceCatalog v0
Validacion: tests con httptest verifican POST JSON a la ruta compacta asumida, `X-Correlation-ID`, filtros `module|role|phase|tags`, respuesta read-only con `effective` + contadores `proposed/quarantine`, 400 publico, 500/timeout sin body privado, JSON invalido y `server_url` sin credenciales. La CLI valida que `effective` solo contenga entradas aprobadas y que `counters.effective` coincida con el tamano devuelto.
Bloqueos: ninguno local; queda acoplamiento minimo documentado a la publicacion de `POST /api/v0/governance/catalog/query` por `orquesta-governance`
Estado: completada ejecutable
```

```text
ID: CLI-006
Objetivo: Definir diagnostico/recuperacion CLI sin acceso a DB, runtime ni filesystem interno.
Write-set: operational_status_client_v0.go, operational_status_client_v0_test.go, docs/contratos.md, docs/decisiones.md, docs/pruebas.md, docs/tareas.md
Simbolo foco: OperationalStatusCliClientV0
Contrato: OperationalStatusQuery v0 + DiagnosticoCompactoV0
Validacion: tests con httptest verifican POST JSON, X-Correlation-ID, timeout, 400 publico, 500 sin body privado, JSON invalido y respuesta compacta read-only valida.
Bloqueos: ninguno; contrato compartido resuelto y DTO importable desde orquesta-observability.
Estado: completada ejecutable
```

```text
ID: CLI-007
Objetivo: Mantener inventario de comandos y flags V1 reutilizables sin copiar codigo.
Write-set: README.md, docs/contratos.md, docs/tareas.md
Simbolo foco: CompatV1InventoryV0
Contrato: contratos locales orquesta-cli
Validacion: inventario documental actualizado con familias V1 observadas, cada flag reutilizable apunta a contrato V2 real o queda en cuarentena/descartado con motivo; sin copiar `cmd/` ni prometer compatibilidad binaria.
Bloqueos: ninguno
Estado: completada documental
```

```text
ID: CLI-008
Objetivo: Planificar harness de pruebas CLI-first para contratos y automatizacion.
Write-set: docs/pruebas.md
Simbolo foco: matriz de pruebas CLI
Contrato: CliOutputEnvelopeV0 + contratos consumidos
Validacion: docs/pruebas.md define matriz CLI-first por capas (unit, contract, integration_fake_server, smoke, blocked_future), cobertura explicita para `SolicitarNuevaApp v0` y `OperationalStatusQuery v0`, y regla de ausencia segura para contratos aun bloqueados
Bloqueos: ninguno
Estado: completada documental
```

```text
ID: CLI-009
Objetivo: Definir estrategia i18n de textos visibles de CLI sin mezclarla con reglas de negocio.
Write-set: docs/contratos.md, docs/pruebas.md; catalogos futuros fuera de esta tarea
Simbolo foco: CliI18nPolicyV0
Contrato: GenerarI18nDocsIniciales v0 como referencia de politica, contratos locales CLI
Validacion: politica local documenta locale por defecto `es`, `supported_locales` iniciales, fallback controlado locale pedido -> locale base -> `es`, y regla de no introducir strings visibles nuevas sin clave/catalogo local o `codigo` estable
Bloqueos: ninguno; el catalogo CLI queda local al modulo hasta que exista contrato compartido especifico para adaptadores
Estado: completada documental
```

```text
ID: CLI-010
Objetivo: Sanear zona amarilla de CLI dividiendo tests de SolicitarNuevaApp y helpers tecnicos de clientes sin cambiar contratos.
Write-set: solicitar_nueva_app_client_v0.go, solicitar_nueva_app_client_helpers_v0.go, solicitar_nueva_app_client_*_v0_test.go, governance_catalog_client_v0.go, governance_catalog_client_helpers_v0.go, operational_status_client_v0.go, operational_status_client_helpers_v0.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: SolicitarNuevaAppCliClientV0, OperationalStatusCliClientV0, GovernanceCatalogCliReaderV0
Contrato: SolicitarNuevaApp v0, OperationalStatusQuery v0, GovernanceCatalog v0
Validacion: gofmt; go test -count=1 ./modulos/orquesta-cli; git diff --check -- modulos/orquesta-cli; wc -l de ficheros Go tocados.
Bloqueos: ninguno.
Estado: completada; tests y helpers quedan separados por escenario/responsabilidad, con ficheros tocados de 213 lineas o menos.
```

```text
ID: CLI-011
Objetivo: Preparar cliente CLI fino para BootstrapProyectoDesdeAppSpec v0 sin binario completo ni fallback local.
Write-set: bootstrap_appspec_client_v0.go, bootstrap_appspec_client_helpers_v0.go, bootstrap_appspec_client_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: BootstrapAppSpecCliClientV0
Contrato: BootstrapProyectoDesdeAppSpec v0
Validacion: gofmt; go test -count=1 ./modulos/orquesta-cli; git diff --check -- modulos/orquesta-cli; wc -l de ficheros Go tocados. Tests con httptest verifican POST JSON a /api/v0/director/bootstrap/appspec, X-Correlation-ID, request_id/correlation_id/idempotency_key tecnicos, 400 publico, 500/timeout sin body privado, JSON/respuesta invalida y server_url sin credenciales.
Bloqueos: ninguno local; si el director publica otra ruta/version, el cambio queda aislado en la constante del cliente y sus tests.
Estado: completada ejecutable
```

```text
ID: CLI-012
Objetivo: Exponer comandos CLI finos para autoprogramacion supervisada sin entrar al nucleo ni al runtime local.
Write-set: server_status_client_v0.go, autoprogramming_client_v0.go, command_handlers_autoprogramming_v0.go, command_runner_v0.go, tests locales, docs locales
Simbolo foco: ServerStatusCliClientV0, AutoprogrammingCliClientV0
Contrato: /api/v0/server/status, /api/v0/autoprogramming/prepare-run, /api/v0/runs/queue/priority, /api/v0/director/stats
Validacion: gofmt; go test -count=1 ./cmd/orquesta-cli ./modulos/orquesta-cli. Tests con httptest verifican estado servidor por GET, cola por API, run stats por API y prepare-run preservando worktree aislada y branch_ref opaca.
Bloqueos: ninguno local; diagnostico mas rico depende de nuevos puertos publicos del servidor.
Estado: completada ejecutable
```

## CONSULTA AL DIRECTOR registrada

- CLI-004: resuelta para cliente CLI read-only con rutas REST candidatas; `registrar` sigue bloqueado.
- CLI-006: resuelta con `OperationalStatusQuery v0` read-only; recuperacion activa sigue fuera de alcance del modulo.

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
