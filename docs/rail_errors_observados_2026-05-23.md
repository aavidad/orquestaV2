# Errores de rail observados 2026-05-23

Objetivo: acumular errores reales de rails/validadores para convertirlos en
casos de prueba rapidos antes de compilar servidor o ejecutar Orquesta completa.

Comando rapido:

```bash
./scripts/test_rails_fast.sh
```

La matriz `TestRailRecordConcurrencyGateExternalMatrixV0` incluye tambien
terminos historicamente problematicos como falsos positivos. La lista operativa
vigente del core se centraliza en
`modulos/orquesta-rails/text_policy_v0.go`: permite palabras
opacas como `runtime`, `provider`, `model`, `codex`, `git`, `db` o `sql` y solo
corta patrones que parecen llevar valor sensible efectivo, por ejemplo
`api_key=`, `client_secret=`, `authorization:`, `bearer ` o material tipo
`-----BEGIN`.

## Formato

```text
ID:
Fecha:
Sintoma:
Campo:
Payload minimo:
Decision:
Test:
Estado:
```

## Casos

```text
ID: RAIL-20260523-001
Fecha: 2026-05-23
Sintoma: `RecordConcurrencyGate: detalle_prohibido`.
Campo: `payload.evidence_refs` / `payload.summary`.
Payload minimo: `secrets_policy`, `oauth-client-secret-policy-doc`, `credential`.
Decision: abrir filtro global por palabras en `RecordConcurrencyGate`; revisar en futuro con sanitizador/clasificador por campo.
Test: `TestRailRecordConcurrencyGateExternalMatrixV0/secrets-policy-reference`.
Estado: cubierto
```

```text
ID: RAIL-20260523-002
Fecha: 2026-05-23
Sintoma: `RecordConcurrencyGate: detalle_prohibido` al crear evento.
Campo: `ConcurrencyGateRecorded.payload`.
Payload minimo: evento de gate con refs opacas que contienen `secrets_policy`.
Decision: excluir `ConcurrencyGateRecorded` del filtro global de eventos; mantener validacion estructural del evento.
Test: `TestRailRecordConcurrencyGateExternalMatrixV0/secrets-policy-reference`.
Estado: cubierto
```

```text
ID: RAIL-20260523-003
Fecha: 2026-05-23
Sintoma: `director_cycle_step_invalido: step.run.command_effects: run invalido`.
Campo: `run.command_effects` y `run.concurrency_gates`.
Payload minimo: `request-ref-app-completion-loop` dentro de subject refs/proyecciones del gate.
Decision: no pasar `concurrency_gates` ni `command_effects` por filtro global de palabras; son proyecciones/efectos estructurados con refs opacas y hashes.
Test: `TestRailRecordConcurrencyGateExternalMatrixV0/completion-run-ref`.
Estado: cubierto
```

```text
ID: RAIL-20260523-004
Fecha: 2026-05-23
Sintoma: `RequestCapacity: detalle_prohibido` bloquea cola de automejora con tareas listas.
Campo: `payload.task_ref`, `payload.summary`, `payload.evidence_refs` y outbox de capacidad/agente.
Payload minimo: refs o textos operativos con `runtime`, `provider`, `model`, `codex`, `$HOME` u OAuth como politica/ref opaca.
Decision: abrir validacion de RequestCapacity/RequestAgent/outbox/director-agent para permitir detalles operativos opacos y cortar solo patrones sensibles efectivos (`client_secret=`, `api_key=`, `authorization: Bearer`, etc.).
Test: `TestRequestCapacityCommandV0PermiteDetallesOperativosOpacos`, `TestRequestAgentCommandV0PermiteDetallesOperativosOpacos` y rechazos sensibles asociados.
Estado: cubierto
```

```text
ID: RAIL-20260524-005
Fecha: 2026-05-24
Sintoma: `detalle_prohibido` durante ingesta/review/cierre despues de entregar agentes.
Campo: `RegisterDelivery`, `RequestReview`, `ReviewResult`, `AcceptReview`, `CloseTask`, `FinalValidation`, `CloseRun`, gates y eventos globales.
Payload minimo: resumenes o refs de entregas que mencionan `runtime`, `provider`, `model`, `codex`, `git`, `db`, `sql`, `prompt policy` o `transcript policy` como contexto opaco.
Decision: centralizar la lista de patrones sensibles en `modulos/orquesta-rails/text_policy_v0.go` y hacer que los validadores de workflow la usen por helper comun; se eliminan listas locales por comando.
Test: `go test -count=1 ./modulos/orquesta-core-workflow`.
Estado: cubierto
```

```text
ID: RAIL-20260524-006
Fecha: 2026-05-24
Sintoma: `director_supervised_burst_step_input: field=step_input_builder: detalle_prohibido`.
Campo: payload de ciclo del director y datos de runtime persistidos.
Payload minimo: trazas locales con detalles operativos reales o de test
(`access_token=...`, `Bearer ...`, HOME, prompt/transcript policy) dentro de
estado/log de runtime.
Decision: abrir temporalmente todos los rails de detalle por
`ORQUESTA_DETAIL_PROHIBITED_RAILS=off`; el servidor usa ese valor por defecto
si no hay valor explicito. No borrar listas ni evidencias. Reendurecer solo con
matrices externas amplias y preservando detalle crudo local para diagnostico.
Test: `TestTextPolicyV0DetalleProhibidoDesactivadoTemporalmente`,
`TestServerEnvironmentWithDetailRailsDefaultV0AddAperturaSiFalta`.
Estado: cubierto como apertura temporal; pendiente matriz de reactivacion.
```

```text
ID: FLAKE-20260523-001
Fecha: 2026-05-23
Sintoma: `go test -count=1 ./...` fallo una vez en `cmd/orquesta-server`: stdout no contenia prompt ejecutado para un agente fake recursivo.
Campo: `TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje`.
Payload minimo: pendiente de aislar.
Decision: tratar como prioridad de fiabilidad; investigar no determinismo en stdout/orden/concurrencia/estado temporal.
Test: repeticion focal `go test -count=1 ./cmd/orquesta-server -run TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje -v`.
Estado: registrado, pendiente de diagnostico
```

## Candidatos pendientes de convertir en matriz

```text
ID: RAIL-CAND-CORE-EVENTS-001
Origen: revision paralela 2026-05-23.
Casos: `RunBlocked.summary` con `completion`, `token budget`, `prompt policy ref`
o `transcript policy ref`.
Decision pendiente: permitir refs/politicas opacas pero seguir rechazando
contenido crudo (`access_token=...`, keys `prompt`, `raw_text`, `transcript`).
Test futuro: `TestRailEventPayloadExternalMatrixV0`.
```

```text
ID: RAIL-CAND-WORKFLOW-TASK-001
Origen: revision paralela 2026-05-23.
Casos: `WorkflowTaskV0` y `CreateMicrotask` con write-set/context refs de
`modulos/orquesta-runtime`, `provider interface`, `modelado de dominio`,
`git diff` y `token budget`.
Decision pendiente: permitir referencias opacas de arquitectura/repo; reservar
rechazo para secretos efectivos o paths inseguros.
Test futuro: `TestRailWorkflowTaskExternalMatrixV0`.
```

```text
ID: RAIL-CAND-CONTEXT-001
Origen: revision paralela 2026-05-23.
Casos: `ContextBundleRequestV0` con `task_kind=web_application`,
`phase=programming`, `capacity_level=normal` y
`write_set=modulos/orquesta-runtime`.
Decision pendiente: normalizar alias razonables en adaptador/director; no
rechazar el modulo real `orquesta-runtime` por palabra.
Test futuro: matriz rapida en `orquesta-context`.
```

```text
ID: RAIL-CAND-DIRECTOR-AGENT-001
Origen: revision paralela 2026-05-23.
Casos: `DirectorAgentDecisionV0` con `summary` que menciona `codex adapter`,
`runtime`, `model`, `provider`, y `record_review_result` con
`accepted_with_notes`.
Decision pendiente: distinguir alias reparables y refs opacas de proveedor real;
normalizar estados equivalentes fuera del core.
Test futuro: matriz rapida en `orquesta-director-agent`.
```

```text
ID: RAIL-CAND-RUNTIME-001
Origen: revision paralela 2026-05-23.
Casos: rol con espacios (`frontend engineer`), `runtime_kind=local`,
`capacity=normal`, `provider_ref=openai`, `model_ref=gpt-5`.
Decision pendiente: normalizar alias de rol/runtime/capacidad; mantener
proveedor/modelo concretos como refs opacas o adaptador opt-in.
Test futuro: matriz rapida en `orquesta-runtime`.
```

```text
ID: FLAKE-CAND-SERVER-001
Origen: revision paralela 2026-05-23.
Casos: stress de `TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje`
con `-count=50 -shuffle=on`, bloque `codex_wave` con procesos fake y `-race`
en `orquesta-runtime`.
Decision pendiente: crear harness rapido de flakes y esperar cierre real de
stdout/ficheros en vez de asumir que `last_message` implica stdout completo.
Test futuro: `./scripts/test_flakes_fast.sh`.
```

```text
ID: RAIL-CAND-DETAIL-REACTIVATION-001
Origen: scanner backlog 2026-05-24.
Casos: `ORQUESTA_DETAIL_PROHIBITED_RAILS=off` quedo como apertura temporal por
defecto tras `director_supervised_burst_step_input: detalle_prohibido`.
Decision pendiente: no reactivar el rail global por env sin matriz externa por
campo. La reactivacion debe permitir refs opacas, vocabulario operativo y
politicas de prompt/transcript, y cortar solo valores sensibles efectivos o
material crudo.
Test futuro: `./scripts/test_rails_fast.sh` mas focos de `orquesta-context`,
`orquesta-director-agent`, `orquesta-core-workflow` y `cmd/orquesta-server`.
Backlog: `T15 detail-rails-reactivation`.
Actualizacion 2026-05-24 cuarta pasada: `cmd/orquesta-server` ya reactiva por
defecto el rail con `ORQUESTA_DETAIL_PROHIBITED_RAILS=on` y scope acotado. El
pendiente ya no es "reactivar todo", sino sincronizar docs/matriz y evitar que
esa reactivacion se use para persistir payloads crudos o bloquear vocabulario
operativo fuera de los scopes probados.
```

```text
ID: RAIL-CAND-AUDIT-PAYLOADS-001
Origen: scanner backlog 2026-05-24 tercera pasada.
Casos: la auditoria JSONL del servidor ya guarda metadata HTTP y diagnostics de
drain, pero el pendiente documentado propone captura opt-in de payloads HTTP
completos. Sin politica por campo, esa captura puede duplicar cuerpos grandes,
transcripts, prompt material, tokens o datos privados en una persistencia
paralela.
Decision pendiente: mantener payload completo apagado por defecto; si se activa,
pasar por sanitizador/rail por campo, retencion corta, evidencia de redaccion y
frontera local de composicion. No usar `ORQUESTA_DETAIL_PROHIBITED_RAILS=off`
como autorizacion para persistir material crudo.
Test futuro: focos de `orquesta-server`, `cmd/orquesta-server` y web/API con
payload sensible simulado y auditoria filtrable.
Backlog: `T19 server-audit-ops-surface`.
```

```text
ID: RAIL-CAND-SQL-REAL-DIALECT-001
Origen: scanner backlog 2026-05-24 tercera pasada.
Casos: el adaptador `orquesta-domain-work-sql` acepta `database/sql` inyectado,
pero el bundle real futuro necesitara DSN, driver, schema y errores de dialecto.
El vocabulario `db`, `sql`, `driver` o `dsn` no debe bloquear refs opacas, pero
valores reales de DSN, passwords o URLs con credenciales no pueden acabar en
eventos, ACKs, audit logs ni docs de smoke.
Decision pendiente: probar dialectos reales solo en composicion opt-in con DSN
externo y redaccion obligatoria; el core/domain-work no importa drivers ni
persistencia global.
Test futuro: contract suite de `DomainWorkJobRecordStorePortV0` contra DB
temporal y caso negativo de secreto en DSN/auditoria.
Backlog: `T20 domain-work-sql-real-dialect-bundle`.
```

```text
ID: RAIL-CAND-REAL-SMOKE-GUARDS-001
Origen: scanner backlog 2026-05-24 tercera pasada.
Casos: la matriz marca smokes reales con distintos niveles de guarda. Algunas
rutas tienen confirm env y OPES temporal obligatorio; `USAGE-METRICS-REAL`
declara que su script no tiene guarda propia aunque puede lanzar Codex real.
Decision pendiente: catalogar smokes por riesgo y exigir confirmacion explicita
para proveedor, OPES, DB, red o efectos externos antes de que el residente los
pueda seleccionar. Un smoke real sin guarda debe quedar excluido de ejecucion
automatica o fallar temprano con bloqueo verificable.
Test futuro: `bash -n scripts/*.sh`, prueba de catalogo en `cmd/orquesta-server`
y caso que rechaza ejecutar smoke real sin confirm env.
Backlog: `T21 real-smoke-guards-and-catalog`.
```

```text
ID: RAIL-CAND-DOMAIN-TESTS-001
Origen: scanner backlog 2026-05-24.
Casos: cierre `domain_work` no-OPES cubierto con app temporal y validador fake;
falta politica productiva por puerto para evidencias/tests de dominio generico.
Decision pendiente: mantener los validadores de dominio fuera del nucleo y
aceptar solo refs/artefactos causales; URL, DB, ruta local o nombre de conector
no son evidencia suficiente de cierre.
Test futuro: focos de `orquesta-domain-work`, `orquesta-app-change`,
`orquesta-app-director-service` y `orquesta-app-codex-stack`.
Backlog: `T14 domain-work-required-tests-productivos`.
```

```text
ID: RAIL-CAND-BACKLOG-STALE-001
Origen: scanner backlog 2026-05-24 segunda pasada.
Casos: `T08`, `T09` y `T10` tenian evidencia local cerrada en docs/codigo, pero
el backlog no incluia `Estado: completada`; si el ACK runtime no esta presente,
el planner puede volver a preparar trabajo ya cerrado.
Decision pendiente: sincronizar backlog con ACKs durables, docs locales de
modulo y cola visible; si la evidencia es ambigua, crear revision documental
acotada en vez de ejecucion amplia.
Test futuro: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`
con caso de seccion cerrada por evidencia documental y ACK ausente.
Backlog: `T17 autoprogramming-backlog-state-sync`.
```

```text
ID: RAIL-CAND-LOCAL-LISTS-002
Origen: scanner backlog 2026-05-24 segunda pasada.
Casos: quedan rails/listas locales fuera del inventario inicial en
`orquesta-app-codex-stack` para refs de contexto externo y evidencia de
review/replan, y fronteras de operador/leases que usan reglas propias o wrappers
locales. Pueden ser fronteras legitimas, pero deben quedar clasificadas antes de
reactivar detalle global.
Decision pendiente: registrar propietario por modulo, matriz externa y relacion
con `orquesta-rails`; no endurecer por vocabulario operativo ni relajar fronteras
de efecto externo sin caso reproducible.
Test futuro: `./scripts/test_rails_fast.sh` mas focos de
`orquesta-app-codex-stack`, `orquesta-operator-mcp` y `orquesta-core-leases`.
Backlog: `T15 detail-rails-reactivation`.
```

```text
ID: RAIL-CAND-DETAIL-DOC-STATE-002
Origen: scanner backlog 2026-05-24 cuarta pasada.
Casos: el codigo de `cmd/orquesta-server` fija
`ORQUESTA_DETAIL_PROHIBITED_RAILS=on` por defecto con scope acotado, mientras
docs de rails fuera de este write-set aun describen el servidor como `off` por
defecto. Esa divergencia puede hacer que el planner reabra T15 de forma
incorrecta o que un operador reactive/desactive rails con una foto antigua.
Decision pendiente: sincronizar registro vivo, rail errors, duplicaciones y
matriz rapida; conservar `off` solo como override explicito del operador y no
como estado declarado del servidor.
Test futuro: `./scripts/test_rails_fast.sh` y
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-rails`.
Backlog: `T22 detail-rails-doc-state-sync`.
```

```text
ID: RAIL-CAND-MCP-RUN-TRANSPORT-001
Origen: scanner backlog 2026-05-24 cuarta pasada.
Casos: el smoke `mcp-real-smoke` cierra transporte MCP temporal por comando,
pero no decide si el servidor residente debe exponer `/mcp` durante `run`.
Si se activa sin guardas, podria duplicar superficies HTTP/MCP, auditoria y
payloads; si no se activa, debe quedar documentado que el comando temporal es la
frontera suficiente.
Decision pendiente: decidir transporte residente opt-in por composicion, con
bind/loopback explicito, payload compacto, errores publicos y auditoria sin
secretos ni transcripts.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client ./cmd/orquesta-server`.
Backlog: `T23 mcp-run-transport-opt-in`.
```

```text
ID: RAIL-CAND-STARTUP-ACK-TEXT-001
Origen: scanner backlog 2026-05-24 quinta pasada.
Casos: compactacion de arranque en `cmd/orquesta-server` detecta ACK
completado leyendo `codex_last_message.txt` y buscando `ack` + `completed`.
Decision pendiente: usar `agent_ack.json`/receipt estructurado y correlado como
fuente de verdad; dejar `codex_last_message.txt` solo como diagnostico.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack`.
Backlog: `T24 startup-structured-ack-compaction`.
```

```text
ID: RAIL-CAND-MCP-ROADMAP-STALE-001
Origen: scanner backlog 2026-05-24 quinta pasada.
Casos: recursos MCP `orquesta.project.roadmap.v0` y
`orquesta.shared_contracts.v0` exponen estados `pendiente_*` hardcodeados que
pueden divergir del backlog vivo y la foto vigente.
Decision pendiente: sincronizar recursos MCP con backlog/docs vigentes o marcar
entradas historicas con freshness/source_refs para no relanzar trabajo cerrado.
Test futuro: `go test -count=1 ./modulos/orquesta-mcp`.
Backlog: `T25 mcp-roadmap-backlog-state-sync`.
```

```text
ID: RAIL-CAND-DOMAIN-QUALITY-001
Origen: scanner backlog 2026-05-24 quinta pasada.
Casos: `topic_expansion_package` usa strings genericas (`TODO`, `placeholder`,
`pendiente_revision`) y recuento de palabras en el stack Codex para decidir
calidad de dominio.
Decision pendiente: mover la politica a puerto/adaptador de dominio con matriz
por campo, issues estructurados y tolerancia a vocabulario valido.
Test futuro:
`go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-app-codex-stack ./modulos/orquesta-opes-bridge`.
Backlog: `T26 domain-work-quality-policy-port`.
```

```text
ID: RAIL-CAND-DOC-STATE-DRIFT-001
Origen: scanner backlog 2026-05-24 sexta pasada.
Casos: documentos de entrada obligatoria presentan estado contradictorio sobre
`CODEX-WAVE-REAL` y `CODEX-RECURSION-REAL`: unas fuentes los declaran cerrados
con proveedor y otras mantienen recursion Codex productiva como pendiente.
Decision pendiente: fijar orden de autoridad documental, sincronizar estado y
marcar historico cualquier documento que no sea foto vigente. El frente real
abierto debe seguir siendo OPES temporal de derivados/cierre, salvo regresion
demostrada de Codex.
Test futuro: prueba documental focal que detecte contradicciones conocidas entre
`AGENTS.md`, `docs/estado_actual_2026-05-17.md`, guia, matriz y backlog.
Backlog: `T27 docs-source-of-truth-state-sync`.
```

```text
ID: RAIL-CAND-RESTART-LIVE-AGENTS-001
Origen: scanner backlog 2026-05-24 sexta pasada.
Casos: `DAEMON-RESTART` valida cola file-based sin agentes reales y la matriz
declara que no cubre rehidratacion de procesos Codex vivos. El contrato runtime
mantiene `resume` fuera de `launch_mode`; si el arranque archiva o completa por
heuristicas locales puede perder trabajo vivo.
Decision pendiente: definir contrato de resume/reconciliacion por refs opacas,
descriptor de runtime, ACK estructurado, checkpoint y wait scope; no cerrar por
`codex_last_message.txt`, summary textual ni ausencia transitoria de pending.
Test futuro: smoke opt-in con agente Codex temporal, reinicio de servidor y
reconciliacion de ACK/delivery/review/cierre o bloqueo causal.
Backlog: `T28 restart-live-agent-reconciliation`.
```

```text
ID: RAIL-CAND-PROVIDER-USAGE-ACCOUNTING-001
Origen: scanner backlog 2026-05-24 sexta pasada.
Casos: telemetry fake y stats REST ya exponen uso, pero falta conector
productivo de cuota/tokens por proveedor. Sin puerto explicito, coste real,
modelo, cuenta o detalles de proveedor pueden mezclarse con stats, auditoria o
ACKs.
Decision pendiente: separar uso/coste por puerto opt-in, redaccion por campo y
errores publicos cuando no haya fuente productiva. El nucleo conserva refs
opacas; proveedor/modelo/cuenta/coste concreto viven en adaptador.
Test futuro:
`go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp ./modulos/orquesta-web ./cmd/orquesta-server`.
Backlog: `T29 provider-usage-quota-accounting`.
```

```text
ID: RAIL-CAND-SHUTDOWN-GENERIC-001
Origen: scanner backlog 2026-05-24 septima pasada.
Casos: apagado cooperativo ya valida `codex_shutdown_checkpoint_ack.v0` con
Codex real, pero el protocolo sigue ligado a ficheros/prompt Codex. Al elevarlo
a runtimes genericos, vocabulario como `shutdown`, `checkpoint`, `runtime`,
`provider` o nombres de ficheros de control no debe disparar falsos positivos.
Decision pendiente: definir contrato neutral de checkpoint por puerto y aplicar
rails por campo: permitir refs/ficheros de control y cortar solo secretos
efectivos, rutas privadas, prompts/transcripts o payloads de proveedor.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-server-shutdown ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T30 runtime-shutdown-checkpoint-port`.
```

```text
ID: RAIL-CAND-RUN-QUEUE-CLAIM-001
Origen: scanner backlog 2026-05-24 septima pasada.
Casos: la cola global ordena y prioriza, pero no reserva candidatos. Un contrato
de claim/lease puede introducir `owner_ref`, `lease_ref`, `ttl`, `heartbeat` y
razones de skip que deben ser refs/metadata opacas, no datos locales de proceso.
Decision pendiente: permitir metadata compacta de lease en cola y supervisor,
rechazar PID/HOME/rutas/host como verdad del nucleo y probar conflictos de
reserva sin relanzar agentes.
Test futuro:
`go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-memory ./modulos/orquesta-run-file ./modulos/orquesta-run-supervisor ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T31 run-queue-reservation-lease`.
```

```text
ID: RAIL-CAND-RESIDENT-WAKEUP-001
Origen: scanner backlog 2026-05-24 septima pasada.
Casos: la politica event-driven pide actuar por senales reales, pero el loop
residente actual se apoya en ticks. Los wakeups futuros pueden traer causas de
cola, delivery, shutdown o capacidad; si transportan payloads crudos duplicarian
auditoria y rails de detalle.
Decision pendiente: wakeup compacto por refs opacas, causa y contadores; sin
prompts, transcripts, payloads HTTP completos, rutas locales, HOME ni tokens.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-run-supervisor ./modulos/orquesta-run-queue ./cmd/orquesta-server`.
Backlog: `T32 resident-supervisor-event-driven-wakeup`.
```

```text
ID: RAIL-CAND-BACKLOG-ACK-001
Origen: scanner backlog 2026-05-24 octava pasada.
Casos: el planner de automejora considera completada una request de backlog si
encuentra `agent_ack.json` bajo `.orquesta-runtime` con `status=completed`,
sin validar schema, correlacion, ack_ref, task_ref ni pruebas obligatorias.
Decision pendiente: correlacionar ACK contra spec/packet o marcarlo ambiguo; no
ocultar secciones pendientes por ACK incompleto, antiguo o de otra request.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack`.
Backlog: `T33 autoprogramming-backlog-ack-correlation`.
```

```text
ID: RAIL-CAND-DIRECTOR-WAVE-GUARDS-001
Origen: scanner backlog 2026-05-24 octava pasada.
Casos: `codex-director-wave` deja `--strict-director-guards` desactivado por
defecto; en modo no estricto rellena `write_set=.` y tests placeholder, y el
prompt permite ampliar ficheros fuera del shard por justificacion textual.
Decision pendiente: modo estricto por defecto para ejecucion real, opt-in
explicito para write-set raiz o tests placeholder, y prompts que separen shard
flexible de alcance autorizado total.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-director-operativo ./modulos/orquesta-app-codex-stack`.
Backlog: `T34 director-wave-strict-guards-default`.
```

```text
ID: RAIL-CAND-STARTUP-CLEANUP-001
Origen: scanner backlog 2026-05-24 octava pasada.
Casos: el servidor fija `ORQUESTA_STARTUP_CLEANUP_MODE=forced_stop` si no hay
valor explicito. En un reinicio con agentes vivos o ACK ambiguos, esa politica
puede compactar cola/control como parada logica antes de reconciliar runtime.
Decision pendiente: default conservador de bloqueo/reconciliacion; `forced_stop`
solo opt-in con evidencia, causa publica y contadores de kept/blocked/compacted.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-run-control ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-state-file`.
Backlog: `T35 startup-cleanup-safe-mode`.
```

```text
ID: RAIL-CAND-ACK-STRICT-001
Origen: scanner backlog 2026-05-24 novena pasada.
Casos: `ValidateCodexAgentAckBytesForSpecV0` rellena campos de identidad desde
la spec antes de validar, y `ReadCodexDeliveryObservationFileV0` tiene cobertura
que acepta un ACK minimo con solo `schema_version` y `status=completed` si la
spec aporta defaults.
Decision pendiente: separar modo diagnostico/legacy de modo terminal estricto;
un ACK terminal nuevo debe traer refs explicitas, tests requeridos y correlacion
propia antes de cerrar delivery, startup compaction o backlog.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T36 codex-ack-strict-terminal-validation`.
```

```text
ID: RAIL-CAND-BACKLOG-PARSER-TESTS-001
Origen: scanner backlog 2026-05-24 novena pasada.
Casos: el planner de backlog solo extrae comandos `go test` desde `Tests:` y
marca secciones como completadas por narrativa amplia como `revalidacion final`.
Se pierden verificaciones declaradas como `git diff --check`,
`bash -n scripts/*.sh`, `./scripts/test_rails_fast.sh` o prueba documental
focal.
Decision pendiente: parsear estado canonico y comandos de verificacion completos
o declarar blocker manual; no sustituir pruebas de la seccion por el test base
ni ocultar trabajo por texto narrativo.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Backlog: `T37 autoprogramming-backlog-parser-fidelity`.
```

```text
ID: RAIL-CAND-CODEX-OBSERVATION-OWNER-001
Origen: scanner backlog 2026-05-24 novena pasada.
Casos: `codexDeliveryObservationUnsafeForCoreV0` mantiene una lista local que
detecta `codex`, `runtime`, `provider`, `git`, `db` o `sql` en observaciones,
mientras la politica comun de `orquesta-rails` ya distingue vocabulario opaco de
secretos efectivos.
Decision pendiente: decidir si ese helper se elimina como detector historico o
se convierte en politica comun por campo; no duplicar listas ni bloquear refs
opacas validas.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-rails ./modulos/orquesta-app-codex-stack`.
Backlog: `T38 codex-delivery-observation-rail-owner`.
```

```text
ID: RAIL-CAND-APP-VCS-EFFECT-GUARDS-001
Origen: scanner backlog 2026-05-24 decima pasada.
Casos: `orquesta.app_vcs.v0` permite `commit`/`push` por MCP/HTTP y el conector
Git hace `git add -A` cuando `commit_paths` esta vacio. El contrato actual no
transporta `write_set`, cierre/review aceptada, tests requeridos ni evidencia de
solape cero.
Decision pendiente: exigir paths/write-set y evidencia causal para commit/push;
dejar `git add -A` y push remoto como opt-in auditado, con redaccion de salida
Git antes de devolver errores o auditar.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-worktree ./cmd/orquesta-server`.
Backlog: `T39 app-vcs-write-set-evidence-guards`.
```

```text
ID: RAIL-CAND-PROMOTION-E2E-STATE-001
Origen: scanner backlog 2026-05-24 decima pasada.
Casos: T13 sigue documentado como pendiente aunque ya existen puerto neutral,
conector Git opt-in, wiring de servidor y pruebas unitarias/fake del stack.
Falta e2e acotado con repo temporal y estado documental que distinga piezas
cerradas, promocion local, archivo idempotente, push pendiente y pendientes
productivos.
Decision pendiente: cerrar T13 con prueba de composicion temporal y recibo
durable; no marcar la cola como terminal promocionada si promocion, push o
archivo queda pendiente/bloqueado.
Test futuro:
`go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T40 autoprogramming-promotion-real-e2e-and-doc-state`.
```

```text
ID: RAIL-CAND-SERVER-AUDIT-PAYLOAD-002
Origen: scanner backlog 2026-05-24 decima pasada.
Casos: `auditEventV0` recibe `payload any` y eventos del supervisor/automejora
guardan structs completos de request/result. Aunque HTTP solo audita metadata,
los payloads internos pueden crecer hacia prompts, transcripts, rutas privadas,
remotos Git o cuerpos de dominio si no hay contrato por evento.
Decision pendiente: sanitizador/esquema por evento antes de escribir JSONL;
guardar refs, contadores, estados y errores publicos, no material crudo.
Mantener payload HTTP completo apagado salvo opt-in con redaccion y retencion
corta.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-observability ./cmd/orquesta-server`.
Backlog: `T41 server-audit-payload-redaction-contract`.
```

```text
ID: RAIL-CAND-STRICT-ACK-WRITESET-001
Origen: scanner backlog 2026-05-24 undecima pasada.
Casos: `ValidateCodexAgentAckBytesForSpecV0` hidrata identidad desde la spec y
la cobertura actual acepta `files` fuera del write-set como rail blando. El
paquete OrquestaV2 externo exige modo estricto: no editar fuera del write-set,
no declarar archivos de control y fallar con `CONSULTA AL DIRECTOR` si falta
alcance.
Decision pendiente: separar modo legacy/advisory de modo terminal estricto por
policy/packet; en estricto, `completed` requiere files reales dentro de alcance,
tests requeridos pasados y evidencia de snapshot si esta disponible.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T42 codex-ack-strict-write-set-terminal-proof`.
```

```text
ID: RAIL-CAND-BACKLOG-SCANNER-MERGE-001
Origen: scanner backlog 2026-05-24 undecima pasada.
Casos: los scanners de automejora usan el mismo write-set documental
(`autoprogramacion`, `rail_errors`, `duplicaciones`) y pueden ejecutarse en
burst. El planner deduplica por refs conocidas/ACK, pero no transporta epoch,
hash de documentos ni lease de merge para bloquear una entrega basada en foto
obsoleta.
Decision pendiente: crear reserva/epoch de scanner y validacion de merge antes
de aceptar ACK; si hay colision, exponer rebase/merge pendiente al director sin
borrar contenido ni marcar completed silencioso.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-run-queue`.
Backlog: `T43 backlog-scanner-doc-merge-lease`.
```

```text
ID: RAIL-CAND-REQUIRED-CONTEXT-REFONLY-001
Origen: scanner backlog 2026-05-24 undecima pasada.
Casos: el paquete del agente puede traer una entrada `required=true` con
`kind=doc_ref`, `mode=ref_only` y `total_bytes=0`. Para tareas que dependen de
docs obligatorios, cerrar con `completed` sin evidencia de lectura/materializacion
puede convertir falta de contexto en backlog inventado o incompleto.
Decision pendiente: distinguir `required ref_only` permitido por diseno de
contexto obligatorio no materializado; exigir evidencia de lectura, resolucion
de contexto truncado o `CONSULTA AL DIRECTOR` antes de cierre terminal.
Test futuro:
`go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-orchestration-core ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T44 required-context-ref-only-guard`.
```

```text
ID: RAIL-CAND-GO-FILE-LINE-BUDGET-001
Origen: scanner backlog 2026-05-24 duodecima pasada.
Casos: el prompt exige ficheros Go por debajo de 300 lineas, pero
`file_too_large` es advisory en `orquesta-autoprogramming` y la matriz acepta
301..520 lineas con followup. El repo ya contiene deuda historica por encima de
300 lineas; sin baseline, endurecer bloquea todo, pero sin modo estricto una
entrega nueva puede agrandar controladores enormes y cerrar como completed.
Decision pendiente: crear baseline de ficheros Go grandes y validar crecimiento
real por snapshot/worktree. Mantener advisory legacy solo para compatibilidad;
en modo OrquestaV2 estricto, bloquear crecimiento nuevo o exigir followup de
particion aceptado por el director.
Test futuro:
`go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T45 autoprogramming-go-file-line-budget-baseline`.
```

```text
ID: RAIL-CAND-PACKET-WRITESET-PRECEDENCE-001
Origen: scanner backlog 2026-05-24 decimotercera pasada.
Casos: el paquete OrquestaV2 trae `write_set_closed` y el prompt Codex ordena no
editar fuera del write-set, pero `programmingObjectiveV0` anade "si debes tocar
otros ficheros del repo ... hazlo y dejalo justificado en el ACK".
Decision pendiente: generar paquetes sin reglas incompatibles. Si el alcance
queda cerrado, una ampliacion requiere decision del director o ACK failed con
`CONSULTA AL DIRECTOR`; no basta una nota.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime`.
Backlog: `T46 agent-packet-write-set-precedence`.
```

```text
ID: RAIL-CAND-ACK-TEST-RECEIPT-001
Origen: scanner backlog 2026-05-24 decimotercera pasada.
Casos: `ValidateCodexAgentAckForSpecV0` exige que `ACK.tests` contenga los
comandos requeridos y detecta evidencia textual de fallo, pero no prueba por si
solo que el agente externo haya ejecutado el comando con exit code exitoso.
Decision pendiente: en modo estricto, completar requires `RequiredTestEvidenceV0`
o recibo estructurado del runtime/adaptador; strings de ACK quedan como
diagnostico o compatibilidad legacy.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-required-test ./modulos/orquesta-app-codex-stack ./modulos/orquesta-orchestration-core ./cmd/orquesta-server`.
Backlog: `T47 external-agent-required-test-receipts`.
```

```text
ID: RAIL-CAND-DESTRUCTIVE-WORKTREE-001
Origen: scanner backlog 2026-05-24 decimotercera pasada.
Casos: `VerifyWorktreeWriteSetV0` detecta paths eliminados y cambios fuera de
write-set, pero la politica terminal no separa truncado fuerte, reemplazo masivo
o rename ambiguo dentro de write-set. El paquete OrquestaV2 prohibe borrar,
mover fuera o truncar archivos existentes sin decision del director.
Decision pendiente: clasificar efectos destructivos desde snapshot/worktree y
bloquear `completed` en modo estricto salvo decision causal explicita.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T48 destructive-worktree-change-proof`.
```

```text
ID: RAIL-CAND-CODEX-SECURITY-PROFILE-001
Origen: scanner backlog 2026-05-24 decimocuarta pasada.
Casos: `cmd/orquesta-server` y `orquesta-app-codex-stack` normalizan sandbox
invalido a `workspace-write` antes de validar, mientras el perfil Codex puro lo
rechaza. Tambien existe `DirectorApprovalPolicy=on-request` en configuracion de
prueba, incompatible con autoprogramacion residente desatendida salvo opt-in de
operador vivo.
Decision pendiente: separar compatibilidad/diagnostico de modo estricto; no
corregir flags de seguridad en silencio y exigir approval no interactivo o
decision explicita del director/operador.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T49 codex-runtime-security-profile-strict-mode`.
```

```text
ID: RAIL-CAND-RUNTIME-CONTROL-WORKTREE-001
Origen: scanner backlog 2026-05-24 decimocuarta pasada.
Casos: el runtime de control puede vivir como `.orquesta-runtime` dentro del
proyecto y el write-set puede ser `.`. Sin exclusion comun, snapshots,
verification, staging promotion o AppVCS pueden tratar prompts, packets, ACKs,
logs o checkpoints como cambios de producto.
Decision pendiente: excluir control dirs y control files de snapshots,
promocion, AppVCS, contexto y ACK terminal, incluso con write-set raiz. La
lectura del propio `agent_packet.json` queda como excepcion de control, no como
artefacto exportable.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T50 runtime-control-files-worktree-exclusion`.
```

```text
ID: RAIL-CAND-CAPACITY-DEFAULT-POLICY-001
Origen: scanner backlog 2026-05-24 decimocuarta pasada.
Casos: varias rutas del servidor y stack Codex defaultan o elevan
`ORQUESTA_CODEX_REASONING_EFFORT` a `high`/`xhigh`, mientras las reglas vigentes
ordenan `medium` por defecto para exploracion y pruebas y reservar `xhigh` para
orden explicita o riesgo tecnico justificado.
Decision pendiente: politica unica por work_profile/dominio/riesgo, con
override auditable para OPES/temarios reales o decisiones amplias, y default
medio para scanners/automejora documental.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./modulos/orquesta-autoprogramming`.
Backlog: `T51 capacity-reasoning-default-policy`.
```

```text
ID: RAIL-CAND-GO-LINE-BUDGET-APP-DIRECTOR-001
Origen: scanner backlog 2026-05-24 decimoquinta pasada.
Casos: medicion local encontro ficheros Go muy por encima de 300 lineas en
`modulos/orquesta-app-director-service`, incluyendo ciclo operativo, continue,
cierre y tests. El rail OrquestaV2 exige no seguir creciendo controladores
enormes, pero falta shard concreto de particion para este modulo.
Decision pendiente: partir por responsabilidad sin cambiar comportamiento y
registrar baseline de deuda historica que no pueda cerrarse en una sola tarea.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-director-service ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file`.
Backlog: `T52 app-director-service-file-split`.
```

```text
ID: RAIL-CAND-GO-LINE-BUDGET-ORCH-CORE-001
Origen: scanner backlog 2026-05-24 decimoquinta pasada.
Casos: `orquesta-orchestration-core` tiene materializador, cierre, plan-state y
runner de tests por encima del rail de 300 lineas. Al ser nucleo de aplicacion,
un refactor mal delimitado puede mezclar puertos neutrales con adaptadores
concretos.
Decision pendiente: separar ficheros por responsabilidad neutral, conservar
puertos/refs opacas y probar idempotencia, WaitAgentRefs, cierre causal y
evidencias requeridas.
Test futuro:
`go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service`.
Backlog: `T53 orchestration-core-file-split`.
```

```text
ID: RAIL-CAND-GO-LINE-BUDGET-CODEX-STACK-001
Origen: scanner backlog 2026-05-24 decimoquinta pasada.
Casos: `modulos/orquesta-app-codex-stack` y `cmd/orquesta-server` acumulan
ficheros largos de composicion, comandos y smokes reales. Sin particion, los
rails de ACK, control files, seguridad Codex, VCS y capacidad pueden volver a
duplicarse dentro de controladores grandes.
Decision pendiente: separar composicion por flujo y mantener los rails T49-T51
como propietarios, sin mover runtime/proveedor/API al nucleo.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery`.
Backlog: `T54 codex-stack-server-file-split`.
```

```text
ID: RAIL-CAND-SERVER-CONTROL-EXPOSURE-001
Origen: scanner backlog 2026-05-24 decimosexta pasada.
Casos: `ORQUESTA_SERVER_ADDR` puede cambiar el bind del servidor y la superficie
HTTP/MCP contiene rutas de control mutables: shutdown, runs/control, cola,
autoprogramacion, AppVCS opt-in y transporte MCP real futuro.
Decision pendiente: loopback por defecto; bind no-loopback solo con opt-in,
principal/credencial o mTLS/TLS de composicion, autorizacion por ruta y auditoria
sin tokens/cabeceras completas/payloads crudos.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-mcp ./cmd/orquesta-server`.
Backlog: `T55 server-control-plane-exposure-guards`.
```

```text
ID: RAIL-CAND-CODEX-CODEHOME-CREDENTIAL-001
Origen: scanner backlog 2026-05-24 decimosexta pasada.
Casos: `codex-wave` y `codex-director-wave` copian `auth.json`, `config.toml`,
skills, plugins, rules y memories desde `CODEX_HOME` a homes de agentes. El
vocabulario `auth`, `token`, `credential`, `HOME`, `skills` o `plugins` no debe
bloquear refs opacas, pero los valores reales no pueden acabar en snapshot,
ACK, auditoria, promocion ni contexto de producto.
Decision pendiente: politica de proyeccion por fichero/categoria, allowlist,
evidencia compacta de categorias copiadas, redaccion de errores y bloqueo
recuperable si falta auth/config requerida.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack`.
Backlog: `T56 codex-code-home-credential-projection`.
```

```text
ID: RAIL-CAND-OUTBOX-DISPATCH-ACK-RECOVERY-001
Origen: scanner backlog 2026-05-24 decimosexta pasada.
Casos: outbox dispatch ya tiene contratos puros de claim/lease/batch/ACK, pero
la composicion residente debe demostrar recuperacion durable ante restart,
claim expirado, ACK parcial y retry sin duplicar efectos externos.
Decision pendiente: persistir/reconciliar `message_id`, `run_id`,
`target_port`, `claim_ref`, `lease_ref` y ACK correlacionado; conservar
`pending`/`ack_failed` con causa publica y no cerrar planes con outbox pendiente.
Test futuro:
`go test -count=1 ./modulos/orquesta-outbox-dispatch ./modulos/orquesta-orchestration-core ./modulos/orquesta-persistence ./modulos/orquesta-app-director-service ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T57 outbox-dispatch-durable-ack-recovery`.
```

```text
ID: RAIL-CAND-CODEX-WAVE-TAIL-RAW-LOG-001
Origen: scanner backlog 2026-05-24 decimoseptima pasada.
Casos: `codex-wave-tail` imprime fragmentos crudos de `codex_stdout.log`,
`codex_stderr.log` o `codex_last_message.txt`. Es diagnostico util, pero puede
exponer prompts, transcripts, HOME, rutas privadas, tokens, payloads HTTP o
diffs completos si se usa fuera de una consola local controlada.
Decision pendiente: summary/redaccion por defecto, opt-in de fragmento crudo,
limites de bytes/lineas, permiso por `wave_ref`/`agent_ref` y prohibicion de
usar logs como evidencia terminal de ACK/delivery/cierre.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-rails`.
Backlog: `T58 codex-wave-log-tail-redaction-access`.
```

```text
ID: RAIL-CAND-CODEX-WAVE-PURGE-RUNTIME-001
Origen: scanner backlog 2026-05-24 decimoseptima pasada.
Casos: `codex-launch-wave` y `codex-launch-director-wave` aceptan
`--purge-runtime`/`ORQUESTA_CODEX_WAVE_PURGE_RUNTIME` y ejecutan borrado del
runtime resuelto. Las guardas actuales reducen riesgo obvio, pero no prueban
raiz permitida, manifest, agentes vivos, checkpoints, ACKs no reconciliados,
director_decisions pendientes, outbox pendiente ni plan state abierto.
Decision pendiente: purga solo con confirmacion explicita, runtime bajo raiz
permitida, dry-run/report, bloqueo por trabajo vivo y evidencia compacta sin
rutas privadas ni contenido de logs/prompts.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-server-shutdown ./modulos/orquesta-agent-process-registry`.
Backlog: `T59 codex-wave-runtime-purge-proof`.
```

```text
ID: RAIL-CAND-CODEX-WAVE-STOP-PID-REGISTRY-001
Origen: scanner backlog 2026-05-24 decimoseptima pasada.
Casos: `codex-wave-stop` carga un registro desde `runtime-dir` y senala PIDs
listados. Si el runtime-dir esta mal seleccionado o el registro esta corrupto,
la herramienta puede intentar parar un proceso que Orquesta no lanzo ni sigue
reconociendo como agente de esa ola.
Decision pendiente: validar descriptor/store de proceso antes de senalar PID:
`run_ref`, `wave_ref`, `agent_ref`, command ref, runtime permitido,
start time/owner opaco y estado vivo; si no hay prueba, bloquear con error
publico `blocked_registry_untrusted`.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-agent-process-registry ./modulos/orquesta-run-control`.
Backlog: `T60 codex-wave-stop-registry-pid-proof`.
```

```text
ID: RAIL-CAND-REQUIRED-TEST-OUTPUT-001
Origen: scanner backlog 2026-05-24 decimoctava pasada.
Casos: `LocalCommandExecutorV0` escribe stdout/stderr de tests requeridos en
artefactos `required-test-output-v0/*.log` con limite de bytes, pero sin
redaccion por campo ni retencion. Un `go test` real puede imprimir HOME, rutas
privadas, payloads HTTP, variables de entorno, tokens simulados o datos de
dominio.
Decision pendiente: persistir evidencia causal compacta y logs diagnosticos
redactados/retencionados por separado; si aparece material sensible no
redactable, bloquear o fallar sin guardar el material crudo.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-required-test ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file ./modulos/orquesta-app-director-service ./cmd/orquesta-server`.
Backlog: `T61 required-test-output-redaction-retention`.
```

```text
ID: RAIL-CAND-DIRECTOR-DECISIONS-PROMPT-RAIL-001
Origen: scanner backlog 2026-05-24 decimoctava pasada.
Casos: `directorDecisionInstructionsV0` sigue instruyendo al director a no usar
palabras como `provider`, `model`, `db`, `sql`, `runtime`, `adapter`, `git` o
`home` en `director_decisions.json`, aunque la politica vigente permite
vocabulario operativo opaco y corta solo secretos efectivos o payloads crudos.
Decision pendiente: sustituir lista textual por politica positiva de refs
opacas y datos no publicables, alineada con el helper comun o una matriz local
de frontera.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-director-agent ./modulos/orquesta-director-agent-file-source ./modulos/orquesta-rails ./modulos/orquesta-runtime-codex-delivery`.
Backlog: `T62 director-decisions-prompt-rail-sync`.
```

```text
ID: RAIL-CAND-DIRECTOR-DECISIONS-SIDECAR-001
Origen: scanner backlog 2026-05-24 decimoctava pasada.
Casos: el descriptor de `director_decisions.json` se deriva desde el `AckPath`
del recibo Codex y la fuente filtra por run/ACK reflejado, pero el sidecar no
tiene recibo propio con hash, producer ACK, correlacion, estado consumido ni
conflicto durable si cambia el payload.
Decision pendiente: exigir recibo estructurado y correlado para convertir el
sidecar en decisiones ejecutables; bloquear stale, fuera de scope, hash
cambiante o ya consumido bajo otro receipt.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-director-agent-file-source ./modulos/orquesta-app-codex-stack ./modulos/orquesta-app-director-service ./cmd/orquesta-server`.
Backlog: `T63 director-decisions-sidecar-receipt-correlation`.
```

```text
ID: RAIL-CAND-DIRECTOR-CYCLE-SOT-001
Origen: scanner backlog 2026-05-24 decimonovena pasada.
Casos: la espina `orquesta-director-cycle`/`scheduler`/`runner`/`tick-input` ya
existe y `orquesta-orchestration-core` la importa, pero la foto raiz y la matriz
no la nombran; `orquesta-orchestration-core/AGENTS.md` conserva pendientes
viejos de review/tests/rework/cierre como si el ciclo offline siguiera abierto.
Decision pendiente: sincronizar documentos de autoridad y AGENTS locales con
estados separados por codigo offline, composicion residente, smoke real y
proveedor/OPES real.
Test futuro:
`go test -count=1 ./modulos/orquesta-director-cycle ./modulos/orquesta-director-scheduler ./modulos/orquesta-director-runner ./modulos/orquesta-director-tick-input ./modulos/orquesta-orchestration-core`.
Backlog: `T64 director-cycle-source-of-truth-sync`.
```

```text
ID: RAIL-CAND-DIRECTOR-CYCLE-RESIDENT-001
Origen: scanner backlog 2026-05-24 decimonovena pasada.
Casos: existen pruebas locales de cycle, scheduler, runner, state-file/outbox y
dispatch, pero falta smoke de composicion residente que demuestre
`DirectorCycleStepV0` con `FileOutboxLedgerV0`, ACK parcial, reinicio y reentrada
sin duplicar comandos ni cerrar con outbox pendiente.
Decision pendiente: anadir smoke servidor temporal con puerto fake/temporal,
claims/ACK reabiertos tras restart y contadores publicos de pending/acked/failed.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack ./modulos/orquesta-state-file ./modulos/orquesta-outbox-dispatch`.
Backlog: `T65 director-cycle-resident-restart-smoke`.
```

```text
ID: RAIL-CAND-NEUTRAL-PROCESS-STOP-001
Origen: scanner backlog 2026-05-24 decimonovena pasada.
Casos: `DIR-P006` sigue documentando parada de proceso real pendiente. El
runtime neutral tiene `ProcessRuntimeConnectorV0` y tests focales, pero falta
recorrer Director/scheduler/outbox hasta `StopRuntimeAgent` contra proceso
temporal real con ACK/evidencia e idempotencia.
Decision pendiente: smoke neutral sin Codex que lance proceso temporal por puerto
explicito, emita stop causal, bloquee refs ajenas/stale y no filtre command/env
crudos.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-director ./modulos/orquesta-director-scheduler ./modulos/orquesta-director-cycle ./modulos/orquesta-orchestration-core ./cmd/orquesta-server`.
Backlog: `T66 neutral-process-stop-e2e`.
```

```text
ID: RAIL-CAND-DIRECTOR-SUPERVISOR-BURST-SOT-001
Origen: scanner backlog 2026-05-24 vigesima pasada.
Casos: `orquesta-orchestration-core` ya importa
`orquesta-director-supervisor` y `orquesta-director-supervised-burst`, pero la
foto raiz, matriz y T64 nombran sobre todo cycle/scheduler/runner/tick-input.
Sin fuente de verdad, futuros agentes pueden duplicar la politica de
repeticion/parada o reabrir pendientes viejos de review/tests/cierre.
Decision pendiente: sincronizar docs de autoridad y AGENTS locales para declarar
la cadena de supervision de pasos, separandola de `run-supervisor` global y sin
dar por cerrado un smoke residente que aun no existe.
Test futuro:
`go test -count=1 ./modulos/orquesta-director-supervisor ./modulos/orquesta-director-supervised-burst ./modulos/orquesta-orchestration-core`.
Backlog: `T67 director-supervisor-burst-source-of-truth-sync`.
```

```text
ID: RAIL-CAND-DIRECTOR-BURST-BUDGET-RESIDENT-001
Origen: scanner backlog 2026-05-24 vigesima pasada.
Casos: el servidor configura presupuestos anidados de supervisor global y drain
del Director (`MaxTicks`, `MaxRunsPerTick`, `MaxExecutions`, `MaxBursts`,
`MaxStepsPerBurst`, `MaxCommands`). Hay pruebas locales, pero falta smoke de
composicion que demuestre corte por `wait_outbox`, `wait_external`,
`stop_max_steps` o `stop_error` sin duplicar comandos ni tratar presupuesto
agotado como exito terminal.
Decision pendiente: probar servidor temporal con estado durable/fake, acciones
finales visibles, outbox pendiente conservado, replay idempotente y distincion
entre falta real de trabajo y trabajo bloqueado por presupuesto.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server ./modulos/orquesta-run-supervisor ./modulos/orquesta-orchestration-core ./modulos/orquesta-director-supervisor ./modulos/orquesta-director-supervised-burst ./modulos/orquesta-outbox-dispatch`.
Backlog: `T68 director-supervised-burst-resident-budget-smoke`.
```

```text
ID: RAIL-CAND-NESTED-SUPERVISOR-STOP-REASON-001
Origen: scanner backlog 2026-05-24 vigesima pasada.
Casos: `run-supervisor`, `director-supervisor` y
`director-supervised-burst` exponen razones de parada locales. Auditoria,
stats, cola e idle autoprogramming pueden interpretar distinto `no_execution`,
`max_ticks`, `wait_outbox`, `wait_external`, `stop_max_steps` o `stop_error` si
cada capa proyecta su propio vocabulario.
Decision pendiente: crear proyeccion publica comun de stop reasons con refs y
contadores compactos; idle autoprogramming solo debe disparar por capacidad
libre real, no por outbox pendiente, espera externa o presupuesto agotado.
Test futuro:
`go test -count=1 ./modulos/orquesta-run-supervisor ./modulos/orquesta-director-supervisor ./modulos/orquesta-director-supervised-burst ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T69 nested-supervisor-stop-reason-contract`.
```

```text
ID: RAIL-CAND-CORE-WORKFLOW-DOC-POLICY-001
Origen: scanner backlog 2026-05-24 vigesimoprimera pasada.
Casos: docs locales de `orquesta-core-workflow` siguen afirmando que comandos,
eventos o JSON rechazan palabras como `runtime`, `provider`, `DB`, `SQL`,
`HOME`, `modelo` o `Codex`, mientras el rail vivo permite vocabulario operativo
opaco y corta solo valores sensibles efectivos.
Decision pendiente: sincronizar docs/pruebas/decisiones locales con
`orquesta-rails`: refs opacas y terminos arquitectonicos pasan; secretos,
rutas privadas, prompts/transcripts crudos y payloads masivos se bloquean.
Test futuro:
`go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-rails`.
Backlog: `T70 core-workflow-docs-rail-policy-sync`.
```

```text
ID: RAIL-CAND-DOMAIN-WORK-FILE-BUDGET-001
Origen: scanner backlog 2026-05-24 vigesimoprimera pasada.
Casos: el rail OrquestaV2 de ficheros Go menores de 300 lineas no solo afecta a
stack/director. `orquesta-domain-work-sql/store_v0.go`,
`orquesta-document-plan-expander/expander_v0.go` y los `job_creator_v0.go` de
memory/file superan el limite o se acercan mezclando responsabilidades.
Decision pendiente: partir por responsabilidad o dejar baseline de no
crecimiento con followups concretos; conservar contract tests y frontera
neutral sin introducir DB/producto en el nucleo.
Test futuro:
`go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-domain-work-memory ./modulos/orquesta-domain-work-file ./modulos/orquesta-domain-work-sql ./modulos/orquesta-document-plan-expander`.
Backlog: `T71 domain-work-adapters-file-budget-split`.
```

```text
ID: RAIL-CAND-DOMAIN-WORK-ARTIFACT-MAP-001
Origen: scanner backlog 2026-05-24 vigesimoprimera pasada.
Casos: el mapa `work_kind -> expected_artifact_type` aparece duplicado en
`orquesta-document-plan-expander`, `orquesta-app-codex-stack`,
`orquesta-opes-bridge` y tests de `cmd/orquesta-server`.
Decision pendiente: fijar owner neutral del mapa y hacer que expander, builder
de entregas, bridge OPES y drain de servidor prueben la misma correspondencia;
unknown work kinds deben tener fallback/issue coherente.
Test futuro:
`go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-document-plan-expander ./modulos/orquesta-app-codex-stack ./modulos/orquesta-opes-bridge ./cmd/orquesta-server`.
Backlog: `T72 domain-work-artifact-contract-map-owner`.
```

```text
ID: RAIL-CAND-FACTORY-CONNECTOR-POLICY-001
Origen: scanner backlog 2026-05-24 vigesimosegunda pasada.
Casos: `modulos/orquesta-factory/appspec_request_v0.go` rechaza nombres de
integracion si contienen substrings como `postgres`, `sqlite`, `runtime`,
`filesystem`, `llm`, `queue`, `deploy`, `database` o `db`. Esa lista evita
proveedores concretos en el contrato base, pero tambien puede cortar
capacidades de dominio validas o refs opacas reparables.
Decision pendiente: sustituir el rail textual por politica de capacidad vs
proveedor concreto. Bloquear DSN, credenciales, SDK/proveedor elegido o backend
impuesto; permitir vocabulario operativo si el director/adaptador puede
normalizarlo sin riesgo.
Test futuro:
`go test -count=1 ./modulos/orquesta-factory ./modulos/orquesta-mcp ./modulos/orquesta-web`.
Backlog: `T73 factory-connector-capability-policy-port`.
```

```text
ID: RAIL-CAND-DEPLOY-CONTRACT-ISLAND-001
Origen: scanner backlog 2026-05-24 vigesimosegunda pasada.
Casos: `DeploymentPlan v0` aparece en factory/MCP y `orquesta-deploy`, pero
`orquesta-deploy` no esta en el mapa raiz de capas y no hay consumidores Go
fuera del propio modulo. Las tareas de deploy pueden quedar como docs/write-set
sin invocar el contrato dry-run ni producir evidencia por puerto.
Decision pendiente: declarar owner de deploy en la foto raiz y conectar el
contrato por composicion opt-in/dry-run. No ejecutar Docker, Kubernetes, cloud,
filesystem productivo ni secretos desde el nucleo.
Test futuro:
`go test -count=1 ./modulos/orquesta-deploy ./modulos/orquesta-factory ./modulos/orquesta-app-planner ./modulos/orquesta-mcp`.
Backlog: `T74 deployment-plan-composition-wiring`.
```

```text
ID: RAIL-CAND-I18N-DOCS-CONTRACT-ISLAND-001
Origen: scanner backlog 2026-05-24 vigesimosegunda pasada.
Casos: `orquesta-i18n-docs` define builder/validator de bundles, loader shape
y docs generadas, pero no tiene consumidores Go fuera del modulo. Web/factory
mantienen mapas i18n locales y la foto vigente aun habla de brechas historicas
de i18n.
Decision pendiente: decidir owner activo. Si `orquesta-i18n-docs` sigue vivo,
factory/web/MCP deben consumir el contrato o una proyeccion por puerto; si no,
marcarlo historico y evitar que sus docs locales se usen como verdad vigente.
Test futuro:
`go test -count=1 ./modulos/orquesta-i18n-docs ./modulos/orquesta-factory ./modulos/orquesta-web ./modulos/orquesta-mcp`.
Backlog: `T75 i18n-docs-active-composition-owner`.
```

```text
ID: RAIL-CAND-APPSPEC-ENTRYPOINTS-001
Origen: scanner backlog 2026-05-24 vigesimotercera pasada.
Casos: MCP expone `orquesta.apps.preparar_orquestacion.v0` por
`orquesta-app-runner`/`AppPlan` y tambien `orquesta.apps.arrancar_director.v0`
por `app-director-service`. Sin estado/freshness claro, una IA puede elegir el
camino historico sin Director V2 para un trabajo que exige juicio, waits,
review/tests y cierre causal.
Decision pendiente: declarar ruta operativa preferente, marcar `app-runner`
como preview/legacy/compatibilidad o enlazarlo con
`OperationalDirectorPlanStateV0` antes de usarlo para ejecucion real.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-runner ./modulos/orquesta-app-planner ./modulos/orquesta-app-director-intake ./modulos/orquesta-app-director-service`.
Backlog: `T76 appspec-entrypoints-director-v2-routing`.
```

```text
ID: RAIL-CAND-APP-CHANGE-READINESS-001
Origen: scanner backlog 2026-05-24 vigesimotercera pasada.
Casos: `orquesta-app-change-director-source/readiness_v0.go` usa
`forbiddenAutoPlanFragmentsV0` con `runtime`, `provider`, `model`, `db`, `sql`,
`codex`, `docker`, `home` y otros fragments. `external_work_v0.go` tambien
reescribe vocabulario operativo en criterios. Puede bloquear refs opacas o
ocultar semantica de dominio fuera del scope de `T15`.
Decision pendiente: mover readiness/sanitizacion a politica por campo alineada
con `orquesta-rails`; permitir vocabulario operativo opaco y cortar solo valores
sensibles efectivos, material crudo o efectos externos no autorizados.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-change-director-source ./modulos/orquesta-app-change ./modulos/orquesta-rails ./modulos/orquesta-app-codex-stack`.
Backlog: `T77 app-change-director-source-readiness-rail-policy`.
```

```text
ID: RAIL-CAND-DIRECTOR-DECISION-PLANSTATE-MERGE-001
Origen: scanner backlog 2026-05-24 vigesimotercera pasada.
Casos: `docs/corte_plan_state_director_decisions_2026-05-21.md` deja pendiente
fusionar nuevas tasks operativas en un `OperationalDirectorPlanStateV0`
existente. El ensure actual devuelve si el state ya existe, por lo que una
decision tardia que abre otra ola necesita contrato de merge/reentrada y replay.
Decision pendiente: anadir merge causal de tasks marcadas por
`director_decision`, wait scope acotado, idempotencia y bloqueo publico si falta
metadata suficiente.
Test futuro:
`go test -count=1 ./modulos/orquesta-director-agent-workflow ./modulos/orquesta-app-director-service ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file ./modulos/orquesta-app-codex-stack`.
Backlog: `T78 director-decisions-existing-planstate-merge`.
```

```text
ID: RAIL-CAND-BACKLOG-DOC-SHARD-001
Origen: scanner backlog 2026-05-24 vigesimocuarta pasada.
Casos: `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`,
`docs/rail_errors_observados_2026-05-23.md` y
`docs/duplicaciones_railes_pendientes_2026-05-24.md` concentran miles de lineas
append-only. `idle_self_improvement_backlog_planner_v0.go` lee un unico backlog
hardcodeado y no conserva fichero/hash por shard. T43 cubre lease de merge,
pero no reduce el hotspot documental.
Decision pendiente: crear indice/shards compatibles con el parser, preservar
linea/hash/fichero por seccion Txx y no borrar ni truncar historico durante la
migracion.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-server`.
Backlog: `T79 autoprogramming-backlog-doc-sharding`.
```

```text
ID: RAIL-CAND-DOMAIN-WORK-HTTP-EGRESS-001
Origen: scanner backlog 2026-05-24 vigesimocuarta pasada.
Casos: `ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL` activa el adaptador HTTP neutral y
`orquesta-domain-work-http` valida esquema/base/path, pero no tiene politica de
host/egress, allowlist, modo smoke ni redaccion especifica del destino. Un error
de composicion podria apuntar a endpoints no temporales o redes internas.
Decision pendiente: anadir politica opt-in de destino, rechazo de credenciales
en URL, allowlist o modo smoke declarado, errores publicos y auditoria compacta
sin URL sensible ni payload crudo.
Test futuro:
`go test -count=1 ./modulos/orquesta-domain-work-http ./modulos/orquesta-domain-work ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T80 domain-work-http-egress-policy`.
```

```text
ID: RAIL-CAND-OBSERVABILITY-GLOBAL-TIMELINE-001
Origen: scanner backlog 2026-05-24 vigesimocuarta pasada.
Casos: `docs/op_096_control_total_estado_proyecto_y_estadisticas.md` declara
operativo el control por agente/proyecto y deja abierto el plano global de
workspace, coste y timeline. `orquesta-observability` y la auditoria del
servidor existen, pero no hay contrato unico para API/MCP/web que agregue
workspace sin shell, transcript crudo o stores internos.
Decision pendiente: definir puerto de lectura global con fuentes declaradas,
paginacion temporal, redaccion por campo y `not_available` para fuentes ausentes
en vez de inferencias ad hoc.
Test futuro:
`go test -count=1 ./modulos/orquesta-observability ./modulos/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-web ./cmd/orquesta-server`.
Backlog: `T81 observability-global-workspace-timeline`.
```

```text
ID: RAIL-CAND-GOVERNANCE-PUBLIC-SHAPE-001
Origen: scanner backlog 2026-05-24 vigesimoquinta pasada.
Casos: `GovernanceCatalogQueryHTTPHandlerV0` espera request con `filters` y
responde `result/errors`, mientras `GovernanceCatalogCliReaderV0` envia
`module|role|phase|tags` en raiz y espera `GovernanceCatalogQueryResultV0`
directo con errores `errores/codigo`. La ruta tampoco aparece cableada en
`orquesta-http-gateway`, `orquesta-app-gateway` ni `cmd/orquesta-server`.
Decision pendiente: fijar un unico shape publico para request/response/error,
actualizar CLI/handler/MCP docs y cablear gateway con provider inyectado; si no
hay provider, devolver error publico recuperable.
Test futuro:
`go test -count=1 ./modulos/orquesta-governance ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T82 governance-catalog-public-route-shape-sync`.
```

```text
ID: RAIL-CAND-OPERATIONAL-STATUS-PUBLIC-SOURCE-001
Origen: scanner backlog 2026-05-24 vigesimoquinta pasada.
Casos: CLI, web y MCP anuncian `OperationalStatusQueryV0` y endpoint
`/api/v0/operational-status/query`, pero `orquesta-observability` solo incluye
DTOs/validadores y adapter en memoria. Falta source residente y wiring HTTP que
derive `DiagnosticoCompactoV0` real desde stats/cola/runs/runtime con redaccion.
Decision pendiente: publicar source por puerto y handler comun; fuentes
ausentes deben producir warning/`not_available`, no datos inventados ni lecturas
directas de shell, stores internos, runtime dirs o transcripts.
Test futuro:
`go test -count=1 ./modulos/orquesta-observability ./modulos/orquesta-server ./modulos/orquesta-cli ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T83 operational-status-public-query-source`.
```

```text
ID: RAIL-CAND-FUNCTION-CONTRACT-PUBLIC-ROUTE-001
Origen: scanner backlog 2026-05-24 vigesimoquinta pasada.
Casos: la CLI implementa `POST /api/v0/core/function-contracts/list` y
`/view`, pero `core` conserva esas operaciones como candidatas documentales y
`core-workflow` solo proyecta refs compactas `FunctionContractPublished`.
Faltan store/index read-only, shape de gateway y caso de payload insuficiente.
Decision pendiente: definir consulta read-only por puerto desde eventos/stores
causales, bloquear cuando solo haya refs sin payload contractual y mantener
`registrar` como operacion no promovida.
Test futuro:
`go test -count=1 ./modulos/orquesta-core ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T84 function-contract-readonly-public-index`.
```

```text
ID: RAIL-CAND-SERVER-STATUS-CONFIG-SNAPSHOT-001
Origen: scanner backlog 2026-05-24 vigesimosexta pasada.
Casos: `/api/v0/server/status` devuelve `StateV0` consumido por CLI, pero el
DTO incluye `project_work_dir` y `runtime_work_dir` crudos y no publica el
snapshot efectivo/redactado de variables residentes que la CLI documenta como
necesario para auditoria de automejora. Eso mezcla estado publico, config local
y paths operativos en una sola respuesta.
Decision pendiente: separar estado publico compacto de snapshot de config
efectiva, redactar paths/HOME/runtime dirs/comandos/proveedor/tokens por campo
y permitir detalle local solo con opt-in de composicion.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-cli ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./cmd/orquesta-server`.
Backlog: `T85 server-status-config-snapshot-redaction`.
```

```text
ID: RAIL-CAND-BOOTSTRAP-APPSPEC-LEGACY-ROUTE-001
Origen: scanner backlog 2026-05-24 vigesimosexta pasada.
Casos: `BootstrapAppSpecCliEndpointV0` apunta a
`/api/v0/director/bootstrap/appspec`, mientras gateway/web usan
`/api/v0/apps/spec` y `/api/v0/apps/director`. No aparece route constant,
handler de app-gateway ni wiring de servidor para esa ruta legacy.
Decision pendiente: decidir si `BootstrapProyectoDesdeAppSpec` sigue vivo como
compatibilidad cableada, queda bloqueado con error publico o migra al flujo
vigente; no dejar cliente fino apuntando a transporte inexistente.
Test futuro:
`go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-director ./modulos/orquesta-factory ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T86 bootstrap-appspec-legacy-route-quarantine`.
```

```text
ID: RAIL-CAND-SERVER-LIVENESS-READINESS-001
Origen: scanner backlog 2026-05-24 vigesimosexta pasada.
Casos: scripts de smoke y daemon tratan `/healthz` como "servidor listo", pero
el handler solo responde liveness `status=ok`. La readiness operativa vive en
`startup_ready/startup_status` de `/api/v0/server/status`, incluyendo cleanup y
reconciliacion.
Decision pendiente: fijar contrato de liveness vs readiness y actualizar scripts
para esperar readiness cuando vayan a preparar runs, drenar OPES, lanzar Codex
o ejecutar automejora; si readiness falta, bloquear o degradar explicitamente.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-cli ./cmd/orquesta-server` y `bash -n scripts/*.sh`.
Backlog: `T87 server-liveness-readiness-contract`.
```

```text
ID: RAIL-CAND-FEDERATED-MODULE-BACKLOG-001
Origen: scanner backlog 2026-05-24 vigesimoseptima pasada.
Casos: el planner residente lee solo
`docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`, pero modulos
mantienen backlog local en `docs/tareas.md` o README (`APG-*`, `RTDELIVERY-*`,
`SSH-*`, `DEP-*`). Sin indice federado, el residente puede no ver pendientes
locales vigentes o duplicarlos como Txx sin alias, owner, linea/hash ni estado.
Decision pendiente: definir indice federado con source path/line/hash,
freshness, owner, alias local y Txx relacionado; el planner solo debe programar
entradas locales promocionadas o vigentes, y dejar docs historicos como
evidencia no ejecutable.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-mcp ./modulos/orquesta-server`.
Backlog: `T88 federated-module-backlog-index`.
```

```text
ID: RAIL-CAND-DIRECTOR-OPERATIVO-DOC-STATE-001
Origen: scanner backlog 2026-05-24 vigesimoseptima pasada.
Casos: `modulos/orquesta-director-operativo/README.md` sigue declarando
pendientes wait por cohorte/ola, review/rework/replan/cierre durable y recursion
Codex real, aunque las fuentes vigentes ya cierran WaitAgentRefs, ciclo offline
del PlanState y `CODEX-WAVE-REAL`/`CODEX-RECURSION-REAL`; el pendiente real
abierto es OPES temporal de derivados/cierre salvo regresion demostrada.
Decision pendiente: sincronizar README/docs locales y documentos obligatorios
del Director Operativo con la foto vigente, marcando historia sin borrar cortes
anteriores y anadiendo check de contradicciones conocidas.
Test futuro:
`go test -count=1 ./modulos/orquesta-director-operativo` y check documental
focal de `CODEX-WAVE-REAL`, `CODEX-RECURSION-REAL` y OPES derivados/cierre.
Backlog: `T89 director-operativo-local-doc-state-sync`.
```

```text
ID: RAIL-CAND-RESIDUAL-GO-FILE-BUDGET-001
Origen: scanner backlog 2026-05-24 vigesimoctava pasada.
Casos: el line-count deja ficheros Go >300 lineas fuera de los shards ya
identificados en T52-T54/T71, especialmente en `orquesta-runtime-codex-delivery`,
`orquesta-runtime-required-test`, `orquesta-run-coordinator`,
`orquesta-external-work-run`, `orquesta-app-gateway`, `orquesta-director`,
`orquesta-core-workflow` y tests de `cmd/orquesta-server`. El rail de tamano
queda como advisory generico si no hay shard residual con baseline.
Decision pendiente: medir baseline por fichero, partir por responsabilidad
local y conservar T52-T54/T71 como owners de los focos principales; no cerrar
ACK terminal estricto por `ACK.files` sin snapshot/worktree real.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-runtime-required-test ./modulos/orquesta-run-coordinator ./modulos/orquesta-external-work-run ./modulos/orquesta-app-gateway ./modulos/orquesta-director ./modulos/orquesta-core-workflow ./cmd/orquesta-server`.
Backlog: `T90 residual-go-file-budget-splits`.
```

```text
ID: RAIL-CAND-SMOKE-SCRIPT-OPS-LIB-001
Origen: scanner backlog 2026-05-24 vigesimoctava pasada.
Casos: varios smokes largos duplican prerequisitos, confirmaciones opt-in,
arranque/parada de servidor temporal, espera de `/healthz`, cleanup, parseo JSON
y redaccion de salida. Esa duplicacion puede divergir de T21/T87 y hacer que un
script lance Codex/OPES/red antes de readiness o sin guarda equivalente.
Decision pendiente: extraer helpers en `scripts/lib` para prerequisitos,
confirmacion, liveness/readiness, cleanup y redaccion, manteniendo por script la
politica de riesgo y sin imprimir HOME, rutas privadas, prompts, transcripts,
tokens ni payloads crudos por defecto.
Test futuro:
`bash -n scripts/*.sh scripts/lib/*.sh` y focos de smokes con confirmaciones
fake/temporales cuando existan.
Backlog: `T91 smoke-script-ops-library`.
```

```text
ID: RAIL-CAND-CAPACITY-DECISION-POLICY-001
Origen: scanner backlog 2026-05-24 vigesimonovena pasada.
Casos: `RequestCapacityDecision` se despacha en el stack con
`CapacityDecisionExecutorV0`, pero el executor usa configuracion estatica/default
de tier/reasoning y no un puerto de politica que consuma `orquesta-capacity`.
Los docs de `orquesta-capacity` tambien conservan casos iniciales como
`Comando: pendiente` aunque ya existen DTOs, fixtures y tests Go.
Decision pendiente: anadir puerto de politica/capacidad inyectable en
composicion, mantener fallback fake/legacy auditable y sincronizar docs locales
para distinguir contrato cerrado, wiring pendiente y cuota/benchmarks reales.
Test futuro:
`go test -count=1 ./modulos/orquesta-capacity ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime ./cmd/orquesta-server`.
Backlog: `T92 capacity-decision-policy-port`.
```

```text
ID: RAIL-CAND-CODEX-PROMPT-TOOLBELT-001
Origen: scanner backlog 2026-05-24 vigesimonovena pasada.
Casos: `cmd/orquesta-server/codex_prompt_hints_v0.go` enumera en el toolbelt
MCP `orquesta.operator.operations.v0`, pero otra linea recomienda usar
`orquesta.operator.directed_query.v0`; el tool existe bajo
`orquesta-operator-mcp` y la exposicion real depende de puertos/transportes
inyectados. Listas libres en prompts, packets y runbooks pueden divergir del
registry real.
Decision pendiente: derivar hints/toolbelt de un descriptor comun o probarlos
contra rutas/tools registrados; distinguir capability disponible, puerto no
configurado y transporte real no arrancado sin prometer conectores opt-in.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-app-gateway`.
Backlog: `T93 codex-prompt-toolbelt-source-sync`.
```

```text
ID: RAIL-CAND-SERVER-AUDIT-WRITE-FAIL-001
Origen: scanner backlog 2026-05-24 trigesima pasada.
Casos: `modulos/orquesta-server/audit_v0.go` ignora el error devuelto por
`AppendAuditEventV0`. Un fallo de escritura JSONL por IO, permisos, espacio o
fichero inaccesible puede dejar al servidor operando sin evidencia durable y
sin diagnostico publico, justo en rutas que preparan automejora, supervisor
ticks, startup checks o mutaciones HTTP.
Decision pendiente: registrar el fallo de auditoria en una proyeccion compacta
no recursiva, con contador/codigo publico/evento afectado y politica de
severidad por composicion. No devolver rutas locales, HOME, permisos crudos,
payloads HTTP, prompts, transcripts ni tokens.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T94 server-audit-write-failure-visibility`.
```

```text
ID: RAIL-CAND-SERVER-STATE-PERSIST-FAIL-001
Origen: scanner backlog 2026-05-24 trigesimoprimera pasada.
Casos: `modulos/orquesta-server/runtime_v0.go` devuelve error si falla el
guardado inicial de `serving`, pero `persistStateV0` ignora errores posteriores
de `saveStateV0`. Ese helper se usa en startup checks, ticks de supervisor,
automejora idle, shutdown y marca de error. Un state store sin permisos, sin
espacio, corrupto o bloqueado por IO puede dejar `/api/status` vivo en memoria
mientras el store durable queda obsoleto para reinicio, CLI o auditoria de
operacion.
Decision pendiente: registrar fallo de persistencia de estado en una proyeccion
compacta no recursiva, separar severidad por transicion y exponer codigo publico
`state_persist_failed` sin rutas locales, permisos exactos, HOME, payloads HTTP,
prompts, transcripts ni tokens. No depender del sink de auditoria para conocer
este fallo.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T95 server-state-persist-failure-visibility`.
```

```text
ID: RAIL-CAND-SERVER-DAEMON-STOP-PID-001
Origen: scanner backlog 2026-05-24 trigesimosegunda pasada.
Casos: `orquesta-server stop` carga `pid` y `addr` desde el state file,
invoca `/api/v0/server/shutdown` con `forced=true` y despues senala ese PID.
Si el state esta stale, el PID fue reutilizado o el addr no corresponde al mismo
daemon, el CLI puede actuar sobre un proceso equivocado o saltarse shutdown
cooperativo como default silencioso.
Decision pendiente: validar descriptor/epoch de proceso y correspondencia
HTTP-state antes de senalar; forced debe ser opt-in con causa/evidencia. No
devolver HOME, rutas locales, argv completos, prompts, transcripts ni tokens.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-server-shutdown ./modulos/orquesta-agent-process-registry`.
Backlog: `T96 server-daemon-stop-process-identity`.
```

```text
ID: RAIL-CAND-SERVER-DAEMON-LOGS-001
Origen: scanner backlog 2026-05-24 trigesimosegunda pasada.
Casos: `orquesta-server start` redirige stdout/stderr del proceso residente a
`stdout.log` y `stderr.log` bajo `StateDir`. Esos logs no tienen owner de
redaccion, rotacion, retencion ni acceso, y pueden duplicar auditoria JSONL,
tail Codex o salida de tests con material crudo.
Decision pendiente: definir politica de logs operacionales del daemon: resumen
redactado por defecto, fragmento crudo solo opt-in local con limite, retencion
corta y bloqueo/redaccion de HOME, rutas privadas, env, comandos, remotos Git,
payloads HTTP, prompts, transcripts, tokens y salida de proveedor.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-observability`.
Backlog: `T97 server-daemon-log-redaction-retention`.
```

```text
ID: RAIL-CAND-EXTERNAL-BRIDGE-INPUT-LEDGER-001
Origen: scanner backlog 2026-05-24 trigesimosegunda pasada.
Casos: el bridge OPES consulta el ledger de entrada antes del submit y lo
actualiza despues de recibir `run_ref`. Si el submit funciona pero falla el
ledger, o si dos drains procesan el mismo job externo en paralelo, puede quedar
una run creada sin claim durable o una entrada `submitted` sobrescrita por otro
`run_ref`.
Decision pendiente: claim durable antes de crear run, recovery por
idempotency/correlation tras fallo de ledger y rechazo de overwrite de
`submitted` sin decision explicita. Errores publicos compactos sin URL sensible,
payload OPES completo, HOME, rutas locales, tokens ni respuestas crudas.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector ./modulos/orquesta-run-queue`.
Backlog: `T98 external-bridge-input-ledger-claim-recovery`.
```

```text
ID: RAIL-CAND-SERVER-HTTP-RESOURCE-001
Origen: scanner backlog 2026-05-24 trigesimotercera pasada.
Casos: `modulos/orquesta-server/runtime_v0.go` crea `http.Server` solo con
`Handler`, y rutas MCP/HTTP mutables decodifican `r.Body` sin limite comun ni
trailing-token check. El transporte MCP real si tiene limite local, pero no
gobierna `prepare-run`, `domain_work`, `run_control` ni el resto de puertos
HTTP.
Decision pendiente: configurar timeouts de servidor y helper comun de decode
JSON por perfil: limite de body, error publico, politica de unknown fields y
auditoria de tamano/codigo sin payload crudo. No exponer body, prompts,
transcripts, tokens, HOME ni rutas privadas.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-web ./cmd/orquesta-server`.
Backlog: `T99 server-http-resource-guardrails`.
```

```text
ID: RAIL-CAND-STARTUP-REVISION-ARCHIVE-001
Origen: scanner backlog 2026-05-24 trigesimotercera pasada.
Casos: la compactacion de startup escribe `queue_v0.before.json`,
`control_v0.before.json`, `queue_removed.json`, `control_removed.json`,
`manifest.json` y runtime archivado bajo un directorio de revision, y proyecta
la ruta de revision en readiness. T35 decide si compactar; falta politica del
artefacto generado.
Decision pendiente: guardar revision con permisos restrictivos, retencion,
tamano maximo, redaccion por campo y `revision_ref` opaco en status. Material
crudo solo opt-in local, con limite y causa publica.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-run-control ./modulos/orquesta-state-file`.
Backlog: `T100 startup-revision-archive-redaction-retention`.
```

```text
ID: RAIL-CAND-DOMAIN-WORK-SUBMISSION-LEDGER-001
Origen: scanner backlog 2026-05-24 trigesimotercera pasada.
Casos: `domain_work_delivery_bridge_v0.go` comprueba ledger por
`idempotency_key`, ejecuta `submit_artifact` y registra accepted/rejected al
final. El ledger file/memory permite reemplazar el record bajo la misma key sin
contrato de claim ni recovery de fallo post-submit.
Decision pendiente: claim durable antes de `submit_artifact`, recovery por
receipt/idempotency si falla el ledger tras efecto externo y conflicto publico
si una key ya aceptada intenta registrar otro receipt/payload. No usar URL, DB,
ruta local o nombre de conector como evidencia de tests de dominio.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-domain-work ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector ./cmd/orquesta-server`.
Backlog: `T101 domain-work-artifact-submission-ledger-recovery`.
```

```text
ID: RAIL-CAND-LEGACY-HTTP-JSON-001
Origen: scanner backlog 2026-05-24 trigesimocuarta pasada.
Casos: handlers HTTP publicos fuera del primer scope de T99 conservan decoders
locales: `orquesta-factory-http` usa `io.ReadAll(r.Body)`, governance aplica su
propio `DisallowUnknownFields` y trailing-token check, y MCP replica decoders por
ruta sin limite comun.
Decision pendiente: unificar frontera JSON por perfil para limite de body,
content-type, trailing tokens, campos desconocidos y error publico; los modos
legacy deben declararse con test. No devolver body crudo, prompts, transcripts,
rutas locales, HOME, tokens ni internals de adaptador.
Test futuro:
`go test -count=1 ./modulos/orquesta-factory-http ./modulos/orquesta-governance ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway ./modulos/orquesta-web ./cmd/orquesta-server`.
Backlog: `T102 legacy-http-json-boundary-policy`.
```

```text
ID: RAIL-CAND-OUTBOUND-HTTP-RESPONSE-001
Origen: scanner backlog 2026-05-24 trigesimocuarta pasada.
Casos: conectores y clientes salientes decodifican o leen respuestas HTTP con
reglas divergentes: OPES y `domain-work-http` hacen `json.NewDecoder` directo
sobre `response.Body`; CLI/governance y function-contract usan `io.ReadAll`;
comandos de servidor leen cuerpos completos para status/diagnostico.
Decision pendiente: limite de respuesta por perfil, status handling redactado,
content-type/trailing-token check y errores publicos compactos. T80 decide
egress/host del HTTP neutral; este rail gobierna respuesta saliente y redaccion.
Test futuro:
`go test -count=1 ./modulos/orquesta-opes-connector ./modulos/orquesta-domain-work-http ./modulos/orquesta-cli ./modulos/orquesta-web ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T103 outbound-http-response-limit-redaction`.
```

```text
ID: RAIL-CAND-FILE-STORE-DURABLE-WRITE-001
Origen: scanner backlog 2026-05-24 trigesimocuarta pasada.
Casos: stores file-based usan contratos de escritura distintos. `orquesta-run-file`
y `orquesta-domain-work-file` hacen temp unico, sync y sync de directorio;
`orquesta-server` y `orquesta-runtime-codex-delivery` usan `path+".tmp"` sin
fsync/dir sync; el ledger de artefactos crea directorio `0755` y no sincroniza
antes/despues de `rename`.
Decision pendiente: politica comun o documentada por modulo para temp unico,
permisos, lock/claim cuando haya multiproceso, fsync/close/rename/sync dir y
limpieza de temp interrumpido. Coordinar con T95, T57, T98 y T101 sin reemplazar
sus contratos especificos.
Test futuro:
`go test -count=1 ./modulos/orquesta-run-file ./modulos/orquesta-domain-work-file ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-server ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T104 file-store-durable-write-policy`.
```

```text
ID: RAIL-CAND-PROCESS-RUNTIME-LAUNCH-IO-001
Origen: scanner backlog 2026-05-24 trigesimoquinta pasada.
Casos: `ProcessRuntimeConnectorV0` arranca `exec.Cmd` con `CommandPath`, `Args`,
`Env` y `WorkingDir` reales ya resueltos, y descarta stdout/stderr con
`io.Discard`. El contrato externo usa refs opacas (`executable_ref`,
`arg_refs`, `env_refs`, `working_dir_ref`), pero falta recibo redacted que
demuestre como se resolvieron y que politica de IO se aplico.
Decision pendiente: recibo de launch/env/IO por politica, sin persistir path,
env ni comando reales; stdout/stderr descartados o capturados solo con limite y
redaccion. Coordinar con stop neutral T66 sin duplicarlo.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-orchestration-core ./modulos/orquesta-agent-process-registry ./modulos/orquesta-agent-process-registry-memory ./cmd/orquesta-server`.
Backlog: `T105 process-runtime-launch-env-io-receipt`.
```

```text
ID: RAIL-CAND-CODEX-WAVE-SUMMARY-REDACTION-001
Origen: scanner backlog 2026-05-24 trigesimoquinta pasada.
Casos: `codex-wave`/`codex-director-wave` publican summaries con rutas reales de
runtime, registry, prompt, ACK, last message, stdout/stderr, HOME/CODE_HOME y
PID. Tail, purge y stop ya tienen tareas separadas, pero el launch/status puede
filtrar material local antes de que esas guardas apliquen.
Decision pendiente: summary publico por refs opacas, contadores y estados;
diagnostico crudo solo opt-in local con limite fuerte. No usar rutas ni logs
crudos como evidencia terminal.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-agent-process-registry`.
Backlog: `T106 codex-wave-public-summary-redaction`.
```

```text
ID: RAIL-CAND-GO-SMOKE-DIAGNOSTIC-REDACTION-001
Origen: scanner backlog 2026-05-24 trigesimoquinta pasada.
Casos: smokes Go y tests opt-in reales usan `t.Fatalf` con body HTTP,
stdout/stderr, prompts o summaries completos. Si falla una ejecucion con Codex
real, OPES temporal o control plane, el log de test puede persistir rutas
privadas, payloads de dominio, prompts, transcripts o tokens simulados.
Decision pendiente: helper comun de diagnostico para Go tests/smokes con limite
de bytes, redaccion por campo, summary compacto y opt-in explicito para crudo
local. Coordinar con T21, T58, T61, T91, T97 y T103.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector ./modulos/orquesta-server`.
Backlog: `T107 real-smoke-go-diagnostic-redaction`.
```

```text
ID: RAIL-CAND-DECISION-COUNCIL-ROUND-WIRING-001
Origen: scanner backlog 2026-05-24 trigesimosexta pasada.
Casos: `orquesta-decision-council` y `orquesta-director-candidates` construyen
planes de propuesta/critica/voto y candidatos schedulables, pero no hay
composicion que los ejecute como rondas vivas del Director con gates, waits y
aceptacion durable. El riesgo es duplicar deliberacion en prompts libres o
saltar directo a un agente sin quorum ni evidencia de votos.
Decision pendiente: materializar rondas de consejo por `WorkflowTaskV0`/
candidatos, gatear propuesta->critica->voto, exigir quorum/evidencia y cerrar
decision solo por comando/evento durable. No meter proveedor, modelo, HOME,
runtime ni familias reales en el modulo puro.
Test futuro:
`go test -count=1 ./modulos/orquesta-decision-council ./modulos/orquesta-director-candidates ./modulos/orquesta-director-scheduler ./modulos/orquesta-director-tick-input ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack`.
Backlog: `T108 decision-council-operational-rounds`.
```

```text
ID: RAIL-CAND-LEASE-PROGRESS-POLICY-OWNER-001
Origen: scanner backlog 2026-05-24 trigesimosexta pasada.
Casos: `orquesta-core-leases` define leases/timeouts puros, mientras
`orquesta-runtime` y `orquesta-runtime-codex-delivery` mantienen politica de
heartbeat/progreso separada y el scheduler recibe `lease_action_candidates`
como carril distinto. Sin puente, stalled/loop/stopped puede tener umbrales y
acciones divergentes.
Decision pendiente: puente por puerto desde `AgentProgressReportV0` +
`AgentLeasePolicyV0` a `AgentTimeoutAssessmentV0`/`AgentLeaseExpired`, con reloj
inyectado por adaptador, refs compactas y replay idempotente. No persistir PID,
HOME, rutas, stdout/stderr, prompts, transcripts, proveedor/modelo ni payloads
de runtime.
Test futuro:
`go test -count=1 ./modulos/orquesta-core-leases ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-director ./modulos/orquesta-director-scheduler ./modulos/orquesta-director-tick-input ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T109 agent-lease-progress-policy-bridge`.
```

```text
ID: RAIL-CAND-SCHEDULER-CYCLE-POLICY-001
Origen: scanner backlog 2026-05-24 trigesimoseptima pasada.
Casos: `DirectorSchedulerTickInputV0` filtra campos operativos con una lista
local que incluye `runtime`, `provider`, `model`, `db`, `sql`, `home`,
`filesystem` y `docker`; el ciclo de dispatch del Director conserva otra lista
local para compactar error codes que contiene `db`, `sql`, `provider`, `home`,
`prompt` y `token`.
Decision pendiente: usar politica comun por campo o wrapper local alineado con
`orquesta-rails`: vocabulario operativo opaco pasa; valores sensibles
efectivos, rutas privadas, prompts/transcripts crudos y payloads masivos
bloquean o se redactan.
Test futuro:
`go test -count=1 ./modulos/orquesta-director-scheduler ./modulos/orquesta-director ./modulos/orquesta-rails ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack`.
Backlog: `T110 director-scheduler-cycle-rail-policy-sync`.
```

```text
ID: CONTEXT-CAND-LOCAL-AGENTS-DOC-001
Origen: scanner backlog 2026-05-24 trigesimoseptima pasada.
Casos: `AGENTS.md` locales de core workflow, core concurrency, core leases,
core replanner y director scheduler apuntan a
`../../docs/reinicio_orquesta_v2/protocolo_anti_bucles.md`, pero el directorio
`docs/reinicio_orquesta_v2` no existe en la foto actual del repo.
Decision pendiente: resolver cada ref obligatoria a doc vigente, marcarla como
historica con sustituto o retirar la obligacion. Si el contexto requerido falta,
el agente debe pedir `CONSULTA AL DIRECTOR` o recibir bundle materializado, no
inventar protocolo.
Test futuro:
`go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-core-workflow ./modulos/orquesta-core-concurrency ./modulos/orquesta-core-leases ./modulos/orquesta-core-replanner ./modulos/orquesta-director-scheduler`.
Backlog: `T111 local-agents-required-doc-refs-sync`.
```

```text
ID: RAIL-CAND-RUN-QUEUE-WORKSET-001
Origen: scanner backlog 2026-05-24 trigesimoseptima pasada.
Casos: `orquesta-core-concurrency` calcula claims/read-write sets y el
scheduler gatea agentes dentro de una run, pero `orquesta-run-queue` solo rankea
candidatos por estado/prioridad/aging y declara bloqueos/leases/despacho fuera
de alcance. Dos runs de automejora con write-set solapado pueden llegar al
supervisor global sin gate comun previo.
Decision pendiente: transportar claims compactos en la cola o proyeccion
asociada y evaluar solapes contra runs vivos/reservados antes de launch.
Coordinar con T31 para leases de cola y con T43 para merge documental de
scanners; no leer Git ni filesystem real desde `orquesta-run-queue`.
Test futuro:
`go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-supervisor ./modulos/orquesta-run-memory ./modulos/orquesta-run-file ./modulos/orquesta-core-concurrency ./modulos/orquesta-director-scheduler ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T112 run-queue-workset-concurrency-bridge`.
```

```text
ID: CONTEXT-CAND-MODULE-HISTORICAL-DOC-001
Origen: scanner backlog 2026-05-24 trigesimoctava pasada.
Casos: ademas de los `AGENTS.md` cubiertos por T111, docs locales de
`orquesta-core`, `orquesta-core-workflow`, `orquesta-capacity`,
`orquesta-governance` y `orquesta-observability` siguen apuntando a
`docs/reinicio_orquesta_v2/*`, arbol que no existe en la foto vigente.
Decision pendiente: clasificar esas refs como historicas/stale o sustituirlas
por fuente vigente; no recrear DBV1/control-plane ni snapshots antiguos solo
para satisfacer contexto.
Test futuro:
`go test -count=1 ./modulos/orquesta-context ./cmd/orquesta-server`.
Backlog: `T113 module-historical-doc-ref-sync`.
```

```text
ID: RAIL-CAND-RUN-QUEUE-FAIRNESS-001
Origen: scanner backlog 2026-05-24 trigesimoctava pasada.
Casos: `RunSchedulingCandidateV0` transporta `fairness_group_ref` y el ranking
tiene aging determinista, pero la politica actual solo usa aging como desempate
dentro de la misma prioridad. Sin owner de fairness por grupo, automejora idle,
smokes reales o trabajo humano pueden monopolizar la cola o quedar hambrientos.
Decision pendiente: definir politica de fairness por grupo/app con reloj
inyectado, reason codes publicos y coordinacion con lease de T31 y work-set de
T112. No elevar trabajos reales con efectos externos por encima de confirmaciones
opt-in.
Test futuro:
`go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-supervisor ./modulos/orquesta-run-memory ./modulos/orquesta-run-file ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T114 run-queue-fairness-group-policy`.
```

```text
ID: CONTEXT-CAND-WEB-INTAKE-SESSION-001
Origen: scanner backlog 2026-05-24 trigesimoctava pasada.
Casos: `modulos/orquesta-web/docs/tareas.md` mantiene `WEB-013` pendiente para
reemplazar formulario largo de nueva app por sesion conversacional de intake.
T76 cubre routing AppSpec hacia Director V2, pero no el contrato web de sesion
parcial, pregunta pendiente, AppSpec parcial, i18n y fallback fino.
Decision pendiente: completar contrato web de intake como adaptador sobre
puertos/API de intake/director; la web no planifica, no elige runtime/proveedor
ni reconstruye AppSpec fuera del contrato canonico.
Test futuro:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-app-director-intake ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway ./cmd/orquesta-server`.
Backlog: `T115 web-app-intake-session-contract`.
```

```text
ID: DOC-CAND-MODULE-TASK-INTEGRITY-001
Origen: scanner backlog 2026-05-24 trigesimonovena pasada.
Casos: `modulos/orquesta-app-codex-stack/docs/tareas.md` declara dos secciones
`APP-CODEX-STACK-012`, y `modulos/orquesta-cli/docs/pruebas.md` mantiene casos
`CLI-P001`, `CLI-P007` y `CLI-P008` como pendientes aunque existen tests locales
para ayuda sin red, estado de servidor sin fallback local e inventario V1 en
cuarentena.
Decision pendiente: crear linter/indice que detecte IDs duplicados y estados
stale en docs locales antes de que el planner cierre o relance trabajo por
heading ambiguo.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-context ./modulos/orquesta-cli ./modulos/orquesta-app-codex-stack`.
Backlog: `T116 module-task-doc-integrity-linter`.
```

```text
ID: RAIL-CAND-APP-CODEX-REVIEW-GATE-001
Origen: scanner backlog 2026-05-24 trigesimonovena pasada.
Casos: la documentacion local de `orquesta-app-codex-stack` deja como pendiente
separar una politica productiva de rechazo/replanificacion por entregas
invalidas. Hoy coexisten review gate, ACK validator, snapshot/worktree, prompt
de 300 lineas y reglas de tests/write-set como fuentes cercanas.
Decision pendiente: definir owner por puerto para issues de entrega y evidencia
causal; no duplicar la regla de tamano ni aplicar heuristicas Go a perfiles no
Go o documentales.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-director-service`.
Backlog: `T117 app-codex-review-gate-policy-owner`.
```

```text
ID: CONTEXT-CAND-LEGACY-GENERATED-DOCS-001
Origen: scanner backlog 2026-05-24 trigesimonovena pasada.
Casos: `docs/plan_microtareas.md` y decisiones de
`orquesta-app-director-intake` nombran artefactos de app generada como
`docs/manual_desarrollador.md`, `docs/manual_sistemas_deploy.md`,
`docs/pruebas_documentales.md` y `docs/pendientes.md`, que no existen como docs
vivos del repo Orquesta.
Decision pendiente: marcar esos planes como historicos/debug o convertir los
nombres en contrato de proyecto generado; no crear archivos raiz vacios ni
tratar esos `test -s` como pruebas obligatorias del nucleo.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-director-intake ./modulos/orquesta-context ./cmd/orquesta-server`.
Backlog: `T118 legacy-generated-doc-artifact-contract-sync`.
```

```text
ID: CONTEXT-CAND-SERVER-FIRST-USAGE-DOC-001
Origen: scanner backlog 2026-05-24 cuadragesima pasada.
Casos: `docs/uso_actual_app_orquesta.md` describe comandos y rutas legacy como
`./orquesta serve`, `/api/status`, `/api/agentes`, `/api/runtime-transcript`,
OpenClaw y AP-077 mientras la composicion vigente usa servidor actual y rutas
`/api/v0/*`.
Decision pendiente: clasificar ese manual como historico o sincronizarlo con la
foto server-first vigente; no dejar que el planner genere codigo contra API V1
ni reabra control-plane legacy.
Test futuro:
`go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-web ./cmd/orquesta-server`.
Backlog: `T119 server-first-usage-doc-route-sync`.
```

```text
ID: CONTEXT-CAND-LEGACY-SQLITE-FORENSIC-DOC-001
Origen: scanner backlog 2026-05-24 cuadragesima pasada.
Casos: docs locales de governance, core y capacity conservan referencias a
snapshots SQLite/DBV1 y comandos `sqlite3` como evidencia forense. Algunas
entradas usan rutas absolutas historicas y pueden parecer prerequisito vivo si
un indice federado no respeta cuarentena/freshness.
Decision pendiente: marcar esas refs como forenses/historicas, retirar rutas
absolutas de fuentes vivas y bloquear que DBV1 o `sqlite3` se programen como
trabajo operativo sin decision del director.
Test futuro:
`go test -count=1 ./modulos/orquesta-governance ./modulos/orquesta-core ./modulos/orquesta-capacity ./modulos/orquesta-context`.
Backlog: `T120 legacy-sqlite-forensic-doc-quarantine`.
```

```text
ID: OPS-CAND-MANUAL-AGENT-COMPAT-001
Origen: scanner backlog 2026-05-24 cuadragesima pasada.
Casos: `docs/operacion_agentes_manuales.md` y el manual de uso mantienen
wrappers `scripts/inicio_agente.sh`, Terminator y sesiones manuales como capa de
compatibilidad, pero no hay contrato/linter que pruebe que siguen subordinados
al daemon y no mutan estado por fuera del control plane.
Decision pendiente: catalogar scripts manuales como recuperacion asistida,
exigir paso por CLI/API vigente o bloqueo verificable y documentar correlacion
con run/task/agent refs sin convertir tmux/Terminator en nucleo.
Test futuro:
`bash -n scripts/inicio_agente.sh scripts/cargar_agentes.sh scripts/terminator_agentes.sh scripts/agente_console.sh` y `go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T121 manual-agent-ops-compatibility-contract`.
```

```text
ID: CONTEXT-CAND-CANONICAL-DOC-PRECEDENCE-001
Origen: scanner backlog 2026-05-24 cuadragesima primera pasada.
Casos: `docs/BIBLIA_APP_ORQUESTA.md` se declara doctrina canonica y
`docs/00_INDICE.md` la lista como `M00`, aunque `estado_actual` ya la trata
como historica/stale y `docs/README.md` conserva el marco "Orquesta v1".
Decision pendiente: marcar los docs historicos desde dentro, fijar orden de
precedencia verificable y evitar que OpenClaw, SQLite, rutas `/api/*` o
control-plane V1 se lean como foto vigente.
Test futuro:
`go test -count=1 ./modulos/orquesta-context ./cmd/orquesta-server`.
Backlog: `T122 canonical-doc-precedence-self-sync`.
```

```text
ID: CONTEXT-CAND-FACTORY-BACKLOG-PREVIEW-001
Origen: scanner backlog 2026-05-24 cuadragesima primera pasada.
Casos: `/api/v0/apps/spec` devuelve `BacklogInicialPropuestoV0` como `backlog`;
la web lo renderiza como preview, pero el contrato HTTP no transporta estado
`preview/no_ejecutable`, freshness ni handoff a Director V2.
Decision pendiente: separar backlog determinista inicial de plan operativo y
exigir refs/estado cuando se convierta en trabajo real.
Test futuro:
`go test -count=1 ./modulos/orquesta-factory ./modulos/orquesta-factory-http ./modulos/orquesta-web ./modulos/orquesta-app-director-intake ./modulos/orquesta-mcp ./cmd/orquesta-server`.
Backlog: `T123 factory-backlog-preview-director-handoff`.
```

```text
ID: CONTEXT-CAND-LEGACY-EXTERNAL-ORCHESTRATOR-DOC-001
Origen: scanner backlog 2026-05-24 cuadragesima primera pasada.
Casos: `BIBLIA_APP_ORQUESTA.md`, `op_088_orquesta_servidor_mcp.md`,
`runtime_worker_contract.md` y OPs cercanas presentan OpenClaw, `tmux`,
`/api/mcp` o servidor MCP como superficies canonicas, aunque la foto vigente
los deja como adaptadores/composiciones opt-in o historia.
Decision pendiente: cuarentenar esos docs para que no alimenten planificacion
automatica ni creen endpoints/runtimes no cableados.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./cmd/orquesta-server`.
Backlog: `T124 legacy-external-orchestrator-doc-quarantine`.
```

```text
ID: CONTEXT-CAND-WORK-PROFILES-EMPTY-MODULE-001
Origen: scanner backlog 2026-05-24 cuadragesima segunda pasada.
Casos: `modulos/orquesta-work-profiles` existe solo como directorio vacio con
`docs/`, mientras `WorkProfileV0` vive en `orquesta-core-workflow` y una
decision local ya descarta crear `orquesta-work-profiles` como owner. Un indice
federado puede tratar ese path como modulo pendiente y duplicar el contrato de
perfiles.
Decision pendiente: marcar el directorio como historico/placeholder no
ejecutable o darle owner explicito; el planner no debe abrir tareas desde
directorios sin contrato efectivo.
Test futuro:
`go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-planner ./modulos/orquesta-context ./cmd/orquesta-server`.
Backlog: `T125 work-profiles-empty-module-quarantine`.
```

```text
ID: OPS-CAND-MODULE-CODEX-LAUNCHERS-001
Origen: scanner backlog 2026-05-24 cuadragesima segunda pasada.
Casos: muchos `modulos/*/arrancar_codex.sh` delegan en
`../_comun/arrancar_codex_modulo.sh`, pero `modulos/_comun` no existe en la
foto revisada. Algunas READMEs locales aun recomiendan esos wrappers como
arranque manual, lo que puede saltarse daemon, cola, write-set, ACK y shutdown
gobernado.
Decision pendiente: inventariar wrappers, catalogar compatibilidad vs ruta
vigente y exigir helper comun/proof `bash -n` o marcar bloqueo publico. No
crear una via paralela de runtime Codex fuera de OrquestaV2.
Test futuro:
`bash -n $(find modulos -name arrancar_codex.sh | sort)` y `go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T126 module-local-codex-launcher-compatibility-contract`.
```

```text
ID: RAIL-CAND-IGNORED-LOCAL-ARTIFACTS-001
Origen: scanner backlog 2026-05-24 cuadragesima segunda pasada.
Casos: `.gitignore` excluye artefactos locales de control/diagnostico como
`.ssl-key.log`, `.orquesta-runtime/`, `.orquesta-server` y
`.orquesta-smoke-work`, y algunos existen en la raiz. Si snapshots, contexto,
AppVCS, promocion o ACK terminal caminan el filesystem sin politica comun,
pueden incluir material local ignorado como si fuera producto.
Decision pendiente: politica ejecutable de exclusion por categoria con recibo
compacto; `.gitignore` ayuda pero no sustituye validacion por puerto ni
redaccion. Artefactos locales ignorados no entran en `ACK.files`, contexto,
commits ni evidencias de cierre.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-context ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./cmd/orquesta-server`.
Backlog: `T127 ignored-local-artifact-exclusion-policy`.
```

```text
ID: RAIL-CAND-AGENT-PROGRESS-SUPERVISOR-001
Origen: scanner backlog 2026-05-24 cuadragesima tercera pasada.
Casos: `modulos/orquesta-director/agent_progress_supervisor_helpers_v0.go`
mantiene una lista local de fragmentos prohibidos que incluye vocabulario
operacional como `provider`, `model`, `db`, `sql`, `home` y `runtime`.
Evidencias opacas o refs causales pueden desaparecer antes de que el Director
evalua progreso, aunque no expongan secretos ni payloads crudos.
Decision pendiente: mover el criterio a politica comun por campo, tolerante a
refs opacas, y bloquear solo secreto efectivo, rutas locales reales,
prompts/transcripts crudos o contenido masivo.
Test futuro:
`go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-core-leases ./modulos/orquesta-rails ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack`.
Backlog: `T128 agent-progress-supervisor-rail-policy-sync`.
```

```text
ID: OPS-CAND-DAEMON-START-ENV-001
Origen: scanner backlog 2026-05-24 cuadragesima tercera pasada.
Casos: `cmd/orquesta-server/daemon.go` arranca el proceso residente con un
entorno derivado de `os.Environ()` y defaults de detail rails. Falta recibo de
categorias permitidas/redactadas y contrato publico para estado, diagnostico y
errores de variables requeridas.
Decision pendiente: definir perfil/allowlist de entorno efectivo para `start`,
redactar snapshot operativo y coordinar con la proyeccion Codex, bootstrap,
logs de daemon y entorno de runtime.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./modulos/orquesta-observability`.
Backlog: `T129 server-daemon-start-env-policy`.
```

```text
ID: RAIL-CAND-ARCHITECTURE-TEST-GUARDS-001
Origen: scanner backlog 2026-05-24 cuadragesima tercera pasada.
Casos: varias pruebas de arquitectura combinan guards de imports/efectos con
listas de terminos por substring como `runtime`, `http`, `mcp` o `sql`. Ese
patron puede bloquear comentarios, refs opacas o nombres de politica validos, y
duplica railes de redaccion/neutralidad fuera de un helper comun.
Decision pendiente: conservar guards estrictos de imports/efectos concretos y
acotar listas textuales por campo o matriz de falsos positivos.
Test futuro:
`go test -count=1 ./modulos/orquesta-run-control ./modulos/orquesta-run-queue ./modulos/orquesta-run-memory ./modulos/orquesta-director-candidates ./modulos/orquesta-app-director-intake ./modulos/orquesta-app-runner ./modulos/orquesta-director-agent-workflow ./modulos/orquesta-app-director-service ./modulos/orquesta-rails`.
Backlog: `T130 architecture-guard-test-rail-policy-owner`.
```

```text
ID: RAIL-CAND-DOC-LOCAL-PATH-001
Origen: scanner backlog 2026-05-24 cuadragesima cuarta pasada.
Casos: docs operativos de auditoria y estado pueden incluir rutas absolutas
locales, state dirs o ficheros de control como ejemplo de diagnostico. Si el
contexto materializado o el ACK los transporta sin clasificacion, pasan a
parecer evidencia de producto o write-set valido.
Decision pendiente: permitir variables y refs opacas, pero bloquear o marcar
como no exportables HOME, state dirs, `.orquesta-runtime`, prompts, transcripts,
logs locales y ficheros de control en contexto/ACK/snapshots.
Test futuro:
`go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T131 documentation-local-path-redaction-linter`.
```

```text
ID: RAIL-CAND-OBSERVABILITY-PRIVACY-001
Origen: scanner backlog 2026-05-24 cuadragesima cuarta pasada.
Casos: `modulos/orquesta-observability` usa listas propias para claves y
fragmentos sensibles, separadas de `orquesta-rails` y de la auditoria JSONL.
Ese rail puede divergir: bloquear refs opacas como `token_policy_ref` o dejar
pasar payloads crudos si otro canal usa una lista distinta.
Decision pendiente: fijar owner de taxonomia/redaccion, distinguir refs de
valores efectivos y mantener proyecciones con `redaction_level` verificable
para API/MCP/web.
Test futuro:
`go test -count=1 ./modulos/orquesta-observability ./modulos/orquesta-rails ./modulos/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-web ./cmd/orquesta-server`.
Backlog: `T132 observability-privacy-taxonomy-rail-sync`.
```

```text
ID: DOC-CAND-MODULE-AGENTS-COVERAGE-001
Origen: scanner backlog 2026-05-24 cuadragesima cuarta pasada.
Casos: modulos de frontera sensible como `orquesta-rails`,
`orquesta-domain-work-http` y `orquesta-factory-http` no tienen guia local
completa para agentes. El fallo no es de runtime, pero si de rail documental:
un agente puede tocar HTTP/rails/factory con solo reglas raiz y sin owner local.
Decision pendiente: exigir `AGENTS.md` local o fuente sustituta explicita para
modulos sensibles antes de que el planner prepare runs automaticas.
Test futuro:
`go test -count=1 ./modulos/orquesta-rails ./modulos/orquesta-domain-work-http ./modulos/orquesta-factory-http ./modulos/orquesta-context ./cmd/orquesta-server`.
Backlog: `T133 module-boundary-local-agent-doc-coverage`.
```

```text
ID: OPS-CAND-SERVER-STATUS-LEGACY-ALIAS-001
Origen: scanner backlog 2026-05-24 cuadragesima quinta pasada.
Casos: `cmd/orquesta-server/commands.go` consulta `/api/status` para
`status` y espera de apagado, mientras `modulos/orquesta-server/handler_v0.go`
acepta tambien `/api/v0/server/status`. La ruta versionada ya es la superficie
publica normal, pero el alias legacy sigue vivo sin contrato de deprecacion.
Decision pendiente: canonizar `/api/v0/server/status`; dejar `/api/status`
solo como alias legacy auditado o bloquearlo con error publico de migracion.
Coordinar con T119/T85/T87 sin duplicar freshness, redaccion ni readiness.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-cli ./modulos/orquesta-web ./cmd/orquesta-server`.
Backlog: `T134 server-status-legacy-alias-sunset`.
```

```text
ID: AUTOPROG-CAND-PLANNER-FALLBACK-SCOPE-001
Origen: scanner backlog 2026-05-24 cuadragesima quinta pasada.
Casos: si el planner de backlog no puede leer el documento o no detecta tareas
pendientes, puede devolver una request fallback basada en la configuracion
generica de automejora y heredar write-set historico de servidor. Un fallo de
scanner documental queda asi convertido en tarea de codigo amplia.
Decision pendiente: el fallback debe bloquear, pedir revision o emitir scanner
acotado a docs salvo opt-in explicito con write-set, pruebas y causa publica.
`planner_empty`, errores de lectura y ACKs ambiguos deben verse en status.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-autoprogramming ./cmd/orquesta-server`.
Backlog: `T135 idle-self-improvement-planner-fallback-safety`.
```

```text
ID: OPS-CAND-RESIDENT-WAIT-BUDGET-001
Origen: scanner backlog 2026-05-24 cuadragesima quinta pasada.
Casos: el supervisor residente recorta `MaxExternalWaits` a 1, mientras otros
flujos del Director admiten presupuestos mayores para Codex real, OPES temporal
u operador. El residente puede tratar una espera externa viva como idle o
capacidad libre antes de que lleguen ACK/delivery causales.
Decision pendiente: definir politica por modo, publicar el valor efectivo y
evitar clamps silenciosos; una run en `wait_external` con refs/lease/ACK
pendiente debe bloquear nueva automejora idle hasta quedar quiescent.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-run-supervisor ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T136 resident-drain-external-wait-budget-policy`.
```

```text
ID: HTTP-CAND-REQUEST-BODY-BOUNDS-001
Origen: scanner backlog 2026-05-24 cuadragesima sexta pasada.
Casos: muchas rutas HTTP publicas de `orquesta-mcp` y `orquesta-web` usan
`json.NewDecoder(r.Body).Decode` directo, mientras `/mcp` si limita JSON-RPC con
`LimitReader`. La politica de limite, content-type, trailing data y campos
desconocidos queda duplicada o ausente por handler.
Decision pendiente: helper comun de lectura JSON HTTP por frontera publica,
limites configurables, errores compactos y rechazo de cuerpo crudo en respuestas
de error.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-web ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T137 public-http-request-body-bounds`.
```

```text
ID: HTTP-CAND-RESPONSE-BODY-BOUNDS-001
Origen: scanner backlog 2026-05-24 cuadragesima sexta pasada.
Casos: comandos y clientes como `run-status`, `status`, web/MCP y conectores
HTTP leen o decodifican respuestas completas sin limite comun; errores HTTP
pueden incluir bodies grandes o material sensible del peer.
Decision pendiente: helper de respuesta con limite, descarte seguro, resumen
redactado y error publico estable; no propagar HTML, transcripts, rutas locales,
tokens ni payloads de dominio en stderr/stdout/auditoria.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector`.
Backlog: `T138 outbound-http-response-bounds-redaction`.
```

```text
ID: OPS-CAND-COMMAND-OUTPUT-PUBLIC-SHAPE-001
Origen: scanner backlog 2026-05-24 cuadragesima sexta pasada.
Casos: `status`, `run-status`, `opes-drain-once`, `mcp-real-smoke` y comandos
`codex-wave-*` escriben bodies o summaries propios en stdout. Algunos incluyen
base URLs, state fallback, diagnostico local o datos de runtime que no tienen
shape/redaccion publica unica.
Decision pendiente: DTO publico versionado por comando con freshness y
`redaction_level`; diagnostico crudo solo opt-in local y nunca como fuente
terminal de ACK/delivery/cierre.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-observability ./modulos/orquesta-rails ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack`.
Backlog: `T139 command-output-public-shape-contract`.
```

```text
ID: DOC-CAND-BACKLOG-OVERLAP-001
Origen: scanner backlog 2026-05-24 cuadragesima septima pasada.
Casos: el backlog contiene pares con solape material ya escrito:
`T102 legacy-http-json-boundary-policy` y
`T137 public-http-request-body-bounds`; `T103 outbound-http-response-limit-redaction`
y `T138 outbound-http-response-bounds-redaction`. Sin canon/alias, el planner
puede lanzar dos tareas equivalentes, dividir criterios o cerrar una mientras
la otra sigue como pendiente.
Decision pendiente: consolidar duplicados existentes con tarea canonica,
aliases y merge de criterios/tests, sin borrar historia. El planner debe
priorizar la canonica y marcar duplicados equivalentes como
`duplicate_backlog_task`.
Test futuro:
`go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T140 backlog-overlap-canonicalization`.
```

```text
ID: OPS-CAND-NONTEST-PANIC-001
Origen: scanner backlog 2026-05-24 cuadragesima septima pasada.
Casos: `rg` encontro `panic(` en codigo no-test:
`cmd/orquesta-server/mcp_real_smoke_v0.go:mustMarshalMCPRealSmokeV0` y
`modulos/orquesta-core-leases/lease_evaluator_validation_v0.go:mustParseAgentLeaseInstantV0`.
Aunque se llamen tras invariantes previas, una frontera publica, smoke opt-in o
validador neutral no debe tumbar el proceso por marshal/parse/invariante rota.
Decision pendiente: convertir esos caminos en errores publicos recuperables o
issues durables; reservar helpers `must*` para tests o inicializacion cerrada
con invariante probada.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-core-leases ./modulos/orquesta-server ./modulos/orquesta-observability`.
Backlog: `T141 public-boundary-no-panic-contract`.
```

```text
ID: OPS-CAND-CLI-INPUT-BOUNDS-001
Origen: scanner backlog 2026-05-24 cuadragesima octava pasada.
Casos: `modulos/orquesta-cli/command_flags_v0.go` lee `stdin` con
`io.ReadAll` y `--input` con `os.ReadFile` antes de aplicar limite comun o
clasificar si el origen es operador local, agente/toolbelt, path sensible o
fichero de control.
Decision pendiente: acotar bytes de entrada por comando, devolver error publico
sin body crudo y bloquear o marcar como opt-in local rutas absolutas, HOME,
`.orquesta-runtime`, prompts, transcripts, logs y ficheros de control.
Test futuro:
`go test -count=1 ./modulos/orquesta-cli ./cmd/orquesta-server`.
Backlog: `T142 cli-json-input-bounds-and-source-policy`.
```

```text
ID: RUNTIME-CAND-CODEX-CONTROL-FILE-BOUNDS-001
Origen: scanner backlog 2026-05-24 cuadragesima octava pasada.
Casos: `agent_ack.json`, `agent_shutdown_checkpoint_ack.json` y lecturas
compatibles de ACK/sidecars se leen completas con `os.ReadFile` en validadores,
planner u observation source. La validacion estructural existe, pero no hay
limite comun ni reason code especifico de sobrelimite.
Decision pendiente: helper de lectura acotada para ficheros de control Codex,
error publico compacto, sin path local ni contenido crudo, y reutilizacion desde
ACK, decisiones, checkpoint y planner.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T143 codex-control-file-size-and-redaction-policy`.
```

```text
ID: DOMAIN-CAND-ARTIFACT-INTAKE-BOUNDS-001
Origen: scanner backlog 2026-05-24 cuadragesima novena pasada.
Casos: `readDomainWorkDeliveryBodyV0` abre el primer fichero de `ACK.files` con
`os.ReadFile` y lo transforma en payload `domain_work`. La ruta se valida contra
`ProjectWorkDir`, pero falta limite por artifact type, deteccion de binario y
redaccion antes de `PayloadFields`.
Decision pendiente: helper de intake por artefacto con limite, reason code
publico y redaccion por campo; refs opacas y markdown/JSON valido pasan, pero
HOME, rutas privadas, prompts/transcripts, tokens, payloads HTTP crudos,
binarios no declarados y cuerpos enormes no salen al conector de dominio.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-domain-work ./modulos/orquesta-runtime-codex-delivery`.
Backlog: `T144 domain-work-delivery-artifact-intake-policy`.
```

```text
ID: OPERATOR-CAND-MCP-CLIENT-DEADLINE-001
Origen: scanner backlog 2026-05-24 cuadragesima novena pasada.
Casos: `OperatorMCPClientConnectorV0.callToolV0` invoca `CallToolV0` con
`context.Background()`. Un conector externo colgado puede bloquear consulta,
burst, outbox o directed query sin timeout publico ni presupuesto visible.
Decision pendiente: contexto/deadline inyectado por composicion, error publico
estable para timeout/cancelacion y redaccion de detalles de transporte. No
confundir ausencia de conector, timeout y consejo del operador.
Test futuro:
`go test -count=1 ./modulos/orquesta-operator-mcp-client ./modulos/orquesta-operator-mcp ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T145 operator-mcp-client-deadline-budget`.
```

```text
ID: HTTP-CAND-FACTORY-JSON-BOUNDARY-001
Origen: scanner backlog 2026-05-24 cuadragesima novena pasada.
Casos: `modulos/orquesta-factory-http/appspec_http_v0.go` usa `io.ReadAll` en
`POST /api/v0/apps/spec` y queda fuera del alcance explicito de T137, que lista
MCP/web/gateway. Tambien coincide con T133 porque `factory-http` no tiene guia
local completa.
Decision pendiente: incluir `factory-http` en helper/politica JSON publica:
limite de body, content-type, trailing data, campos desconocidos, errores
compactos y guia local o sustituto documental. La respuesta sigue siendo preview
de factory, no plan ejecutable.
Test futuro:
`go test -count=1 ./modulos/orquesta-factory-http ./modulos/orquesta-factory ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway`.
Backlog: `T146 factory-http-json-boundary-coverage`.
```

```text
ID: RUNTIME-CAND-CODEX-PROGRESS-FAILURE-LOG-BOUNDS-001
Origen: scanner backlog 2026-05-24 quincuagesima pasada.
Casos: `codexProgressReadFailureLogV0` lee stdout/stderr/last-message con
`os.ReadFile` completo y recorta despues a 64 KiB para clasificar no-ACK,
interrupcion o capacidad. Un log enorme puede cargar memoria y despues acabar
resumido como decision de progreso sin haber pasado por el rail tail/redaccion.
Decision pendiente: usar lectura tail acotada antes de cargar logs de progreso,
mantener codigos compactos y no exponer stdout/stderr, prompts, transcripts,
HOME, rutas privadas, tokens ni salida cruda de proveedor.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T147 codex-progress-failure-log-tail-bounds`.
```

```text
ID: FILE-CAND-LEDGER-SNAPSHOT-READ-BOUNDS-001
Origen: scanner backlog 2026-05-24 quincuagesima pasada.
Casos: ledgers/snapshots JSON file-based como
`cmd/orquesta-server/external_bridge_input_ledger.go` y
`modulos/orquesta-app-codex-stack/domain_work_delivery_file_ledger_v0.go`
leen el fichero completo antes de validar schema, records o corrupcion. Si el
snapshot crece demasiado o queda corrupto, el error no distingue sobrelimite,
corrupcion recuperable ni riesgo de duplicar efectos externos.
Decision pendiente: helper/politica de lectura acotada por tipo, `max_records`,
errores publicos compactos y recuperacion que no trate ledger ilegible como
vacio. Coordinar con T98, T101 y T104 sin sustituir claim/recovery ni escritura
durable.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack ./modulos/orquesta-domain-work-file ./modulos/orquesta-run-file ./modulos/orquesta-state-file ./modulos/orquesta-runtime-codex-delivery`.
Backlog: `T148 file-ledger-snapshot-read-bounds`.
```

```text
ID: WORKTREE-CAND-SNAPSHOT-READ-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima primera pasada.
Casos: `CaptureWorktreeSnapshotV0` recorre el worktree y
`hashWorktreeFileV0` lee cada fichero completo con `os.ReadFile`. El snapshot
ya se usa como base futura para ACK estricto, line budget y efectos
destructivos, pero no aplica `max_files`, `max_file_bytes`, `max_total_bytes`
ni hash streaming.
Decision pendiente: presupuesto de snapshot por composicion, hash por streaming,
ignore prefixes comunes y errores publicos compactos sin path absoluto ni
contenido de fichero. No usar `ACK.files` como sustituto cuando falte snapshot
valido.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T149 worktree-snapshot-read-budget`.
```

```text
ID: REVIEW-CAND-PROJECT-TREE-SCAN-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima primera pasada.
Casos: los gates `codexReviewGate*`, `reviewRework*` y
`domainWorkRecoveryFilesUnderDirV0` usan `filepath.WalkDir` para comprobar
globs/carpetas o recuperar artefactos, con listas de ignore copiadas y sin
contexto, max entradas, max profundidad ni reason code cuando el scan agota
presupuesto.
Decision pendiente: helper/politica comun de scan de proyecto para existencia y
recovery, tolerante con alias/rutas hijas pero bloqueando traversal, HOME,
control files, prompts, transcripts, logs y binarios enormes como evidencia de
producto.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-worktree`.
Backlog: `T150 project-tree-scan-budget-for-review-recovery`.
```

```text
ID: CODEX-CAND-WAVE-FILE-INPUT-BOUNDS-001
Origen: scanner backlog 2026-05-24 quincuagesima primera pasada.
Casos: `codexWavePromptTextV0`, `codexDirectorObjectiveTextV0` y
`codexDirectorDomainContextBlocksFromFilesV0` leen ficheros de operador con
`os.ReadFile` completo antes de crear prompts/contexto para agentes Codex.
Decision pendiente: limite por fichero, validacion texto/UTF-8, politica de
origen y errores publicos redactados. No aceptar `.orquesta-runtime`,
`.orquesta-codex-runtime`, logs, ACKs, checkpoints, prompts/transcripts previos,
HOME ni rutas privadas como contexto salvo opt-in local auditado.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree`.
Backlog: `T151 codex-wave-operator-file-input-bounds`.
```

```text
ID: RAIL-CAND-CODEX-CODEHOME-COPY-BOUNDS-001
Origen: scanner backlog 2026-05-24 quincuagesima segunda pasada.
Casos: la proyeccion de `CODEX_HOME` en olas Codex usa allowlist de nombres,
pero copia ficheros/directorios con `os.Stat`, `os.ReadFile` y `WalkDir` sin
presupuesto de bytes/ficheros ni politica explicita de symlinks, modos o
entradas omitidas.
Decision pendiente: aplicar presupuesto de copia, `Lstat`/resolucion segura,
modos seguros y recibo compacto de categorias copiadas/omitidas. No copiar
secretos, HOME, memorias/plugins completos ni rutas privadas como evidencia.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack`.
Backlog: `T152 codex-code-home-copy-bounds-symlink-policy`.
```

```text
ID: RAIL-CAND-WORKFLOW-PAYLOAD-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima segunda pasada.
Casos: outbox valida `maxOutboxPayloadBytesV0`, pero comandos/eventos del
workflow serializan o decodifican `json.RawMessage` sin presupuesto comun antes
de persistir en `orquesta-state-file`.
Decision pendiente: limite por tipo para comandos/eventos, validacion previa a
`json.Unmarshal`/compactacion y uso de refs de artefacto para payloads grandes
o crudos.
Test futuro:
`go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-state-file ./modulos/orquesta-orchestration-core ./modulos/orquesta-director ./modulos/orquesta-app-director-service`.
Backlog: `T153 workflow-command-event-payload-budget`.
```

```text
ID: RAIL-CAND-EFFECT-DEADLINE-CONTEXT-001
Origen: scanner backlog 2026-05-24 quincuagesima segunda pasada.
Casos: puertos/adaptadores con efectos externos pueden recibir `context nil` o
ser invocados desde `context.Background()` y quedar gobernados solo por timeout
local o por el cliente inyectado.
Decision pendiente: deadline/cancelacion por politica de composicion para
runtime launch/stop, HTTP domain_work, OPES REST, MCP operador, bridge externo y
shutdown; errores publicos compactos para timeout/cancelacion.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./modulos/orquesta-operator-mcp-client ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T154 effect-port-deadline-context-policy`.
```

```text
ID: RAIL-CAND-IDLE-BACKLOG-ACK-SCAN-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima tercera pasada.
Casos: `completedBacklogRequestRefsV0` camina toda `.orquesta-runtime` para
encontrar `agent_ack.json` y deduplicar requests de backlog completadas. En
runs con olas Codex, homes de agentes, plugins y children, ese scan puede tocar
material ajeno al planner sin max de dirs/ficheros/profundidad/tiempo.
Decision pendiente: indice o scan acotado por refs esperadas, limite de bytes
por ACK, schema/correlacion terminal y degradacion observable
`backlog_ack_scan_budget_exhausted`/`backlog_ack_scan_ambiguous`. Coordinar con
T33, T79, T135 y T143.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Backlog: `T155 idle-backlog-runtime-ack-scan-budget`.
```

```text
ID: RAIL-CAND-APP-VCS-GIT-OUTPUT-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima tercera pasada.
Casos: `GitAppVCSConnectorV0` y promocion de staging ejecutan Git con
`CombinedOutput()` para `status`, `commit`, `push` y `rev-parse`; `git status
--porcelain --untracked-files=all` puede devolver demasiadas rutas antes de que
AppVCS aplique write-set o evidencias de promocion.
Decision pendiente: captura limitada de stdout/stderr Git, `max_changed_paths`,
errores publicos `git_output_too_large`/`git_status_too_many_paths`, redaccion
de remotos/rutas/HOME/tokens y timeout `git_command_timeout`. Coordinar con T39,
T105, T139 y T154.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp ./cmd/orquesta-server`.
Backlog: `T156 app-vcs-git-output-and-path-budget`.
```

```text
ID: RAIL-CAND-CODEX-CONTROL-FILE-ROOT-SYMLINK-001
Origen: scanner backlog 2026-05-24 quincuagesima cuarta pasada.
Casos: `ReadCodexAgentAckFileV0`, `ReadCodexShutdownCheckpointAckFileV0`,
`directorDecisionFileExistsV0` y `codexProgressReadTailV0` leen ficheros de
control/progreso desde paths de descriptor con `os.ReadFile`, `os.Stat` u
`os.Open`. T143 cubre tamano/redaccion, pero no raiz autorizada, symlinks ni
entradas no regulares antes de abrir.
Decision pendiente: helper comun de lectura por raiz/descriptor que use
`Lstat`/apertura segura, rechace symlinks, dirs, dispositivos o paths fuera de
raiz y devuelva reason codes compactos sin path ni contenido.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T157 codex-control-file-root-and-symlink-policy`.
```

```text
ID: RAIL-CAND-OPES-BRIDGE-DESTINATION-SUMMARY-001
Origen: scanner backlog 2026-05-24 quincuagesima cuarta pasada.
Casos: `opesDrainConfigFromEnvV0` acepta `ORQUESTA_OPES_BASE_URL`/`OPES_BASE_URL`
y `ORQUESTA_BASE_URL` como strings de composicion; `opesDrainSummaryV0` expone
esas bases completas en el summary publico. El bridge tiene confirmacion y
filtro por job, pero no politica OPES especifica de destino ni redaccion de
summary.
Decision pendiente: politica de destino OPES temporal/productivo, rechazo de
credenciales en URL, modo productivo opt-in y summary con refs/categorias en vez
de URL completa o payload crudo.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-opes-connector ./modulos/orquesta-opes-bridge`.
Backlog: `T158 opes-bridge-destination-and-summary-policy`.
```

```text
ID: RAIL-CAND-CODEX-CONTROL-FILE-WRITE-DURABILITY-001
Origen: scanner backlog 2026-05-24 quincuagesima quinta pasada.
Casos: `codex_resolver_v0.go`, `codex_wave_command_v0.go`,
`codex_wave_control_v0.go` y `codex_shutdown_checkpoint_v0.go` escriben
`agent_packet.json`, `agent_prompt.txt`, wrappers, registry o
`orquesta_shutdown_request.json` con `os.WriteFile` directo sobre ruta final.
T143/T157 cubren lectura, pero no escritura atomica, permisos, `fsync`, rechazo
de symlinks ni receipt compacto.
Decision pendiente: helper/puerto de escritura de control files con raiz
autorizada, temp+rename, modos cerrados, conflicto idempotente y receipt por
hash/bytes/tipo sin path local ni contenido crudo.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T159 codex-control-file-durable-write-policy`.
```

```text
ID: RAIL-CAND-DIRECTOR-DECISIONS-BATCH-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima quinta pasada.
Casos: `DirectorAgentDecisionFileSourceV0` limita bytes por fichero, pero
`decisionsFromDescriptorsV0` y `compositeDirectorDecisionSourceV0` agregan
descriptors/fuentes sin presupuesto visible para numero total de decisions,
`create_microtask` o tasks nuevas antes de materializar workflow/outbox.
Decision pendiente: limite por request de descriptors, decisions, microtasks,
tasks y bytes acumulados; exceso con reason code publico y contadores compactos
sin bodies JSON, rutas, prompts ni transcripts. Coordinar con T63 y T153.
Test futuro:
`go test -count=1 ./modulos/orquesta-director-agent-file-source ./modulos/orquesta-director-agent-workflow ./modulos/orquesta-app-codex-stack ./modulos/orquesta-orchestration-core`.
Backlog: `T160 director-decisions-batch-budget-and-source-limit`.
```

```text
ID: RAIL-CAND-CLOCK-REF-GENERATION-001
Origen: scanner backlog 2026-05-24 quincuagesima sexta pasada.
Casos: `codexWaveRefV0` genera refs con resolucion de segundo, varios
adaptadores rellenan `OccurredAt`/`RequestedAt` con `time.Now` directo y web/CLI
tienen fallbacks de request id basados en `UnixNano`. En concurrencia o replay,
un timestamp puede parecer identidad causal aunque solo sea evidencia temporal.
Decision pendiente: owner comun de reloj/ref generator por composicion, reloj
inyectable en tests, colision observable y prohibicion de usar timestamps como
unica identidad terminal.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./modulos/orquesta-web ./modulos/orquesta-cli`.
Backlog: `T161 clock-and-ref-generation-policy`.
```

```text
ID: RAIL-CAND-PUBLIC-CLIENT-IDEMPOTENCY-001
Origen: scanner backlog 2026-05-24 quincuagesima sexta pasada.
Casos: clientes web/CLI/MCP y comandos del servidor propagan `request_id`,
`correlation_id` e `idempotency_key` de forma desigual; algunas mutaciones
generan ID local, otras aceptan idempotency opcional y otras reutilizan
correlacion como identidad. Un retry tras timeout puede duplicar efecto o dejar
traza incompleta.
Decision pendiente: politica comun para mutaciones publicas reintentables:
idempotency estable, correlation transversal, headers coherentes, receipt/estado
observable y errores redactados.
Test futuro:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./modulos/orquesta-mcp ./cmd/orquesta-server ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway`.
Backlog: `T162 public-client-mutation-idempotency-policy`.
```

```text
ID: RAIL-CAND-STATE-FILE-EVENT-LOG-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima septima pasada.
Casos: `AppendRunEventsV0` lee todo el documento de eventos del run, recorre
todos los eventos para deduplicar por `event_id` y reescribe el snapshot JSON
completo en cada append. En runs residentes largas, el coste y el riesgo de
snapshot grande/corrupto crecen antes de que replay o cierre causal puedan
degradar de forma observable.
Decision pendiente: presupuesto por append/run, indice durable por `event_id`,
lectura paginada o ventana causal, compaction compatible y reason codes para
log excesivo/corrupto sin asumir historial vacio.
Test futuro:
`go test -count=1 ./modulos/orquesta-state-file ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service`.
Backlog: `T163 state-file-event-log-index-compaction-budget`.
```

```text
ID: RAIL-CAND-WORKFLOW-TASK-PARENT-INDEX-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima septima pasada.
Casos: `LoadWorkflowTasksByParentV0` hace `os.ReadDir` del directorio de tasks
del run y abre cada JSON para filtrar por `parent_task_ref`. Recursion real,
waits por parent y cierre de arbol pueden depender de un scan sin presupuesto ni
indice, y una ausencia/corrupcion puede parecer "sin hijos" si no se distingue.
Decision pendiente: indice parent/child durable con presupuesto de entradas y
bytes, rebuild acotado con reason code, bloqueo si el indice falta o hay outbox
pendiente, y conservacion de wave/cohort/depth para waits por parentesco.
Test futuro:
`go test -count=1 ./modulos/orquesta-state-file ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack`.
Backlog: `T164 workflow-task-store-parent-index-budget`.
```

```text
ID: RAIL-CAND-SERVER-CONFIG-GLOBAL-ENV-DEFAULTS-001
Origen: scanner backlog 2026-05-24 quincuagesima septima pasada.
Casos: `serverConfigFromEnvV0`, `setDefaultStartupCleanupModeV0` y
`ensureServerDetailRailsDefaultV0` fijan defaults con `os.Setenv` en el entorno
global del proceso. Eso mezcla input explicito del operador con defaults de
composicion y puede afectar tests, smokes, comandos hijos o lecturas posteriores
de config.
Decision pendiente: config efectiva sin mutar entorno global, proyeccion al
daemon marcada como `explicit`/`defaulted`/`derived`, summary redactado por
categorias y tests que demuestren lecturas repetidas sin contaminacion global.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-rails ./modulos/orquesta-app-codex-stack`.
Backlog: `T165 server-config-global-env-defaults-policy`.
```

```text
ID: OPS-CAND-RESIDENT-ASYNC-SHUTDOWN-001
Origen: scanner backlog 2026-05-24 quincuagesima octava pasada.
Casos: `RuntimeV0.RunV0` lanza `server.Serve`, supervisor ticks async e idle
self-improvement en goroutines; durante shutdown usa
`server.Shutdown(context.Background())`, persiste `stopped` con background y no
espera las goroutines internas antes de terminar.
Decision pendiente: quiescencia async con deadline, estados `stopping`/
`async_work_draining`/`stop_timeout`, receipts compactos y bloqueo de writes
posteriores a `stopped`.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server ./modulos/orquesta-observability`.
Backlog: `T166 resident-runtime-async-shutdown-quiescence`.
```

```text
ID: OPS-CAND-EXTERNAL-BRIDGE-LOOP-LIFECYCLE-001
Origen: scanner backlog 2026-05-24 quincuagesima octava pasada.
Casos: `cmd/orquesta-server run` arranca `runOPESBridgeLoopV0` en una goroutine
paralela al runtime; `writeExternalBridgeTickV0` publica cada tick solo por
stderr. El status/auditoria residente no conserva ultimo tick, ultimo error,
estado `stopping` ni join/cancelacion del bridge.
Decision pendiente: registrar lifecycle compacto del bridge externo, coordinar
apagado con el runtime y publicar readiness/status sin URL completa, payload
OPES, respuestas crudas ni rutas locales.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-opes-bridge`.
Backlog: `T167 external-bridge-resident-loop-lifecycle-state`.
```

```text
ID: STATE-CAND-RUN-EVENT-LOAD-VALIDATION-001
Origen: scanner backlog 2026-05-24 quincuagesima octava pasada.
Casos: `StoreV0.LoadRunV0` valida `schema_version` y refs externas del documento
de run, pero no ejecuta `ValidateOrchestrationRunV0` sobre la proyeccion
cargada. `LoadRunEventsV0` valida schema/ref del documento de eventos, pero no
valida cada `OrchestrationEventV0` cargado ni duplicados antes de entregar el
historial a replay/cierre.
Decision pendiente: validar run/eventos al cargar desde state-file, bloquear
cierre/replay ante snapshot invalido y exponer reason code compacto sin asumir
run vacia, sin eventos o completed.
Test futuro:
`go test -count=1 ./modulos/orquesta-state-file ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service`.
Backlog: `T168 state-file-run-event-load-validation`.
```

```text
ID: MCP-CAND-REAL-TRANSPORT-REGISTRATION-COLLISION-001
Origen: scanner backlog 2026-05-24 quincuagesima octava pasada.
Casos: `mcpRealTransportRegistryV0.RegisterResourceV0` y `RegisterToolV0`
guardan envelopes en mapas por nombre/URI y sobrescriben duplicados
silenciosamente. Si dos descriptors colisionan, `/mcp` puede ocultar un
resource/tool sin error de registro.
Decision pendiente: rechazar colisiones de tool/resource en el transporte real,
propagar error desde `RegisterMCPTransportV0` y demostrar que no queda catalogo
parcial tras fallo.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp`.
Backlog: `T169 mcp-real-transport-registration-collision-guard`.
```

```text
ID: RAIL-CAND-OUTBOUND-REDIRECT-POLICY-001
Origen: scanner backlog 2026-05-24 quincuagesima novena pasada.
Casos: los clientes HTTP revisados en `orquesta-domain-work-http`,
`orquesta-opes-connector`, `orquesta-web`, `orquesta-mcp`,
`orquesta-app-gateway` y `cmd/orquesta-server` declaran timeout, pero no
`CheckRedirect` ni una politica comun de cadena 3xx. Go sigue redirects por
defecto, por lo que la URL inicial puede estar validada y el destino efectivo
terminar fuera de origen/politica.
Decision pendiente: declarar `redirect_policy`, revalidar cada salto contra la
politica de destino, bloquear cross-origin o esquema no permitido y no reenviar
headers sensibles salvo permiso explicito redactado.
Test futuro:
`go test -count=1 ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T170 outbound-http-redirect-policy`.
```

```text
ID: RAIL-CAND-PUBLIC-QUERY-FORM-BOUNDS-001
Origen: scanner backlog 2026-05-24 quincuagesima novena pasada.
Casos: `auditHTTPHandlerV0` conserva `raw_query`; endpoints web/server usan
`ParseForm`, `FormValue` o `URL.Query().Get` para controles publicos. T137
limita cuerpos JSON, pero no impone presupuesto/redaccion comun sobre query
string ni form params antes de auditoria/status.
Decision pendiente: reemplazar raw query por resumen redactado, limitar tamano
total, numero de claves, repeticion y longitud por valor, y clasificar rutas,
URLs, tokens, prompts o filtros libres antes de log/auditoria.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway`.
Backlog: `T171 public-query-form-parameter-bounds-redaction`.
```

```text
ID: RAIL-CAND-HTTP-RESPONSE-WRITE-VISIBILITY-001
Origen: scanner backlog 2026-05-24 quincuagesima novena pasada.
Casos: muchos handlers publicos en server/web/MCP/gateway usan
`_ = json.NewEncoder(w).Encode(...)` o ignoran errores de `w.Write(...)`. Si la
serializacion o escritura falla, el dominio puede haber tenido exito mientras
la entrega HTTP queda invisible para status, auditoria o smokes.
Decision pendiente: helper o patron comun para distinguir fallo de dominio,
fallo de serializacion y fallo de escritura; registrar `response_write_failed`
compacto cuando el status ya fue emitido y no filtrar payloads internos.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-observability`.
Backlog: `T172 http-response-encode-write-error-visibility`.
```

```text
ID: RAIL-CAND-BROWSER-ORIGIN-CSRF-001
Origen: scanner backlog 2026-05-24 sexagesima pasada.
Casos: rutas web/API mutables aceptan POST por navegador o cliente local sin
owner visible para `Origin`, `Referer`, CSRF token o intent ref. T55 cubre
auth/bind remoto y T102/T137 cubren JSON/form/query, pero un POST de navegador
hacia loopback o bind opt-in puede activar control plane si solo se valida body.
Decision pendiente: declarar por ruta si acepta navegador, CLI/MCP o gateway;
exigir same-origin o intent token para mutaciones browser y registrar decision
compacta sin cookies, query cruda, URL completa ni payload.
Test futuro:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-app-gateway ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./cmd/orquesta-server`.
Backlog: `T173 browser-origin-csrf-intent-guard`.
```

```text
ID: RAIL-CAND-MCP-JSONRPC-STRICTNESS-001
Origen: scanner backlog 2026-05-24 sexagesima pasada.
Casos: `/mcp` decodifica un unico objeto JSON-RPC con `LimitReader`, pero no
valida estrictamente `jsonrpc=2.0`, trailing tokens, tipo/tamano de `id`,
batch/notification ni presupuesto de `params` por metodo. T23/T169 cubren
transporte opt-in y colisiones de registro; falta owner de protocolo.
Decision pendiente: contrato JSON-RPC estricto o modo legacy declarado para
version, id, batch, notifications, params, `Content-Type`/`Accept` y errores,
sin eco de argumentos, resource payloads, rutas, prompts, transcripts ni tokens.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway`.
Backlog: `T174 mcp-jsonrpc-protocol-strictness`.
```

```text
ID: RAIL-CAND-MCP-OUTPUT-BUDGET-001
Origen: scanner backlog 2026-05-24 sexagesima primera pasada.
Casos: `cmd/orquesta-server/mcp_real_transport_v0.go` envuelve payloads de
`resources/read` y `tools/call` como texto completo sin limite comun de salida,
freshness ni redaccion por resource/tool. Un recurso grande o un resultado de
herramienta con diagnostico amplio puede superar presupuesto o filtrar material
operativo antes de que T174 actue sobre el request.
Decision pendiente: presupuesto de salida MCP por resource/tool, modo summary
por defecto para payloads grandes, errores compactos y redaccion antes de
serializar JSON-RPC.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-observability`.
Backlog: `T175 mcp-tool-resource-output-budget`.
```

```text
ID: RAIL-CAND-CONTROL-PLANE-SECURITY-HEADERS-001
Origen: scanner backlog 2026-05-24 sexagesima primera pasada.
Casos: handlers web/MCP/gateway fijan `Content-Type`, `Allow` y a veces
`X-Correlation-ID`, pero no comparten `Content-Security-Policy`,
`X-Content-Type-Options`, `Referrer-Policy`, anti-frame ni `Cache-Control`.
T173 cubre origen/CSRF; este rail cubre respuesta/cache.
Decision pendiente: helper o politica de headers por perfil HTML, JSON, MCP,
status/read-only y mutacion, con cache/freshness explicita y sin headers que
filtren cookies, query, URLs completas ni datos privados.
Test futuro:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway ./cmd/orquesta-server`.
Backlog: `T176 control-plane-http-security-cache-headers`.
```

```text
ID: RAIL-CAND-REST-BASE-URL-ENDPOINT-001
Origen: scanner backlog 2026-05-24 sexagesima primera pasada.
Casos: clientes REST de web/MCP/CLI y composicion unen base URL y endpoint con
concatenacion o `strings.TrimRight/TrimLeft`; algunos normalizan esquema/host y
otros aceptan `BaseURL` como string ya confiable. Antes de T80/T170/T103 falta
un rail comun para userinfo, query, fragment, base path y endpoint absoluto.
Decision pendiente: normalizador compartido o politica equivalente para base
URL y endpoint relativo, con rechazo de destinos ambiguos y errores publicos
sin URL completa ni credenciales.
Test futuro:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T177 rest-client-base-url-endpoint-policy`.
```

```text
ID: RAIL-CAND-HTTP-CLIENT-TRANSPORT-PROXY-001
Origen: scanner backlog 2026-05-24 sexagesima segunda pasada.
Casos: clientes salientes de web, CLI, MCP, OPES, domain-work y comandos del
servidor crean `http.Client{Timeout: ...}` o cliente nil con transporte por
defecto. Asi proxy, TLS, keepalive y pool de conexiones quedan como decision
implicita del entorno/proceso, incluso para loopback o control plane interno.
Decision pendiente: factory/perfil de transporte por cliente, proxy deny por
defecto para interno/loopback, proxy explicito o allowlisted para egress, y
auditoria por categoria sin URL completa, userinfo, tokens ni rutas privadas.
Test futuro:
`go test -count=1 ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./modulos/orquesta-web ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T178 outbound-http-client-transport-proxy-policy`.
```

```text
ID: RAIL-CAND-INPROCESS-HTTP-TRANSPORT-001
Origen: scanner backlog 2026-05-24 sexagesima segunda pasada.
Casos: `InProcessTransportV0` y el roundtripper del smoke MCP llaman al handler
directamente y acumulan la respuesta en memoria. Los tests/gateway pueden pasar
sin ejercer limite de salida, cancelacion observable, panic recovery o fallos de
escritura que aparecerian en HTTP real.
Decision pendiente: recorder acotado, propagacion/verificacion de deadline,
errores publicos `inprocess_timeout`/`inprocess_response_too_large` y paridad de
headers/status/correlacion con frontera HTTP real.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-gateway ./modulos/orquesta-web ./modulos/orquesta-mcp ./cmd/orquesta-server ./modulos/orquesta-observability`.
Backlog: `T179 inprocess-http-transport-budget-parity`.
```

```text
ID: RAIL-CAND-COMMAND-STDIO-WRITE-001
Origen: scanner backlog 2026-05-24 sexagesima tercera pasada.
Casos: comandos de `cmd/orquesta-server` y runners CLI ignoran errores de
`json.NewEncoder(stdout).Encode(...)` o `fmt.Fprintf(stdout/stderr, ...)`.
T139 define shape publico de comandos y T172 cubre escritura HTTP, pero no hay
owner visible para pipe roto, stdout cerrado o writer de test que falla.
Decision pendiente: comprobar errores de escritura stdio, devolver exit code o
reason code publico y no usar stdout/stderr textual como evidencia terminal de
cierre.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-cli ./modulos/orquesta-observability`.
Backlog: `T180 command-stdio-write-error-visibility`.
```

```text
ID: RAIL-CAND-DOCPLAN-REF-UNIQUENESS-001
Origen: scanner backlog 2026-05-24 sexagesima cuarta pasada.
Casos: `DomainDocumentPlanV0` valida presencia y forma compacta de secciones,
visuales, reviews y entregables, pero no hay rail visible para refs duplicados.
El expander genera `RequestID`/`IdempotencyKey` desde esos refs, asi que un
duplicado puede producir jobs derivados indistinguibles. Ademas, arrays raw
malformados pueden colapsar a `nil` durante canonicalizacion y perder campo
causal.
Decision pendiente: diagnostico estable por campo para arrays raw invalidos,
rechazo de `section_ref`, `visual_ref`, `review_ref` y `deliverable_ref`
duplicados, y garantia de que el expander no emite jobs con misma idempotencia.
Test futuro:
`go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-document-plan-expander ./modulos/orquesta-opes-bridge`.
Backlog: `T181 domain-document-plan-ref-uniqueness-and-diagnostics`.
```

```text
ID: RAIL-CAND-MCP-TOOL-EXEC-BUDGET-001
Origen: scanner backlog 2026-05-24 sexagesima cuarta pasada.
Casos: T174 limita forma/protocolo JSON-RPC y T175 limita salida de
tools/resources, pero el transporte MCP real invoca handlers con el contexto
del request y sin presupuesto de ejecucion por perfil. Un handler lento o que no
observe cancelacion puede retener `/mcp` sin reason code publico propio.
Decision pendiente: deadline/cancel cause por tool/resource antes de invocar
handler, perfiles separados para lectura, mutacion y autoprogramacion larga, y
observabilidad compacta sin argumentos, payloads, prompts ni rutas privadas.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-observability`.
Backlog: `T182 mcp-tool-execution-budget-and-cancellation`.
```

```text
ID: RAIL-CAND-WEB-HTML-RENDER-ERROR-001
Origen: scanner backlog 2026-05-24 sexagesima cuarta pasada.
Casos: renderers HTML de `modulos/orquesta-web` ejecutan templates sobre el
writer HTTP y varios caminos ignoran el error de `template.Execute`. T172 cubre
fallos de escritura HTTP genericos, pero falta owner para distinguir template
invalido, socket tardio y fallback localizado en vistas HTML.
Decision pendiente: comprobar errores de render/escritura, publicar reason code
estable, no reejecutar efectos y registrar solo contadores compactos sin
formularios, query cruda, cookies, payloads, prompts, HOME, rutas privadas ni
tokens.
Test futuro:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-app-gateway ./modulos/orquesta-observability ./cmd/orquesta-server`.
Backlog: `T183 web-html-render-error-contract`.
```

```text
ID: RAIL-CAND-DETERMINISTIC-REF-HASH-COLLISION-001
Origen: scanner backlog 2026-05-24 sexagesima quinta pasada.
Casos: refs y firmas en app-change, MCP, orchestration-core, runtime Codex
delivery y app Codex stack usan FNV32, SHA1 truncado o `strings.Join` con
separadores textuales. Eso es aceptable para diagnostico/advisory, pero puede
ser ambiguo o colisionable si alimenta identidad causal, idempotencia,
recuperacion o evidencia durable.
Decision pendiente: builder canonico no ambiguo para refs causales, margen de
digest suficiente, distincion explicita entre huella visual y ref causal, y
conflicto reparable cuando una ref existente corresponde a otra fuente.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-change-director-source ./modulos/orquesta-mcp ./modulos/orquesta-orchestration-core ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack`.
Backlog: `T184 deterministic-ref-hash-collision-proof`.
```

```text
ID: RAIL-CAND-SMOKE-TEMP-ROOT-DELETION-001
Origen: scanner backlog 2026-05-24 sexagesima quinta pasada.
Casos: smokes y runners aceptan raices por `ORQUESTA_SMOKE_ROOT` o
`ORQUESTA_PARALLEL_TEST_TMP` y luego ejecutan `rm -rf` sobre esa raiz durante
cleanup. Si una variable apunta al proyecto, HOME, `.orquesta-runtime` o una
ruta compartida, el borrado puede ser mucho mas amplio que el smoke.
Decision pendiente: helper comun de cleanup con prefijo permitido,
marcador/manifest de creacion, bloqueo de rutas prohibidas y conservacion de
raices no verificadas.
Test futuro:
`bash -n scripts/*.sh scripts/lib/*.sh` y prueba focal del helper con raiz
valida, raiz sin marcador y ruta prohibida.
Backlog: `T185 smoke-script-temp-root-deletion-guard`.
```

```text
ID: RAIL-CAND-DOMAIN-WORK-JOB-FINGERPRINT-001
Origen: scanner backlog 2026-05-24 sexagesima sexta pasada.
Casos: `orquesta-domain-work-memory`, `orquesta-domain-work-file` y
`orquesta-domain-work-sql` derivan fingerprints/job refs con FNV64 base36 desde
payload canonico local. Es suficiente como huella corta de referencia, pero
queda cerca de idempotencia, replay y conflicto durable entre adaptadores.
Decision pendiente: builder canonico compartido o modo legacy documentado,
digest con margen suficiente para refs nuevas y conflicto reparable si dos
requests distintas colisionan.
Test futuro:
`go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-domain-work-memory ./modulos/orquesta-domain-work-file ./modulos/orquesta-domain-work-sql`.
Backlog: `T186 domain-work-job-ref-fingerprint-collision-proof`.
```

```text
ID: RAIL-CAND-HTTP-AUDIT-CLIENT-IDENTITY-001
Origen: scanner backlog 2026-05-24 sexagesima sexta pasada.
Casos: `auditHTTPHandlerV0` persiste `remote_addr` completo en eventos
`http_request`. T171 cubre query/form y T132 privacidad general, pero falta
politica concreta para IP:puerto, loopback, redes privadas y headers
`X-Forwarded-*`.
Decision pendiente: guardar categoria/hash/redaccion, aceptar headers de proxy
solo con perfil confiable y no convertir identidad declarada por cliente en
autorizacion o evidencia durable.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-observability ./cmd/orquesta-server`.
Backlog: `T187 http-audit-client-identity-redaction-policy`.
```

```text
ID: RAIL-CAND-HTTP-METHOD-CONTRACT-001
Origen: scanner backlog 2026-05-24 sexagesima sexta pasada.
Casos: handlers HTTP/MCP/web/gobernanza devuelven 405 con patrones locales; en
algunas rutas se espera `Allow` y en otras no hay contrato comun para `OPTIONS`.
Decision pendiente: contrato por perfil para metodo no permitido, header
`Allow`, `OPTIONS` sin efectos y shape de error publico sin payload crudo.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-web ./modulos/orquesta-governance ./modulos/orquesta-factory-http ./cmd/orquesta-server`.
Backlog: `T188 http-method-allow-options-contract`.
```

```text
ID: RAIL-CAND-SERVER-SIGNAL-SHUTDOWN-001
Origen: scanner backlog 2026-05-24 sexagesima septima pasada.
Casos: `cmd/orquesta-server run` crea contexto con `signal.NotifyContext` solo
para `os.Interrupt`; `orquesta-server stop` tambien senala el PID con
`os.Interrupt`. En modo daemon o service manager puede llegar SIGTERM, segunda
senal o timeout de cierre sin contrato publico ni estado observable propio.
Decision pendiente: politica por plataforma para interrupcion/terminacion,
deadline de gracia, segunda senal/escalado y status/auditoria compacta de
`stopping_by_signal`/`stop_timeout` sin PID crudo, HOME, rutas, env ni logs.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-server-shutdown`.
Backlog: `T189 server-resident-signal-shutdown-policy`.
```

```text
ID: RAIL-CAND-HTTP-GATEWAY-ROUTE-MANIFEST-001
Origen: scanner backlog 2026-05-24 sexagesima septima pasada.
Casos: `orquesta-http-gateway` registra rutas exactas y prefijo
`/api/v0/apps/` en `ServeMux`; `orquesta-app-gateway` monta AppVCS con otro mux
en `/api/v0/apps/vcs`. La precedencia depende del mux y puede cambiar al sumar
rutas bajo prefijos existentes sin rail de colision/shadowing.
Decision pendiente: manifiesto canonico de rutas/prefijos con owner, test de
colisiones exactas, prefijos ambiguos y dispatch de overlays; evidencia solo con
refs compactas de ruta, sin query, cookies, headers, payloads ni rutas privadas.
Test futuro:
`go test -count=1 ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-mcp ./cmd/orquesta-server`.
Backlog: `T190 http-gateway-route-manifest-collision-guard`.
```

```text
ID: RAIL-CAND-PROCESS-RUNTIME-STOP-ESCALATION-001
Origen: scanner backlog 2026-05-24 sexagesima septima pasada.
Casos: `ProcessRuntimeConnectorV0.signalProcessStopV0` envia `os.Interrupt` y
solo ejecuta `Kill` si `Signal` falla. Un proceso que recibe la senal pero la
ignora queda sin deadline de gracia, escalado, estado `stopping` ni reason code.
Decision pendiente: parada cooperativa con timeout, escalation/kill
observable, codigos para senal no soportada/proceso detenido/timeout/kill
fallido y tests con proceso que sale, ignora senal y ya estaba cerrado.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-runtime-required-test ./modulos/orquesta-orchestration-core ./cmd/orquesta-server`.
Backlog: `T191 process-runtime-stop-signal-escalation-policy`.
```

```text
ID: RAIL-CAND-SERVER-STATUS-MESSAGE-001
Origen: scanner backlog 2026-05-24 sexagesima octava pasada.
Casos: `status_tracker_v0.go` construye razones de automejora idle a partir de
mensajes, acciones y evidencias de adaptadores; `supervisor_loop_v0.go` emite
eventos de auditoria con requests/results/plans seleccionados.
Decision pendiente: proyeccion de mensajes operativos con codigo de razon, refs
opacas, limites de bytes y redaccion antes de persistir/exponer estado.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server ./modulos/orquesta-observability`.
Backlog: `T192 server-status-operational-message-projection`.
```

```text
ID: RAIL-CAND-FACTORY-APPSPEC-TIME-001
Origen: scanner backlog 2026-05-24 sexagesima octava pasada.
Casos: `appspec_usecase_v0.go` acepta `now`, pero si llega cero usa
`time.Now().UTC()`; `appspec_http_v0.go` ya inyecta reloj desde adaptador.
Decision pendiente: reloj obligatorio por composicion o fallback convertido en
politica explicita/versionada con pruebas de determinismo.
Test futuro:
`go test -count=1 ./modulos/orquesta-factory ./modulos/orquesta-factory-http ./modulos/orquesta-web ./modulos/orquesta-mcp`.
Backlog: `T193 factory-appspec-time-source-contract`.
```

```text
ID: RAIL-CAND-GOVERNANCE-CATALOG-BUDGET-001
Origen: scanner backlog 2026-05-24 sexagesima octava pasada.
Casos: `governance_catalog_public_query_v0.go` proyecta entradas efectivas del
catalogo para consultas publicas sin owner visible para presupuesto de salida,
frescura y source refs.
Decision pendiente: proyeccion publica bounded con limites, version/freshness,
source refs y conteos para estados no publicos por defecto.
Test futuro:
`go test -count=1 ./modulos/orquesta-governance ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway`.
Backlog: `T194 governance-catalog-output-budget-freshness`.
```

```text
ID: RAIL-CAND-MCP-TOOL-SCHEMA-DESCRIPTOR-001
Origen: scanner backlog 2026-05-24 sexagesima novena pasada.
Casos: tools de `orquesta-mcp` declaran `InputSchema`/`Output` como strings
compactos escritos a mano; el transporte MCP real los reexpone y las
capabilities de operador declaran `OutputShape`/`InputRefs` en otra fuente.
Decision pendiente: descriptor verificable por tool contra DTO, validador,
handler y registro real; si falta puerto o composicion, publicar error publico
`not_configured`/`unavailable` sin prometer schema ejecutable.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T195 mcp-tool-input-schema-descriptor-sync`.
```

```text
ID: RAIL-CAND-CLI-COMMAND-CATALOG-001
Origen: scanner backlog 2026-05-24 sexagesima novena pasada.
Casos: `modulos/orquesta-cli/command_runner_v0.go` mantiene texto de ayuda
ES/EN, dispatch por `hasCLIPathV0` y `FlagSet` por comando como fuentes
manuales separadas. Un comando nuevo puede quedar ejecutable pero no anunciado,
o anunciado con flags desactualizadas.
Decision pendiente: catalogo canonico de comandos CLI con help localizada,
dispatch y flags verificables; errores de comando desconocido no deben volcar
argumentos completos, URLs con credenciales, rutas privadas ni payloads inline.
Test futuro:
`go test -count=1 ./modulos/orquesta-cli ./cmd/orquesta-server`.
Backlog: `T196 cli-command-catalog-help-dispatch-sync`.
```

```text
ID: RAIL-CAND-CLI-RESPONSE-BOUNDS-001
Origen: scanner backlog 2026-05-24 septuagesima pasada.
Casos: clientes REST de `orquesta-cli` para FunctionContract,
OperationalStatus, gobernanza, status/run control y comandos afines leen bodies
con `io.ReadAll` o decoders directos antes de producir envelopes publicos.
Decision pendiente: extender la politica de respuestas HTTP salientes a CLI con
limite por comando, content-type/trailing JSON, redaccion de no-2xx y reason
codes compactos; no devolver body crudo ni truncar JSON silenciosamente.
Test futuro:
`go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-web ./modulos/orquesta-mcp ./cmd/orquesta-server`.
Backlog: `T197 cli-rest-response-bounds-redaction-parity`.
```

```text
ID: RAIL-CAND-MCP-RESOURCE-DESCRIPTOR-001
Origen: scanner backlog 2026-05-24 septuagesima pasada.
Casos: resources MCP como operational-status, shared/core contracts, roadmap,
governance y operator capabilities publican shapes, public errors, refs
canonicas y guardrails como strings estaticos separados de DTOs/validadores.
Decision pendiente: verificar resources contra fuente canonica, version,
freshness, presupuesto de salida y owner real; si falta fuente/puerto, publicar
`descriptor_stale`, `not_configured` o `unavailable` sin inventar schema.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-observability ./modulos/orquesta-governance ./modulos/orquesta-core ./cmd/orquesta-server`.
Backlog: `T198 mcp-resource-descriptor-source-sync`.
```

```text
ID: RAIL-CAND-MCP-PUBLIC-ERROR-CATALOG-001
Origen: scanner backlog 2026-05-24 septuagesima pasada.
Casos: helpers MCP/HTTP y operador usan strings locales de error publico
(`metodo_no_permitido`, `request_body_invalido`, `*_no_configurado`,
`*_error`) con mappings distintos entre HTTP, JSON-RPC, CLI/web y resources.
Decision pendiente: catalogo comun de error codes publicos con i18n key,
retryability, severidad y mapping por frontera; `err.Error()` no allowlisted se
reduce a codigo generico sin payload crudo.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-web ./modulos/orquesta-cli ./modulos/orquesta-i18n-docs ./cmd/orquesta-server`.
Backlog: `T199 mcp-public-error-code-catalog`.
```

```text
ID: RAIL-CAND-CMD-SERVER-REST-CLIENT-001
Origen: scanner backlog 2026-05-24 septuagesima primera pasada.
Casos: comandos locales de `cmd/orquesta-server` (`status`, `run-status`,
`stop`) usan clientes HTTP propios. `getStatusBodyV0` y `postRunStatusBodyV0`
leen respuestas con `io.ReadAll`; `run-status` incorpora body no-2xx completo
en stderr; `requestServerShutdownV0` decodifica JSON sin limite ni trailing-data
check.
Decision pendiente: helper o prueba de paridad para limite de respuesta,
content-type, trailing JSON y redaccion de errores del binario servidor, sin
volcar HTML, URLs con credenciales, rutas privadas, HOME, tokens, prompts,
transcripts ni payloads de dominio.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-observability`.
Backlog: `T200 cmd-server-management-rest-client-policy`.
```

```text
ID: RAIL-CAND-OUTBOUND-RETRY-BACKOFF-001
Origen: scanner backlog 2026-05-24 septuagesima segunda pasada.
Casos: `runExternalBridgeLoopV0` reintenta ticks con intervalo fijo y los
conectores `domain_work-http`/OPES devuelven errores compactos sin politica
comun de `Retry-After`, jitter, presupuesto de reintentos, rate limit ni circuit
breaker por destino/ref.
Decision pendiente: definir retry/backoff/rate por adaptador de composicion,
con reason codes publicos y sin reintentar mutaciones no idempotentes cuando
falte ledger o `idempotency_key` causal.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./modulos/orquesta-opes-bridge ./modulos/orquesta-server`.
Backlog: `T201 outbound-connector-retry-backoff-rate-policy`.
```

```text
ID: RAIL-CAND-OUTBOUND-CORRELATION-HEADERS-001
Origen: scanner backlog 2026-05-24 septuagesima segunda pasada.
Casos: `modulos/orquesta-domain-work-http/client_v0.go` fija solo
`Content-Type` y `modulos/orquesta-opes-connector/http_v0.go` fija
`Accept`/`Content-Type`; `correlation_id` e `idempotency_key` viajan en JSON
pero no como cabeceras compactas para proxies, apps externas o ledgers HTTP.
Decision pendiente: propagar `X-Correlation-ID`, `Idempotency-Key` y/o ref
equivalente desde campos ya validados; bloquear valores no compactos y no poner
prompts, transcripts, rutas privadas, HOME, tokens ni payloads de dominio en
headers.
Test futuro:
`go test -count=1 ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack`.
Backlog: `T202 outbound-domain-correlation-idempotency-headers`.
```

```text
ID: RAIL-CAND-OPES-PAGINATION-WINDOW-001
Origen: scanner backlog 2026-05-24 septuagesima segunda pasada.
Casos: `ListExternalJobsV0` lee una sola respuesta de OPES con `limit` y
`opesBridgeScanLimitV0` capado a 100. Si los primeros jobs ya estan en ledger,
OPES no garantiza orden/cursor o la respuesta ignora limit, el bridge puede
revisar siempre la misma ventana y no llegar a jobs pendientes posteriores.
Decision pendiente: contrato de cursor/orden/ventana para OPES o bloqueo
publico `pagination_not_supported`/`window_exhausted`; la secuencia por tipos no
debe tratar `Seen > 0` como progreso si no hubo envio ni avance real.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-opes-connector ./modulos/orquesta-opes-bridge`.
Backlog: `T203 opes-bridge-pagination-window-policy`.
```
