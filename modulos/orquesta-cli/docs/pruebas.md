# Pruebas locales: orquesta-cli

Registra pruebas obligatorias del modulo.

## Harness CLI-first v0

Matriz minima para slices presentes y futuros bloqueados. La CLI se prueba desde
fuera hacia dentro: envelope estable, transporte publico, fake server y ausencia
de fallback local.

| Capa | Objetivo | Contratos cubiertos | Evidencia minima |
| --- | --- | --- | --- |
| `unit` | DTO local, normalizacion de contexto y envelope estable | `CliInvocationContextV0`, `CliOutputEnvelopeV0` | golden JSON, ids generados, rechazo de `server_url` con credenciales |
| `unit` | transporte REST tecnico reutilizable sin negocio | `CliRESTTransportAdapterV0` | base URL normalizada, timeout por defecto, POST JSON, `X-Correlation-ID`, cliente inyectado |
| `contract` | cliente fino por contrato publico | `SolicitarNuevaApp v0`, `OperationalStatusQuery v0` | 200 canonico, 400 publico, 500/timeout compactos, JSON invalido |
| `integration_fake_server` | composicion CLI-adaptador-servidor simulado sin negocio local | contratos actuales y futuros desbloqueados | ruta/version correctas, `X-Correlation-ID`, headers, timeouts |
| `smoke` | ergonomia minima sin efectos laterales | ayuda, locale, config, comandos read-only | exit code, sin DB/runtime/filesystem local, sin binarios V1 |
| `blocked_future` | preservar hueco contractual sin inventar implementacion | mutaciones de `FunctionContract v0` y otros futuros | comando ausente, `contrato_no_configurado` o bloqueo publico, nunca fallback local |

### Cobertura por contrato

| Contrato/familia | Unit | Contract | Integration fake server | Smoke | Estado |
| --- | --- | --- | --- | --- | --- |
| `SolicitarNuevaApp v0` | si | si | si | si | activo |
| `OperationalStatusQuery v0` + `DiagnosticoCompactoV0` | si | si | si | si | activo |
| `FunctionContract v0` listar/ver | si | si | si | si | activo read-only |
| `FunctionContract v0` registrar | si | no | no | si, solo rechazo | bloqueado |
| `GovernanceCatalog v0` listar/ver | si | si | si | si | activo |
| `BootstrapProyectoDesdeAppSpec v0` | si | si | si | si | activo |
| futuros contratos read-only | segun DTO local | planificado | planificado | si | pendiente |

### Reglas del harness

- Cada comando nuevo entra primero por `unit` y `contract`; `integration_fake_server`
  solo cuando ya exista puerto publico.
- Ninguna prueba CLI puede depender de DB, runtime, HOME, worktree o filesystem
  operativo local.
- Los tests de contratos futuros bloqueados validan ausencia segura o
  `contrato_no_configurado`; no crean mocks de negocio local.
- La evidencia preferida es `go test` por paquete con `httptest`; si en el futuro
  existe binario CLI, los smoke podran duplicarse como capa externa sin cambiar
  los contratos.

## Pruebas previstas

```text
Caso: CLI-P001 ayuda sin efectos laterales
Tipo: smoke
Comando: orquesta-cli --help
Evidencia esperada: salida de ayuda estable, exit 0, sin llamadas de red, DB ni filesystem operativo
Ultima ejecucion: pendiente
Riesgos: reintroducir inicializacion local heredada de V1
```

```text
Caso: CLI-P002 envelope JSON de exito
Tipo: unit
Comando: orquesta-cli app spec solicitar --input request_minima_valida.json --json
Evidencia esperada: CliOutputEnvelopeV0 con ok=true, contract=SolicitarNuevaApp, version=v0, app_spec y backlog canonicos
Ultima ejecucion: 2026-05-04, go test -count=1 .; TestCliOutputEnvelopeV0JSONEstable y TestSolicitarNuevaAppCliClientV0ExitoPropagaCorrelacionYEnvelopeCanonico
Riesgos: envolver el payload con shape propio incompatible
```

```text
Caso: CLI-P003 propagacion de correlacion
Tipo: contract
Comando: orquesta-cli app spec solicitar --input request_minima_valida.json --correlation-id cli-test --json
Evidencia esperada: fake server recibe X-Correlation-ID=cli-test y el envelope devuelve el mismo correlation_id
Ultima ejecucion: 2026-05-04, go test -count=1 .; TestNormalizeCliInvocationContextV0GeneraRequestYCorrelationID y TestSolicitarNuevaAppCliClientV0ExitoPropagaCorrelacionYEnvelopeCanonico
Riesgos: perder trazabilidad entre CLI y factory
```

```text
Caso: CLI-P004 validacion 400 de factory
Tipo: contract
Comando: orquesta-cli app spec solicitar --input request_i18n_invalida.json --json
Evidencia esperada: ok=false, errores publicos del contrato, exit no cero documentado
Ultima ejecucion: 2026-05-04, go test -count=1 .; TestSolicitarNuevaAppCliClientV0Respuesta400DevuelveErroresPublicos
Riesgos: duplicar validacion de negocio en CLI
```

```text
Caso: CLI-P005 error de transporte no filtra internals
Tipo: unit
Comando: orquesta-cli app spec solicitar --input request_minima_valida.json --json
Evidencia esperada: ante 500/timeout devuelve error_transporte, retryable cuando aplique y no incluye body privado remoto
Ultima ejecucion: 2026-05-04, go test -count=1 .; TestSolicitarNuevaAppCliClientV0Status500NoFiltraBodyPrivado y TestSolicitarNuevaAppCliClientV0TimeoutDevuelveErrorTransporte
Riesgos: exponer stack traces, rutas o detalles de proveedor
```

```text
Caso: CLI-P006 respuesta invalida
Tipo: unit
Comando: orquesta-cli app spec solicitar --input request_minima_valida.json --json
Evidencia esperada: ante JSON mal formado o envelope sin app_spec/backlog devuelve respuesta_invalida
Ultima ejecucion: 2026-05-04, go test -count=1 .; TestSolicitarNuevaAppCliClientV0RespuestaInvalida y TestSolicitarNuevaAppCliClientV0AceptaAliasRemotoPeroSalidaEsCanonica
Riesgos: aceptar alias o shapes no canonicos como salida propia
```

```text
Caso: CLI-P007 bloqueo de fallback local
Tipo: smoke
Comando: orquesta-cli app spec solicitar --input request_minima_valida.json --server-url http://127.0.0.1:1 --json
Evidencia esperada: falla por error_transporte; no intenta DB local ni comandos V1
Ultima ejecucion: pendiente
Riesgos: violar server-first por compatibilidad
```

```text
Caso: CLI-P008 inventario V1 en cuarentena
Tipo: contract
Comando: revision documental de CompatV1InventoryV0
Evidencia esperada: cada comando/flag reutilizado tiene contrato V2; runtime/worktree/modelo/pool quedan bloqueados si no hay puerto publico
Ultima ejecucion: pendiente
Riesgos: copiar comportamiento de `cmd/` V1 o hardcodear control plane
```

```text
Caso: CLI-P009 FunctionContract pendiente no implementa negocio
Tipo: contract
Comando: orquesta-cli contratos funcion registrar --json
Evidencia esperada: `RegistrarFunctionContract` devuelve `registrar_function_contract_bloqueado`; nunca lee tareas V1, DB, filesystem ni intenta ruta REST mutante
Ultima ejecucion: 2026-05-04, go test -count=1 ./modulos/orquesta-cli; TestFunctionContractCliClientV0RegistrarBloqueado
Riesgos: reintroducir microprogramacion hardcodeada
```

```text
Caso: CLI-P010 Governance read-only
Tipo: contract
Comando: orquesta-cli gobernanza catalogo listar --json
Evidencia esperada: `GovernanceCatalogCliReaderV0` hace `POST /api/v0/governance/catalog/query`, propaga `request_id`, `correlation_id` y `X-Correlation-ID`, consume respuesta publica `{effective,counters}` con ids compactos y devuelve `GovernanceCatalogQueryResultV0` canonico sin activar ni mutar reglas
Ultima ejecucion: 2026-05-04, go test -count=1 .; TestGovernanceCatalogCliReaderV0ExitoPropagaCorrelacionYEnvelopeCanonico y TestGovernanceCatalogCliReaderV0Respuesta400DevuelveErroresPublicos
Riesgos: convertir CLI en control plane de gobernanza
```

```text
Caso: CLI-P011 diagnostico compacto read-only
Tipo: smoke
Comando: orquesta-cli doctor contratos --json
Evidencia esperada: `OperationalStatusCliClientV0` consume `OperationalStatusQuery v0`, propaga `X-Correlation-ID` y devuelve `DiagnosticoCompactoV0` canonico sin inspeccionar internals
Ultima ejecucion: 2026-05-04, go test -count=1 .; TestOperationalStatusCliClientV0ExitoPropagaCorrelacionYEnvelopeCanonico y TestOperationalStatusCliClientV0Respuesta400DevuelveErroresPublicos
Riesgos: introducir shape paralelo o diagnostico que lea DB, runtime o filesystem privado
```

```text
Caso: CLI-P014 diagnostico transporte y JSON invalido
Tipo: unit
Comando: orquesta-cli doctor contratos --json
Evidencia esperada: 500/timeout devuelven `error_transporte` sin body privado; JSON invalido o diagnostico no valido devuelve errores publicos compactos
Ultima ejecucion: 2026-05-04, go test -count=1 .; TestOperationalStatusCliClientV0Status500NoFiltraBodyPrivado, TestOperationalStatusCliClientV0TimeoutDevuelveErrorTransporte y TestOperationalStatusCliClientV0RespuestaInvalida
Riesgos: filtrar secretos remotos o aceptar respuestas no canonicas
```

```text
Caso: CLI-P012 i18n de errores publicos
Tipo: unit
Comando: orquesta-cli app spec solicitar --input request_i18n_invalida.json --locale es --json
Evidencia esperada: errores con codigo estable y mensaje_i18n; fallback documentado si falta catalogo
Ultima ejecucion: 2026-05-04, go test -count=1 .; TestCliOutputEnvelopeV0JSONEstable y TestSolicitarNuevaAppCliClientV0Respuesta400DevuelveErroresPublicos
Riesgos: strings no localizables en comandos nuevos
```

```text
Caso: CLI-P013 server_url sin credenciales
Tipo: unit
Comando: orquesta-cli app spec solicitar --server-url https://usuario:secreto@example.test --json
Evidencia esperada: rechazo con configuracion_cli_invalida, campo server_url, sin filtrar usuario ni secreto
Ultima ejecucion: 2026-05-04, go test -count=1 .; TestSolicitarNuevaAppCliClientV0RechazaServerURLConCredenciales
Riesgos: exponer credenciales o aceptar endpoints no publicos como configuracion valida
```

```text
Caso: CLI-P015 matriz harness SolicitarNuevaApp
Tipo: contract
Comando: revision documental de harness CLI-first
Evidencia esperada: `SolicitarNuevaApp v0` queda cubierto por unit, contract, integration_fake_server y smoke sin fallback local
Ultima ejecucion: 2026-05-04, revision documental de docs/pruebas.md
Riesgos: dejar huecos entre DTO local, transporte y ausencia de smoke
```

```text
Caso: CLI-P016 matriz harness OperationalStatus
Tipo: contract
Comando: revision documental de harness CLI-first
Evidencia esperada: `OperationalStatusQuery v0` queda cubierto por unit, contract, integration_fake_server y smoke read-only
Ultima ejecucion: 2026-05-04, revision documental de docs/pruebas.md
Riesgos: mezclar diagnostico compacto con recuperacion activa o con internals
```

```text
Caso: CLI-P017 contratos futuros bloqueados
Tipo: smoke
Comando: orquesta-cli contratos funcion listar --json; orquesta-cli gobernanza catalogo listar --json
Evidencia esperada: `FunctionContract` listar/ver usan rutas REST candidatas read-only y fallan por transporte si no hay servidor; `GovernanceCatalog` usa transporte publico compacto; ninguno hace fallback local
Ultima ejecucion: 2026-05-04, go test -count=1 ./modulos/orquesta-cli; TestFunctionContractCliClientV0ListarOK, TestFunctionContractCliClientV0VerOK y tests GovernanceCatalog existentes
Riesgos: mezclar operaciones read-only activas con mutaciones todavia bloqueadas
```

```text
Caso: CLI-P018 locale por defecto y fallback controlado
Tipo: contract
Comando: orquesta-cli app spec solicitar --input request_i18n_invalida.json --json
Evidencia esperada: locale por defecto `es`; si falta clave del locale pedido, fallback controlado a locale base o `es`, manteniendo `codigo` y shape estable
Ultima ejecucion: 2026-05-04, revision documental de docs/contratos.md y docs/pruebas.md
Riesgos: introducir texto libre no catalogado o cambiar semantica del error por traduccion
```

```text
Caso: CLI-P019 strings visibles sin catalogo
Tipo: contract
Comando: revision documental de comandos nuevos
Evidencia esperada: ayuda, warnings y errores visibles de slices nuevos referencian clave local o `codigo` estable; no aparecen strings nuevas hardcodeadas
Ultima ejecucion: 2026-05-04, revision documental de docs/contratos.md y docs/pruebas.md
Riesgos: acoplar UX e i18n al negocio o romper consistencia entre locales
```

```text
Caso: CLI-P020 transporte REST compartido sin negocio
Tipo: unit
Comando: go test -count=1 .
Evidencia esperada: helper comun normaliza `server_url`, aplica timeout por defecto, construye POST JSON con `X-Correlation-ID` y reutiliza `http.Client` inyectado; los clientes `SolicitarNuevaAppCliClientV0` y `OperationalStatusCliClientV0` lo consumen sin cambiar contratos publicos
Ultima ejecucion: 2026-05-04, go test -count=1 .; TestNewCLIRESTClientConfigV0NormalizaBaseYTimeout, TestPrepareCLIRESTRequestV0SeteaHeadersCanonicos y TestHTTPClientFromConfigV0RespetaClienteInyectado
Riesgos: duplicar transporte entre clientes o mezclar parseo funcional en la capa tecnica comun
```

```text
Caso: CLI-P021 governance transporte y validacion compacta
Tipo: unit
Comando: orquesta-cli gobernanza catalogo listar --json
Evidencia esperada: 500/timeout devuelven `error_transporte` sin body privado; JSON invalido o `effective` mal formado devuelve `respuesta_invalida`; `server_url` con credenciales se rechaza
Ultima ejecucion: 2026-05-04, go test -count=1 .; TestGovernanceCatalogCliReaderV0Status500NoFiltraBodyPrivado, TestGovernanceCatalogCliReaderV0TimeoutDevuelveErrorTransporte, TestGovernanceCatalogCliReaderV0RespuestaInvalida y TestGovernanceCatalogCliReaderV0RechazaServerURLConCredenciales
Riesgos: aceptar catalogos efectivos mal promovidos o filtrar detalles privados del servidor
```

```text
Caso: CLI-P022 saneamiento de tests y helpers de clientes
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-cli
Evidencia esperada: tests de SolicitarNuevaApp quedan separados por exito, errores, configuracion y helpers compartidos; clientes REST mantienen JSON, headers, errores publicos y validaciones existentes.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: el saneamiento no anyade cobertura nueva; verifica que la division no cambia contratos ni comportamiento.
```

```text
Caso: CLI-P023 bootstrap AppSpec desde director
Tipo: contract
Comando: orquesta-cli app spec bootstrap --json
Evidencia esperada: `BootstrapAppSpecCliClientV0` hace `POST /api/v0/director/bootstrap/appspec`, propaga `X-Correlation-ID`, `request_id`, `correlation_id` e `idempotency_key`, y devuelve `BootstrapProyectoDesdeAppSpecResultV0` canonico con registro compacto, refs opacas, StartRun y RunStarted.
Ultima ejecucion: 2026-05-04, go test -count=1 ./modulos/orquesta-cli; TestBootstrapAppSpecCliClientV0ExitoPropagaCorrelacionYEnvelopeCanonico.
Riesgos: duplicar en CLI la composicion del director o llamar core/workflow localmente.
```

```text
Caso: CLI-P024 bootstrap transporte y errores compactos
Tipo: unit
Comando: orquesta-cli app spec bootstrap --json
Evidencia esperada: 400 devuelve errores publicos del director/core-workflow; 500/timeout devuelven `error_transporte` sin body privado; JSON o resultado invalido devuelve `respuesta_invalida`; `server_url` con credenciales se rechaza sin filtrar usuario/secreto.
Ultima ejecucion: 2026-05-04, go test -count=1 ./modulos/orquesta-cli; TestBootstrapAppSpecCliClientV0Respuesta400DevuelveErroresPublicos, TestBootstrapAppSpecCliClientV0Status500NoFiltraBodyPrivado, TestBootstrapAppSpecCliClientV0TimeoutDevuelveErrorTransporte, TestBootstrapAppSpecCliClientV0RespuestaInvalida y TestBootstrapAppSpecCliClientV0RechazaServerURLConCredenciales.
Riesgos: exponer stack traces del director o aceptar una respuesta no compacta.
```

```text
Caso: CLI-P025 autoprogramacion cliente fino
Tipo: contract
Comando: orquesta-cli servidor estado --json; orquesta-cli autoprogramacion estado ver --run-ref RUN_REF --json; orquesta-cli autoprogramacion supervisar --max-ticks 1 --json; orquesta-cli autoprogramacion cola listar --json; orquesta-cli autoprogramacion run ver --run-ref RUN_REF --json; orquesta-cli autoprogramacion run controlar --run-ref RUN_REF --action pause --json
Evidencia esperada: la CLI consume solo HTTP/API publica, propaga X-Correlation-ID y no lee stores, runtime, worktrees ni filesystem interno; estado, supervisor y control de run viajan por endpoints publicos, no por runtime local.
Ultima ejecucion: 2026-05-23, go test -count=1 ./cmd/orquesta-cli ./modulos/orquesta-cli; TestRunOrquestaCLIV0ServidorEstadoUsaAPI, TestRunOrquestaCLIV0AutoprogramacionColaListarUsaAPI, TestRunOrquestaCLIV0AutoprogramacionEstadoYSupervisarUsanAPI y TestRunOrquestaCLIV0AutoprogramacionRunControlarUsaAPI.
Riesgos: convertir CLI en control plane local o recomponer estado de cola/run fuera del servidor.
```

```text
Caso: CLI-P026 prepare-run autoprogramacion preserva refs opacas
Tipo: contract
Comando: orquesta-cli autoprogramacion preparar --input request.json --json
Evidencia esperada: el cliente envia el envelope MCP/HTTP sin reescribir branch_ref ni worktree_ref, exige la respuesta publica del servidor y falla por transporte si no hay API.
Ultima ejecucion: 2026-05-23, go test -count=1 ./cmd/orquesta-cli ./modulos/orquesta-cli; TestAutoprogrammingCliClientV0PrepararRunPreservaWorktreeYRamaOpaca.
Riesgos: normalizar refs opacas en CLI o lanzar agentes directamente desde el cliente.
```

```text
Caso: CLI-P027 configuracion residente visible para operador y CLI
Tipo: contract
Comando: orquesta-cli servidor estado --json; orquesta-cli autoprogramacion estado ver --run-ref RUN_REF --json; orquesta-cli autoprogramacion cola listar --json; orquesta-cli autoprogramacion run ver --run-ref RUN_REF --json; orquesta-cli autoprogramacion supervisar --max-ticks 1 --json; orquesta-cli autoprogramacion run controlar --run-ref RUN_REF --action pause --json; revision documental de ORQUESTA_SERVER_ALLOW_REPEATED_RUNS, ORQUESTA_SERVER_ALLOW_REPEAT, ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_* y limites residentes
Evidencia esperada: el operador encuentra las variables `ORQUESTA_SERVER_SUPERVISOR_MAX_TICKS`, `ORQUESTA_SERVER_ALLOW_REPEATED_RUNS`, `ORQUESTA_SERVER_MAX_RUNS_PER_TICK`, `ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK`, `ORQUESTA_SERVER_DRAIN_MAX_BURSTS`, `ORQUESTA_SERVER_DRAIN_MAX_STEPS`, `ORQUESTA_SERVER_DRAIN_MAX_DISPATCHES`, `ORQUESTA_SERVER_DRAIN_MAX_COMMANDS`, `ORQUESTA_SERVER_DRAIN_MAX_OUTBOX`, `ORQUESTA_SERVER_DRAIN_MAX_DECISIONS`, `ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS`, `ORQUESTA_SERVER_TICK_INTERVAL_MS` y `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_*` en docs CLI/runbook; cada variable declara frontera o default operativo; `ORQUESTA_SERVER_ALLOW_REPEATED_RUNS` aparece con nombre exacto y `ORQUESTA_SERVER_ALLOW_REPEAT` aparece solo como alias no valido; los perfiles por entorno quedan como variables del proceso servidor, no como flags CLI; la CLI observa estado, cola, run, supervision y control por API publica, sin leer entorno, statefile ni runtime local; si falta snapshot publico de un valor efectivo, la salida debe tratarlo como configuracion no visible y no reconstruirlo.
Ultima ejecucion: 2026-05-23, revision documental de rework `task-ref-self-improvement-ed850d0c5d8d`; go test -count=1 ./modulos/orquesta-cli.
Riesgos: ocultar la configuracion residente al operador, o convertir la CLI en fuente local de configuracion del servidor.
```

```text
Caso: CLI-P028 FunctionContract read-only listar/ver
Tipo: contract
Comando: orquesta-cli contratos funcion listar --json; orquesta-cli contratos funcion ver --json
Evidencia esperada: `FunctionContractCliClientV0` hace `POST /api/v0/core/function-contracts/list` y `POST /api/v0/core/function-contracts/view`, propaga `X-Correlation-ID`, request_id y correlation_id, y devuelve resumenes o `FunctionContractV0` canonico de core sin fallback local.
Ultima ejecucion: 2026-05-04, go test -count=1 ./modulos/orquesta-cli; TestFunctionContractCliClientV0ListarOK y TestFunctionContractCliClientV0VerOK.
Riesgos: inventar shape paralelo o validar reglas de microtarea dentro de CLI.
```

```text
Caso: CLI-P029 FunctionContract transporte y errores publicos
Tipo: unit
Comando: orquesta-cli contratos funcion listar --json; orquesta-cli contratos funcion ver --json
Evidencia esperada: 400 devuelve `FunctionContractErrorV0` publico, timeout devuelve `error_transporte`, JSON/shape invalido devuelve `respuesta_invalida`, `server_url` con credenciales se rechaza sin filtrar usuario/secreto y `registrar` devuelve `registrar_function_contract_bloqueado`.
Ultima ejecucion: 2026-05-04, go test -count=1 ./modulos/orquesta-cli; TestFunctionContractCliClientV0Respuesta400DevuelveErroresPublicos, TestFunctionContractCliClientV0TimeoutDevuelveErrorTransporte, TestFunctionContractCliClientV0RespuestaInvalida, TestFunctionContractCliClientV0RechazaServerURLConCredenciales y TestFunctionContractCliClientV0RegistrarBloqueado.
Riesgos: filtrar body privado remoto o convertir registrar en mutacion no promovida.
```

## Plantilla

```text
Caso:
Tipo: unit | contract | integration | smoke
Comando:
Evidencia esperada:
Ultima ejecucion:
Riesgos:
```
