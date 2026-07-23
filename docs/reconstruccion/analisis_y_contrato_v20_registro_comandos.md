# V20: registro único de comandos y bindings públicos

Fecha de decisión: 2026-07-22 Europe/Madrid. Integración corregida: 2026-07-23.
Base acreditada V19: `42692512b0048f116d660a53fbe6650e243bb4e3`.

Estado: `integrated_implementation_complete_pending_seal`. Dos revisores rechazaron
integrar la rama privada V20 completa: no tenía SQLite, bootstrap, recovery ni
E2E y su catálogo anterior a V19 omitía `CouncilPolicy`, `OpenCouncilRound` y
`SkipCouncil`. La implementación final se reconstruyó selectivamente sobre el
E de V19, con 25 comandos, una sola autoridad MCP y P/S/E propio.

Las referencias posteriores a «preflight privado», 23 comandos o dependencia
pendiente de V17 describen la primera exploración histórica. Esta cabecera, el
fixture y el workset integrados mandan sobre esas notas. Se conservan las notas
históricas que explican decisiones, pero no describen el estado vivo. El runtime
acreditado se amplió por composición; no se creó un runtime paralelo.

## Declaración obligatoria de tarea

```text
capability IDs: GOV-17, UI-02
invariante: una definición canónica produce todas las superficies públicas
autoridad que escribe: casos de uso application-side; Goal conserva writer único
puertos afectados: identidad request-bound, auditoría durable de comandos, estado SQLite
adaptadores afectados: HTTP, MCP, CLI, SDK Go pequeño, SQLite, bootstrap
write-set preflight: análisis V20, manifest V20, acceptance/fixture V20
dependencias causales: V07, V10, V17, V18 y V19 acreditadas
código antiguo retirado como autoridad: registro/handlers/error map MCP manuscritos
test de contrato: TestAcceptanceV20CommandRegistry
negativo/mutación: drift de schema/auth/i18n/error/handler por binding; lifecycle en interfaz
E2E/gate: misma invocación HTTP/MCP/CLI/SDK, SQLite/restart y audit digest exacto
presupuesto: 6.400 LOC manuales, 3.000 generadas, 500 SQL; un registry y un writer
```

Manifest autoritativo del frente:
`docs/reconstruccion/worksets/v20_command_registry.json`.

## 1. Alcance exacto y fronteras

V20 posee únicamente:

- `GOV-17`: auditoría durable de comandos, decisiones, efectos y cierres;
- `UI-02`: servidor MCP real sobre el registro común;
- definición canónica versionada de comando/query, schemas de payload,
  handler, permiso, audience, política de replay, clave i18n, códigos máquina
  y nombres de bindings;
- dispatcher y handlers en el lado de aplicación;
- proyección determinista compilada de definiciones; bindings manuales
  HTTP/MCP/CLI derivados de ella en runtime y SDK HTTP genérico compatible;
- migración/recovery de la auditoría de invocaciones;
- paridad semántica y E2E público proporcional.

V20 no posee:

- traducción y formato total BCP-47, plurales, fechas, números, moneda o zona
  horaria: V21;
- el E2E Codex completo: V22;
- Wizard/dossier: V23;
- web administrativa o mutación pública de configuración `OPS-07`: V24;
- registro general de tools/resources/skills y su SDK: V26;
- plugins, Forge, deploy, notificaciones u operación final: V28, V29 y V32;
- ningún lifecycle, scheduler, cola, provider, TestAttestor o writer paralelo.

`ProcessNext`, `prepare_workspace`, `commit_change`, `attest_test`, integración
automática, terminalidad y recovery interno no se convierten en comandos
públicos. Son acciones del único scheduler/application writer. Un binding solo
invoca un caso de uso admitido.

## 2. Fuentes leídas y autoridad

Se leyeron completos:

1. `AGENTS.md` del worktree e `internal/AGENTS.md`;
2. `docs/reconstruccion/ruta_total_100.md`;
3. `docs/reconstruccion/estado_y_handoff_rebuild.md`;
4. `docs/reconstruccion/plan_agentes_independientes_v17_v34.md` del árbol de
   integración;
5. `docs/reconstruccion/analisis_y_contrato_v17_artefactos_atestador.md` del
   árbol de integración;
6. `product/roadmap.json`: `AC-V20-COMMAND-REGISTRY`, `GOV-17`, `UI-02` y la
   vertical `command_registry`;
7. configuración V07, identidad/RBAC V10, aplicación y superficies públicas
   actuales del corte base;
8. durante el preflight histórico, cambios V17 todavía no sellados,
   exclusivamente en lectura; el producto integrado consume después los
   contratos V17/V18/V19 acreditados, sin tipos provisionales.

La fila canónica V20 exige: una definición produce HTTP, MCP, CLI y SDK
pequeño; auth, schema, i18n y códigos son idénticos; ninguna interfaz escribe
lifecycle. Cualquier detalle de este documento subordinado que contradiga el
roadmap debe cambiar antes de producto, no mediante compatibilidad paralela.

## 3. Inventario V07: configuración canónica

### 3.1 Autoridad existente

V07 ya aporta una autoridad suficiente:

- `config/registry.json`, schema 2 y revisión `2026-07-21.16`;
- `Snapshot` inmutable con getters generados, hash de registro y hash efectivo;
- precedencia única `default -> file -> env` en el ingress canónico;
- `Manager` sobre `DocumentStore`, CAS de revisión, confirmación,
  idempotencia, receipt y `pending_restart`;
- effective config JSON redactado, generado y nunca usado como input;
- `Doctor` read-only `reuse|replace|new`;
- errores máquina `config_*`, sin texto humano en el paquete;
- adapter TOML con replace/fsync/journal/receipt y recovery propios.

Claves ya relevantes para V20:

| Clave | Uso V20 |
|---|---|
| `server.listen` | listener HTTP existente, loopback |
| `server.mcp_path` | montaje MCP existente |
| `server.max_request_bytes` | límite común de payload |
| `server.read_timeout` / `write_timeout` / `idle_timeout` | servidor común |
| `server.shutdown_timeout` | parada del runtime |
| `api.max_list_limit` | límite de queries generadas |
| `api.locale` | locale transitorio V20; V21 amplía semántica |
| `identity.provider` y claves local/OIDC | autenticación request-bound |
| `project.default` | bootstrap local, no inferencia en payload público |

### 3.2 Decisión V20 sobre configuración

Preflight y producto V20 no crean claves de configuración. El endpoint HTTP
de invocación, nombres MCP y jerarquía CLI son parte del contrato público
versionado del registro de comandos, no un default operativo mutable. Se fijan
en la definición canónica:

```text
HTTP: POST /api/v1/commands/<command-id>
MCP:  <command-id>
CLI:  orquesta command <connection/auth/envelope flags> -- <segments>
```

El listener, path MCP, límites, locale y shutdown reutilizan getters V07.
Si aparece una necesidad de despliegue no representada, el
frente debe pedir `L-CONFIG`, registrar primero la semántica y regenerar. No se
admite constante ambiental, alias, default o `os.Getenv` en V20.

El runtime no lee `internal/commands/registry.json` desde disco. Ese JSON es
input de build/codegen; producción usa exclusivamente la proyección Go
generada, inmutable y ligada a su digest. Así no aparece una ruta operativa
nueva ni una segunda configuración mutable. `commandgen --check` compara la
proyección compilada con la fuente durante build/test.

El CLI fino no inventa `server_url`, token ni fichero de credencial con
defaults/env propios. Hasta que V24 posea su configuración de usuario, esos
datos de conexión son argumentos explícitos del constructor SDK/ejecución CLI;
si se decide persistirlos, antes debe reservarse `L-CONFIG` y declararse la
clave en el registro V07.

La mutación de configuración por HTTP/MCP/CLI/web sigue asignada expresamente
a V24. V20 no expone `Manager.Update` ni `Doctor` como comandos públicos para
adelantar `OPS-07` o V32.

## 4. Inventario V10: identidad, RBAC y auditoría

### 4.1 Modelo existente

V10 separa correctamente:

- `PrincipalRef`: identidad autenticada y autorizable;
- `ActorRef`: procedencia histórica, sin concesión implícita de permisos;
- `ProjectRef`: scope explícito de cada petición;
- principal humano o service con método de autenticación estable;
- siete roles: `platform_admin`, `project_owner`, `project_admin`,
  `contributor`, `reviewer`, `operator`, `viewer`;
- default deny y doce permisos actuales;
- membresía por proyecto, revisión CAS, revocación y audit receipt;
- `Access` application-side que liga principal autenticado y proyecto;
- `AccessRepository.Authorize` como autoridad de permisos, causalmente
  idempotente y persistida por la misma composición SQLite;
- ocultación de existencia cross-project: `goals.get` y `artifacts.read`
  pueden devolver `not_found` ante ausencia de membership.

Permisos consumidos por los comandos V20:

```text
project.membership.manage
goals.create
goals.amend
goals.get
goals.list
goals.direct
effects.approve
changes.integrate
artifacts.read
project.status
```

`project.hierarchy.manage` y `budgets.manage` siguen sin caso de uso público
V20. Que existan en la política no autoriza inventar un handler.

### 4.2 Regla de autenticación/autorización V20

Las operaciones ligadas solo a principal siguen esta única cadena:

```text
credential transport adapter
  -> Principal request-bound
  -> commands.Dispatcher
  -> commands application handler
  -> application.Access
  -> caso de uso existente
  -> AccessRepository.Authorize
  -> mismo writer/StateRepository
```

El payload nunca contiene principal, actor autenticado, roles ni receipt de
autorización. `project_ref` viaja en el envelope común y se valida antes de
seleccionar recursos.

Para mailbox no se presupone que una `ExecutionRef` aportada por el cliente sea
auténtica. El envelope puede transportar `claimed_execution_ref`, siempre fuera
del payload, pero un resolver de autoridad inyectado en composición debe ligarla
al principal autenticado, proyecto y Execution exactos antes de que el
dispatcher construya `NewExecutionAccess`.
Un binding que solo tenga Bearer principal y no pueda demostrar esa relación
debe rechazar `forbidden`; no puede activar mailbox con un cast o contexto
falsificable. V20 congela ese seam contra los contratos sellados existentes y
falla cerrado cuando la composición no inyecta resolver; no introduce un store
de identidad paralelo. La autoridad durable productiva queda diferida a V22.

El permiso del descriptor es metadata ejecutable y un ratchet de paridad; no
sustituye la autorización del caso de uso. Acceptance debe demostrar que cada
handler usa el caso de uso que aplica ese mismo permiso. Un descriptor con
`goals.get` conectado a `IntegrateChange`, por ejemplo, es error de generación.

### 4.3 Auditoría durable de comandos

Los recibos V10, decisiones V12, controles V14, efectos V15, integración V16,
atestaciones V17 y eventos de cierre ya son hechos durables. V20 añade la capa
uniforme que registra qué comando público se admitió y qué resultado máquina
produjo sin convertir la interfaz en writer de Goal.

V20 añade dos hechos append-only detrás de un puerto consumido por
`commands.Dispatcher` e implementado por la misma instancia SQLite activa:
`command_invocations` para admisión y `command_outcomes` para el resultado
terminal. El `CommandAuditRecord` es la proyección de ambos, no una fila
`admitted` actualizada in-place. No crea DB, store físico, lifecycle, scheduler
ni cola.

Para acreditar la amplitud de `GOV-17`, una proyección read-only parte de cinco
raíces públicas: `director.plan.propose`, `effects.decide`, `goals.control`,
`council.round.open` y `council.skip`. Para cada una enlaza
`invocation/outcome -> authorization receipt -> fact raíz`. Desde la raíz
recorre, cuando existen, `effect intent -> attempt -> receipt -> terminal
event/attestation`. Los descendientes son opcionales mientras el Goal sigue
vivo; si existen, su integridad causal es obligatoria, y el cierre terminal
debe presentar la cadena completa aplicable. La proyección no copia esos facts
a `command_outcomes`, no les cambia owner y no escribe lifecycle.

Identidad canónica de una invocación:

```text
registry_digest
+ command_id/version
+ schema_digest
+ request_ref
+ input_digest
+ principal_ref
+ project_ref
+ optional authenticated_execution_ref
```

El outcome conserva también `output_digest`, calculado sobre el envelope
canónico devuelto, para demostrar qué respuesta se produjo sin guardar payload
o artifact bytes. El registro conserva digest, scope, estado, código máquina y tiempos. No
conserva secreto, credential, token de Director, payload bruto, artifact bytes,
argv, environment, path físico ni texto traducido.

Estados mínimos:

```text
admitted -> completed | rejected | failed
```

No son lifecycle de Goal. Una invocation sin outcome tras crash es un hecho
recuperable, no un PASS. La aplicación subyacente conserva su propia
idempotencia por `request_ref`; comandos mutantes reobtienen su receipt y
completan el outcome sin aplicar dos veces. Queries son read-only y pueden
reevaluarse; el nuevo output se compara/declara explícitamente y nunca se finge
que el audit guardó bytes que no persiste. Una petición no autenticada se corta
antes del dispatcher y no inventa un `PrincipalRef` ni un command audit; su
telemetría de seguridad pertenece a la frontera de autenticación/V32.

## 5. Inventario histórico de interfaces en la base V19

Esta sección caracteriza el punto de partida anterior a V20. No describe la
composición post-V20, que se define en las secciones 7 y 10.

### 5.1 MCP en la base V19

`internal/interfaces/mcp` exponía seis tools manuscritas:

```text
orquesta.goals.create
orquesta.goals.amend
orquesta.goals.get
orquesta.goals.list
orquesta.artifacts.read
orquesta.system.status
```

La implementación de la base V19:

- declara DTOs privados `CreateGoalInput`, `GoalView`, etc.;
- registra cada tool manualmente;
- traduce manualmente DTO -> aplicación y aplicación -> DTO;
- contiene su propio mapa `publicCode` con cinco errores;
- consulta el catálogo directamente para descripciones y mensajes;
- importa `application`, `goal`, `identity` y `ports`;
- llama al `Orchestrator` desde el adapter;
- usa el SDK MCP oficial sobre Streamable HTTP;
- recibe autenticación Bearer mediante middleware del bootstrap.

La conducta se caracterizó antes del reemplazo. V20 retiró registro, handlers y
clasificador de errores MCP como autoridades: la composición registra
exclusivamente las 25 tools derivadas del registro común. `tools.go` conserva
solo nombres y DTO V04 pasivos exigidos por contratos históricos; no registra
tools, no despacha y no decide schema, errores ni lifecycle.

### 5.2 HTTP en la base V19

El runtime base montaba únicamente el handler MCP en `server.mcp_path`; no
existía API HTTP de comandos independiente. Conservaba estas protecciones:

- listener loopback;
- `CrossOriginProtection` y protección localhost del SDK MCP;
- límite estricto de body;
- timeouts y shutdown del `http.Server`;
- Bearer auth local token u OIDC request-bound.

V20 monta el binding HTTP derivado del registro en el mismo mux y servidor; no
crea otro daemon, puerto, listener o composición.

### 5.3 CLI en la base V19

`cmd/orquesta` solo reconoce:

```text
orquesta serve [--config path]
orquesta version
```

No existía CLI de casos de uso. V20 conserva `serve` y `version`, incorpora el
subcomando fino `command` y resuelve su path contra
`internal/interfaces/cli`, derivado de las definiciones compiladas. El CLI usa
el SDK HTTP pequeño; no abre SQLite, CAS, Orchestrator ni lifecycle local.

### 5.4 SDK en la base V19 y resultado V20

No existía SDK público de comandos. V20 crea un paquete pequeño
`sdk/commands` con:

- `Client.Invoke(ctx, Request) (Result, error)`;
- espejo público deliberado de `Request`, `Result`, `Failure` y códigos
  máquina, sin importar paquetes `internal`;
- una tabla local de validación `failure -> HTTP status`, paralela a la tabla
  del adapter HTTP y ratcheted por pruebas de paridad;
- `http.Client` inyectable;
- cero reglas de lifecycle, RBAC, provider o retry oculto.

El SDK general de tools/resources/skills sigue en V26.

## 6. Dependencia sellada: V19

V20 parte exclusivamente del E acreditado de V19
`42692512b0048f116d660a53fbe6650e243bb4e3`. El gate valida el receipt
`AC-V19-COUNCIL`, su fuente detached-clean y la cadena P/S/E. El prototipo V20
anterior sirve solo de referencia; no es ancestro ni producto integrable.

Al congelar el registro se revalidaron los casos de uso V19. En particular:

1. `goals.create` y `director.plan.propose` transportan `CouncilPolicy`;
2. `orquesta.council.round.open` proyecta `OpenCouncilRound`;
3. `orquesta.council.skip` proyecta `SkipCouncil` y exige `council.skip`;
4. la migración de auditoría usa el ordinal posterior a `014_council.sql`;
5. los errores del Consejo conservan códigos máquina estables;
6. el MCP derivado del registro sustituye la autoridad manuscrita; los símbolos
   V04 pasivos no forman una segunda autoridad ni registran tools.

## 7. Arquitectura V20

### 7.1 Flujo único

```text
internal/commands/registry.json
       |
       v
commandgen --check/--write
       |
       +-> definitions_generated.go (definitions + schemas + binding metadata)
       |
       +-> HTTP/MCP/CLI adapters derive bindings from CanonicalDefinitions
       +-> SDK receives the same public envelope over HTTP
       |
transport auth -> Dispatcher -> application handler -> Orchestrator use case
                         |
                         +-> CommandAuditRepository (misma SQLite, dos facts)
```

El registro JSON es la única fuente declarativa. `commandgen` hace decode y
validación estrictos y emite solo `definitions_generated.go`. El runtime
revalida defensivamente la proyección compilada con un validador separado.
Dos mapas de permiso por handler —generador y runtime— son duplicación
consciente; acceptance exige igualdad exacta entre registro, generado,
handlers y bindings para impedir drift.

### 7.2 Definición canónica

Cada entrada contiene exactamente:

```text
id, version, kind, handler, permission, audience, execution_bound, replay_mode,
input_schema, output_schema, description_key, error_codes,
http{method,path}, mcp{tool}, cli{path}
```

Propiedades:

- ID estable más versión explícita; V20 inicial usa `version="1"` y todo
  envelope debe coincidir con ella;
- IDs y nombres son únicos en todas las superficies;
- schema JSON estricto: unknown fields rechazados, límites explícitos, refs
  locales acíclicas y números con rango;
- handler es una clave 1:1 a función application-side, no nombre reflejado en
  runtime;
- `replay_mode` es `application_receipt` para comandos mutantes y
  `read_reexecute` para queries; no promete almacenar una respuesta ausente;
- permission debe existir en `identity` y coincidir con el caso de uso;
- `description_key` y errores son claves, nunca texto;
- binding metadata se valida contra colisiones y formato;
- orden de archivo es canónico y se conserva en la salida generada;
- todos los digests usan bytes canónicos, sin reloj/path/environment.

### 7.3 Dispatcher y handlers application-side

`commands.Dispatcher` recibe un `Invocation` común:

```text
CommandID
CommandVersion
RequestRef
ProjectRef
ClaimedExecutionRef optional
Payload JSON
```

Principal procede del contexto autenticado. `ClaimedExecutionRef`, cuando
existe, no tiene autoridad hasta pasar por el resolver inyectado. El
dispatcher:

1. resuelve una definición exacta;
2. liga principal/proyecto y, si aplica, resuelve la claim de Execution fuera
   del payload;
3. valida schema estricto y calcula digests;
4. persiste admisión auditiva;
5. llama a un handler 1:1 en `internal/commands/application_handlers.go`;
6. el handler construye tipos de aplicación y llama al caso de uso;
7. mapea error una vez a `Failure{Code, MessageKey}`;
8. persiste un outcome inmutable completion/rejection/failure con output digest;
9. devuelve un `Result` neutral.

Las interfaces dependen solo de la API del dispatcher. Quedan prohibidos en
HTTP/MCP/CLI/SDK:

- importar `application` o `goal`;
- llamar `Orchestrator`, `StateRepository` o SQLite;
- construir/transicionar Goal/WorkItem/Execution;
- mapear errores propios;
- definir DTOs de negocio privados;
- decidir permiso, idempotencia, cierre o retry.

### 7.4 Schemas y DTO

`input_schema` y `output_schema` describen el payload/data de una operación, no
repiten el envelope. Un schema fijo y compartido compone
`command_id/version/request_ref/project_ref/claimed_execution_ref/payload` y
`data|failure/audit_ref` de igual forma en cada binding.

El schema en el registro es la autoridad de red. El generador emite
descriptores `Definition`; los handlers y adapters mantienen structs Go
manuales necesarios para sus fronteras, pero ninguno decide el schema público.
Acceptance compara schema, envelope y resultado de HTTP/MCP/CLI/SDK con la
misma definición compilada.

El envelope común contiene `request_ref`, `project_ref`, `payload` y solo para
audience execution una `claimed_execution_ref`. Principal, actor autenticado,
rol, tiempo del servidor, receipts, tokens y la ExecutionRef ya verificada no
son campos falsificables. Listas no-nil y timestamps UTC se normalizan en la
proyección común antes del binding.

Para resultados grandes se devuelven refs; el límite actual de artifact read
se conserva. V20 no transforma el SDK en transporte de corpus.

### 7.5 Errores e i18n

Códigos V20, invariantes por locale y transporte:

```text
invalid_request
unauthenticated
forbidden
not_found
conflict
unavailable
internal
```

Cada código tiene `MessageKey = error.<code>`. V20 transporta código+key; no
renderiza mensajes de error por locale. El catálogo compartido solo aporta las
keys y las instrucciones MCP ya existentes. HTTP usa status
400/401/403/404/409/503/500; MCP structured result, CLI JSON y SDK devuelven el
mismo code/key/data. El status HTTP o exit code CLI no sustituyen el código
máquina.

V20 añade solo las claves de descripción de sus comandos y
`error.unauthenticated`/`error.unavailable` a los catálogos `es/en` actuales.
El registry contiene claves, nunca traducciones. V21 será owner de todos los
locales, fallback, plurales y formatters. V20 no añade selección de locale,
fallback ni formato propios y no afirma i18n total.

### 7.6 Bindings

HTTP:

- un handler genérico que deriva cada definición bajo
  `POST /api/v1/commands/<command-id>`;
- mismo body/envelope/schema;
- mismo Bearer adapter y límites V07;
- comandos execution-bound exigen resolver inyectado; un header o campo
  de cliente por sí solo nunca autentica Execution;
- JSON error/success canónico;
- ningún path param reinterpreta refs de payload.

MCP:

- una tool por definición, nombre igual a command ID;
- input schema exacto del registro;
- annotations derivadas solo de `kind`;
- wrapper registry-derived delega al dispatcher;
- no queda registro, `publicCode` ni handler MCP privado; `CreateGoalInput` y
  otros DTO V04 pasivos permanecen solo para contratos históricos.

CLI:

- path derivado de `Definition.CLI.Path`, por segmentos posteriores a
  `orquesta.`;
- payload JSON explícito en `--payload`, sin parser de negocio alternativo;
- endpoint, credential file y límites son argumentos explícitos, sin
  env/default nuevo fuera de V07;
- llama al SDK HTTP, no a la DB;
- salida canónica a stdout; diagnóstico no sensible a stderr;
- exit `0` para success, `1` para failure remoto/operativo y `2` para uso o
  credencial inválidos; el código máquina permanece en el envelope.

SDK:

- `Invoke` genérico sobre command ID y versión explícitos;
- cliente HTTP sustituible y endpoint/credential provider inyectados;
- no reintenta comandos mutantes salvo petición explícita del caller con el
  mismo `request_ref`;
- no guarda token, lifecycle, cache ni estado autoritativo.

## 8. Catálogo integrado

El catálogo integrado contiene 25 operaciones: las 23 caracterizadas por el
prototipo más las dos acciones públicas del Consejo V19. El registro se congela
contra casos de uso application-side reales y schemas cerrados; no contra DTO
de transporte ni reflexión de structs.

| Command ID | Clase | Caso de uso | Permiso | Audience |
|---|---|---|---|---|
| `orquesta.goals.create` | command | `Submit` | `goals.create` | principal |
| `orquesta.goals.amend` | command | `Amend` | `goals.amend` | principal |
| `orquesta.goals.get` | query | `GetGoal` | `goals.get` | principal |
| `orquesta.goals.list` | query | `ListGoals` | `goals.list` | principal |
| `orquesta.artifacts.read` | query | `GetArtifact` | `artifacts.read` | principal |
| `orquesta.system.status` | query | `Status` | `project.status` | principal |
| `orquesta.projects.memberships.grant` | command | `GrantMembership` | `project.membership.manage` | principal |
| `orquesta.projects.memberships.revoke` | command | `RevokeMembership` | `project.membership.manage` | principal |
| `orquesta.director.claim` | command | `ClaimDirector` | `goals.direct` | principal |
| `orquesta.director.renew` | command | `RenewDirector` | `goals.direct` | principal |
| `orquesta.director.plan.propose` | command | `ProposeDirectorPlan` | `goals.direct` | principal |
| `orquesta.council.round.open` | command | `OpenCouncilRound` | `goals.direct` | principal |
| `orquesta.council.skip` | command | `SkipCouncil` | `council.skip` | principal |
| `orquesta.goals.control` | command | `Control` | `goals.direct` | principal |
| `orquesta.effects.decide` | command | `DecideEffect` | `effects.approve` | principal |
| `orquesta.changes.list` | query | `ListPendingChanges` | `goals.list` | principal |
| `orquesta.changes.integrate` | command | `IntegrateChange` | `changes.integrate` | principal |
| `orquesta.mailbox.admit` | command | `AdmitMailbox` | `goals.direct` | execution |
| `orquesta.mailbox.claim` | command | `ClaimMailbox` | `goals.get` | execution |
| `orquesta.mailbox.mark_delivered` | command | `MarkMailboxDelivered` | `goals.get` | execution |
| `orquesta.mailbox.consume` | command | `ConsumeMailbox` | `goals.get` | execution |
| `orquesta.mailbox.get` | query | `GetMailbox` | `goals.get` | execution |
| `orquesta.mailbox.list` | query | `ListMailbox` | `goals.get` | execution |
| `orquesta.mailbox.acknowledge` | command | `AcknowledgeMailbox` | `goals.get` | execution |
| `orquesta.mailbox.block` | command | `BlockMailbox` | `goals.get` | execution |

No se añaden comandos vacíos para permisos sin caso de uso. Los adapters
internos, incluido cualquier atestador sellado, no son comandos públicos: solo
los casos de uso application-side caracterizados pueden entrar en el catálogo.

## 9. Generación determinista y recovery

### 9.1 Generación

`commandgen` ofrece `--check` y `--write`. En ambos modos:

- lee solo la ruta explícita del registro;
- valida JSON estricto y EOF;
- canonicaliza schemas y calcula SHA-256 del source semántico;
- ordena únicamente donde el contrato declara conjunto; conserva orden de
  comandos y propiedades donde sea significativo;
- emite únicamente `internal/commands/definitions_generated.go`, con header
  `Code generated` y `RegistrySourceSHA256`;
- no incluye timestamp, host, cwd, path absoluto, Go cache o env;
- escribe por temp privado + rename solo en `--write`;
- segunda ejecución produce bytes idénticos;
- `--check` falla ante drift sin mutar.

El gate combinado `commandgen --check` + acceptance deja rojas estas mutaciones:

1. duplicar command ID, handler, HTTP path, MCP tool o CLI path;
2. permiso desconocido o distinto del handler;
3. i18n key inexistente;
4. error code desconocido o distinto por binding;
5. schema no estricto, ambiguo, recursivo o inválido;
6. quitar un binding o cambiar su schema;
7. editar generado a mano;
8. incluir reloj/path/environment en output.

### 9.2 Migración SQLite

El ordinal quedó reservado como `015_command_registry.sql`. La migración crea
dos tablas append-only y sus índices de replay/scope. Campos mínimos:

```text
command_invocations:
ref, command_id, command_version, registry_digest, schema_digest,
request_ref, input_digest, principal_ref, project_ref,
authenticated_execution_ref not null (vacío cuando no aplica), admitted_at

command_outcomes:
ref, invocation_ref, status, error_code not null (vacío en completed),
output_digest, completed_at
```

Constraints bloquean:

- outcome sin invocation o sin `completed_at`;
- `completed` con error;
- `rejected|failed` sin código;
- scope, digest o ref vacíos;
- dos identidades semánticas distintas para la misma clave idempotente;
- update/delete de invocation u outcome;
- replay con identidad principal/proyecto/Execution distinta.

No se persiste JSON de schema ni payload completo por fila: se liga su digest
al registry/source versionado. La DB preserva identidad e inmutabilidad; la
existencia y pertenencia de una Execution se valida en el resolver antes de la
admisión, no mediante una foreign key inventada por la tabla de auditoría.

### 9.3 Crash/restart/replay

Fronteras exigidas:

```text
before_admission
after_admission_before_handler
after_handler_before_completion
after_completion_before_response
```

Resultados:

- antes de admisión: retry crea una única invocación;
- tras admisión: replay exacto continúa; scope/digest cambiado es conflict;
- tras handler mutante: el mismo request ref obtiene replay/receipt del caso de
  uso y completa audit sin segundo efecto;
- tras handler query: puede reevaluarse porque no produce efecto; el outcome
  liga el output realmente devuelto, sin prometer bytes históricos ausentes;
- tras outcome antes de response: un comando reconstruye por su receipt; una
  query puede reevaluarse y solo devuelve replay si el output digest coincide;
  si el estado cambió devuelve `conflict` y exige un nuevo `request_ref`;
- un error/denegación nunca se reinterpreta como success;
- queries pueden reevaluar datos, pero sus facts históricos no mutan;
- restart conserva registry/schema/input/output digest histórico;
- schema/version no soportado falla cerrado, no migra semántica en memoria.

Recovery V20 valida todas las filas y enlaces antes de abrir. Tamper de status,
digest, principal/project, código, tiempos o duplicidad hace fallar recovery.
Backup/restore V09 debe conservarlas; no se crea sidecar.

## 10. Paridad y E2E

### 10.1 Matriz de paridad

Para cada definición, acceptance compara:

```text
command id/version
input/output schema digest
kind/read-only annotation
handler
permission/audience/execution binding
description key
error-code set
HTTP path/method
MCP tool
CLI path
SDK constant
```

Casos conductuales comunes:

- success;
- unknown field, type/range/ref invalid;
- unauthenticated;
- forbidden y aislamiento cross-project;
- not found sin oracle;
- conflict CAS/idempotencia;
- unavailable;
- internal sin fuga;
- locale distinto con código/key invariantes;
- mismo request replay;
- mismo request con payload/principal/project/command distinto.

### 10.2 E2E real V20 y frontera V22

Composición aislada:

- `cmd/orquesta` real;
- listener loopback y token local;
- SQLite y CAS reales;
- un principal humano local autenticado;
- `system.status` con la misma semántica por HTTP, MCP oficial, CLI como
  subproceso real y SDK;
- `goals.create` y replay/restart por HTTP, con cierre del Goal y cero segundo
  launch;
- resolver execution-bound neutral compartido por HTTP/MCP, probado con fake
  acotado; ausencia de resolver en composición rechaza siempre `forbidden`;
- restart entre admisión/consulta y recuperación de audit;
- shutdown por el runtime ya acreditado y terminación del subproceso CLI; V20
  no añade daemon, scheduler, cola ni proceso residente.

La composición V20 no dispone todavía de un binding durable entre un service
principal y un `ExecutionRef`: ejecuciones y principales se persisten por
separado, y el secreto del proveedor Codex no es identidad Orquesta. Por ello
los ocho comandos mailbox se registran y generan en V20, pero permanecen
fail-closed en la composición por defecto. V22 debe crear/injectar esa autoridad
desde el adapter Codex y acreditar principal + proyecto + ejecución exacta,
incluidos sucesor, revocación y restart. Devolver la claim sin comprobarla está
prohibido.

El gate real V20 recorre:

1. create/replay/restart con payload V19 compatible;
2. status por los cuatro bindings, incluido CLI subproceso;
3. rechazo execution-bound sin resolver y uso compartido del seam HTTP/MCP;
4. verificación de audit digests/resultados y cero doble lifecycle;
5. prueba estructural de que interfaces no escriben lifecycle.

Get/list, Director, control, effect, artifact, memberships, aislamiento y la
máquina mailbox completa conservan sus tests application/SQLite específicos y
la paridad registry→binding de V20. No se presentan falsamente como un único
E2E público. El gate Codex V22 es quien debe encadenarlos por bindings reales
con service principals y fixtures causales.

No se atribuye cierre Codex V22. Un agente fake/fixture puede producir el caso
de aplicación; la disponibilidad Codex completa sigue en su vertical.

## 11. Contrato integrado y gate

Archivos:

```text
acceptance/fixtures/v20_command_registry.json
acceptance/v20_command_registry_test.go
```

El test valida:

- identidad/base/capabilities y receipt V19 acreditado;
- 25 comandos del catálogo y presupuesto;
- schemas V19, incluida política y acciones del Consejo;
- registry compilado, cero clave de configuración nueva y alcance i18n solo de
  keys `es/en`;
- AC-V20 aún `planned` o, después del gate, `executable` con rutas exactas;
- `GOV-17`/`UI-02` solo `declared|accredited`, nunca estado parcial.

El rojo inicial integrado, antes de portar el producto, era exactamente:

```text
V20_RED integrated command registry product is absent on the accredited V19 base
```

Con producto presente, activa gates de definición, handlers, generados,
lifecycle, migración, recovery y pruebas conductuales. Cada nombre declarado
en `required_behavior_tests` debe corresponder a una función de test real; una
lista aspiracional ya no puede producir verde.

El roadmap permanece `planned` durante la implementación y cambia solo en E,
después de ejecutar la fuente S detached-clean:

```text
status: executable
test_ref: acceptance/v20_command_registry_test.go
fixture: acceptance/fixtures/v20_command_registry.json
receipt: product/evidence/v20_command_registry.json
command: gate V20 exacto, sin ./...
```

Las capabilities permanecen `declared` hasta receipt P/S/E.

## 12. Presupuesto de simplicidad

| Superficie | Máximo |
|---|---:|
| producto manual total | 6.400 LOC |
| registry/dispatcher/handlers application-side | 2.600 LOC |
| adapters HTTP/MCP/CLI | 2.000 LOC |
| SDK | 500 LOC |
| SQLite/recovery/bootstrap | 1.300 LOC |
| salida generada, medida aparte | 3.000 LOC |
| migración SQL | 500 LOC |
| command definitions | 25 |
| command registries | 1 |
| lifecycle writers | 1 |

Tests no justifican duplicar producción. Un fichero manual sobre unas 350
líneas debe dividirse por responsabilidad. Los generados no contienen reglas y
su tamaño cuenta; superar un límite exige ADR y retirada compensatoria.

Medición física pre-P tras cerrar la amplitud GOV-17:

```text
manual=4526
commands/application=2590 (owned=2566, integration=24)
bindings=572
SDK=254
SQLite/recovery/bootstrap=979 (owned=950, integration=29)
contract_data=131
generated=33
SQL=62
```

Métricas de cierre:

```text
registries=1
runtime handler dispatch maps=1
handler-permission validation maps=2 (commandgen/runtime, paridad ratcheted)
public commands=25
public wire DTO mirrors=3 (HTTP/MCP/SDK; schema authority=registry)
legacy V04 passive DTO sources=1
business error classifiers=1 (commands)
HTTP status maps=2 (HTTP adapter/SDK response validator)
interface lifecycle writers=0
state repositories active=1
new goroutines/schedulers/queues=0
```

Estas duplicaciones de frontera son explícitas y pequeñas. No son fuentes de
schema, permisos ni lifecycle. Su paridad se prueba; convertirlas en nuevas
autoridades o añadir otra copia exige rediseño.

## 13. Write-set real y secuencia de cierre

Contrato/documentación/acceptance:

```text
docs/reconstruccion/analisis_y_contrato_v20_*.md
docs/reconstruccion/worksets/v20*.json
acceptance/v20_*.go
acceptance/fixtures/v20_*.json
```

Producto V20 propio:

```text
internal/commands/**
internal/ports/command_audit*.go
internal/interfaces/httpapi/**
internal/interfaces/cli/**
sdk/commands/**
internal/adapters/state/sqlite/command_audit*.go
internal/adapters/state/sqlite/migrations/*_command_registry.sql
internal/adapters/state/sqlite/recovery_validation_v20.go
internal/bootstrap/command_*.go
cmd/orquesta/command.go
```

Integración sobre hosts acreditados:

- `internal/application/access.go`, `effect_decision.go`, `effect_model.go` y
  `planning.go`: errores estables consumidos por el clasificador de comandos;
- `internal/interfaces/mcp/**`: `L-BINDINGS`;
- `cmd/orquesta/main.go` y `main_test.go`: `L-BINDINGS`;
- `internal/bootstrap/runtime.go`: `L-BOOTSTRAP`;
- tests E2E MCP/bootstrap existentes: adaptación al envelope V20;
- `internal/adapters/state/sqlite/errors.go`, `recovery_validation.go`,
  `recovery_validation_versions.go` y tests de versión: `L-STATE`;
- la migración `015_command_registry.sql` entra por el loader `embed` existente;
  `migrations.go` no necesitó modificación;
- catálogos `es/en`: `L-BINDINGS`;
- inventario de bugs, trazabilidad, receipt Codex genérico, write-set checker,
  roadmap y gates focales V20: `L-TRACE`, integrador;
- P/S/E: `L-SEAL`.

No se toca `config/**`, no se añade variable y no se introduce provider,
scheduler, cola, DB o runtime paralelo. `product/evidence/**` solo cambia por
refresco causal del receipt Codex genérico y, en E, por evidencia V20.

Implementación, focales, race, recovery, E2E y contrarrevisión están completos.
Secuencia restante:

1. P: congelar subjects exactos y commit de producto sin acreditación V20;
2. S: ligar ese P en fixture desde fuente detached-clean;
3. E: ejecutar gate exacto, publicar receipt/output V3 y promover solo
   `AC-V20`, `GOV-17` y `UI-02`;
4. handoff sellado a V21.

## 14. Riesgos y decisiones abiertas acotadas

1. **API V17 provisional durante el preflight.** Resuelto: V20 parte de
   V17/V18/V19 acreditados y no consume tipos provisionales.
2. **Ordinal SQLite.** Resuelto como `015_command_registry.sql` sobre el loader
   `embed` existente.
3. **Auditoría y atomicidad.** El audit son facts separados de invocation y
   outcome, no writer de Goal ni una fila mutable. La mutación de negocio
   conserva su transacción; replay causal por receipt cierra la ventana entre
   handler y outcome. Queries comparan output digest y no inventan cache.
4. **MCP V19 tenía semántica útil.** Se caracterizó y migró. Solo quedan DTO
   V04 pasivos; no existe fallback ni segundo registro runtime.
5. **i18n total es V21.** V20 exige key/code parity solo para sus superficies
   y los dos catálogos ya existentes.
6. **CLI auth.** El CLI usa SDK HTTP y credential explícita; no abre acceso
   local privilegiado ni lee env ad hoc.
7. **Mailbox execution-bound.** El cliente solo reclama una Execution fuera
   del payload. El resolver inyectado debe probar principal, proyecto y
   Execution exactos; sin resolver, V20 falla cerrado. La autoridad durable de
   service principals, sucesión, revocación y restart pertenece a V22.
8. **GOV-17 breadth.** Cinco raíces públicas enlazan autorización y fact raíz;
   la proyección añade descendientes de efecto y terminales existentes. No se
   duplican en otra tabla autoritativa.

## 15. Estado de la integración

```text
hecho: implementación integrada; 25 comandos; auditoría SQLite; cinco raíces GOV y descendientes; cuatro superficies; restart/race/E2E
estado: integrated_implementation_complete_pending_seal
invariante: Orchestrator conserva lifecycle; transportes solo proyectan dispatcher
autoridad final: roadmap sigue planned hasta receipt P/S/E V20
tests: focales, paridad HTTP/MCP/CLI/SDK, restart, race, recovery y E2E reales verdes antes de P
legacy: DTO V04 pasivos retenidos; cero registro/handler/error authority MCP legacy
bloqueo: pending_PSE_only
siguiente: P/S/E V20; después V21 i18n total
```
