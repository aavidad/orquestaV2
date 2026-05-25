# Decisiones locales: orquesta-cli

Las decisiones de este archivo solo afectan a `orquesta-cli`. Si afectan a otro modulo, deben elevarse al director y registrarse en `../../CONTRATOS.md`.

## Decisiones iniciales

```text
Fecha: 2026-05-04
Decision: La CLI V2 arranca como cliente secundario y adaptador fino.
Motivo: El contrato global define que la fuente de verdad esta en modulos propietarios y que la CLI consume contratos publicos.
Alternativas: portar comandos V1 completos; crear control plane local; llamar internals.
Impacto: el backlog prioriza envelopes, transporte y contratos antes que paridad de comandos.
Contratos afectados: contratos locales orquesta-cli, SolicitarNuevaApp v0
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: No se implementa fallback local ni acceso directo a DB/runtime/filesystem.
Motivo: Server-first y hexagonal son reglas globales; la CLI no puede corregir caidas del servidor saltandose contratos.
Alternativas: permitir `--local` generico como V1; abrir DB en modo lectura; usar scripts de rescate no contratados.
Impacto: diagnostico y recuperacion quedan bloqueados hasta tener puertos publicos de solo lectura o herramientas aisladas.
Contratos afectados: CliInvocationContextV0, CliOutputEnvelopeV0
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: `SolicitarNuevaApp v0` es el primer flujo funcional objetivo de la CLI.
Motivo: El contrato global ya autoriza a `orquesta-cli` como consumidor y define ruta REST, envelope y errores publicos.
Alternativas: empezar por tareas/agentes/runtime; empezar por compatibilidad V1.
Impacto: CLI-002 queda como primer slice ejecutable cuando se pase de documentacion a codigo.
Contratos afectados: SolicitarNuevaApp v0, SolicitarNuevaAppCliClientV0
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: El inventario V1 es fuente de nombres/flags, no de implementacion.
Motivo: V1 contiene comandos acoplados a `cmd/`, DB, runtime local y control plane; copiarlos violaria los contratos V2.
Alternativas: migracion mecanica de Cobra; compatibilidad binaria; copiar handlers y protegerlos con flags.
Impacto: los flags `--json`, `--tsv`, `--limit`, `--estado`, `--proyecto`, `--agente`, `--desde` se pueden conservar solo si el contrato consumido los soporta.
Contratos afectados: CompatV1InventoryV0
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: La salida automatizable se normaliza con `CliOutputEnvelopeV0`.
Motivo: Scripts y CI necesitan shape estable y errores publicos sin detalles privados del transporte.
Alternativas: imprimir payload remoto directo; salidas ad hoc por comando; texto humano como salida primaria.
Impacto: todos los comandos nuevos deben tener pruebas golden de envelope y ausencia de secretos.
Contratos afectados: CliOutputEnvelopeV0
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: Comandos de FunctionContract y Governance quedan documentados pero bloqueados hasta transporte publico.
Motivo: El contrato global autoriza a CLI como consumidor, pero no todos los puertos de consulta/registro tienen transporte concreto.
Alternativas: leer documentos o DB directamente; replicar validadores en CLI; omitirlos del backlog.
Impacto: CLI-004 y CLI-005 no deben pasar a codigo hasta resolver contratos/transportes.
Contratos afectados: FunctionContract v0, GovernanceCatalog v0
Estado: superada; `CLI-005` queda desbloqueada con ruta REST compacta asumida y `CLI-004` queda desbloqueada solo para listar/ver read-only con rutas REST candidatas
```

## CONSULTA AL DIRECTOR

```text
Fecha: 2026-05-04
Modulo origen: orquesta-cli
Modulos afectados: orquesta-core, orquesta-observability, orquesta-runtime, orquesta-persistence
Bloqueo: diagnostico y recuperacion son responsabilidades locales, pero no existe contrato compartido de consulta compacta para estado operativo.
Pregunta concreta: se debe crear un contrato publico read-only para diagnostico/progreso compacto consumible por CLI?
Opcion recomendada: si, propiedad de observability o core, con referencias opacas y sin DB/runtime directo.
Impacto: desbloquea CLI-006 y evita portar diagnosticos V1 acoplados a internals.
Decision del director: crear y promover `OperationalStatusQuery v0` como contrato read-only propiedad de `orquesta-observability`, consumible por CLI/MCP/Web con referencias opacas y sin DB/runtime directo. Recuperacion activa queda fuera.
Estado: resuelta
```

```text
Fecha: 2026-05-04
Decision: `CLI-006` se implementa como cliente REST fino contra `OperationalStatusQuery v0` y devuelve `DiagnosticoCompactoV0` canonico.
Motivo: El contrato compartido ya existe y `orquesta-observability` aporta DTO/validador puro reutilizable; CLI no debe reinterpretar ni duplicar reglas del diagnostico.
Alternativas: DTO local espejo; diagnostico solo documental; leer recursos MCP o internals en vez de puerto publico.
Impacto: `doctor contratos` queda desbloqueado para consultas compactas read-only; recuperacion activa, repair o acceso a sinks reales siguen fuera del modulo.
Contratos afectados: OperationalStatusQuery v0, DiagnosticoCompactoV0, OperationalStatusCliClientV0
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: El transporte REST comun de CLI se extrae a una capa tecnica local reutilizable y pequena.
Motivo: `SolicitarNuevaAppCliClientV0` y `OperationalStatusCliClientV0` repetian la misma mecanica tecnica de `server_url`, timeout, POST JSON, headers y `http.Client`; mantener eso duplicado complica slices pequenos y deriva facil a inconsistencias.
Alternativas: dejar duplicacion entre clientes; crear framework de transporte mas grande; mover parseo funcional al helper comun.
Impacto: `CLI-003` queda cerrado con reutilizacion real; la capa comun solo concentra reglas tecnicas y cada cliente mantiene su decodificacion y mapping de errores por contrato.
Contratos afectados: CliRESTTransportAdapterV0, SolicitarNuevaAppCliClientV0, OperationalStatusCliClientV0
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Modulo origen: orquesta-cli
Modulos afectados: orquesta-core
Bloqueo: FunctionContract v0 menciona a CLI como adaptador fino para consultar o registrar contratos, pero no define transporte ni operaciones publicas.
Pregunta concreta: que superficie publica debe usar CLI para `listar`, `ver` y `registrar` FunctionContractV0?
Opcion recomendada: definir endpoints o puerto versionado antes de implementar CLI-004.
Impacto: evita que CLI replique microprogramacion V1 o acceda a tareas historicas.
Decision del director: CLI podra consumir operaciones candidatas `listar FunctionContractV0` y `ver FunctionContractV0` solo cuando core las promocione como contrato compartido. `registrar FunctionContractV0` queda bloqueado hasta cerrar OrchestrationRun, CommandHandler y Outbox.
Estado: resuelta para cliente CLI read-only; `registrar` sigue bloqueada
```

```text
Fecha: 2026-05-04
Decision: `CLI-004` se implementa como cliente REST fino read-only de `FunctionContract v0` con rutas candidatas `/api/v0/core/function-contracts/list` y `/api/v0/core/function-contracts/view`.
Motivo: Core ya documento operaciones candidatas `listar/ver FunctionContractV0` y `FunctionContract v0` esta promovido como contrato global minimo; la CLI puede consumir esas rutas como adaptador secundario sin inventar negocio ni fallback local.
Alternativas: dejar CLI-004 documental; leer docs/tareas locales; exponer registrar como POST mutante; copiar microprogramacion V1.
Impacto: `FunctionContractCliClientV0` propaga request/correlation por REST, devuelve resumenes o `FunctionContractV0` publico de core, mapea `FunctionContractErrorV0` a errores CLI y falla por transporte si el servidor no publica la ruta. `RegistrarFunctionContract` queda bloqueado explicitamente con `registrar_function_contract_bloqueado`.
Contratos afectados: FunctionContract v0, FunctionContractCliClientV0, CliOutputEnvelopeV0
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: `CLI-005` se implementa como cliente REST fino read-only sobre una ruta compacta asumida para `GovernanceCatalog v0`.
Motivo: `GovernanceCatalog v0` ya esta promovido como contrato compartido y `orquesta-governance` expone DTOs/validadores reutilizables; faltaba solo fijar un acoplamiento minimo de transporte para no dejar la CLI esperando a un servidor final.
Alternativas: dejarlo documental; leer docs/fixtures locales; devolver el catalogo completo `GovernanceCatalogV0`; crear un DTO espejo solo de CLI.
Impacto: la CLI consume `POST /api/v0/governance/catalog/query` con envelope `{request_id, correlation_id, filters}`, propaga `X-Correlation-ID`, consume respuesta publica `{request_id, correlation_id, effective, counters}` y normaliza errores HTTP `{errors:[{code, field}]}` a errores CLI. Devuelve `GovernanceCatalogQueryResultV0` canonico y valida localmente que `effective` solo contenga entradas efectivas aprobadas.
Contratos afectados: GovernanceCatalog v0, GovernanceCatalogCliReaderV0
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: Los tests de `SolicitarNuevaAppCliClientV0` se dividen por escenario y los helpers tecnicos repetidos de clientes se separan en ficheros locales pequenos.
Motivo: varios ficheros CLI estaban en zona amarilla y eran candidatos a crecer con nuevos comandos; separar escenarios y helpers reduce contexto por microtarea.
Alternativas: mantener tests monoliticos; mover transporte a otro modulo; cambiar contratos de clientes para reducir codigo.
Impacto: se preservan los contratos publicos y queda mas barato depurar fallos de transporte, configuracion y respuestas por cliente.
Contratos afectados: SolicitarNuevaApp v0, OperationalStatusQuery v0, GovernanceCatalog v0
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: `CLI-011` se implementa como cliente REST fino de `BootstrapProyectoDesdeAppSpec v0` con ruta local asumida `/api/v0/director/bootstrap/appspec`.
Motivo: El contrato global autoriza a CLI como consumidor secundario y exige que Web/MCP/CLI consuman el puerto del director en vez de duplicar la composicion de `SolicitarNuevaApp`, registro de core y StartRun.
Alternativas: recomponer el flujo en CLI; invocar `BootstrapProyectoDesdeAppSpecV0` local; copiar comandos V1; dejar solo documentacion.
Impacto: `BootstrapAppSpecCliClientV0` solo normaliza datos tecnicos de invocacion, envia POST JSON al director y mapea errores publicos compactos. Los tests usan fake server HTTP y no llaman core/workflow ni al caso de uso local del director.
Contratos afectados: BootstrapProyectoDesdeAppSpec v0, CliOutputEnvelopeV0
Estado: aceptada_local
```

```text
Fecha: 2026-05-23
Decision: La CLI de autoprogramacion consume solo endpoints HTTP publicos del servidor.
Motivo: T06 exige cliente fino para estado, cola y runs sin entrar al nucleo ni al runtime.
Alternativas: leer stores/run-state locales; reutilizar comandos internos de `cmd/orquesta-server`; arrancar agentes desde CLI.
Impacto: `servidor estado`, `autoprogramacion preparar`, `autoprogramacion cola listar` y `autoprogramacion run ver` quedan como adaptadores REST, con refs opacas preservadas.
Contratos afectados: ServerStatus v0, Autoprogramming prepare-run v0, RunQueuePriority v0, DirectorStats v0
Estado: aceptada_local
```

## Plantilla

```text
Fecha:
Decision:
Motivo:
Alternativas:
Impacto:
Contratos afectados:
Estado:
```
