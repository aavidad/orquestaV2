# CODEX: LEE ESTO ANTES DE TOCAR NADA

## H1b — ORDEN DEL OPERADOR (2026-07-12 ~14:30): CABLEAR, NO BORRAR

El operador ha decidido: **si las tools son validas, se cablean; NO se
borran**. Esa es la linea. Ninguna de las seis se retira del registro sin
autorizacion explicita suya.

### Analisis del revisor (ya hecho, no lo repitas)

Verifique el estado real de las seis y **ninguna tiene ejecutor concreto
implementado** (cero implementaciones declaradas del port en todo el repo):

| Tool | Binding | Estado real |
|---|---|---|
| `orquesta.apps.ejecutar_orquestacion.v0` | `EjecutarOrquestacion` | Existe `NewMCPEjecutarOrquestacionAppToolExecutorV0(ports orquestaapprunner.RunPreparedAppOrchestrationPortsV0)`. **Es el mas cercano a cablearse**: hay que construir esos ports en la composicion. |
| `orquesta.apps.solicitar_nueva.v0` | `NuevaApp` | Solo hay handler de transporte; sin executor concreto. |
| `orquesta.director_agent.apply_decision.v0` | — | Sin implementacion. |
| `orquesta.domain_work.v0` | `DomainWork` | El stack lo propaga desde `config.DomainWork`, pero **nadie construye un DomainWork real** en `cmd/orquesta-server`. |
| `orquesta.runtime.models.v0` | `RuntimeModels` | `RuntimeModelManagerPortV0` es una interfaz **sin implementacion** en el repo. |
| `orquesta.tool.capabilities.list.v0` | `ToolCapabilities` | La registraste tu en `9d8c312b8`; su materializador sigue pendiente (lo dijiste tu mismo). |

**Conclusion:** "cablear" aqui **no es enchufar algo que ya existe: es
implementarlo**. Es trabajo de verdad, no de fontaneria.

### Como quiero que lo hagas

Una tool por vez, en este orden (de mas cerca a mas lejos):

1. `ejecutar_orquestacion` (tiene executor; construye sus ports).
2. `domain_work` (el stack ya lo propaga; falta la implementacion real).
3. `runtime.models` (implementa `RuntimeModelManagerPortV0` — **OJO: sin tocar
   modelos ni routing; solo exponer lo que el routing ya decide**).
4. `tool.capabilities.list` (tu materializador pendiente).
5. `apps.solicitar_nueva` y `director_agent.apply_decision`.

Para cada una:
- **Un commit por tool**, con su test.
- Si al abrirla ves que la implementacion real exige decisiones de producto
  que no estan tomadas, **PARA y escribelo aqui**. No la inventes ni la
  simules: una tool que responde algo falso es peor que una tool muerta.
- **No debilites** el guard exhaustivo
  (`TestMCPBootstrapComposicionCanonicaCableaCatalogoYSuperficiesV0`). El
  rojo actual (6 tools) es correcto: es tu lista de trabajo. Ira bajando a
  medida que cierres cada una.
- Prohibido seguir vigente: modelos, aliases, routing, seguridad.

### Estado que ya cerraste (bien hecho)

- Pin del runner a Codex `0.144.1` (`fee72de10`) y alineado en las imagenes
  generales (`363b75e5b`). Verificado por el revisor: el contrato de deploy
  pasa. Con eso el runner ya puede hablar con los modelos `gpt-5.6-*`.
- Relanza el goal H1b-A cuando quieras: el bloqueo de version esta resuelto.

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
