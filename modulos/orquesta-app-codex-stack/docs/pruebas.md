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
- `TestDefaultDomainWorkArtifactSubmissionBuilderV0NoFiltraPathsComoPayloadRefs`
  valida que el builder no convierte rutas de `ack.files` en `payload_refs`
  invalidas ni filtra paths internos al contrato de dominio;
- `TestDefaultDomainWorkArtifactSubmissionBuilderV0EntregaDocumentPlanOPES`
  valida que `plan_tema` se entrega como `document_plan`, usa
  `content_type=application/json` y conserva `sections`/`deliverables` como
  JSON validable;
- `TestValidateDomainWorkDeliveryQualityV0AceptaDocumentPlanValido` y
  `TestValidateDomainWorkDeliveryQualityV0RechazaDocumentPlanIncompleto`
  validan que el stack no acepta planes documentales incompletos;
- `TestCodexStackV0OPESPlanTemaDeliveryEnviaDocumentPlan` valida el flujo
  vertical corto: cambio externo `plan_tema`, drenaje, ACK de agente y envio de
  `document_plan` al puerto `DomainWork`.
- `TestCodexStackV0OPESPlanTemarioOperadoresSupervisorXHighEnviaDocumentPlan`
  valida el smoke local completo para un job OPES `plan_temario` de operadores:
  el bridge crea `/api/v0/external-work/run`, `POST /api/v0/runs/supervise`
  empuja la run, el agente fake entrega `document_plan`, el puente envia
  `submit_artifact` a `DomainWork`, el paquete de agente marca `capacity=xhigh`
  y el wrapper Codex materializado incluye `model_reasoning_effort="xhigh"`;
  tambien comprueba que el contexto del agente contiene la regla OPES de
  derivacion descendente A1/A2 o A1 -> B/C1 -> C2/AP y el metodo de
  asimilacion.
- `DrainRunV0` ejecuta `submitPendingDomainWorkArtifactsV0` tambien despues de
  `ContinueAppDirectorV0`; asi una entrega registrada dentro del mismo ciclo no
  queda como run `quiescent` antes de enviar el artefacto al conector de dominio.
- `TestCodexStackResidentDirectorV0IdleSinCandidatosV0` valida que el Director
  residente del stack queda idle cuando la cola no tiene runs ejecutables.
- `TestCodexStackResidentDirectorV0EjecutaRunEnColaConBriefingLoopV0` valida
  el adaptador real con runtime fake: el residente elige el run de `RunQueue`,
  usa el briefing loop, materializa outbox y despacha hasta registrar un agente
  Codex iniciado sin lanzar procesos reales.
- `TestCodexStackResidentDirectorV0ProcesaLoteSegunMaxRunsPerTickV0` valida que
  el residente puede ejecutar mas de un run por tick cuando los limites
  normalizados lo permiten.
- `TestCodexStackResidentBriefingSourceV0NoReutilizaStopMaxStepsInternoV0`
  evita que un `stop_max_steps` interno de un burst se convierta en parada
  terminal del residente.
- `TestCodexStackResidentBriefingSourceV0EmiteConsejoConEstadoEstructuralV0`
  valida que el residente solo propone `materialize_decision_council` cuando el
  run tiene estado durable suficiente (`Brainstorms`, `Votes`) y contratos
  publicados.
- `TestCodexStackResidentBriefingSourceV0NoEmiteConsejoFueraDeFaseDeCreacionV0`
  cubre que no se fuerza el consejo en fases donde `CreateMicrotask` no es
  causalmente aplicable.
- `TestCodexStackResidentCouncilHandlerV0MaterializaUnaVezConContratoPublicadoV0`
  valida la materializacion idempotente de tareas `task-council-*` por puertos
  del stack, abre `brainstorming_arquitectura` y deja candidatos de propuesta
  visibles para `WorkflowTaskCandidateProviderV0`.
- `TestCodexStackResidentCouncilV0AbreVotacionCuandoTerminaBrainstormV0`
  valida que, tras propuestas y criticas entregadas, la fuente residente emite
  `open_decision_council_vote_phase`, el handler abre `votacion_y_decision` y
  los votos quedan schedulables.
- `TestCodexStackResidentCouncilV0AceptaDecisionConVotosEstructuradosV0`
  valida la cadena live del consejo residente: propuestas y criticas
  entregadas, apertura de votacion, votos `architecture_vote.v0` simulados por
  `VoteSource`, evaluacion de quorum/evidencia/familias y aplicacion
  idempotente de `AcceptDecision`.
- `TestCodexStackResidentCouncilV0NoProponeAceptarSinVoteSourceV0` valida que
  el residente no emite una accion de aceptacion que no puede aplicar por falta
  de fuente estructurada de votos.
- `TestCodexStackResidentCouncilHydrateVotesV0NormalizaMetadataDurableV0`
  valida que `TaskRef` une voto y tarea, `VoteRef` puede ser independiente y
  `agent_ref`/`family_ref` durables mandan sobre valores devueltos por el
  adaptador.
- `TestCodexLaunchSpecResolverV0MaterializaPacketConsejoPropuestaCriticaVoto`
  valida que propuesta, critica y voto del consejo se resuelven como area
  `decision_council`, no como director generico, con objetivo especifico,
  artefacto esperado y contexto `decision_council_context.v0`.
- `TestCodexLaunchSpecResolverV0DetectaConsejoPorMetadataSinRolPayloadV0`
  valida que el resolver detecta una tarea de consejo por metadata durable
  aunque el payload no traiga `role`.
- `TestCodexStackResidentCouncilHandlerV0NoMaterializaSinContratoFuncionalV0`
  confirma que sin contrato funcional publicado el handler deja evidencia
  pendiente y no inventa microtareas.
- las refs publicas del paquete de agente son neutrales y no filtran el
  conector real;
- el resolver de tareas de programacion conserva en el paquete de agente el
  linaje recursivo neutral leido desde `WorkflowTaskStore`;
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
- `TestCodexStackV0ReviewGateAceptaFicheroGrandeComoAviso` valida que una
  entrega registrada con mas de 300 lineas conserva la evidencia como rail
  blando y no dispara `changes_requested`.
- `TestCodexStackV0ReviewChangesRequestedReplanificaYArrancaAgente` valida el
  flujo vertical completo: entrega registrada, revision con evidencia real,
  `changes_requested`, `RequestRework`, `retry_task`, decision de capacidad y
  arranque de un nuevo agente sin intervencion manual del test.
- `TestReviewReworkReplanSourceV0MantieneEvidenciaOperativaOpaca` valida que
  refs operativas opacas como runtime/Codex se conservan como evidencia y solo
  se filtran valores sensibles efectivos.
- El stack cablea `OperationalPlanStateWriter` y `OperationalPlanStateStore`
  hacia `ContinueAppDirectorV0`; la prueba focal del state vive en
  `orquesta-app-director-service` y `TestOperationalPlanStateStoreV0UsaStoreExplicitoOWriterLegible`
  cubre el fallback de composicion.
- `TestNuevaAppWebCodexStackRealReviewReworkOptInV0` queda desactivado por
  defecto y valida con Codex reales el ciclo: app multiagente, entregas de
  programacion, revision `changes_requested` por incidencia real no blanda,
  `RequestRework`, `retry_task`, agente real de rework y ACK del rework. La
  version actual exige ademas que la entrega del rework sea aceptada por el
  review gate con evidencia real.
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
- `TestCodexSupervisorV0LanzaPrimeroYContinuaHastaDoneV0` fija el contrato
  minimo de supervisor Codex: primer tick `LaunchV0`, ticks siguientes
  `ContinueV0(ctx, "sigue")` y parada al observar `done`.
- `TestCodexSupervisorV0CortaPorMaxTicksSinDoneV0` fija que `pending`/`stopped`
  no cierran la sesion y que el supervisor corta por `max_ticks` si no llega
  `done`.
- `TestCodexSupervisorStackLifecycleV0SupervisaRunExistenteSinCanalParaleloV0`
  prueba el adaptador real de stack sobre una run ya creada: `SuperviseCodexV0`
  hace `launch` y luego `continue`, ambos por `DrainRunV0`, sin relanzar agentes.
- `TestCodexSupervisorStackLifecycleV0AvanzaArbolRecursivoFakeSinManualPorNivelV0`
  prueba el mismo adaptador con arbol recursivo fake 1->2->4: el supervisor
  empuja `sigue` por `DrainRunV0`, conserva parent/child refs y waits acotados,
  y cierra sin llamadas manuales por nivel ni relanzar agentes.
- `TestCodexSupervisorStackLifecycleV0UsaSupervisorGlobalExistenteV0` prueba el
  camino sin `RunRef`, que reutiliza `RunGlobalSupervisorV0` y la cola global.
- `TestCodexSupervisorRuntimeStateFromLoopV0MapeaEstadosDelNucleoV0` fija la
  traduccion de estados del loop del nucleo a snapshot del supervisor Codex.
- `TestCodexStackRunSupervisorAPIV0EmpujaRunExistenteSinRelanzarAgentes` prueba
  `POST /api/v0/runs/supervise` sobre `stack.Handler`: reentra por el adaptador
  real, conserva `run_ref`, expone evidencia y no relanza agentes ya vivos.
- `TestCodexStackRealRequiredTestRunnerEndToEndOptInV0` queda desactivado por
  defecto y valida con un agente Codex real acotado el ciclo del Director
  Operativo: task con `RequiredTests`, `WaitAgentRefs`, ACK/entrega, review
  causal aceptada, runner local de tests, `RequiredTestEvidenceV0` durable y
  cierre de plan/run. Cerrado por `CODEX-REQTEST-REAL-E2E`; no sustituye OPES
  temporal real de derivados/cierre.
- `TestCodexAckRequiredTestRunnerV0MaterializaContextoRefOnlyDesdeNotaSinReceipt`
  y `TestOperationalClosureSourceV0CierraConContextoRefOnlySinTestReceipt`
  cubren el caso ResiGRX: un required test contextual `ref_only` no necesita
  recibo shell si el ACK lista el test, trae nota `contexto_ref_only_resuelto`
  y conserva refs causales de entrega/review; los tests ejecutables siguen
  exigiendo `test_receipts` validos.
  El cierre no-OPES temporal con app HTTP/file, submitter real opt-in,
  `codex-fake`, review, required-tests y plan cerrado queda cubierto por
  `EXT-NO-OPES`.
- `TestCodexStackRecursiveTreeFakeRuntimeV0` valida el arbol recursivo completo
  1->2->4 con runtime fake: waits por `parent_task_ref`/`wave_ref`, entregas,
  review causal aceptada, `RequiredTestRunner`, evidencias por task y cierre de
  las 7 tareas por `PlanState`.
- `TestCodexStackGeneratedRecursiveTreeFakeRuntimeV0` valida la ruta offline en
  la que solo existe el padre inicial: una review `changes_requested` genera
  hijos por `split_task`, esos hijos generan nietos por la misma ruta real de
  replan, y el test comprueba `WorkflowTaskStore`, refs del run, waits por
  parent/wave y 7 lanzamientos con runtime fake.
- `TestCodexStackRealRecursiveTreeOptInV0` queda desactivado por defecto y
  reutiliza el mismo recorrido con procesos Codex reales, sin OPES y con doble
  confirmacion; espera ACK/entregas reales por nivel antes de abrir hijos o
  nietos, y despues exige review causal, runner y cierre del arbol. Cerrado el
  2026-05-23 por `CODEX-RECURSION-REAL` con `high`, permisos amplios y
  `scripts/smoke_codex_real_recursive_tree.sh` en 341.05s.

Estado de huecos restantes:

- `CODEX-WAVE-REAL` ya cierra la ola/cohorte amplia formal con varios agentes
  Codex vivos, `PlanState`, review causal, `RequiredTestEvidenceV0` y cierre.
- `CODEX-REQTEST-REAL-E2E` ya cierra el caso acotado de un agente Codex vivo
  con runner y cierre causal.
- `CODEX-RECURSION-REAL` ya cierra el arbol 1->2->4 con proveedor real,
  split_task generado offline y wrapper real opt-in
  (`scripts/smoke_codex_real_recursive_tree.sh`). No reabrir estos smokes salvo
  regresion demostrada.
- `EXT-NO-OPES` cierra la ruta temporal no-OPES con `codex-fake`, no una
  politica productiva de tests de dominio ni un proveedor Codex real.
- OPES `plan_temario` real quedo cerrado para `document_plan` y creacion de
  derivados pendientes; falta smoke real completo de derivados/cierre OPES.
- El consejo residente ya llega offline/fake hasta `AcceptDecision` con votos
  estructurados y packet Codex especifico para propuesta, critica y voto. Falta
  fuente real de artefactos `architecture_vote.v0`.
- Falta un smoke canonico opt-in de `Director residente + Codex real`. Los
  smokes reales vigentes cubren ola/recursion Codex, pero el smoke residente
  actual usa `codex-fake`; no debe marcarse como evidencia de proveedor real
  hasta crear script o flag dedicado.
- Gemini/Claude aun heredan contratos de ACK/receipt con nombres `codex_*`.
  Pendiente neutralizar a `orquesta_agent_ack.v0` y mantener `codex_*` solo
  como alias legacy del adaptador Codex.

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
- el adaptador de `CodexSupervisorAgentLifecyclePortV0.ContinueV0` usa el manejo
  normal de agentes de Orquesta: tick/drain/replan/outbox `LaunchRuntimeAgent`,
  registry, ACK y progreso. No debe crear un canal paralelo al dispatcher
  existente.

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

Smoke real requerido con runner y cierre operativo:

```bash
ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_CONFIRM=1 \
ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_EXECUTE_CODEX=1 \
ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_CODEX_EXECUTION_CONFIRMED=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_REASONING_EFFORT=medium \
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run '^TestCodexStackRealRequiredTestRunnerEndToEndOptInV0$' \
  -timeout 720s -v
```

Resultado 2026-05-22: cerrado como `CODEX-REQTEST-REAL-E2E` mediante
`./scripts/smoke_codex_real_required_test_runner.sh` en 64.49s.

Evidencia validada:

- un unico agente Codex real arranca bajo `WaitAgentRefs`;
- registra ACK/entrega y pasa por review causal aceptada;
- `RequiredTestRunner` ejecuta los tests declarados y persiste
  `RequiredTestEvidenceV0`;
- `ContinueAppDirectorV0` cierra plan/run por fuente de cierre del stack Codex;
- no toca OPES ni core.

Pendiente no cubierto por estos smokes:

- OPES temporal real de derivados/cierre.

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
  tokens y capacidad por agente;
- `DirectorRunStatsV0.UsageSummary` acumula tokens para que MCP/web y el
  director puedan responder cuanto se ha usado en una app;
- no se introduce proveedor, HOME, OAuth, DB ni API concreta en el core.

Reconciliacion T209 2026-05-27 de reporte runtime redactado:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackRuntimeUsageMetricsSourceV0|TestCodexStackAgentUsageSourceV0UneMetricasInyectadas|TestCodexStackV0DirectorStatsIncluyeProcesoYProgresoPorPuertos'
```

Cobertura:

- lee `codex_usage_accounting.json` redactado por descriptor registrado;
- reporta tokens y `quota_status` cuando el reporte es seguro;
- si el reporte falta, esta vacio, no trae cuota fiable o contiene campos
  sensibles, publica `unknown` con `quota_observed_unavailable` y evidence ref
  compacto;
- no expone proveedor, modelo, HOME, OAuth, coste, rutas privadas, prompts,
  transcripts ni completions.

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

Validacion external-work-run 2026-05-13:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run TestCodexStackV0ExternalWorkRunCreaRunSinDirectorInicial
```

Evidencia:

- el endpoint `POST /api/v0/external-work/run` crea run en `programacion`;
- registra pregunta de director durable para app-change;
- encola la run con `app_ref=opes`;
- no arranca agentes ni director inicial antes del tick global.

Validacion autoprogramacion incremental 2026-05-23:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run TestCodexStackV0ExternalWorkRunSelfProgrammingSupervisaSinBloqueoGoBootstrapV0
```

Evidencia:

- un `external-work/run` con `work_kind=self_programming`, write-set de modulo
  existente y test focal de modulo no se trata como bootstrap de app Go nueva;
- `DrainRunV0` consume la decision de `app-change`, crea la microtarea y lanza
  un agente fake;
- no aparece el blocker `director_decision_source`.

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

Validacion visual asset OPES 2026-05-13:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run TestDefaultDomainWorkArtifactSubmissionBuilderV0EntregaVisualAssetOPES
```

Evidencia esperada:

- `generate_visual_asset` se entrega como `artifact_type=visual_asset`;
- `format=svg` produce `content_type=image/svg+xml`;
- `title`, `caption`, `alt_text`, `placement`, `language_code` y refs de OPES
  permanecen como payload de dominio;
- el SVG generado por el agente viaja como `body`, sin filtrar rutas locales.

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

Validacion de contexto externo amplio 2026-05-13:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'TestProgrammingTaskV0TrabajoExternoUsaUnidadTrabajoNoMicrotareaMinima|TestCodexLaunchSpecResolverV0PermiteContextoAmplioEnTrabajoDocumentalLargo|TestCodexLaunchSpecResolverV0MarcaContextoExternoTruncadoComoRiesgoDeCierre|TestCodexLaunchSpecResolverV0MarcaPresupuestoTotalAgotado'
```

Evidencia esperada:

- `ApplyExternalDomainWorkV0` se lanza como unidad de trabajo externa, no como
  microtarea minima;
- `draft_content_block` admite una ventana editorial mas amplia sin truncar un
  paquete razonable;
- si `agent_packet.context` contiene entradas requeridas truncadas, el packet
  exige validar contexto no truncado o justificar materializacion externa.
- si el paquete externo excede el presupuesto total, queda una entrada
  `external_context_budget` truncada para impedir cierres silenciosos.

Validacion recuperacion neutral DomainWork/ACK 2026-05-14:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server
```

Resultado local: `ok`.

Casos cubiertos:

- `TestCodexStackV0OPESExternalWorkRecuperaEntregaSinACKConArtefactoValido`;
- `TestDomainWorkRecoveryArtifactFilesV0AceptaDirectorioPermitido`;
- `TestDomainWorkRecoveryDirectSubmitEligibleV0RechazaAgenteAssessment`;
- `TestDomainWorkRecoveryAckFailureDetectaSandboxEnStderr`;
- `TestCodexStackV0RunGlobalTickRecuperaDomainWorkParadoAntesFiltroControl`.

Evidencia:

- la recuperacion solo acepta artefactos reales bajo write-set y contrato
  `ApplyExternalDomainWorkV0`;
- el ACK sintetico valida contra la misma spec que un ACK escrito por agente;
- el ledger evita replay por `idempotency_key`;
- un agente parado/terminal puede entregar por `direct submit` si el artefacto
  es valido y el ACK fallo por sandbox;
- los agentes de assessment no pueden usar esta ruta de entrega directa.

Smoke real OPES 2026-05-14:

- Orquesta server vivo en la run
  `/home/alberto/Trabajo/OPES/opes-uso/runs/temario_psicologia_autonomo_20260514T204913`;
- OPES API y program-runner siguen vivos;
- el bridge `opes-drain-once` sigue enviando jobs pendientes a Orquesta;
- el job `d9e501a5352f011a0e8fd6d9f67cac16` paso de pendiente a completado
  tras recuperar una entrega valida sin ACK escrito;
- nuevos jobs (`a178cbdda95b5f1f8d52030b090c49b2`,
  `586320266a25fc2b4d59ee23e0c2c9a3`,
  `ab6200e9bd75cd6f92b6e2d9c40c6e41`) quedaron reenviados por el bridge para
  continuar el temario.

Validacion runner generico de puentes externos 2026-05-14:

```bash
go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector ./modulos/orquesta-app-codex-stack
```

Resultado local: `ok`.

Casos cubiertos:

- `external_bridge_loop` no ejecuta ticks si esta desactivado;
- un error transitorio de tick se registra y no tumba el servidor;
- `opes_bridge_loop` usa el fallback del server para `ORQUESTA_BASE_URL`;
- el ledger generico evita reenviar el mismo job externo OPES pendiente;
- OPES queda como adaptador opt-in y no entra en el core.

Revalidacion de drenaje por microtarea durable 2026-05-14:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'TestCodexStackV0GatewayAPIYWebArrancanEquipoDirectorConRuntimeInyectado|TestCodexStackV0CleanupRuntimeTrasACKRegistrado|TestDrainRunV0IgnoraArtefactoYaRegistradoPorLoopGestionado'
```

Resultado local: `ok`.

Evidencia:

- los ACK de director inicial siguen entrando como `PhaseArtifactRegistered`;
- el cleanup de runtime se dispara tras ACK registrado;
- el drenaje idempotente ignora ACK ya reflejado por el loop gestionado.

Validacion normalizacion `content_block` OPES 2026-05-15:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'TestDefaultDomainWorkArtifactSubmissionBuilderV0NormalizaSourceRefsRicosContentBlockOPES|TestDefaultDomainWorkArtifactSubmissionBuilderV0EntregaTopicSummaryOPES'

go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server \
  ./modulos/orquesta-app-codex-stack ./modulos/orquesta-opes-bridge \
  ./modulos/orquesta-opes-connector
```

Resultado local: `ok`.

Smoke real:

- job OPES `6e7468ddeb377d6151f8ad4974113c09` estaba `pending` con ACK y
  contenido escrito, pero sin artefacto OPES por `source_refs` ricos;
- tras compilar y reiniciar Orquesta, el mismo run entrego por DomainWork sin
  tocar OPES a mano;
- OPES creo el artefacto `0c5d157e88ada08dcc60cdc2dfd41fd6` y marco el job
  como `completed`;
- el bloque vivo creado en OPES es `ebfae56a4741bd3f0e0e3470bf849d0d`, tipo
  `doctrine`, capitulo `37274eef9db47452a80acff3bedcf832`, unas 2627 palabras.

Validacion tolerancia de nombres OPES 2026-05-18:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'TestDefaultDomainWorkArtifactSubmissionBuilderV0AceptaEnvelopeConNombreLibreOPES|TestDefaultDomainWorkArtifactSubmissionBuilderV0NormalizaSourceRefsRicosContentBlockOPES|TestDefaultDomainWorkArtifactSubmissionBuilderV0DerivaSourceRefsDesdeCitasOPES|TestDefaultDomainWorkArtifactSubmissionBuilderV0EntregaVisualAssetOPES'

go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector
```

Resultado local: `ok`.

Cobertura:

- el `artifact_type` del sobre de agente puede llamarse de otra forma si el
  job externo ya fija el contrato esperado;
- aliases como `tema_id`, `id_capitulo`, `titulo`, `contenido`, `idioma` y
  `fuentes` se normalizan a `topic_id`, `chapter_id`, `title`, `body`,
  `language_code` y `source_refs`;
- si un bloque trae `citations` con `ref`/`source_ref` y omite `source_refs`,
  el adaptador deriva una lista compacta deduplicada sin perder las citas;
- `topic_summary` conserva `markdown` para no romper contratos existentes;
- `audio_tema`, `generacion_audio`, `tts_topic` y equivalentes se normalizan a
  `audio_asset` sin exponer proveedor, modelo, GPU ni rutas;
- la entrega sigue bloqueada si el payload no es materializable.

Validacion director `crear_app_completa` con mas manga 2026-05-22:

```bash
go test -count=1 ./modulos/orquesta-app-director-intake ./modulos/orquesta-app-codex-stack
```

Resultado local: `ok`.

Cobertura:

- `TestPrepareAppDirectorIntakeV0PropagaContextoFuncionalEnSummary` fija que
  el summary del director lleva contexto funcional compacto de `AppSpecV0`;
- `TestDirectorTaskV0IncluyeContratoDeDecisionesEjecutables` exige que
  `crear_app_completa` normal no use `CONSULTA AL DIRECTOR` para evitar una
  tarea de programacion si ya hay objetivo suficiente;
- `TestCompositeDirectorDecisionSourceV0NormalizaOpenPhaseConPhaseDestino`
  cubre el caso real en que el agente pone la fase destino como
  `decision.phase_id` de `open_phase`;
- `TestCompositeDirectorDecisionSourceV0NormalizaPublishContractDecisionRefAlAcceptPrevio`
  cubre el caso real en que el agente apunta el contrato a la decision de
  publicacion en vez del `accept_decision` previo;
- `TestCompositeDirectorDecisionSourceV0NoInventaPublishContractSinAcceptPrevio`
  confirma que la tolerancia no inventa causalidad si no hay
  `accept_decision` previo;
- `TestCompositeDirectorDecisionSourceV0NoRompeWriteSetMaximoPorCompletarWeb`
  cubre el caso real en que el director entrega una tarea amplia ya valida y
  Orquesta no debe invalidarla anadiendo otra ruta al `write_set`;
- `TestDrainRunV0ConsumeDecisionFileVerticalGoConGlobsDeDirectorReal` reproduce
  una salida realista del director con una tarea vertical Go amplia y comprueba
  que se materializa y lanza programacion;
- `TestCodexRuntimeConfigV0ConservaReasoningEffortMedium`,
  `TestCodexStackCapacityConfigFromEnvV0ConservaReasoningCodexMedium`,
  `TestCodexLaunchWaveCommandV0DryRunNoExigeCodexReal` y
  `TestCodexLaunchDirectorWaveCommandV0DryRunConstruyePlanYPromptsPorAgente`
  fijan que los agentes reales no bajan de `high` aunque el operador haya
  dejado `medium`/`low` en el entorno o flags;
- el prompt del director sigue por debajo del limite de tamano existente.

Validacion rework por write-set faltante 2026-05-22:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack
```

Resultado local: `ok`.

Cobertura:

- `TestCodexStackV0ReviewGateAceptaDestinoWriteSetFaltanteComoAviso` fija que
  un destino ausente como `web` queda como rail blando aceptado;
- `TestReviewReworkReplanSourceV0DescribeWriteSetFaltanteParaElAgente` fija
  que el follow-up de rework dice que hay que completar el faltante;
- `TestProgrammingTaskV0IncluyeContextoDeReworkSinRehacerTodo` fija que el
  agente recibe instrucciones de conservar lo valido y corregir lo indicado;
- `TestCompositeDirectorDecisionSourceV0NoDuplicaWebSiWebAdminInternoLoCubre`
  y `TestCodexStackRealSmokeProjectTargetExistsV0AceptaWebAdminInternoComoWeb`
  cubren la equivalencia `web` -> `internal/webadmin` en apps Go;
- el smoke real multiagente queda preparado para abrir revision y esperar el
  rework antes de la verificacion final.

Validacion sanitizador local de contexto sensible:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestLocalSensitiveDataSanitizerV0|Test.*Sanitizer|Test.*Egress'
```

Cobertura:

- `TestLocalSensitiveDataSanitizerV0SaneaContextoAntesDelPacket` verifica que
  el packet no conserva token, secreto, ruta HOME ni prompt literal y que deja
  evidencia durable con policy publica de saneamiento;
- `TestLocalSensitiveDataSanitizerV0DudaYActivaRevisionDirector` verifica que
  material privado ambiguo se degrada a refs y agrega criterios de revision por
  director/humano con policy de consulta al director.
- `TestEgressSanitizerConfigV0CentralizaRuntimePorProveedor` verifica la
  proyeccion canonica de Codex, Gemini, Claude y egress sanitizer sin filtrar
  rutas crudas.
- `TestEgressSanitizerConfigV0InyectaPrivacyFilterLocalOptIn` verifica que
  `openai_privacy_filter_local` se declara como modelo local opt-in y que la
  evidencia conserva categoria sin guardar el secreto.
- `TestLocalSensitiveDataSanitizerV0NoBloqueaPorProviderModelRuntime` fija que
  palabras blandas de runtime/config no bloquean ni fuerzan revision.
- `TestEgressSanitizerConfigV0RespetaSanitizerExplicito` fija que un puerto
  explicito prevalece sobre la config canonica.

Validacion de calidad `domain_work` con issues estructurados:

```bash
GOCACHE=/tmp/orquesta-go-cache go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestValidateDomainWorkDeliveryQualityV0|TestDefaultDomainWorkArtifactSubmissionBuilderV0'
GOCACHE=/tmp/orquesta-go-cache go test -count=1 ./modulos/orquesta-domain-work
```

Cobertura:

- `TestValidateDomainWorkDeliveryQualityV0ReportaIssueEstructuradoDePalabras`
  fija que el rechazo por palabras minimas conserva campo causal
  `payload.chapters.blocks` y conteos observados;
- los rechazos por placeholder visible y `document_plan` incompleto exponen un
  `DomainWorkIssueV0` local sin cambiar el error publico compatible
  `domain_work_artifact_quality_gate_failed`.
