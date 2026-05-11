# Pruebas: orquesta-runtime-codex-delivery

```bash
go test -count=1 ./modulos/orquesta-runtime-codex-delivery
```

Cobertura:

- lee un ACK valido desde path externo y devuelve una observacion neutral;
- omite deliveries ya registradas en el run;
- propaga ACK invalido como error publico sin incluir la ruta local del ACK;
- registra descriptors mediante `CodexReceiptRecordingSpecResolverV0`;
- permite que el source lea posteriormente el descriptor registrado;
- recupera descriptors con `FileCodexReceiptDescriptorStoreV0` tras recrear
  instancia;
- reemplaza descriptors por `descriptor_ref` sin duplicar;
- no filtra paths locales si el snapshot JSON del store esta corrupto;
- filtra descriptors por run, agente arrancado y delivery ya registrada;
- observa agentes Codex sin ACK listo y emite progreso `stalled` solo si no hay
  avance observable;
- no escala una firma repetida sin ACK a `loop_detected` si no hay senal real
  de accion repetida;
- no emite supervision si el ACK ya esta listo o si los logs compactos avanzan;
- exige registro de proceso antes de construir supervision de progreso;
- recupera heartbeats/reportes de `FileCodexProgressStateStoreV0` tras recrear
  instancia;
- no filtra paths locales si el snapshot JSON del estado de progreso esta
  corrupto;
- conecta `CodexProgressObservationSourceV0` con
  `ProgressSupervisionCandidateProviderV0`;
- valida que Orquesta no para un proceso real solo porque el ACK tarde y la
  firma compacta no cambie;
- valida que una consulta `stalled` se despacha al director sin convertirse en
  parada automatica por falso bucle;
- el smoke de programacion real queda cableado con supervision de progreso
  conservadora para detectar agentes sin ACK en ciclos futuros;
- valida un loop Orquesta completo con capacidad, request de agente, launcher,
  ACK simulado, observacion de entrega y `DeliveryRegistered`;
- captura baseline de worktree antes del launch si se inyecta recorder;
- rechaza un ACK valido si el diff real contiene cambios fuera del write-set;
- convierte una entrega registrada en observacion de revision;
- acepta la revision si ACK, tests, write-set y ficheros reales son validos;
- pide cambios si falta un test obligatorio, falta un fichero o un fichero
  supera el limite de lineas;
- sigue emitiendo la observacion de review gate tras `RequestReview` para que
  el scheduler pueda registrar resultado y pedir rework en ciclos posteriores;
- deja de emitir la observacion cuando el rework de esa entrega ya esta
  proyectado;
- valida que el source de revision no queda bloqueado por el filtro de
  deliveries ya registradas del store;
- permite un smoke opt-in con Codex real: Orquesta lanza el proceso, tolera que el
  ACK aun no exista, espera el recibo con timeout y registra entrega o artefacto
  de fase en un segundo loop segun la fase del receipt;
- no filtra paths ni detalles operacionales al nucleo.

Smoke real ejecutado en este entorno:

```bash
ORQUESTA_CODEX_SMOKE=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=120 \
go test ./modulos/orquesta-runtime-codex-delivery \
  -run TestCodexReceiptDeliveryLoopV0SmokeCodexRealOptIn \
  -count=1 -timeout 150s
```

Resultado: `ok`, 36.019s.

Smoke de mini app Go ejecutado en este entorno:

```bash
ORQUESTA_CODEX_APP_SMOKE=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
go test ./modulos/orquesta-runtime-codex-delivery \
  -run TestCodexReceiptDeliveryLoopV0SmokeCodexRealAppOptIn \
  -count=1 -timeout 300s
```

Resultado: `ok`, 222.247s. Lectura: valida el pipeline real de app minima, pero
esta duracion es demasiado alta para usar "app completa" como una sola tarea de
agente. La produccion debe dividir app completa en microtareas con ACK/progreso
por agente.

Smoke de formulario MCP con director Codex real ejecutado en este entorno:

```bash
ORQUESTA_CODEX_MCP_FORM_SMOKE=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/orquesta-smokes/mcp-form-director-20260509113045/project \
ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-smokes/mcp-form-director-20260509113045/project/.orquesta-codex-runtime \
go test ./modulos/orquesta-runtime-codex-delivery \
  -run TestMCPFormularioDirectorCodexRealOptInV0 \
  -count=1 -timeout 260s -v
```

Resultado: `ok`, 152.022s.

Evidencia generada:

- `runtime/agent_packet.json`: paquete de director emitido por Orquesta.
- `runtime/agent_ack.json`: ACK valido con `status=completed`.
- `project/docs/arquitectura.md`: 284 lineas.
- `project/docs/plan_microtareas.md`: 349 lineas.

Lectura: valida la ruta formulario -> MCP -> Orquesta -> director -> Codex real
-> ACK/observacion. En este corte aun no registraba el artefacto durable del
director; queda como evidencia historica del hueco detectado.

Smoke de formulario MCP con director Codex real y artefacto de fase:

```bash
ORQUESTA_CODEX_MCP_FORM_SMOKE=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=240 \
ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/orquesta-smokes/mcp-form-director-20260509120532/project \
ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-smokes/mcp-form-director-20260509120532/project/.orquesta-codex-runtime \
go test ./modulos/orquesta-runtime-codex-delivery \
  -run TestMCPFormularioDirectorCodexRealOptInV0 \
  -count=1 -timeout 260s -v
```

Resultado: `ok`, 128.025s.

Evidencia generada:

- `runtime/agent_packet.json`: paquete de director emitido por Orquesta.
- `runtime/agent_ack.json`: ACK valido con `status=completed`.
- `project/docs/arquitectura.md`: 200 lineas.
- `project/docs/plan_microtareas.md`: 91 lineas.
- el segundo loop de Orquesta registro `PhaseArtifactRegistered`;
- no se registro `DeliveryRegistered` para `brainstorming_arquitectura`.

Lectura: valida ruta completa formulario -> MCP -> Orquesta -> director ->
Codex real -> ACK -> `DeliveryCandidateProviderV0` -> scheduler ->
`RegisterPhaseArtifact` -> core. La documentacion generada respeta hexagonal,
i18n y persistencia por puerto/conector, e incluye consultas al director.

Revalidacion 2026-05-09 tras activar equipo de directores acotado en
`orquesta-app-director-intake`:

```bash
ORQUESTA_CODEX_MCP_FORM_SMOKE=1 \
go test ./modulos/orquesta-runtime-codex-delivery \
  -run TestMCPFormularioDirectorCodexRealOptInV0 \
  -count=1 -timeout 240s -v
```

Resultado: `ok`, 164.026s. Lectura: el formulario sin autonomia alta mantiene
un solo director real, y la ruta formulario -> MCP -> Orquesta -> Codex real ->
ACK -> `PhaseArtifactRegistered` sigue verde.

Prueba real multiagente desde formulario MCP con autonomia alta:

```bash
ORQUESTA_CODEX_MCP_FORM_TEAM_SMOKE=1 \
go test ./modulos/orquesta-runtime-codex-delivery \
  -run TestMCPFormularioDirectorTeamCodexRealOptInV0 \
  -count=1 -timeout 420s -v
```

Resultado: `ok`, 172.041s.

Evidencia validada por el test:

- Orquesta arranco 4 procesos Codex reales en paralelo: director, web, API y
  persistencia.
- Cada agente escribio `agent_ack.json` en su runtime aislado por run/agente.
- El loop registro los 4 ACKs como `PhaseArtifactRegistered`.
- Se verificaron documentos en `docs/arquitectura.md`,
  `docs/plan_microtareas.md`, `docs/web.md`, `docs/api.md` y
  `docs/persistencia.md`.

Fallo previo corregido: el source devolvia primero un artefacto ya registrado y
el scheduler no alcanzaba los demas candidatos. Se corrigio filtrando
`phase_artifacts` ya consumidos en el store y en el source defensivo. Tests
unitarios asociados:

- `TestCodexDeliveryObservationSourceV0OmiteArtefactoFaseYaRegistrado`.
- `TestInMemoryCodexReceiptDescriptorStoreV0FiltraArtefactoFaseRegistrado`.

Prueba real multiagente de programacion:

```bash
ORQUESTA_CODEX_PROGRAMMING_TEAM_SMOKE=1 \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=420 \
go test ./modulos/orquesta-runtime-codex-delivery \
  -run TestProgrammingTeamCodexRealOptInV0 \
  -count=1 -timeout 480s -v
```

Resultado historico antes de anadir director real y README: `ok`, 125.354s.

La version actual anade director real y documentacion minima; no se reejecuto
en este corte para no gastar cuota. Duracion esperada: 4-8 minutos. El smoke de
programacion usa `StartAppDirectorV0` y `RunManagedProgressiveLoopV0`; la prueba
ya no espera ACKs manualmente desde el test. Orquesta relanza el loop tras cada
`wait_external`, consume `director_decisions.json`, observa ACKs disponibles y
puede detectar estancamiento entre ciclos.

Evidencia validada por el test:

- Orquesta arranca un director Codex real y registra su artefacto de fase.
- El director entrega decisiones ejecutables para API, web y documentacion.
- Orquesta arranca agentes de programacion en paralelo por batch de hasta 2.
- Los write-sets quedan separados: API, `web/index.html` y `README.md`.
- Cada agente escribio `agent_ack.json` en runtime aislado por run/agente.
- En `workspace-write`, el runtime de control se ubico bajo
  `project/.orquesta-codex-runtime`; ya no depende de un sibling externo.
- El loop registra las entregas de programacion como `DeliveryRegistered`.
- La app generada paso `go test ./...` ejecutado por la prueba.
- El test exige API REST, tests generados, web con i18n basico, README minimo y
  ficheros por debajo de 300 lineas antes de aceptar la app.
- El smoke queda cableado con baseline/diff de `orquesta-runtime-worktree` para
  rechazar cambios reales fuera de write-set antes de `DeliveryRegistered`.
- No quedaron procesos `codex exec` asociados al smoke al terminar.

Fallos previos que quedan como reglas de diseno:

- Primer intento: fallo en 152.042s porque el agente API creo codigo y test,
  pero no declaro `go test ./...` en el ACK. Se corrigio la plantilla del
  prompt Codex para incluir `files` y `tests` esperados.
- Segundo intento: fallo en 360.020s porque la tarea web era demasiado abierta;
  genero ficheros pero no cerro con ACK. Se redujo a microtarea de un unico
  HTML con limite de lineas.
- Tercer intento: los agentes crearon la app y los tests pasaron, pero no
  pudieron escribir ACK fuera del proyecto con `workspace-write`. Se corrigio de
  raiz exigiendo runtime interno al proyecto para ese sandbox.

Revalidacion de supervision Codex sin ACK:

```bash
go test ./modulos/orquesta-runtime-codex-delivery \
  -run 'TestCodexProgressObservationSourceV0|TestCodexProgressObservationV0ParaProcesoRealEnBucle|TestCodexProgressObservationV0StalledNoBloqueaParadaPorBucle|TestFileCodexProgressStateStoreV0' \
  -count=1
```

Resultado: `ok`. Lectura: el adaptador detecta estancamiento compacto, escala a
`loop_detected`, evita falsos positivos si hay avance o ACK, alimenta el
provider del nucleo sin filtrar paths ni logs y permite que el loop gestionado
pare el proceso real registrado por `ProcessAgentStopperV0`. Tambien valida que
un aviso `stalled` ya despachado al target `director` no bloquea una parada
posterior si la evidencia empeora a bucle.

Revalidacion 2026-05-11 de materializacion de progreso:

```bash
go test ./modulos/orquesta-runtime-codex-delivery \
  -run 'TestCodexProgressObservationSourceV0|TestCodexProgressObservationV0StalledEscalaYParaPorBucle|TestCodexProgressReportWithProcessFailureContextV0' \
  -count=1 -v
```

Resultado: `ok`.

Evidencia anadida:

- `TestCodexProgressObservationSourceV0ReemiteSiElRunNoHaMaterializadoDecision`
  asegura que una lectura previa de progreso no roba el candidato al scheduler.
- `TestCodexProgressObservationSourceV0ReportaProcesoParadoSinACKEnPrimerTick`
  cubre proceso terminado sin ACK desde el primer tick.
- `TestCodexProgressReportWithProcessFailureContextV0ClasificaCuotaSinFiltrarProveedor`
  compacta cuota externa agotada sin exponer proveedor, modelo, HOME ni paths.
- `TestCodexProgressObservationV0StalledEscalaYParaPorBucle` confirma que una
  pregunta previa por stalled no bloquea la escalada posterior a loop.
