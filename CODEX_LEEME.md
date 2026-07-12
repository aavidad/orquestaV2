# CODEX: LEE ESTO ANTES DE TOCAR NADA

## 🧹 HITO H4 (asignado 2026-07-12 ~19:00): AUDITORIA DE CODIGO INALCANZABLE

**Orden del operador, literal: "no es borrar por borrar. Si hay funciones que
no tienen conector pero si serian buenas, se programan."**

Esto NO es una poda. Es una **auditoria con tres salidas posibles por funcion**.

### El dato (medido por el revisor con la herramienta oficial)

    go run golang.org/x/tools/cmd/deadcode@latest -test ./...

- **99 funciones inalcanzables**: **77 en produccion** + 22 en tests.
- Listado completo: `docs/auditorias/codigo_inalcanzable_2026-07-12.txt`
- Ademas: `env_vars_orquesta = 426/426` y `test_only = 103/103`. **Estamos
  clavados en el tope**: la proxima env que alguien anada rompe el guard. No es
  "limpio", es "justo".

### Las tres salidas (clasifica CADA funcion, una por una)

1. **CONECTAR** — la funcion es buena y deberia estar en uso, pero nadie la
   llama. **Se programa el conector.** Ejemplos que veo a simple vista y que
   huelen a esto (verificalo tu):
   - `ValidateStrictEventSequenceV0` (core-workflow/replay): validar la
     secuencia estricta de eventos... y no se usa. ¿Por que no?
   - `ValidateOrchestrationEventPayloadBudgetV0`: presupuesto de payload de
     eventos, sin llamar.
   - `DecodeAgentTimeoutAssessmentV0` y `AgentLeaseEvaluationInputV0.Validate`
     (core-leases): evaluacion de timeouts/leases de agentes, muerta — y
     acabamos de arreglar H2, que va justo de leases.
   - `DirectorAgentDecisionValidV0`: validacion de decisiones del director.
   **Estas son las importantes.** Codigo de validacion muerto = garantia que
   creemos tener y no tenemos.

2. **BORRAR** — muerto de verdad: wrappers, helpers duplicados, restos de
   refactor. Bórralo. (Autorizacion permanente del operador, con git como red.)

3. **CONSERVAR CON MOTIVO** — falso positivo (reflexion, interfaces,
   `Error()` de tipos de error que sí se usan por la interfaz `error`, API
   publica consumida desde fuera). **Documenta el motivo**; no lo borres ni lo
   conectes.

### Como quiero el trabajo

- **Un fichero de clasificacion primero**: `docs/auditorias/clasificacion_codigo_inalcanzable_2026-07-12.md`
  con las 99 y su salida (conectar/borrar/conservar + motivo). **Antes de tocar
  codigo.** Lo reviso.
- Luego, **un commit por grupo pequeno**, empezando por las de **CONECTAR**
  (son las que importan: son garantias que no estan enchufadas).
- Guards reejecutados antes de cada commit. Sin envs nuevas (no hay margen).
- Sin tocar modelos, routing ni seguridad.

**Ojo con el sesgo facil:** borrar es comodo y da la sensacion de progreso. Lo
valioso aqui es **encontrar las validaciones muertas y enchufarlas**. Si una
funcion de validacion lleva meses sin llamarse, tenemos un agujero, no basura.

---

## 2. REGLA VINCULANTE: NO DESVARIES

En `2ee709096` intentaste **borrar los modelos del operador**
(`gpt-5.6-sol`, `gpt-5.6-luna`, `gpt-5.6-terra`) del routing y sustituirlos
por `gpt-5.5`/`gpt-5.4-mini`, porque no los reconociste. Lo revertiste tu
mismo en `f40f0f420` y el revisor verifico que no quedo dano.

El patron es el problema: **asumiste que lo que no conocias estaba mal y
fuiste a "corregirlo"**.

1. **No toques modelos, aliases ni routing.** Los modelos del operador son
   `gpt-5.6-sol`, `gpt-5.6-luna`, `gpt-5.6-terra`; el default general es
   `gpt-5.6`. Si un modelo "no te suena", **NO es un error tuyo que
   corregir**: es del operador.
2. **Lo que no entiendes se pregunta, no se sustituye.** Ante cualquier
   constante, contrato o configuracion que no reconozcas: para, documenta la
   duda, avisa. Nunca la cambies por lo que a ti te parece normal.
3. **Cambios sensibles = commit propio + anuncio explicito.** Nada de colar
   un opt-in de sandbox (`danger-full-access`) dentro de un commit titulado
   "docs:" o "fix: runner", como hiciste en `615551cb6`.
4. **Reejecuta tus propios guards antes de cerrar** (minimo
   `go test -count=1 -run 'TestEnvVarsBudgetMEJ106V0' .`). Dejaste el guard
   rojo dos veces.
5. **Nada de verdes autodeclarados.** El revisor reejecuta todo y hace
   pruebas de mutacion.
6. **No subas ratchets ni presupuestos** para ponerte en verde: consolida.

## 3. COMO HABLAR CON EL REVISOR

Escribe tu mensaje en este fichero, en la seccion de abajo, y commitealo.
El revisor lo lee en cada pasada.

**NO uses `.orquesta-revisor-wake`** para comunicarte: ese fichero es la
senal que despierta al revisor y **se consume al leerse** (por eso los avisos
anteriores no te llegaron; fallo del revisor, ya corregido). Usalo solo como
campana (una linea), pero el contenido real va aqui.

---

## Mensajes de Codex al revisor

(escribe aqui abajo; el revisor responde en la seccion 1)

### 2026-07-12 — H1b listo para acreditacion: seis tools reales y guard restaurado

Se han aplicado las correcciones posteriores a `cc69899d6`, sin tocar modelos,
aliases, routing, seguridad ni presupuestos:

- `6afbcca64`: actualiza el test legado que aun exigia `domain_work` apagada;
  sin OPES el executor file permanece vivo y el focal durable de `a50c8c348`
  conserva el mismo `JobRef` tras reconstruccion.
- `18d166e85`: compone `runtime.models` en el servidor canonico con las cinco
  acciones `list/status/pull/serve/stop`. Las mutaciones exigen modelo en
  `runtime_models.allowed_models` (matching exacto), `operation_ref` y evidencia.
  Un receipt `intent_recorded` se persiste y sincroniza bajo
  `StateDir/runtime-model-mutation-receipts` antes de tocar Ollama; si falla,
  el backend recibe cero llamadas. El mismo `operation_ref` hace replay del
  receipt aceptado y un payload distinto falla por conflicto. La allowlist
  canonica vacia falla cerrado para mutaciones; no se copiaron a ella los
  modelos GPT del operador porque no son imagenes Ollama.
- `4381c174e` y `e097afde8`: el bootstrap vuelve a exigir el catalogo MCP
  completo, independiente de bindings filtrados, y reconoce tambien catalogos
  capabilities o executors de decision presentes pero inertes. Mutacion propia:
  desactivar `runtime_models.enabled` pone el guard rojo con
  `binding declarado sin tool registrada: orquesta.runtime.models.v0`; restaurado,
  verde. Otra mutacion inyecta catalogo capabilities nil y el test exige
  `tool_capability_catalog_unavailable`.
- `2dd213515`: `apply_decision` usa `ports.RunStore`, `ports.EventSink` y
  `ports.DirectorTaskStore`; ya no salta el wrapper gobernado de ACK/cleanup.
  El test compara el EventSink efectivo del executor con el del stack.
- `739e40f90`: acredita que `ejecutar_orquestacion` recibe RunStore, EventSink,
  OutboxLedger, dispatchers y batch dispatchers reales; vaciar el mapper legacy
  deja el test rojo. El executor MCP ya conserva sus pruebas funcionales de
  bootstrap, completion y director autonomo.

Evidencia reejecutada tras todos los cambios, en paralelo local:

- `go test -count=1 ./modulos/orquesta-app-codex-stack` — verde (26.480 s);
- `go test -count=1 ./cmd/orquesta-server` — verde (63.479 s);
- runtime, runtime-ollama, MCP, capability-file y domain-work-file — verdes;
- guard `TestEnvVarsBudgetMEJ106V0` — verde, presupuesto 426 sin ratchet.

El arbol queda limpio. Solicito acreditacion H1b con pruebas de mutacion del
revisor. No declaro cierre hasta esa acreditacion.

### 2026-07-12 — H3 implementado para acreditacion (`c35fbb256`)

El cierre goal-first `accepted` llama ahora a la promocion desde el wrapper
serializado, antes de publicar la cola como cerrada y sin depender del drain
legacy. La ruta con `GoalRef` conserva el integrador local de workspace; no se
ha anadido push ni publicacion automatica.

Decisiones causales aplicadas tras auditoria paralela:

- promocion deshabilitada o sin port deja el goal autoprogramming en
  `promotion-pending` y no cierra la cola;
- la finalizacion durable exige marker derivado de `run_ref`, `goal_ref`,
  promotion ref, integration receipt, commit y archive ref; una cadena falsa
  sin receipt no acredita nada;
- observe y el caller legacy comparten coordinador de promocion por run;
- un tick residente recupera directamente un accepted sin marker tras restart,
  sin reobservar backend ni volver a ejecutar el atestador;
- los conflictos CAS se reintentan solo si son conflictos tipados; errores de
  I/O/validacion conservan su causa.

Evidencia: suite completa `orquesta-app-codex-stack` verde; focales `-race`
verdes; focal productivo de integracion workspace en `cmd/orquesta-server`
verde; guard de envs verde (426). Prueba de mutacion propia: eliminado
temporalmente el hook de `goal_first_queue_sync_v0.go`,
`TestCodexStackAutoprogrammingPromotionV0GoalFirstE2ERepoTemporalReplayV0`
queda rojo con `promotions:0 archives:0`; restaurado, verde. Tests nuevos
cubren promocion inmediata sin drain, recovery tras restart, ausencia de port,
marker falso y carrera observe-vs-drain.

### 2026-07-12 — runner Docker local vinculante y retirada total del remoto

El operador ha corregido expresamente el alcance: todo el trabajo se ejecuta
en los Docker **locales** de este equipo. Queda prohibido volver a usar
`uso.dipgra.cloud` o cualquier runner remoto hasta nueva orden. Los commits
utiles ya presentes en el canon local hasta `a2145ee00` se conservan; no hay
diff remoto H1b/H2/H3 pendiente de copiar.

Se ha levantado `orquesta-self-programming-local` desde `a2145ee00`, gobernado
por API directa en `127.0.0.1:19039`. Evidencia de despliegue: usuario
`10001:10001`, rootfs read-only, no privilegiado, `cap_drop=ALL`,
`no-new-privileges`, sin Docker socket, sin mount de `$HOME`, Codex `0.144.1`,
Go `1.25.11`, tmux `3.3a`, `GOTMPDIR=/workspace/cache/go`, auth aislada y
config de atestacion owner-only con snapshot de modulos Go montado read-only.
Solo el clon local de Orquesta se monta desde el host; estado, runtime, caches
y homes viven en volumenes Docker privados.

Durante el bootstrap aparecieron dos fronteras locales y se resolvieron sin
rebajar el aislamiento exterior:

1. `/home` esta al 100 % y `fsync` quedaba bloqueado en
   `FileAuditSinkV0`/`FileStateStoreV0`. La traza SIGQUIT lo demostro. Estado,
   runtime, caches y homes se movieron a volumenes Docker sobre el storage
   local de Docker; el API volvio a responder `status=ok`.
2. El sandbox interno de Codex devolvio `runtime_sandbox_unavailable` /
   `runtime_sandbox_bwrap_failure`. Por la orden previa del operador —acceso
   completo dentro del Docker, sin acceso exterior salvo el repo Orquesta— el
   perfil local usa `ORQUESTA_CODEX_SANDBOX=danger-full-access` junto a
   `ORQUESTA_CODEX_CONTAINER_SANDBOX_BOUNDARY_CONFIRMED=1`. Es una decision de
   seguridad explicita, no un default productivo ni una relajacion oculta.

Los primeros goals H2 fueron detenidos por `POST /api/v0/runs/control` y el
servidor confirmo `shutdown_ready=true` antes del recreate. Se relanzaron por
`POST /api/v0/autoprogramming/prepare-run` cuatro reworks locales paralelos,
cada uno obligado a crear dos subagentes antes de editar:

- H2a lifecycle: `goal-ref-task-autoprogramming-2f3fe35c8f21-g01`;
- H2b leases/reclaim: `goal-ref-task-autoprogramming-87f331c08080-g01`;
- H2c serializacion por run: `goal-ref-task-autoprogramming-292141a97fdc-g01`;
- H2d HTTP 202 durable: `goal-ref-task-autoprogramming-a1c0c74f5e83-g01`.

No se llamara manualmente a `observe_goal` mientras H2 siga abierto; el
observer residente realiza la observacion para no provocar la carrera que se
esta reparando. H3 y H1b siguen despues de H2, sin cambiar el orden vinculante.

### 2026-07-12 — asignacion explicita del operador

El operador ha asignado como objetivo persistente: cierre total de Orquesta,
sus conectores y tools. No reabro H0a-H0d ni el nucleo: ambos constan
acreditados. En auditoria read-only encontre un residual concreto posterior al
cierre de plataforma: `orquesta.tool.capabilities.list.v0` (commit
`9d8c312b8`) esta registrado en `orquesta-mcp`, pero no aparece cableado en
`orquesta-app-codex-stack` ni `cmd/orquesta-server`, y la tarea canonica del SDK
declara pendiente composicion/materializador real. Solicito que confirmes este
residual como siguiente hito H1b o indiques el write-set/criterio alternativo.
Hasta respuesta no modificare codigo productivo, modelos, routing ni seguridad;
seguire solo con auditoria y pruebas read-only.

### 2026-07-12 — H1b bloqueado por version del binario del runner

H1b-A se lanzo por la API nativa como
`goal-ref-task-autoprogramming-aa8aea5f63ee-g01`, pero quedo `blocked` en un
segundo, cero tokens y sin diff. La causa ya esta reproducida fuera del goal:

- el backend de Orquesta ejecuta explicitamente `/usr/local/bin/codex`, version
  `0.142.3`, fijada en `Dockerfile.self-programming`;
- esa version devuelve HTTP 400 para `gpt-5.6-terra`: el modelo requiere una
  version mas reciente de Codex;
- `/workspace/home/.local/bin/codex` version `0.144.1`, ya presente dentro del
  mismo contenedor aislado, responde `PROVIDER_OK` con `gpt-5.6-terra` bajo
  sandbox read-only.

No toco modelo, alias ni routing. Solicito autorizacion y write-set para alinear
el binario canonico/pin del runner con `0.144.1` (o la correccion que indiques),
reconstruir y relanzar H1b-A causalmente. El goal fallido se limpiara por
run-control gobernado; no se reutilizara como falso verde.

### 2026-07-12 — bitacora de decisiones H1b y CLIs Docker

Decisiones tomadas y evidencia:

1. El primer rework H1b-A (`5d162e0b9a39-g01`) produjo diff material, pero
   lanzo varias suites `cmd/orquesta-server` simultaneas. El goal termino
   `invalid/blocked`; run-control y shutdown gobernado retiraron backend y
   procesos. No se acredita ni se reutiliza como verde.
2. La revision secuencial del diff recuperable encontro dos fallos reales:
   `TestRegisterMCPTransportV0ExponeOperacionesExistentes` seguia exigiendo
   publicar tools sin binding, y `TestMCPTransportV0NuevaAppQuedaOptInSinPuerto`
   hacia panic al invocar una tool ya omitida. El write-set anterior no incluia
   ese test. Por eso se descarta la integracion directa y se relanza causalmente.
3. H1b se divide en dos goals paralelos con write-sets disjuntos: A1 gobierna
   registro MCP condicional y todos sus tests; A2 cablea ejecutores reales en
   `orquesta-app-codex-stack` y documenta las seis decisiones. Los tests se
   ejecutaran secuencialmente por goal.
4. Decision funcional por tool: `ejecutar_orquestacion` y `apply_decision`
   usan ejecutores reales existentes; `solicitar_nueva` recibe executor real
   in-process desde composition root; `domain_work` y `runtime.models` solo se
   registran cuando su puerto opt-in existe; `tool.capabilities.list` usa
   catalogo file real bajo `StateDir/tool-capabilities`, sin env nueva.
5. El operador pidio actualizar Codex, Claude y Gemini a sus ultimas versiones
   estables, tambien en Docker. Registry verificado: Codex `0.144.1`, Claude
   Code `2.1.207`, Gemini CLI `0.50.0`; no se usan preview/nightly. Host queda
   en esas tres versiones. Runner self y Dockerfiles generales fijan Codex
   `0.144.1` (`fee72de10`, `363b75e5b`). Falta incorporar Claude/Gemini a las
   imagenes que deban ejecutarlos y reconstruir/probarlas; se hara en cambio de
   build separado, sin tocar modelos, routing ni seguridad.

### 2026-07-12 — orden posterior del operador: Orquesta paralela con subagentes

El operador ha dado una orden posterior y explicita: usar Orquesta con agentes
en paralelo y exigir subagentes de cada agente. Esta orden sustituye solo la
secuencialidad anterior; se conservan un commit por tool, worktrees aislados,
tests propios, el guard exhaustivo sin debilitar y las prohibiciones sobre
modelos, aliases, routing, seguridad y ratchets.

Antes del lanzamiento se detuvieron por run-control los goals A1/A2 antiguos,
se obtuvo shutdown gobernado `shutdown_ready=true`, y se reconstruyo el runner
aislado sobre `2e57edbd53008e6afbc1961e934ad9e27c6cfa38`. Evidencia independiente:
Codex `0.144.1`, contrato `deploy/self-programming` verde dentro del contenedor,
usuario `10001:10001`, rootfs read-only, sin Docker socket, no privilegiado,
`cap_drop=ALL` y `no-new-privileges`.

Se lanzaron cuatro goals goal-first paralelos, cada uno con la obligacion de
crear al menos dos subagentes (auditoria y pruebas) antes de editar:

- `goal-ref-task-autoprogramming-808db03fe542-g01`: solo
  `orquesta.apps.ejecutar_orquestacion.v0`;
- `goal-ref-task-autoprogramming-94bfa5591bcc-g01`: solo
  `orquesta.tool.capabilities.list.v0`;
- `goal-ref-task-autoprogramming-7ab8ac4b5d02-g01`: solo
  `orquesta.apps.solicitar_nueva.v0`;
- `goal-ref-task-autoprogramming-2c129af2f743-g01`: solo
  `orquesta.director_agent.apply_decision.v0`.

Auditoria read-only previa detecto que la tabla superior esta desactualizada en
un punto material: hay implementaciones concretas existentes para las seis
tools. En especial:

- `domain_work_stack_v0.go` ya construye ejecutores reales file durable, HTTP
  neutral u OPES temporal; el stack los propaga cuando `domain_work` esta
  habilitado. Los contratos vigentes lo declaran opt-in.
- `runtimeModelManagerFromConfigV0` ya construye
  `OllamaModelManagerV0`, con `list/status/pull/serve/stop`; los contratos
  vigentes lo declaran opt-in. Este port gestiona disponibilidad y no expone
  decisiones de routing.
- `MCPToolCapabilitiesListToolExecutorV0`,
  `NewMCPNuevaAppToolExecutorV0` y
  `NewMCPDirectorAgentDecisionToolExecutorV0` tambien existen; falta su
  composicion canonica, no su implementacion base.

Por tanto, para `domain_work` y `runtime.models` queda una decision real de
producto que no inventare: ¿deben dejar de ser opt-in en la configuracion
canonica? Para DomainWork eso elegiria por defecto el backend file durable bajo
`StateDir`; para runtime.models obligaria a elegir proveedor/endpoint y
expondria operaciones mutantes de Ollama. Ademas, la instruccion «solo exponer
lo que el routing ya decide» no coincide con el contrato actual del port. Pido
al revisor resolver expresamente estas dos decisiones mientras avanzan las
otras cuatro tools sin ambiguedad.

### 2026-07-12 — correccion del operador sobre canal de gobierno

El operador precisa que Orquesta debe usarse por su API o por MCP, no mediante
operacion ad hoc por SSH. Las cuatro llamadas anteriores alcanzaron la API
HTTP del runner, pero lo hicieron transportando `curl` por SSH porque el puerto
remoto solo escucha en loopback. Ese canal queda rechazado para el trabajo
siguiente: no se usaran ni acreditaran los resultados de esos goals remotos.

Se levantara el mismo perfil Docker aislado localmente, con todos sus binds
dentro de este repo y API publicada solo en `127.0.0.1:19039`. A partir de ahi,
prepare/observe/control/shutdown e integracion se gobernaran exclusivamente por
API HTTP directa o MCP. No se montara `$HOME`, el Docker socket ni ninguna ruta
del host exterior a `/home/alberto/Trabajo/orquesta`.

### 2026-07-12 — precision inmediata del operador sobre API por SSH

La interpretacion anterior fue demasiado restrictiva y queda corregida por el
operador: SSH al host/contenedor esta permitido como transporte y para tareas
de despliegue o diagnostico. Lo obligatorio es que el **control de Orquesta**
se haga por sus contratos API o MCP, igual que lo hara el operador en uso
normal; no se puede manipular a mano su estado durable, worktrees, sesiones o
procesos para fabricar resultados.

Las cuatro ejecuciones paralelas anteriores son por tanto validas: todas se
crearon mediante `POST /api/v0/autoprogramming/prepare-run`; SSH solo alcanzo
la API ligada a loopback. Se mantienen y se gobernaran por
`prepare-run/status/observe/runs-control/shutdown` o por las tools MCP
equivalentes. No se integrara ningun diff leyendo o alterando directamente los
worktrees del runner. El perfil Docker local duplicado no se levantara.

### 2026-07-12 — dos fallos reales descubiertos por la ola H1b

La ola gobernada por API materializo codigo util, pero Orquesta rechazo
correctamente los cierres sin atestacion independiente. La investigacion
encontro dos fallos de plataforma, por lo que no se acredita H1b todavia:

1. La imagen declara `/usr/local/go/bin` en `ENV PATH`, pero los comandos de
   agente usan login shell y `/etc/profile` reconstruye PATH sin esa ruta.
   Reproduccion dentro del contenedor: `go: not found`, aunque el binario
   existe. Fix acotado: exponer `go` y `gofmt` mediante symlinks en
   `/usr/local/bin`, ruta conservada por login shell, y fijarlo en el contrato.
   El preflight posterior encontro otra frontera del mismo toolchain: `/tmp`
   es `noexec`, por lo que `go test` fallaba al ejecutar `go-build*/test`.
   `GOTMPDIR=/workspace/cache/go` queda fijado al bind aislado, writable y
   ejecutable; el comando real pasa con ese valor.
2. Auditoria read-only encontro que el cierre Goal-first aceptado no llama
   automaticamente a promocion/integracion: el unico caller productivo de
   `maybePromoteClosedAutoprogrammingRunV0` vive en drain legacy, mientras
   Goal-first hace short-circuit y `observe_goal` solo sincroniza/cierra cola.
   Tras reparar el toolchain se abrira un goal causal separado para cablear
   promocion desde el cierre Goal-first aceptado, con tests y sin push.

Todas las runs afectadas fueron detenidas por `runs/control` y el servidor dio
`shutdown_ready=true`. Sus diffs no se integran ni cuentan como verdes.

### 2026-07-12 — mensaje urgente al revisor: carrera de atestacion Goal-first

Auditoria read-only del cierre invalido identifica una tercera causa
estructural, ademas del PATH:

- El ciclo sano `ObserveGoalWorkV0` captura y congela snapshot, adquiere claim,
  ejecuta/persiste attestations y solo despues valida cierre.
- La reparacion `repairGoalFirstReceiptFromMaterializedRefsV0` /
  `repairGoalFirstReceiptFromMaterializedResultV0` llama directamente al
  `GoalClosureValidator`, sin capturar snapshot ni atestar. Eso produce
  `goal_required_test_final_snapshot_missing` y persiste un terminal blocked.
- Los endpoints REST de observe cancelan a los 2 s. Si cancelan despues de
  adquirir claim, el fallo del claim usa el mismo `ctx` cancelado. El store no
  tiene lease, expiracion ni reclaim de claim `pending`; otra observacion puede
  saltar el test y validar como `required_test_attestation_missing`.
- La observacion manual REST puede competir con el observer residente; el CAS
  devuelve estado ganador sin fusionar receipts.

Pido al revisor confirmar este frente causal. Propuesta de reparacion separada,
para ejecutar con Orquesta una vez arreglado el PATH: (1) repair receipt debe
delegar al lifecycle/capturar+atestar antes de validar; (2) claim con
lease/owner/expiry y reclaim gobernado; (3) serializacion por run entre observer
residente/manual; (4) HTTP observe desacoplado (`202` + poll/wakeup) para que el
deadline no cancele trabajo durable. Hasta ese fix no repetire `observe` REST
sobre cierres que esten atestando; usare MCP directo o successor causal.

### 2026-07-12 — H3 reabierto por prueba live y reparado para reacreditacion

La app real pedida por el operador encontro una regresion que los E2E anteriores
enmascaraban. Ejecucion local por API/MCP:

- request `request-ref-native-tool-final-003`;
- run `run-ref-native-tool-final-003`;
- goal `goal-ref-task-autoprogramming-9625cae418dd-g01`;
- cierre `accepted`, snapshot final y atestacion independiente verificada;
- resultado del implementador con required test `passed` y `evidence_refs=[]`;
- promocion detenida en `promotion-pending`, sin receipt, commit ni archive.

Causa: `autoprogrammingPromotionGoalRequiredTestEvidenceV0` solo leia evidencia
autorreportada desde `LastResult`. Ignoraba la autoridad ya persistida en
`LastClosure.AttestationVerifications`. El evaluador devolvia por tanto
`required_test_not_passed`. La proyeccion materializada repetia la misma doble
fuente de verdad y publicaba `required_test_evidence_missing`. Ademas, recovery
convertia `complete=false, err=nil` en exito silencioso.

Decision aplicada, sin tocar modelos, routing, seguridad, envs ni ratchets:

1. Cuando el contrato exige atestacion independiente, promocion solo acepta un
   cierre accepted con verification `Verified && Independent`, TestRef requerido
   y AttestationRef no vacio. No usa evidencia del implementador.
2. Contratos legacy que no exigen atestacion independiente conservan la via
   previa para no cambiar su contrato.
3. La proyeccion materializada reconcilia esas verificaciones aceptadas sin
   reescribir el receipt del implementador y expone sus refs auditables.
4. Recovery incompleto publica `goal_first_promotion_recovery_pending`; ya no
   desaparece como falso exito.
5. El E2E Goal-first fue endurecido: el implementador entrega evidencia vacia,
   el lifecycle atestigua de forma independiente y aun asi deben ocurrir
   promocion, commit, archive, marker y replay sin segundo commit. Volver a leer
   solo `LastResult` deja ese E2E rojo.

Evidencia local tras el cambio:

- suite completa `./modulos/orquesta-app-codex-stack` verde (25.845 s);
- focal E2E + pending con `-race` verde;
- guard `TestEnvVarsBudgetMEJ106V0` verde, sin subir presupuesto.

H3 no se vuelve a declarar cerrado hasta reconstruir el Docker local, recuperar
el mismo run por API/MCP, verificar receipt+commit+archive+ficheros canonicos y
obtener reacreditacion independiente del revisor.

### 2026-07-12 — bloqueo live posterior: identidad Git del integrador Docker

Tras reconstruir el runner en `c68c82960`, la observacion API del mismo run
reutilizo snapshot+atestacion y alcanzo por fin el puerto de integracion. Quedo
`blocked_integration` sin receipt. Diagnostico read-only del workspace fisico:

- los dos `.go` estaban realmente materializados y staged sobre base
  `ac5649269`;
- canonical estaba limpio en `c68c82960`, con la base como ancestro;
- ni workspace ni canonical tenian `user.name`/`user.email`;
- el source commit no llegaba a crearse.

El fixture Git ocultaba esta frontera porque configuraba identidad antes de
crear los worktrees. El conector ejecutaba tanto `commit -m` como
`cherry-pick -x` sin identidad propia. Decision: identidad tecnica fija solo
por comando con `git -c`, sin escribir config, tocar HOME, anadir env ni usar la
identidad personal del operador:

- `Orquesta Integration`;
- `orquesta-integration@localhost.invalid`.

Test nuevo elimina config local, aisla global/system, exige identidad exacta de
autor y committer en source+canonical, repos limpios y replay con mismo HEAD.
Suite `orquesta-runtime-worktree`, E2E H3 focal y guard de envs verdes. Falta
reconstruir runner, recuperar otra vez el mismo run y comprobar integracion
completa antes del cierre operativo.

### 2026-07-12 — app real integrada: evidencia final para cierre operativo

Runner reconstruido en `64128cb80`, `startup_ready=true`. Se observo por API el
mismo run `request-ref-native-tool-final-003`; no se relanzo goal ni se altero
estado durable. Resultado final:

- closure accepted y atestacion independiente reutilizada;
- promotion ref `promotion-ref-635e58f3af60`;
- integration receipt `integration-receipt-ref-promotion-ref-635e58f3af60`;
- commit promovido `dda4f5e19328f9330a0568046a2b2de03f06ae89`;
- archive `archive-ref-c7b423a70d31`;
- marker `promotion-complete:dabf6acd1f4d37787977bc6eb8c0689c`;
- `closure_issues=[]`;
- ambos ficheros presentes en canonical y test de modulo verde.

Los refs historicos `promotion-pending` y `blocked_integration` permanecen por
diseno append-only, pero el estado vigente queda acreditado por marker completo,
receipt, commit y archive. La promocion coincidio con la acreditacion documental
`101c6a6ff`; se conservaron ambos hijos mediante merge `583955307`, sin rebase ni
reescritura del commit acreditado. Solicito cierre final del frente y confirmacion
de que no queda residual tecnico en nucleo/conectores/tools.

### 2026-07-12 — residual post-archive encontrado tras el cierre y corregido

La repeticion final del contrato encontro un ultimo fallo reproducible: tras
archivar y reiniciar, `autoprogramming/status` conservaba el cierre completo,
pero `POST /api/v0/autoprogramming/goal/observe` y la tool MCP equivalente
devolvian 500 para el mismo run. El servicio siempre reobservaba el backend,
aunque `GoalWorkState` ya fuese `complete` con `LastClosure.Accepted=true`; el
backend/workspace archivado ya no debe ser dependencia de un replay terminal.

Fix: el fast-path durable que ya evitaba reobservar forced stops reconoce tambien
un cierre complete+accepted con LastResult/LastClosure persistidos, refleja el run
idempotentemente y devuelve snapshot sin tocar el backend. Test causal nuevo
exige cero llamadas al observer; el test concurrente acredita que la segunda
llamada, serializada despues del primer cierre, reutiliza estado y no genera una
segunda evidencia. Antes del fix ese focal reobservaba y el live devolvia 500.

Evidencia: suite completa `orquesta-app-director-service` verde, suite completa
`orquesta-app-codex-stack` verde (25.556 s), guard env verde. Solicito
reacreditacion. Docker reconstruido en `65d41f467`, `startup_ready=true`; replay
del mismo run archivado devuelve HTTP 200 tanto por REST como por MCP,
`goal_status=complete`, `closure_status=accepted`, `closure_issues=[]`, marker
`promotion-complete` preservado y `isError=false` en MCP.
