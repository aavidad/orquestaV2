# Pruebas: orquesta-app-codex-stack

Validacion de este subtrabajo:

```bash
git diff --check -- modulos/orquesta-app-codex-stack
go test -count=1 ./modulos/orquesta-app-codex-stack
go test -count=1 ./modulos/orquesta-app-director-intake ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack
```

Cobertura Go actual:

- `BuildStackV0` exige opt-in y puertos explicitos;
- el guard de arquitectura bloquea imports legacy `cmd` y DB hardcodeada;
- `POST /api/v0/apps/director` arranca una cohorte directora por batch con
  runtime fake inyectado;
- `POST /nueva-app` usa el cliente REST interno y arranca otra cohorte
  independiente;
- `POST /api/v0/domain-work` delega en el executor `DomainWork` inyectado sin
  que el stack importe OPES ni conectores reales;
- `TestCodexStackV0OPESExternalWorkRESTCreaMicrotareaSinWriteSetLocal` valida
  el flujo REST que usara OPES: arranque de director, `POST
  /api/v0/apps/opes/changes` con `external_work`, microtarea de dominio externo
  sin `allowed_write_set` local y consulta de `/api/v0/director/stats`;
- las refs publicas del paquete de agente son neutrales y no filtran el
  conector real;
- dos solicitudes con el mismo nombre visible no colisionan porque el intake
  usa identidad de spec, no solo slug.
- `TestNuevaAppWebCodexStackRealOptInV0` queda desactivado por defecto y valida
  `/nueva-app` con agente real cuando `ORQUESTA_CODEX_STACK_SMOKE=1`.
- `TestNuevaAppWebCodexStackRealMultiagentOptInV0` queda desactivado por
  defecto y valida `/nueva-app` con 4 Codex reales en paralelo cuando
  `ORQUESTA_CODEX_STACK_MULTIAGENT_SMOKE=1`.
- `TestCodexStackV0DirectorStatsIncluyeProcesoYProgresoPorPuertos` valida que
  `/director-stats` usa los puertos inyectados del stack para exponer control
  de parada y progreso de agentes sin ACK; solo el director inicial puede
  aparecer protegido, no los directores especializados.
- `TestCodexStackV0ReviewGateAceptaEntregaConEvidenciaReal` valida que el
  stack conecta review gate y acepta una entrega con fichero real manejable.
- `TestCodexStackV0ReviewGatePideCambiosSiFicheroEsDemasiadoGrande` valida que
  una entrega registrada pasa a `changes_requested` si supera 300 lineas.
- `TestCodexStackV0ReviewChangesRequestedReplanificaYArrancaAgente` valida el
  flujo vertical completo: entrega registrada, revision con evidencia real,
  `changes_requested`, `RequestRework`, `retry_task`, decision de capacidad y
  arranque de un nuevo agente sin intervencion manual del test.
- `TestNuevaAppWebCodexStackRealReviewReworkOptInV0` queda desactivado por
  defecto y valida con Codex reales el ciclo: app multiagente, entregas de
  programacion, revision `changes_requested` por evidencia real de fichero
  demasiado grande, `RequestRework`, `retry_task`, agente real de rework y ACK
  del rework. La version actual exige ademas que la entrega del rework sea
  aceptada por el review gate con evidencia real.
- `TestCompositeDirectorDecisionSourceV0RechazaVoteRefIncoherente` valida que
  el stack no consume un lote del director con refs causales rotas entre
  votacion, decision, contrato y microtareas.
- `TestCodexStackRealSmokeDetectaACKFaltanteConProyectoCompilable` reproduce
  localmente el caso real de programacion paralela con write-set materializado,
  app Go compilable y un ACK faltante; el helper devuelve
  `project_compiles_but_ack_missing` en vez de esperar al deadline global.
- `TestStackShutdownCheckpointV0SolicitaAckSiHayAgenteEnVuelo` valida que el
  shutdown no forzado escribe request de checkpoint y deja pendiente al agente
  vivo si aun no hay ACK.
- `TestStackShutdownCheckpointV0RegistraCuandoTodosLosAgentesResponden` valida
  que el stack registra checkpoint solo despues de ACK de checkpoint valido.
- `TestCodexStackRealShutdownCheckpointOptInV0` queda desactivado por defecto
  y valida con Codex real el ciclo: agente vivo, `POST /api/v0/server/shutdown`
  con `forced=false`, request de shutdown, ACK de checkpoint y registro en una
  segunda llamada.

Guardas esperadas para pruebas futuras:

- unitarios sin Codex real, sin DB real y sin credenciales;
- smokes reales desactivados por defecto;
- activacion solo con `ORQUESTA_CODEX_STACK_OPT_IN=1`;
- proveedor/modelo/DB/HOME/CODEX_HOME/PATH siempre configurados por operador;
- timeout acotado y procesos observables por registry;
- ACK valido antes de registrar entrega o artefacto;
- verificacion de write-set antes de aceptar ACK;
- la supervision no convierte silencio temporal de logs/ACK en bucle terminal;
- los helpers de drenaje real no deben confundir `tasks=[]` temporal con falta
  de progreso cuando `LastSequence` avanzo;
- el smoke de app completa debe exigir una ola de programacion paralela real:
  varios `AgentStarted` de programacion antes del primer `DeliveryRegistered`
  de esa ola;
- si el write-set de una microtarea esta completo y el proyecto Go generado
  compila pero falta ACK valido, el fallo esperado es causal
  `project_compiles_but_ack_missing`, no `context deadline exceeded`;
- el smoke de review/rework no debe modificar evidencia mientras queden agentes
  externos pendientes o ACKs completados sin registrar, porque eso falsea el
  verificador real de write-set;
- errores publicos sin paths locales, tokens, prompts ni transcripts.

Smoke real review/rework opt-in:

```bash
ORQUESTA_CODEX_STACK_REVIEW_REWORK_SMOKE=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=900 \
ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/orquesta-smokes/review-rework-20260512/project \
ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-smokes/review-rework-20260512/project/.orquesta-codex-runtime \
go test ./modulos/orquesta-app-codex-stack \
  -run TestNuevaAppWebCodexStackRealReviewReworkOptInV0 \
  -count=1 -timeout 1100s -v
```

Contrato validado por el smoke:

- Orquesta arranca una app multiagente con agentes Codex reales, no fakes ni
  subagentes manuales;
- recoge entregas reales de programacion mediante ACK y write-set;
- espera una cohorte estable sin agentes pendientes ni ACKs completados sin
  registrar antes de inyectar una incidencia artificial;
- altera una evidencia textual entregada para superar el limite de 300 lineas;
- abre `revision` y el review gate lee evidencia real por puerto inyectado;
- proyecta resultado `changes_requested` sin aceptar la entrega original;
- emite `RequestRework` y replanificacion `retry_task`;
- arranca un agente Codex real de rework localizado por contrato de task, no
  por prefijo interno del nombre;
- espera y valida el ACK del agente de rework;
- registra la entrega del rework en una reentrada posterior de `DrainRunV0`;
- evalua la entrega del rework con el review gate real y exige `accepted`;
- limpia procesos con los conectores de runtime, sin depender de HOME, DB,
  proveedor ni filesystem dentro del core.

Riesgo observado en este smoke:

- un director real puede aplicar varias decisiones y aumentar `LastSequence`
  antes de que existan tareas materializadas para programacion o rework;
- por tanto, `tasks=[]` durante una pasada de drenaje es estado intermedio,
  no prueba de ausencia de progreso;
- el helper de smoke debe considerar progreso cualquier avance de secuencia,
  proyeccion, ACK, descriptor o proceso observable, y solo fallar cuando no hay
  avance durante el presupuesto configurado.

Prueba real de cambio a mitad de ejecucion:

```bash
ORQUESTA_CODEX_STACK_OPT_IN=1 \
ORQUESTA_CODEX_STACK_COMMAND='ORQUESTA_CODEX_STACK_CHANGE_SMOKE=1 go test ./modulos/orquesta-app-codex-stack -run TestNuevaAppWebCodexStackRealCambioMitadOptInV0 -count=1 -timeout 1200s -v' \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=900 \
ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/orquesta-smokes/app-codex-stack-change-real-4/project \
ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-smokes/app-codex-stack-change-real-4/project/.orquesta-runtime \
./modulos/orquesta-app-codex-stack/arrancar_codex.sh
```

Resultado 2026-05-10: `ok`, 284.168s.

Evidencia validada:

- Orquesta lanzo 4 Codex reales iniciales en paralelo: `director`, `web`,
  `api` y `persistencia`;
- el director emitio `director_decisions.json` y Orquesta lo consumio sin
  intervencion manual;
- Orquesta lanzo agentes de programacion derivados de esas decisiones;
- el cambio entro por `/app-change` mientras habia trabajo de programacion
  pendiente;
- Orquesta creo y lanzo el agente adicional
  `agent-ref-task-ref-app-change-*`;
- el agente de cambio escribio `docs/change-request-midrun.md`;
- no quedaron procesos Codex/go test vivos tras finalizar.

Repeticion final tras corregir drenaje de ACK tardio:

- workdir:
  `/tmp/orquesta-smokes/app-codex-stack-change-real-5/project`;
- resultado 2026-05-10: `ok`, 314.187s;
- Orquesta volvio a lanzar 4 Codex reales iniciales;
- consumio decisiones reales del director;
- lanzo agentes de programacion y, en paralelo, el agente de cambio;
- `docs/change-request-midrun.md` fue creado por el agente lanzado por
  Orquesta;
- no quedaron procesos Codex/go test vivos tras finalizar.

Smoke manual de referencia:

```bash
ORQUESTA_CODEX_STACK_OPT_IN=1 \
ORQUESTA_CODEX_STACK_COMMAND='ORQUESTA_CODEX_STACK_SMOKE=1 go test ./modulos/orquesta-app-codex-stack -run TestNuevaAppWebCodexStackRealOptInV0 -count=1 -timeout 300s -v' \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=240 \
ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/orquesta-smokes/app-codex-stack-real/project \
ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-smokes/app-codex-stack-real/project/.orquesta-runtime \
./modulos/orquesta-app-codex-stack/arrancar_codex.sh
```

Este comando es una referencia operativa, no un default del modulo. El operador
debe anadir modelo, stores, DSN o workdirs solo cuando el smoke concreto los
requiera.

Evidencia 2026-05-10:

- comando anterior ejecutado con `ORQUESTA_CODEX_MODEL=gpt-5.5`;
- PASS en 88.09s;
- workdir: `/tmp/orquesta-smokes/app-codex-stack-real-2/project`;
- el agente real creo `docs/arquitectura.md` y `docs/plan_microtareas.md`;
- Orquesta valido `agent_ack.json` y registro `PhaseArtifactRegistered`.
- repeticion tras dividir helpers de test: PASS en 82.09s;
- workdir de repeticion:
  `/tmp/orquesta-smokes/app-codex-stack-real-3/project`.

Smoke multiagente real ejecutado el 2026-05-10:

```bash
ORQUESTA_CODEX_STACK_OPT_IN=1 \
ORQUESTA_CODEX_STACK_COMMAND='ORQUESTA_CODEX_STACK_MULTIAGENT_SMOKE=1 go test ./modulos/orquesta-app-codex-stack -run TestNuevaAppWebCodexStackRealMultiagentOptInV0 -count=1 -timeout 420s -v' \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=300 \
ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/orquesta-smokes/app-codex-stack-multiagent-real-12/project \
ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-smokes/app-codex-stack-multiagent-real-12/project/.orquesta-runtime \
./modulos/orquesta-app-codex-stack/arrancar_codex.sh
```

Resultado: `ok`, 148.207s.

Evidencia validada:

- 4 procesos Codex reales lanzados por Orquesta en paralelo;
- ACKs: `web`, `director`, `persistencia`, `api`;
- documentos: `docs/web.md`, `docs/arquitectura.md`,
  `docs/plan_microtareas.md`, `docs/persistencia.md`, `docs/api.md`;
- drenaje final correcto con artefactos de fase registrados;
- sin procesos Codex/go test vivos tras finalizar.

Prueba de decisiones ejecutables del director:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexAreaV0|TestDirectorTaskV0|TestCodexStackV0ConsumeDecisionFileYArrancaProgramacion|TestCodexStackV0GatewayAPIYWebArrancanEquipoDirectorConRuntimeInyectado' -v
```

Resultado: `ok`.

Evidencia validada:

- el objetivo del director principal incluye `run_id`, `brainstorm_ref`,
  `director_decisions.json` y schemas esperados;
- las areas especializadas no reciben contrato de decision ejecutable;
- un runtime fake escribe `director_decisions.json`;
- Orquesta consume el fichero por puerto, crea microtarea, abre
  `programacion` y arranca un agente de implementacion;
- el rol `implementacion` se clasifica como area `programacion`.

Prueba real de decisiones ejecutables tras endurecer contrato:

```bash
ORQUESTA_CODEX_STACK_OPT_IN=1 \
ORQUESTA_CODEX_STACK_COMMAND='ORQUESTA_CODEX_STACK_MULTIAGENT_SMOKE=1 go test ./modulos/orquesta-app-codex-stack -run TestNuevaAppWebCodexStackRealMultiagentOptInV0 -count=1 -timeout 900s -v' \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=600 \
ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-10/project \
ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-10/project/.orquesta-runtime \
./modulos/orquesta-app-codex-stack/arrancar_codex.sh
```

Resultado 2026-05-10: `ok`, 194.108s.

Evidencia validada:

- Codex real escribe `director_decisions.json` con campos exactos
  `phase_id`, `decision_ref`, `command_type`, `command_ref` y payload tipado;
- `evidence_refs` contiene solo ids compactos, sin espacios, slash, rutas ni
  etiquetas humanas;
- si un agente tarda, `AgentStalledV0` puede enviar pregunta al director sin
  bloquear el registro posterior de ACK/artefactos.
- Orquesta reentra con `ContinueAppDirectorV0`, consume decisiones tardias y
  lanza al menos un agente de `programacion` sin intervencion manual.
- el POST inicial no consume decisiones ni espera programacion completa;
- `DrainRunV0` lanza 4 agentes reales de programacion:
  `agent-ref-task-agenda-domain-001`, `agent-ref-task-agenda-usecases-001`,
  `agent-ref-task-agenda-api-001` y `agent-ref-task-agenda-web-001`;
- no quedan procesos Codex vivos tras finalizar el smoke.

Prueba real de ciclo completo de programacion:

```bash
rm -rf /tmp/orquesta-smokes/app-codex-stack-director-decisions-real-12 && \
mkdir -p /tmp/orquesta-smokes/app-codex-stack-director-decisions-real-12/project && \
ORQUESTA_CODEX_STACK_OPT_IN=1 \
ORQUESTA_CODEX_STACK_COMMAND='ORQUESTA_CODEX_STACK_MULTIAGENT_SMOKE=1 go test ./modulos/orquesta-app-codex-stack -run TestNuevaAppWebCodexStackRealMultiagentOptInV0 -count=1 -timeout 1200s -v' \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=900 \
ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-12/project \
ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-12/project/.orquesta-runtime \
./modulos/orquesta-app-codex-stack/arrancar_codex.sh
```

Resultado 2026-05-10: `ok`, 500.099s.

Evidencia validada:

- Orquesta arranca 4 Codex reales iniciales: `api`, `web`, `persistencia` y
  `director`;
- el director emite `director_decisions.json`;
- el primer `DrainRunV0` consume decisiones y lanza 3 agentes reales de
  programacion en paralelo:
  `agent-ref-task-programacion-dominio-agenda-v0`,
  `agent-ref-task-programacion-entrega-agenda-v0` y
  `agent-ref-task-programacion-calidad-agenda-v0`;
- los 3 agentes de programacion escriben ACK valido;
- el segundo `DrainRunV0` registra las entregas de programacion;
- se validan los write-sets de todos los descriptores, incluidos directorios;
- la app generada pasa pruebas Go en `internal/agenda/...`,
  `internal/agenda/delivery` y `web/agenda`;
- todos los ficheros Go generados quedan por debajo de 300 lineas;
- no quedan procesos Codex/go test vivos tras finalizar.

Intento real previo 2026-05-10:

- workdir:
  `/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-11/project`;
- resultado: FAIL en 810.117s;
- Orquesta si completo la orquestacion real: 4 ACKs iniciales, decisiones del
  director, 3 agentes de programacion lanzados y 3 ACKs de programacion;
- causa raiz: el verificador de smoke trataba un write-set de directorio como
  fichero (`internal/agenda/domain`);
- decision aplicada: no tocar el contrato de agentes; el verificador acepta
  directorios si contienen artefactos y sigue validando write-sets.

Observacion de calidad:

- `web/agenda/openapi.yaml` quedo en 310 lineas; no rompe la guarda actual
  porque la regla automatizada se aplica a ficheros Go. Si se quiere limitar
  tambien specs largas, debe cerrarse como politica separada.

Repeticion real 2026-05-10:

- workdir:
  `/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-6/project`;
- resultado: FAIL en 178.226s;
- los 4 agentes reales terminaron y escribieron ACK;
- el director escribio `director_decisions.json` con estructura tipada correcta;
- causa raiz: `evidence_refs` incluia rutas/texto humano como identificadores,
  por lo que el puerto estricto de decisiones rechazo el fichero antes de
  arrancar programacion.

Segunda repeticion real 2026-05-10:

- workdir:
  `/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-7/project`;
- resultado: FAIL en 204.249s;
- los 4 agentes reales terminaron y el director emitio 10 decisiones con
  `evidence_refs` limpias;
- causa raiz nueva: refs encadenadas inconsistentes entre voto y aceptacion, y
  vocabulario sensible literal en campos de decision;
- decision aplicada: reforzar prompt, no relajar validadores ni traducir refs.

Tercera repeticion real 2026-05-10:

- workdir:
  `/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-8/project`;
- resultado: FAIL en 174.221s;
- los 4 agentes reales terminaron y el director emitio decisiones con refs
  limpias e IDs encadenados coherentes;
- causa raiz nueva: `minimum_recommended_capacity` localizado como `alta`;
- decision aplicada: fijar enum literal `low`, `medium`, `high`, `xhigh` y
  usar `high` para la votacion inicial.

Prueba local de drenaje tardio:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestDrainRunV0ConsumeDecisionFileTardioYArrancaProgramacion|TestCodexStackV0WebDrenaACKMultiagenteTardio' -v
```

Resultado 2026-05-10: `ok`.

Evidencia validada:

- un `director_decisions.json` que aparece despues del arranque inicial se
  consume en `DrainRunV0`;
- el run abre `programacion`;
- se arranca un agente de microtarea;
- ACKs tardios no quedan bloqueados por preguntas no bloqueantes al director.

Smoke real multiagente ejecutado el 2026-05-11:

```bash
ORQUESTA_CODEX_STACK_MULTIAGENT_SMOKE=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=1200 \
ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/orquesta-smokes/multiagent2-20260511/project \
ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-smokes/multiagent2-20260511/project/.orquesta-codex-runtime \
go test ./modulos/orquesta-app-codex-stack -run TestNuevaAppWebCodexStackRealMultiagentOptInV0 -count=1 -timeout 1400s -v
```

Resultado: `ok`, 698.155s.

Evidencia validada:

- Orquesta lanzo 4 Codex reales iniciales: director, api, web y persistencia;
- el director emitio `director_decisions.json`;
- Orquesta consumio decisiones y lanzo 5 agentes reales de programacion en
  paralelo: domain-contracts, channels-i18n, connectors, quality-docs y
  delivery;
- todos los agentes escribieron ACK valido;
- se registraron entregas y artefactos de fase;
- los ficheros Go generados quedaron por debajo de 300 lineas.

Hallazgo posterior:

- la app generada tenia codigo real y modular, pero no era una app Go autonoma:
  faltaba `go.mod`, no habia entrypoint `cmd/`, y existian imports relativos;
- validacion manual: `GO111MODULE=off go test ./...` fallo en connectors por
  contrato desalineado;
- decision aplicada: endurecer contrato y smoke para exigir `go.mod`,
  entrypoint bajo `cmd/`, imports de modulo y `go test ./...`.

Pruebas locales tras la decision:

```bash
go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director-agent ./modulos/orquesta-director-agent-workflow ./modulos/orquesta-app-planner ./modulos/orquesta-app-codex-stack ./orquestacionnucleoapp ./modulos/orquesta-app-runner ./modulos/orquesta-mcp ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex
```

Resultado 2026-05-11: `ok`.

Pruebas locales 2026-05-11 de drenaje estricto y proceso parado sin ACK:

```bash
go test ./modulos/orquesta-app-codex-stack \
  -run 'TestDrainRunV0TrasEntregaBootstrapLanzaFronteraDependiente|TestDrainRunV0ProcesoParadoSinACKNoQuedaEsperandoIndefinido|TestNuevaAppWebCodexStackRealMultiagentOptInV0' \
  -count=1 -v
```

Resultado local sin smoke real: `ok` para frontera dependiente y proceso parado
sin ACK. El smoke real estricto queda opt-in.

Evidencia:

- tras entregar bootstrap, Orquesta lanza en paralelo las tareas dependientes de
  dominio, HTTP, web y documentacion;
- si un runtime queda `stopped` sin ACK, el stack no deja agentes pendientes
  indefinidos y proyecta assessment/parada/confirmacion;
- el helper de smoke real ya no acepta como exito una app que solo haya
  completado bootstrap.

Intento real estricto 2026-05-11:

- workdir: ruta local de smoke real aislada fuera del repo;
- resultado: abortado manualmente tras detectar cuota externa agotada;
- los procesos Codex iniciales devolvieron error de uso/cuota antes de ACK;
- no quedaron procesos `codex exec` vivos asociados al smoke;
- el hallazgo origino la regla de proceso parado sin ACK documentada arriba.

Pruebas locales 2026-05-11 de uso/cuota por conector:

```bash
go test ./modulos/orquesta-app-codex-stack \
  -run 'TestCodexStackAgentUsageSourceV0UneMetricasInyectadas|TestCodexStackV0DirectorStatsIncluyeProcesoYProgresoPorPuertos' \
  -count=1 -v
```

Resultado: `ok`.

Evidencia:

- el stack sigue devolviendo `not_configured` si no hay proveedor de metricas;
- con `CodexStackAgentUsageMetricsProviderPortV0` fake se proyectan cuota,
  tokens y coste por agente;
- `DirectorRunStatsV0.UsageSummary` acumula tokens/coste para que MCP/web y el
  director puedan responder cuanto se ha usado en una app;
- no se introduce proveedor, HOME, OAuth, DB ni API concreta en el core.

Smoke real multiagente repetido el 2026-05-11 con app Go/API/web:

- workdir: `/tmp/orquesta-smokes/multiagent-20260511143049/project`;
- comando: `ORQUESTA_CODEX_STACK_MULTIAGENT_SMOKE=1 ... go test ./modulos/orquesta-app-codex-stack -run TestNuevaAppWebCodexStackRealMultiagentOptInV0 -count=1 -timeout 25m -v`;
- resultado: FAIL por `context deadline exceeded` al llegar al timeout global
  de 1200s del smoke;
- Orquesta lanzo 4 Codex reales iniciales en paralelo: `director`, `api`,
  `web` y `persistencia`;
- el director escribio `docs/arquitectura.md`, `docs/plan_microtareas.md` y
  `director_decisions.json`;
- Orquesta consumio decisiones y lanzo agentes reales de programacion;
- completaron con ACK valido las microtareas `task-agenda-api-web-001`
  (`go.mod`, `cmd/server`, `internal/bootstrap`) y `task-agenda-api-web-002`
  (`internal/domain`, `internal/application`, `internal/ports`);
- el tercer agente genero `internal/http`, `internal/web` e `internal/i18n`,
  pero no alcanzo a escribir ACK antes del timeout global;
- no quedaron procesos `codex exec` vivos tras terminar el smoke;
- la app parcial generada paso:

```bash
GOCACHE=/tmp/orquesta-smokes/multiagent-20260511143049/gocache go test ./...
```

Resultado: `ok`.

Calidad observada:

- app Go autonoma con `go.mod` y entrypoint `cmd/server/main.go`;
- API/web/i18n/dominio/casos de uso separados por paquetes pequenos;
- todos los ficheros Go quedaron por debajo de 300 lineas;
- los agentes de programacion que cerraron ACK ejecutaron `go test ./...`;
- la persistencia sigue detras de puertos, sin SQLite/Postgres por defecto.

Causa raiz del timeout:

- el smoke tenia un limite global de run, pero no un presupuesto por agente,
  tarea o ACK terminal;
- algunos procesos Codex seguian vivos despues de escribir ACK mientras Orquesta
  ya habia registrado la entrega;
- una microtarea de HTTP/web/i18n era demasiado grande para el presupuesto real
  de una prueba controlada.

Decisiones y fixes aplicados despues del hallazgo:

- `ProcessAgentStopperV0` valida identidad de proceso (`process_ref`,
  `session_ref`, `launch_ref`) antes de parar;
- los agentes de direccion protegidos no se convierten en `StopRuntimeAgent`
  ante `loop_detected`; generan assessment y pregunta al director;
- el stack hace cleanup terminal del runtime tras registrar ACK valido, sin
  emitir `AgentStopConfirmed` ni contaminar `StoppedAgents`;
- la prueba de stack se actualizo para distinguir la ref explicita del director
  inicial protegido frente a directores especializados, que siguen siendo
  parables si tienen proceso registrado.

Validacion local posterior:

```bash
go test ./modulos/orquesta-app-codex-stack -count=1
go test ./orquestacionnucleoapp ./modulos/orquesta-director ./modulos/orquesta-director-scheduler ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-outbox-dispatch -count=1
git diff --check -- modulos/orquesta-app-codex-stack orquestacionnucleoapp modulos/orquesta-director modulos/orquesta-director-scheduler modulos/orquesta-runtime
```

Resultado 2026-05-11: `ok`.

Smoke real multiagente repetido el 2026-05-11 con contrato de bootstrap:

- workdir: `/tmp/orquesta-smokes/multiagent4-20260511/project`;
- resultado: FAIL en 380.150s;
- mejora validada: el director real emitio 4 microtareas de programacion en
  paralelo, todas con `required_tests: go test ./...`;
- mejora validada: el plan ya asigno `go.mod` y `cmd/server/main.go`;
- causa raiz nueva: el write-set usaba `internal/bootstrap/*.go`, el agente
  escribio archivos concretos dentro del patron y el ACK fallo porque el
  validador exigia el glob literal como artifact;
- decision aplicada: el conector Codex acepta artifacts concretos que satisfacen
  globs cerrados sin abrir el write-set.

Smoke real multiagente repetido el 2026-05-11 tras endurecer verificador:

- workdir: `/tmp/orquesta-smokes/multiagent3-20260511/project`;
- resultado: FAIL por timeout controlado de 1200s;
- Orquesta lanzo 4 agentes iniciales reales y despues 2 agentes reales de
  programacion en paralelo;
- ambos agentes de programacion dejaron ACK y `CONSULTA AL DIRECTOR`;
- causa raiz: el director creo microtareas sin `required_tests` y sin write-set
  para `go.mod`/`cmd/server`; los agentes no podian crear esos archivos sin
  violar el contrato;
- decision aplicada: `required_tests` obligatorio en programacion y validacion
  temprana del lote de decisiones Go antes de lanzar agentes.
- prueba local posterior: un fichero `director_decisions.json` con microtareas
  reales que traen `decision.phase_id=programacion` se normaliza en el borde y
  `DrainRunV0` materializa todas las microtareas antes de abrir agentes.

Pruebas locales tras la decision:

```bash
go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director-agent ./modulos/orquesta-director-agent-workflow ./modulos/orquesta-app-director-service ./modulos/orquesta-app-change-director-source ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./orquestacionnucleoapp
```

Resultado 2026-05-11: `ok`.

Revalidacion 2026-05-12 de proceso parado por capacidad limitada:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'TestDrainRunV0ProcesoParadoPorCapacidadLimitadaNoMarcaBasura|TestDrainRunV0ProcesoParadoSinACKNoQuedaEsperandoIndefinido' -v
```

Resultado: `ok`.

Evidencia:

- un proceso parado sin ACK con senal de capacidad externa limitada no queda
  como agente pendiente;
- Orquesta registra `#verdict:capacity_limited` y `#action:stop_agent`;
- no se marca `garbage` ni `loop_detected`;
- el smoke real de review/rework imprime `issues` estructurados tambien en el
  primer drain para diagnosticar fallos largos sin leer manualmente todos los
  logs.

Revalidacion global 2026-05-12:

```bash
go test -count=1 ./...
```

Resultado: `ok`.

Smoke real review/rework 2026-05-12:

- workdir: `/tmp/orquesta-smokes/app-codex-stack-review-rework-real-20260512102054/project`;
- modelo configurado por conector: `gpt-5.5`, razonamiento `xhigh`;
- Orquesta lanzo 4 agentes reales en paralelo: director, API, web y
  persistencia;
- API, web y persistencia escribieron ACK y documentacion;
- el director escribio ACK y `director_decisions.json`;
- resultado: FAIL controlado en 424.627s por orden causal, no por timeout ni
  cuota.

Causa raiz:

- `director_decisions.json` se consumio antes de que el ACK del propio director
  estuviera reflejado como `PhaseArtifactRegistered`;
- las decisiones abrieron `programacion`;
- despues el ACK de `brainstorming_arquitectura` fallo correctamente con
  `payload.phase_id` porque la fase actual ya habia cambiado.

Fix de base aplicado:

- `director_decisions.json` queda tratado como sidecar causal del ACK productor;
- el conector Codex no expone el fichero de decisiones hasta que el `ack_ref`
  ya aparece en `PhaseArtifacts` o `Deliveries`;
- no se ignora `transicion_invalida` ni se mete sleep.

Regresion local:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'TestDrainRunV0RegistraACKDirectorAntesDeConsumirDecisionFile|TestCodexStackV0ConsumeDecisionFileYArrancaProgramacion|TestDrainRunV0ConsumeDecisionFileTardioYArrancaProgramacion' -v
```

Resultado: `ok`.

Regresion local 2026-05-12 del caso app compilable sin ACK:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'TestCodexStackRealSmokeDetectaACKFaltanteConProyectoCompilable|TestNuevaAppWebCodexStackRealMultiagentOptInV0' -v
```

Resultado local sin opt-in real: `ok` para la regresion fake; el smoke real
sigue desactivado si no se define `ORQUESTA_CODEX_STACK_MULTIAGENT_SMOKE=1`.

Evidencia fijada:

- Orquesta consume decisiones del director y lanza varias microtareas de
  programacion;
- al menos dos agentes de programacion quedan `AgentStarted` antes de la
  primera entrega de esa ola, probando paralelismo del stack;
- una microtarea materializa su write-set pero no escribe ACK;
- el proyecto Go temporal tiene `go.mod`, `cmd/server` y pasa `go test ./...`;
- el helper del smoke devuelve `project_compiles_but_ack_missing` con
  `task_ref` y `ack_ref`, en vez de agotar el timeout global.

Revalidacion 2026-05-13 de shutdown controlado en stack:

```bash
go test ./cmd/orquesta-server ./modulos/orquesta-server-shutdown ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack -count=1
```

Resultado: `ok`.

Evidencia:

- `BuildStackV0` cablea `/api/v0/server/shutdown` con
  `orquesta-server-shutdown`;
- el caso de uso lista runs por cola, prepara checkpoint en modo no forzado,
  solicita stop por RunControl, ejecuta el supervisor global y reconstruye
  readiness con stats del director;
- `orquesta-server stop` llama primero al endpoint de shutdown y solo envia
  senal al servidor si `shutdown_ready=true`;
- el test de stats de progreso del stack valida el contrato estable:
  agentes arrancados = agentes progresando + agentes sin senal, sin exigir que
  todos caigan siempre en una sola categoria observable.

Smoke real opt-in de shutdown cooperativo Codex:

```bash
ORQUESTA_CODEX_STACK_SHUTDOWN_SMOKE=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=240 \
ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/orquesta-smokes/app-codex-stack-shutdown-real/project \
ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-smokes/app-codex-stack-shutdown-real/project/.orquesta-runtime \
go test ./modulos/orquesta-app-codex-stack \
  -run TestCodexStackRealShutdownCheckpointOptInV0 \
  -count=1 -timeout 300s -v
```

Validacion sin opt-in:

```bash
go test ./modulos/orquesta-app-codex-stack -run Test.*Shutdown.* -count=1
```

Resultado local 2026-05-13 sin opt-in: `ok`, 0.005s.

Validacion OPES/domain-work 2026-05-13:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'TestCodexStackV0OPESExternalWork|TestCodexLaunchSpecResolverV0MaterializaContextoDominioExterno|TestBuildStackV0CableaDomainWorkOptIn' -v
```

Resultado: `ok`.

Evidencia:

- `external_work.input_fields` llega al `agent_packet.context.entries` como
  contexto de dominio acotado;
- una entrega real fake de `draft_content_block` invoca `DomainWork`
  `submit_artifact`;
- el artefacto usa `job_ref`, `content_block`, `body` leido desde fichero del
  ACK y refs externas `run_ref/task_ref/delivery_ref`;
- el replay se controla por ledger e idempotencia de `delivery_ref`.

Validacion de stats OPES/job externo 2026-05-13:

```bash
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./modulos/orquesta-run-file \
  -run 'TestMCPDirectorStatsToolExecutorV0ResuelveRunPorJobExterno|TestMCPRunControlExecutorV0ResuelveRunPorJobExterno|TestCodexStackV0OPESExternalWorkRESTCreaMicrotareaSinWriteSetLocal|TestRunFileStoreAppChangePersistsAfterRecreateAndReplacesV0|TestFileDomainWorkArtifactSubmissionLedgerV0PersisteYRecupera'
```

Resultado: `ok`.

Evidencia:

- `orquesta.director.stats.v0` resuelve `run_ref` desde `external_job_ref`
  mediante puerto inyectado;
- `orquesta.runs.control.v0` puede aplicar `pause/resume/stop/cancel` sobre la
  run asociada a `external_job_ref`;
- la respuesta incluye `external_job` con `job_ref`, `run_ref`, `task_ref`,
  `agent_ref`, `status` y `delivery_refs` cuando existan;
- `RunFileStoreV0` no pierde `external_work` tras reinicio;
- el ledger de entregas de dominio puede persistir y deduplicar
  `idempotency_key` despues de recrear el conector.
