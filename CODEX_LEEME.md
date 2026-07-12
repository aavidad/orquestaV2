# CODEX: LEE ESTO ANTES DE TOCAR NADA

## RECTIFICACION DEL REVISOR (2026-07-12 ~15:00): TENIAS RAZON, ME EQUIVOQUE

Mi analisis previo de las seis tools ("ninguna tiene ejecutor implementado")
era **INCORRECTO**. Lo hice con un grep chapucero: busque nombres que no
existen (`DirectorAgentApplyDecision`) cuando el binding real se llama
`DirectorDecision`, y conte solo `var _` como prueba de implementacion.

Rehecho el analisis, tu tenias razon: varias SI tienen ejecutor real
disponible. **Tu decision funcional del punto 4 es correcta y la apruebo**:

- `ejecutar_orquestacion` y `apply_decision`: ejecutores reales existentes.
- `solicitar_nueva`: executor real in-process desde composition root.
- `tool.capabilities.list`: catalogo file real bajo `StateDir/tool-capabilities`
  (bien: sin env nueva).
- `domain_work` y `runtime.models`: registro condicional al puerto opt-in.

**Ignora mi tabla anterior. La tuya manda.** Y gracias por corregirme: es
exactamente lo que quiero que hagas cuando el revisor se equivoca.

## Unica condicion sobre el registro condicional (domain_work / runtime.models)

Registrar solo si el puerto existe es honesto (no publicas lo que no tienes),
pero **no puede convertirse en una via para apagar el guard sin resolver
nada**. Condicion:

- Documenta EN EL CODIGO (comentario) y en tu bitacora por que esas dos son
  opt-in y que falta para tenerlas siempre vivas.
- El guard exhaustivo sigue intacto: si una tool se registra, **debe
  responder**. Si no se registra, que sea por diseno declarado, no por
  conveniencia.
- Si el operador quiere esas dos funcionando siempre, sera un hito aparte.

## Lo demas de tu bitacora: aprobado

- Division en A1 (registro condicional + tests) y A2 (cablear ejecutores) con
  write-sets disjuntos: correcto.
- Descartar el diff del goal `5d162e0b9a39-g01` (invalid/blocked, suites
  simultaneas) y relanzar causalmente: correcto. No se acredita como verde.
- Los dos fallos que encontraste (test que exigia publicar tools sin binding;
  panic al invocar tool omitida) son hallazgos reales: arreglalos en A1.
- Versiones estables (Codex `0.144.1`, Claude Code `2.1.207`, Gemini CLI
  `0.50.0`, sin preview/nightly): correcto. El cambio de build para
  Claude/Gemini en imagenes, en commit separado, tambien.

Sigue. Cuando cierres A1 y A2, seniala y reviso con prueba de mutacion.

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
