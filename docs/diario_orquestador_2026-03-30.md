# Diario del orquestador — 2026-03-30

## 2026-03-31 03:xx aprox. — el bootstrap pasa a poseer de verdad su mailbox y deja de duplicarse por `session_resume`

Hallazgo:

- el caso vivo de `Codex2` ya no era un problema generico de `send_instruction`
- el `start #80154` habia arrancado con `bootstrap.mailbox_count=1`, es decir, el mensaje `64424` ya estaba dentro de la continuidad pendiente
- aun asi, el batch `procesarRuntimeMailboxSessionResumeBatch()` materializo una segunda `send_instruction` (`#80157`) para ese mismo `mailbox_id`
- eso demostraba una fuga de contrato: la `bootstrap lease` y el batch de mailbox competian por la misma verdad

Decision:

- si un `mailbox_id` ya esta cubierto por una `bootstrap lease` pendiente, ese mensaje pertenece al bootstrap hasta su `ack`
- ningun batch puede crear otra `send_instruction` para ese mismo `mailbox_id`
- si aun existe una `send_instruction` vieja para ese mailbox, debe cerrarse como `superseded` por `covered_by_bootstrap_lease`
- ademas, una `send_instruction` creada desde mailbox deja de ser cola durable propia: si no hay runtime entregable inmediato, la orden se completa y la verdad queda solo en el mailbox

Cambios:

- `db/controlplane_entities.go`
  - nuevo helper exportado `RuntimeMailboxCubiertoPorBootstrapPendiente(...)`
  - nuevo cierre `completarRuntimeOrderSendInstructionCubiertaPorBootstrap(...)`
  - nuevo cierre `completarRuntimeOrderSendInstructionDiferidaAMailbox(...)`
  - `ejecutarRuntimeOrderSendInstruction(...)` deja de reencolar cuando la verdad ya esta cubierta por bootstrap o cuando el mailbox ya es la fuente durable suficiente
- `cmd/controlplane_support.go`
  - `procesarRuntimeMailboxSessionResumeBatch()` salta el mensaje si ya esta cubierto por la lease de bootstrap
- tests nuevos:
- `db/controlplane_entities_test.go`
  - `TestRuntimeOrderSendInstructionMailboxCubiertaPorBootstrapSeCompletaSinReintento`
  - `TestRuntimeOrderSendInstructionMailboxSeCompletaDejandoLaVerdadEnMailboxSiNoHayHandleEntregable`
  - `cmd/controlplane_support_test.go`
    - `TestProcesarRuntimeMailboxSessionResumeBatchRespetaLeaseBootstrapPendiente`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(RuntimeOrderSendInstructionMailboxCubiertaPorBootstrapSeCompletaSinReintento|RuntimeOrderSendInstructionMailboxSupersedeSesionObsoleta|GuardarSesionActivaPreservaMetadataRicaDelHandle|RuntimeOrderSendInstructionCodexSupervisadoSigueCayendoAMailbox)' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(ProcesarRuntimeMailboxSessionResumeBatchEncolaSendInstruction|ProcesarRuntimeMailboxSessionResumeBatchEncolaSendInstructionParaWatchdog|ProcesarRuntimeMailboxSessionResumeBatchRespetaLeaseBootstrapPendiente|ProcesarRuntimeMailboxSessionResumeBatchCoalesceNudgeAunqueHayaOrdenAbierta)' -count=1` => OK

## 2026-03-31 00:xx aprox. — decision de no reescritura sobre framework externo y cierre de doctrina del nucleo

Objetivo:

- cortar el riesgo de seguir reprogramando sintomas sin una direccion fija
- fijar por escrito si Orquesta se reescribe o no sobre LangChain/LangGraph/CrewAI/AutoGen
- dejar una doctrina estable del nucleo de orquestacion para las siguientes pasadas

Conclusion tomada:

- no se reescribe Orquesta sobre LangChain, LangGraph, CrewAI ni otro framework externo
- si se copiaran patrones, se copiaran como arquitectura y semantica, no como sustitucion del producto

Motivo:

- el problema real no es de prompts ni de grafo conversacional
- el problema real es de control operativo duradero:
  - daemon unico
  - cola durable
  - lease y ack
  - handoff con continuidad
  - supervisor de runtime
  - reconciliacion tras reinicio o caida
- cambiar de framework ahora destruiria continuidad y abriria otra transicion a medias

Referencias estudiadas:

- AutoGen Core como patron de supervisor/manager y mensajes entre agentes
- OpenAI Agents para handoff filtrado y resumen de continuidad
- Kubernetes controller/operator como referencia de reconciliacion declarativa
- Celery/Temporal como referencia de cola durable con retry/backoff/ack

Decision doctrinal fijada en la biblia:

- `orquesta server` sigue siendo el unico plano de control
- `runtime_orders` debe evolucionar a cola formal con `pending/leased/running/completed/failed/...`
- `runtime_mailbox` debe tipificarse con claridad y no mezclar señales internas con mensajes entregables al agente
- si un `kind` de mailbox puede quedar pendiente, debe tener una ruta oficial de entrega o consumo

Hallazgo operativo ya visible antes de tocar codigo:

- `Codex1` ya no tenia la orden `#79976` colgada en `ejecutando`; el daemon la reencolo en `pendiente`
- aun asi seguian quedando `8` mensajes pendientes en mailbox para `Codex1` sin runtime vivo ni tarea activa
- esto confirma que el siguiente frente correcto ya no es el arranque del daemon sino la semantica y reconciliacion de mailbox

## 2026-03-31 01:xx aprox. — reconciliacion de watchdog pendiente sin ruta real de entrega

Objetivo:

- corregir una incoherencia concreta del control plane sin abrir otro refactor ciego
- evitar que `watchdog` quede pendiente indefinidamente en `runtime_mailbox` cuando ya no existe handle activo para el agente

Hallazgo:

- `watchdog` si tiene sentido como señal persistente cuando existe un runtime activo al que sondar
- pero tambien puede quedar pendiente mucho despues de que el runtime/handle haya desaparecido
- en ese estado deja de ser una señal entregable y pasa a ser ruido operativo

Cambio aplicado:

- `procesarRuntimeMailboxBatch()` ahora empieza por reconciliar `watchdog` pendientes sin handle activo
- esos mensajes se marcan como `entregado` + `consumido`
- se deja auditoria explicita `runtime_mailbox_watchdog_sin_handle`

Criterio doctrinal que queda reforzado:

- un mensaje pendiente solo puede seguir vivo si existe una ruta oficial de entrega o consumo
- si `watchdog` ya no tiene runtime/handle activo al que aplicarse, debe consumirse como señal obsoleta
- esto no elimina `watchdog` como primitive; solo evita su acumulacion como mentira operativa

## 2026-03-31 02:xx aprox. — unificacion oficial de `orquestador` -> `orquesta`

Objetivo:

- cortar otra fuente de confusion recurrente entre nombre de app, slug de proyecto y ruta fisica del repo
- evitar que agentes futuros sigan mezclando `orquesta` y `orquestador` como si fueran dos proyectos vivos distintos

Hallazgo:

- el estado vivo seguia usando `orquestador` como slug del proyecto activo aunque la ruta real ya era `/home/alberto/Trabajo/orquesta`
- no existian dos proyectos raiz activos distintos para la misma ruta; era el mismo registro persistido con slug legacy

Accion ejecutada por la via oficial:

- `./orquesta proyecto descubrir /home/alberto/Trabajo/orquesta`

Resultado:

- el proyecto activo `id=1` queda normalizado como `slug=orquesta`
- las asignaciones activas de `Codex1-6` pasan a referenciar `orquesta`
- las tareas activas/asignadas siguen accesibles bajo `--proyecto orquesta`
- `orquestador` deja de existir como proyecto vivo en la lista

Lectura doctrinal:

- para esta base actual el proyecto canonico vivo es `orquesta`
- `orquestador` queda ya como nombre legacy en documentos antiguos o en historico, pero no como slug operativo
- cuando la via oficial puede normalizar sin fusion destructiva, se prefiere eso antes que borrar o manipular entidades a mano

## 23:55 aprox. — verificación oficial de persistencia y limpieza de mantenimiento

Objetivo:

- cerrar otro resto importante de deriva SQLite-céntrica en el camino oficial de operaciones
- dejar la verificación de persistencia dentro de Orquesta, backend-aware, en vez de empujar al operador hacia scripts o hábitos manuales ambiguos

Hallazgos:

- ya existía `Backup` por backend en `db/backend.go`, pero faltaba la verificación oficial equivalente
- el camino de mantenimiento seguía quedando partido:
  - `persistencia info` explicaba el backend y el daemon
  - `scripts/verificar_bd.sh` seguía siendo la única verificación explícita, y además solo para SQLite
- `docs/manual_tecnico_sistemas.md` todavía arrastraba lenguaje de “SQLite como arquitectura” y una recomendación operativa mala (`pkill -9 orquesta`)
- además `cmd/respaldo.go` conservaba un helper `literalSQLite` ya muerto, que solo añadía ruido conceptual

Cambios aplicados:

- se añade el contrato `Verify` al backend de persistencia
- se incorpora `db.VerificarPersistenciaActual()` con informe estructurado:
  - `driver`
  - `target`
  - `sano`
  - lista de comprobaciones
- SQLite verifica:
  - conexión
  - `PRAGMA quick_check`
  - presencia de tablas core
- MySQL/Postgres verifican:
  - conexión
  - presencia de tablas core
- se expone la ruta oficial `GET /api/persistencia/verificar`
- se añade `orquesta persistencia verificar`
- la biblia deja fijado que `persistencia info` y `persistencia verificar` son la vía oficial de observabilidad/verificación, y que los scripts backend-específicos quedan para rescate
- `manual_tecnico_sistemas.md` deja de vender SQLite como si fuera el diseño entero y sustituye la receta de `pkill -9` por `server doctor` + `persistencia info` + `persistencia verificar`
- se elimina el helper muerto `literalSQLite` de `cmd/respaldo.go`

Lectura doctrinal consolidada:

- el camino oficial ya no es “abre un script y mira SQLite”
- el camino oficial pasa por Orquesta y sus adaptadores
- los scripts que queden deben decir con claridad si son rescate y para qué backend sirven

## 00:10 aprox. — el backup automatizado deja de fingir que es genérico

Objetivo:

- eliminar otra fuente de despiste para agentes futuros: una unit systemd de backup que por nombre parecía general, pero en realidad solo valía para SQLite

Cambios aplicados:

- `scripts/orquesta-backup.service` se sustituye por `scripts/orquesta-backup-sqlite.service`
- `scripts/orquesta-backup.timer` se sustituye por `scripts/orquesta-backup-sqlite.timer`
- `scripts/instalar_servicio.sh` pasa a instalar y anunciar esos nombres explícitos
- `scripts/backup_bd.sh` y `scripts/verificar_bd.sh` dejan más claro que son utilidades de rescate/automatización solo-SQLite
- `manual_tecnico_sistemas.md` añade `orquesta respaldo bd` como snapshot manual por el camino oficial

Lectura doctrinal:

- no es aceptable dejar nombres neutros a piezas que no lo son
- el camino oficial multi-backend pasa por Orquesta
- las piezas SQLite-only deben identificarse como SQLite-only en nombre y documentación

## 00:20 aprox. — limpieza de doctrina residual SQLite-first

Hallazgos:

- `docs/op_088_orquesta_mcp_server.md` seguía mostrando la integración MCP contra `SQLite: orquesta.db`
- `docs/politica_backups_es.md` y `docs/politica_backups_en.md` parecían políticas generales, pero en realidad describían solo restauración/copia de ficheros SQLite
- el índice aún anunciaba esa política como si fuera “de la BD” en general

Cambios aplicados:

- el diagrama de OP-088 pasa a hablar de `Storage Adapter: backend activo`
- las políticas de backup se reetiquetan como políticas del backend SQLite
- ambas políticas remiten ya a `orquesta respaldo bd` y `orquesta persistencia verificar` como camino oficial
- el índice refleja explícitamente que esa política es para SQLite

Lectura doctrinal:

- no basta con tener la biblia correcta si otras piezas siguen sugiriendo una arquitectura vieja
- toda referencia residual que empuje a leer “Orquesta = SQLite file” debe corregirse o desaparecer

## 00:30 aprox. — corrección de OPs que todavía filtraban arquitectura vieja

Hallazgos:

- `docs/op_088_orquesta_servidor_mcp.md` seguía describiendo prompts MCP como si leyesen `reglas/skills/workflows` “de SQLite”
- `docs/op_089_relevo_agentes_por_token.md` seguía formulando el checkpoint en términos de “escribir en SQLite”
- `docs/op_049_matriz_voto.md` mantenía el riesgo como “que SQLite no quede expuesto a demasiada concurrencia”, cuando la regla correcta ya es proteger el single-writer y el backend activo

Cambios aplicados:

- OP-088 larga: briefing y auditoría reformulados contra servicios/adaptador de persistencia, no contra SQLite
- OP-089: checkpoint/handoff reformulados contra `runtime_checkpoints` vía adaptador de persistencia activo
- OP-049: riesgo reformulado en términos de concurrencia lateral y ruptura del single-writer

Lectura doctrinal:

- las OP también son arquitectura viva
- si una OP vieja describe bien la intención pero mal el mecanismo, hay que corregirla para que no vuelva a arrastrar implementación equivocada

## 16:00 aprox. — foco actual

Objetivo en curso:

- verificar desde la propia app de Orquesta, en modo `server-first`, qué control real existe hoy sobre `Codex1-6`
- separar lo que está codificado de verdad de lo que sigue siendo wrapper, compatibilidad o rescate
- definir el flujo operativo correcto sin recurrir a `sqlite`

Hallazgos ya confirmados:

- `Orquesta server` es la vía operativa válida; `CLI`, `web` y `API` son clientes del servicio
- `agente preparar` y `agente tick` ya van por API y sirven para bundle/heartbeat operativo
- el control activo de ciclo de vida existe como `orquesta agente control <arrancar|pausar|continuar|detener> <agente>`
- la observabilidad real del control plane vive en `runtime listar`, `runtime handles`, `runtime ordenes`, `runtime mailbox`, `runtime checkpoints` y `runtime diagnostico`
- los scripts `inicio_agente.sh` y `agente_console.sh` siguen presentes, pero el propio repo los marca como compatibilidad/rescate, no como camino oficial

Trabajo en curso:

- contrastar esos comandos con la instancia viva para identificar señales sanas y señales de degradación por agente
- convertir esta lectura en un runbook operativo concreto para validar control real sobre `Codex1-6` sin tocar BD

Riesgo principal que se está atacando:

- evitar seguir “programando en bucle” sobre síntomas y centrar la validación en contratos observables del control plane

## 18:00 aprox. — evidencia viva contra Orquesta

Trazabilidad enviada a agentes:

- nudges encolados a `Codex1-6` vía `orquesta runtime nudge`
- órdenes creadas: `#79844` a `#79849`

Comandos verificados con salida útil:

- `orquesta server status`
- `orquesta status`
- `orquesta sesion listar`
- `orquesta runtime listar --proyecto orquestador`
- `orquesta runtime handles --agente Codex1`
- `orquesta runtime ordenes --agente Codex1 --proyecto orquestador`
- `orquesta runtime mailbox --to Codex1 --proyecto orquestador`
- `orquesta runtime diagnostico --agente Codex1 --proyecto orquestador --limit 5`
- `orquesta agente tick Codex1..Codex6 --proyecto orquestador`

Señales confirmadas:

- `status` ve `Codex1-6` como agentes activos en la plataforma
- `sesion listar` marca activas las sesiones de `Codex1-5`; `Codex6` queda sin sesión activa
- `tick` devuelve acción operativa útil:
  - `Codex1`, `Codex3`, `Codex4`, `Codex5`: `continuar_trabajo`
  - `Codex2`: `votar_propuestas_pendientes`
  - `Codex6`: `esperar_o_pedir_tarea`
- el nudge nuevo a `Codex1` aparece en `runtime ordenes` como `completada`
- en `runtime mailbox` de `Codex1` siguen quedando mensajes pendientes (`nudge`, `governance_refresh`, `watchdog`, `autonomia`)
- `runtime diagnostico` muestra handle activo actual para `Codex1` y también un volumen excesivo de runtimes/handles históricos cerrados

Desajuste importante observado:

- `status`, `sesion` y `runtime*` responden por la ruta server-first
- `tarea listar` y `propuesta listar` no han quedado consistentes todavía en esta instancia: unas veces exigen servidor aunque hay daemon, y con `ORQUESTA_SERVER_URL=http://127.0.0.1:16543` han devuelto `connection refused`

Conclusión provisional:

- sí existe control plane real observable desde Orquesta sin `sqlite`
- todavía hay incoherencias en la superficie cliente para algunas rutas (`tarea` y `propuesta`) y sigue habiendo backlog pendiente en mailbox/runtimes históricos

## 20:15-20:35 aprox. — arranque real de flota por vía de rescate

Objetivo:

- comprobar personalmente si Orquesta puede arrancar de verdad `Codex1-6` y ponerlos a trabajar sin bajar a `sqlite`

Acciones ejecutadas:

- se levantó el daemon local con `./orquesta server run --addr 127.0.0.1:16543`
- se confirmó `server status` sano en host: daemon activo y `healthz` visible
- se releyó el estado real de proyecto y tareas desde Orquesta
- se preparó un plan manual de lanzamiento de flota
- se ejecutó `scripts/preparar_lanzamiento_agentes.sh --ejecutar /tmp/orquesta-flota-20260330.plan`
- se crearon worktrees dedicados:
  - `orquesta-codex1`
  - `orquesta-codex2`
  - `orquesta-codex3`
  - `orquesta-codex4`
  - `orquesta-codex5`
  - `orquesta-codex6`
- el plan generó además nuevas tareas asignadas en Orquesta:
  - `#421` Codex1
  - `#422` Codex2
  - `#423` Codex3
  - `#424` Codex4
  - `#425` Codex5
  - `#426` Codex6
- se lanzaron seis consolas reales vía `scripts/agente_console.sh` y quedaron procesos `node .../bin/codex` vivos en host, uno por agente
- se inyectaron instrucciones manuales en cada PTY para orientar el trabajo sobre sus frentes reales

Evidencia confirmada:

- el proyecto lógico activo es `orquesta`; `orquestador` ya no existe como proyecto vivo en esta base
- `./orquesta proyecto listar` muestra:
  - `orquesta`
  - `orquesta-archivado-20260329-130458`
- `./orquesta tarea listar --proyecto orquesta --estado asignada --tsv` devuelve las tareas `410`, `411`, `414`, `415`, `417`, `418`, `419`, `420` y las nuevas `421-426`
- los wrappers `agente_console.sh Codex1..6` siguen vivos, con sus procesos `node .../bin/codex` hijos activos

Bloqueo estructural detectado:

- `scripts/agente_console.sh` llama a `orquesta sesion inicio` y `orquesta agente preparar`
- `sesion inicio` sí puede abrir sesión y devuelve el briefing largo del agente
- `agente preparar` falla por API con timeout aun teniendo el daemon activo:
  - `Get "http://127.0.0.1:16543/api/agente/preparar?...": net/http: timeout awaiting response headers`
- como `agente preparar` falla, el launcher cae al `BOOTSTRAP_PROMPT_DEFAULT`
- ese prompt por defecto deja al runtime en modo:
  - "si no hay mas instrucciones del usuario, mantente en espera"

Conclusión operativa:

- sí he arrancado personalmente la flota y he dejado seis agentes vivos en host
- no está cerrado todavía el arranque autónomo correcto porque el launcher de rescate depende de `agente preparar`, y hoy ese endpoint no responde a tiempo
- por eso la flota puede quedar abierta pero mal bootstrappeada si no se le inyecta trabajo manual después

## 21:25-21:35 aprox. — cierre del cuello real en `prepare` y `status`

Objetivo:

- dejar de trabajar a ciegas sobre síntomas y cerrar el cuello del núcleo medido en el daemon real

Diagnóstico confirmado:

- `agente preparar` no estaba roto de forma genérica; el problema era latencia excesiva
- con instrumentación en servidor se midió primero `BuildPrepare` en ~31-34s
- el tramo dominante era `BuildLaunchBootstrapPrompt -> EsSupervisorAutonomiaOperativo`, con ~23s
- además `/api/status` hacía doble trabajo:
  - `statusService.FetchStatus()`
  - `buildEstadoResumen()`
- la CLI `status` usaba además un cliente HTTP distinto con timeout de `3s`, inferior al coste real del endpoint

Cambios aplicados:

- fast-path en selección de supervisor operativo para reutilizar `SupervisorAgente` configurado cuando sigue siendo válido
- `governance_overrides` fuera del camino de lectura lenta, pasando a bootstrap/migración
- `agente preparar` ya no consume bootstrap al inspeccionar
- `/api/status` deja de recomputar el resumen completo por segunda vez
- los votos de propuestas abiertas se cargan ahora en batch con una sola consulta (`ResumenVotosPorPropuestas`)
- `status` reutiliza el cliente HTTP normal de servidor en vez de un cliente separado de `3s`

Verificación real en host, contra daemon actualizado:

- `ORQUESTA_SERVER_URL=http://127.0.0.1:16543 ./orquesta agente preparar Codex1 --proyecto orquestador --conector codex-cli --campo mode`
  - responde `resume`
- `curl /api/agente/preparar?...`
  - `HTTP 200`
  - `time_total=8.683676`
- `./orquesta status`
  - vuelve a responder completo
- `curl /api/status`
  - `HTTP 200`
  - `time_total=3.086607`

Lectura operativa:

- se ha salido del bucle real: el launcher ya no cae por timeout al pedir `prepare`
- `status` vuelve a ser usable como señal de salud del sistema
- el núcleo no está “acabado al 100%”, pero estos dos bloqueos centrales ya no están abiertos
- el siguiente trabajo ya no es adivinar si la app responde, sino validar y endurecer el gobierno vivo de runtimes/agents sobre esta base estable

Validación end-to-end del launcher de rescate:

- se ejecutó `scripts/agente_console.sh Codex1 orquestador /tmp/orquesta-validate-launch codex-cli /tmp/codex-perfil`
- `/tmp/codex-perfil` era un lanzador falso que solo capturaba los argumentos recibidos
- el wrapper completó `sesion inicio`, pidió `agente preparar`, lanzó el runtime y cerró sesión guardando continuidad
- el fichero capturado `/tmp/orquesta-fake-codex-launch.txt` ya no contiene el prompt genérico de “mantente en espera”
- contiene un bootstrap real de Orquesta con:
  - rol del agente
  - proyecto
  - directorio de trabajo
  - perfil/modelo/razonamiento
  - continuidad previa
  - rol operativo de orquestación autónoma
  - tareas activas y reglas efectivas

Conclusión adicional:

- el camino `scripts/agente_console.sh -> orquesta sesion inicio -> orquesta agente preparar -> runtime_launch`
  vuelve a estar operativo como flujo de compatibilidad/rescate

## 22:30-23:30 aprox. — supervisor local residente y cierre de deriva circular

Objetivo:

- alinear el gobierno de runtimes locales con `ARQUITECTURA.md`
- sacar la propiedad del proceso local de `db`/polling oportunista y moverla a `internal/controlruntime`
- dejar de depender de inspecciones laterales para entender por qué una `runtime_order` queda pendiente

Lectura arquitectónica aplicada:

- `ARQUITECTURA.md` insiste en que Orquesta debe depender de un contrato estable de runtime y centralizar el acceso concurrente
- eso invalida seguir tratando el runtime local como un PID sin dueño que luego se "redescubre" desde `db`
- la pieza correcta era un supervisor local residente en el adaptador `controlruntime`, no otro parche en SQL o en el runner

Cambios aplicados:

- se añadió un supervisor local residente en `internal/controlruntime/supervisor_local.go`
- `ArrancarPlan` local registra el proceso arrancado bajo supervisión residente y persiste `supervisor_ref`/`supervision_mode`
- `ProcesoVivo`, `PausarProceso`, `ContinuarProceso`, `DetenerProceso` y `EnviarInstruccionProceso` consultan primero ese supervisor
- `db/controlplane_entities.go` ya no observa el runtime local solo como PID; aplica estado local observado desde `controlruntime`
- el supervisor detecta y propaga `external_session_id` de runtimes Codex cuando puede, para reforzar la vía durable `session_resume`

Riesgo de programación circular detectado y corregido:

- en una primera iteración el supervisor estaba sobrescribiendo `driver=process_pty_cli` con `driver=local_runtime_supervisor`
- eso rompía conceptualmente la ruta `DetectExternalSessionID -> session_resume`, o sea, el cambio "arreglaba" una capa y degradaba otra
- también apareció otra deriva real: al reutilizar una entrada del supervisor por PID, podía quedarse con metadata vieja
- ambos puntos quedaron corregidos:
  - el supervisor ya publica `supervisor_driver` sin pisar el `driver` real del runtime
  - si reengancha por PID, refresca su metadata con el contexto nuevo antes de devolver estado

Validación hecha:

- `go test ./internal/controlruntime ./db ./cmd -run ...` pasa en la batería dirigida del supervisor, control plane y recuperación local
- se añadió test específico para garantizar que `ConsultarEstadoLocal` detecta `external_session_id` sin sobrescribir el `driver`
- el binario `./orquesta` recompila bien

Mejora operativa añadida dentro de Orquesta:

- `orquesta runtime ordenes` ahora muestra `DISPONIBLE` y `DETALLE`
- esto permite ver desde la propia app si una orden está diferida por reintento o simplemente pendiente, sin bajar a `sqlite`

Lectura real contra la base actual:

- `ORQUESTA_FORCE_LOCAL=1 ./orquesta runtime ordenes --agente Codex1 --estado pendiente`
  muestra una sola orden viva:
  - `#79959` `send_instruction`
- esa orden no aparece como diferida ni con `retry_after`; sale pendiente con `available_at = created_at`
- por tanto, el siguiente foco real ya no es el algoritmo de reencolado, sino el consumidor del control plane o el acceso al daemon en este entorno

Limitación del entorno actual:

- desde esta sandbox no se puede abrir socket ni consultar `127.0.0.1:16543` (`socket: operation not permitted`)
- `server doctor` y `server status` no sirven aquí para concluir si el daemon real del host está procesando o no la cola
- el log local visible desde la sandbox mezcla arranques fallidos del propio entorno con `SQLITE_BUSY` y `listen ... operation not permitted`, así que no es fuente fiable para juzgar el daemon del host

Conclusión operativa:

- el núcleo del runtime local ha avanzado de verdad: ya existe supervisor residente y contrato observable para procesos locales
- el siguiente bloqueo serio no parece estar en `start/stop/status`, sino en la ejecución efectiva de `send_instruction` pendiente sobre la cola viva
- gracias a la salida nueva de `runtime ordenes`, ese frente ya se puede seguir desde Orquesta sin recurrir a inspección directa de BD

## 21:45-22:15 aprox. — corrección estructural de `send_instruction` y mailbox durable

Objetivo:

- revisar `ARQUITECTURA.md` antes de seguir tocando el núcleo
- evitar programación cíclica sobre síntomas de runtime
- fijar el contrato correcto de entrega para `send_instruction`

Contraste arquitectónico realizado:

- `ARQUITECTURA.md` confirma tres restricciones relevantes:
  - el runtime debe quedar detrás de un contrato estable de conector
  - `orquesta serve` debe ser el escritor principal
  - el acceso concurrente debe centralizarse, no dispersarse en atajos por SQLite o por handle puntual

Hallazgo importante al mirar la instancia viva desde Orquesta:

- el problema ya no era simplemente "falta un supervisor"
- `send_instruction` había funcionado muchas veces en el día, pero volvió a fallar en caliente con órdenes recientes
- además, el flujo actual trataba cada `send_instruction` como si perteneciera a un `handle` concreto
- cuando ese handle quedaba viejo o el runtime aún no estaba listo, la orden podía:
  - fallar de forma terminal
  - o incluso volver a crear mailbox duplicada
- la causa viva observada en `Codex1` fue concreta:
  - `session_resume fallo: chdir /tmp/orquesta-validate-launch: no such file or directory`
  - es decir, la entrega seguía intentando ejecutar sobre un `working_dir` histórico de validación, no sobre la worktree/runtime activos

Conclusión de diseño:

- eso sí era programación cíclica potencial:
  - seguir añadiendo supervisión sin corregir la semántica de entrega
  - o seguir parcheando reinicios/handsoff sin arreglar la intención duradera del mailbox
- la solución correcta en este punto era redefinir `send_instruction` como intención durable del control plane, no como envío oportunista a un PID o handle viejo

Cambios aplicados:

- `db/controlplane_entities.go`
  - `send_instruction` ahora rebindea al handle activo del agente/proyecto si la orden apunta a uno obsoleto
  - si la orden proviene del mailbox y el runtime todavía no está entregable, la orden se reencola con `backoff` en lugar de quedar `fallida`
  - si falla `session_resume`, la orden nacida del mailbox también se reencola y conserva trazabilidad del error
  - se evita duplicar mailbox cuando el mensaje original ya existe y solo falta una entrega válida
- `db/controlplane_entities_test.go`
  - test nuevo: reencolado sin duplicar mailbox cuando no hay handle entregable
  - test nuevo: reencolado cuando falla `session_resume`
  - test nuevo: rebind al handle activo para evitar `working_dir` obsoleto en `session_resume`

Verificación:

- `go test ./db -run 'TestRuntimeOrderSendInstruction(...)' -count=1` OK
- `go test ./cmd -run 'TestProcesarRuntimeMailbox(...)' -count=1` OK
- `go build -o ./orquesta .` OK

Lectura operativa:

- este cambio no cierra todavía todo el núcleo de orquestación
- pero sí corrige un contrato central:
  - el mailbox vuelve a comportarse como cola durable
  - `send_instruction` deja de ser frágil por apuntar a un runtime puntual
  - la entrega en caliente queda mejor alineada con la arquitectura de Orquesta y con el objetivo de gobierno estable sobre agentes vivos

## 22:45 aprox. — cierre del hueco real del supervisor local

Contexto de diseño:

- se revisó `ARQUITECTURA.md` como referencia de contrato
- la pieza correcta no es otro parche en `db`, sino un propietario residente del runtime en `internal/controlruntime`

Hallazgo importante:

- en el árbol actual ya existía una primera implementación del supervisor local
- no era humo: registra procesos locales, distingue modo `resident`/`attached` y ya alimenta `ConsultarEstadoLocal`
- el hueco serio que seguía abierto era otro:
  - el supervisor no exponía pronto el `external_session_id` de Codex como primera fuente canónica
  - por eso `session_resume` podía tardar demasiado en activarse y el mailbox seguía degradando a backlog

Decisión aplicada:

- completar el supervisor local existente en vez de rehacerlo
- hacer que el supervisor residente sondee y cachee el `external_session_id` de Codex durante la ventana inicial de arranque
- hacer que `DetectExternalSessionID` consulte primero el estado del supervisor local antes de volver a escanear el almacén de sesiones

Objetivo operativo:

- reducir el tiempo en que un runtime Codex vivo queda en `bootstrap_only` efectivo por falta de identificación de sesión
- facilitar que el control plane pase antes a `session_resume`
- evitar volver al patrón cíclico de mailbox pendiente + restart oportunista

## 23:10 aprox. — consolidacion de doctrina canonica unica

Motivo del cambio:

- la arquitectura y las reglas del proyecto estaban repartidas entre demasiados `.md`
- eso hacia demasiado facil que un agente trabajase con una lectura parcial y perdiese el contrato del sistema
- antes de seguir con `SQLITE_BUSY` y server-first habia que fijar una fuente unica y obligatoria

Decisión aplicada:

- se crea `docs/BIBLIA_APP_ORQUESTA.md` como doctrina canonica unica de la app
- esa biblia consolida arquitectura, politicas, runbooks, vision, roadmap y la direccion de producto ya estable
- se deja explicitado que:
  - la doctrina estatica vive en la biblia
  - el estado vivo de tareas, propuestas, sesiones y runtimes se consulta en Orquesta
  - si hay conflicto entre documentos, manda la biblia hasta alinear el resto

Cambios aplicados:

- `docs/BIBLIA_APP_ORQUESTA.md`
  - nuevo documento canonico
- `docs/00_INDICE.md`
  - la biblia entra en el indice como referencia primaria
- `ARQUITECTURA.md`
  - pasa a declarar la biblia como consolidacion canonica
- `docs/manual_programador.md`
  - obliga a leer la biblia antes de extender flujos operativos
- `db/runtime_bootstrap_prompt.go`
  - el bootstrap de agentes exige leer la biblia
- `cmd/mcp.go`
  - el briefing MCP incluye la referencia canonica
- `cmd/sesion.go`
  - `sesion inicio` muestra la ruta de la doctrina canonica

Efecto buscado:

- cortar la deriva entre documentos y conversaciones
- evitar que agentes autonomos construyan arquitectura a partir de recuerdos parciales
- fijar una base estable para el siguiente trabajo estructural sobre persistencia, single-writer y eliminacion de `SQLITE_BUSY`

## 23:40 aprox. — raiz estructural de SQLITE_BUSY en el arranque del daemon

Hallazgo confirmado:

- el problema no era solo contencion de SQLite ni "muchas escrituras"
- `serve` y `server run` podian tocar la BD demasiado pronto, incluso antes de saber si el proceso iba a ser realmente el daemon valido
- en los logs se veia el patron exacto:
  - `database is locked (SQLITE_BUSY)`
  - `attempt to write a readonly database`
  - y despues fallo de `listen tcp`
- eso delata arranques fantasma o no validos que primero abren persistencia y solo despues descubren que no pueden poseer el puerto o que van con flags de recuperacion heredadas

Decision aplicada:

- dejar de abrir BD desde el arranque generico de CLI para `serve` y `server run`
- reservar primero el listener del servidor
- solo el proceso que ya posee el puerto abre la BD y arranca el control plane
- el hijo autolanzado del daemon deja de heredar `ORQUESTA_FORCE_LOCAL` y `ORQUESTA_FORCE_LOCAL_DB`

Cambios aplicados:

- `cmd/root.go`
  - `serve` y `server run` dejan de abrir BD via `cobra.OnInitialize`
- `cmd/servidor_unificado.go`
  - el listener se reserva antes de `ensureServerDBOpen()`
  - el servidor sirve con `Serve/ServeTLS` sobre listener ya poseido
- `cmd/server.go`
  - el hijo autolanzado limpia flags de recuperacion local antes de arrancar
- `cmd/root_gating_test.go`
  - se actualizan expectativas para que `serve/server run` no requieran BD previa
- `cmd/servidor_unificado_test.go`
  - test basico del listener previo al arranque

Doctrina consolidada:

- no volver a aceptar un servidor que abra SQLite antes de demostrar que es el dueño legitimo del listener
- no volver a mezclar modo `server` con modo `recovery/local`

## 00:10 aprox. — validacion viva del daemon corregido

Validacion operativa realizada:

- se recompila `./orquesta` con la correccion de arranque
- se comprueba que un segundo `server run` sobre `127.0.0.1:16543` ya falla por `bind: address already in use`
- y ya no aparece el patron previo de tocar SQLite antes del bind

Hallazgo adicional:

- el daemon viejo seguia vivo sin `statefile` del scope actual
- por eso `server doctor` podia ver `healthz` pero no encontraba el estado local en disco
- no era ya un problema del nuevo arranque sino un proceso previo levantado con contrato viejo

Operacion aplicada:

- se reinicia el daemon con el binario nuevo
- queda residente en `127.0.0.1:16543`
- publica `statefile` en `/tmp/orquesta-localrpc-2ea48e3b1141.json`
- `./orquesta server doctor` vuelve a dar `Health RPC: OK`
- `./orquesta status` vuelve a responder por servidor

Evidencia final:

- `server doctor`
  - `State: pid=3196890 addr=127.0.0.1:16543 scope=2ea48e3b1141 db=/home/alberto/Trabajo/orquesta/orquesta.db`
  - `Health RPC: OK`
- log del daemon
  - solo banner de arranque
  - sin `SQLITE_BUSY`
  - sin `attempt to write a readonly database`

Conclusion:

- queda cerrada la causa estructural detectada hoy para `SQLITE_BUSY` en el arranque del daemon
- a partir de aqui el siguiente trabajo ya no es "seguir probando arranques", sino seguir retirando rutas hibridas y cerrar el resto del server-first/single-writer desde una base estable

## 00:25 aprox. — `server status/stop` ya no dependen ciegamente del statefile

Hallazgo:

- puede existir un daemon sano que responda `healthz` aunque el `statefile` falte o se haya quedado fuera de sincronizacion
- si `status` y `stop` dependen solo del `statefile`, se convierten en falsos negativos operativos y empujan a rescate manual innecesario

Decision:

- aceptar `healthz` como fallback de autoridad solo para inspeccion y parada
- ese fallback solo es valido cuando `scope` y el backend activo coinciden con el proceso actual
- la comparacion correcta no es una `DBPath` “especial”, sino `storage_driver + storage_target`
- `statefile + healthz` siguen siendo el estado normal esperado; el fallback no sustituye la coherencia, solo evita operar a ciegas

Cambios:

- `cmd/server.go`
  - `loadServerInfoWithHealthFallback` valida `health.OK`
  - conserva contexto del error de `statefile`
  - reconstruye `ServerInfo` desde `healthz` para `server status` y `server stop`
- `cmd/server_health_fallback_test.go`
  - caso de recuperacion por `healthz`
  - rechazo por `scope` ajeno
  - rechazo por `storage target` ajeno
  - rechazo por `storage driver` ajeno

Validacion:

- `go test ./cmd -run 'Test(LoadServerInfoWithHealthFallbackUsaHealthzSiFaltaStatefile|LoadServerInfoWithHealthFallbackRechazaScopeAjeno|LoadServerInfoWithHealthFallbackRechazaDBAjena|ShouldDelegateToLocalServer|ShouldDelegateToLocalServerExcluyeComandosDeRecuperacion|CommandNeedsDBWithDelegationCoverage|BuildLocalServerProcessEnvLimpiaRecuperacionLocal|EscucharServidorUnificadoReservaListenerAntesDelArranque|NormalizarAddrServidorLocal)' -count=1` => OK

## 21:25 aprox. — el contrato del daemon deja de razonar en terminos de SQLite

Hallazgo:

- varios checks de `server` seguian comparando `DBPath` como si el backend fuese siempre SQLite
- eso era una fuga conceptual: la persistencia ya no debe validarse por “ruta sqlite”, sino por conector/backend activo

Decision:

- el estado y `healthz` del daemon pasan a publicar y validar `storage_driver` y `storage_target`
- `db_path` queda solo como campo de compatibilidad hacia atras
- los comandos `server status`, `server stop` y `server doctor` deben mostrar y verificar el backend real, no asumir SQLite

Cambios:

- `internal/rpclocal/types.go`
  - añade `storage_driver` y `storage_target` a `State` y `HealthResponse`
- `internal/rpclocal/server.go`
  - publica esos campos en `healthz`
- `cmd/server.go`
  - valida `scope + storage_driver + storage_target`
  - deja de comparar solo `DBPath`
  - `server doctor` y `server status` muestran `Storage`
  - el `statefile` nuevo se guarda con metadatos de backend
- `cmd/persistencia.go`
  - `persistencia info` deja de mostrar `DB objetivo`
  - pasa a exponer `Storage driver` y `Storage target`
- `cmd/server_health_fallback_test.go`
  - cubre recuperacion desde `healthz`
  - rechazo por `scope` ajeno
  - rechazo por `storage target` ajeno
  - rechazo por `storage driver` ajeno
- `cmd/persistencia_test.go`
  - valida la nueva salida backend-agnostica
- `cmd/architecture_test.go`
  - deja de tolerar `db.CurrentDBPath()` en `persistencia.go`
  - el uso de `db_path` queda restringido a compatibilidad puntual del servidor

Validacion:

- `go test ./cmd ./internal/rpclocal -run 'Test(LoadServerInfoWithHealthFallback|ShouldDelegateToLocalServer|ShouldDelegateToLocalServerExcluyeComandosDeRecuperacion|CommandNeedsDBWithDelegationCoverage|BuildLocalServerProcessEnvLimpiaRecuperacionLocal|EscucharServidorUnificadoReservaListenerAntesDelArranque|NormalizarAddrServidorLocal|BaseURL)' -count=1` => OK
- `go test ./cmd -run 'TestCmdNoUsaSQLDirectoNiAperturasFueraDeExcepcionesControladas' -count=1` => OK
- reinicio controlado del daemon con el binario nuevo
- `./orquesta server doctor` =>
  - `Storage driver: sqlite`
  - `Storage target: /home/alberto/Trabajo/orquesta/orquesta.db`
  - `Health RPC: OK`
- `ORQUESTA_SERVER_INFO=/tmp/orquesta-missing-state.json ORQUESTA_SERVER_ADDR=127.0.0.1:16543 ./orquesta server status` =>
  - `Statefile: ausente; usando healthz del daemon activo`
  - `Storage: sqlite /home/alberto/Trabajo/orquesta/orquesta.db`
- `ORQUESTA_SERVER_INFO=/tmp/orquesta-missing-state.json ORQUESTA_SERVER_ADDR=127.0.0.1:16543 ./orquesta persistencia info` =>
  - `Descubrimiento: fallback a healthz por statefile ausente`
  - `Ruta activa: servidor local`
- `ORQUESTA_SERVER_INFO=/tmp/orquesta-missing-state.json ORQUESTA_SERVER_ADDR=127.0.0.1:16543 ./orquesta status` =>
  - sigue resolviendo el daemon por server-first
  - responde sin caer a modo local ni exigir rescate manual

Limpieza asociada:

- se elimina `scripts/arrancar_codex1_orquestador.sh`
- motivo: hardcodeaba una BD recuperada en `/tmp` y un daemon viejo en `:16546`
- no era el camino canonico actual y solo podia reintroducir operativa paralela y confusion
- se eliminan `scripts/orquesta-vigilante.service` y `scripts/vigilante.sh`
- motivo: mantenian un segundo plano de control separado para `antigravity`, incompatible con el daemon unico server-first
- `scripts/instalar_servicio.sh` deja de instalar o anunciar ese vigilante legacy
- `scripts/backup_bd.sh` y `scripts/verificar_bd.sh` se declaran y validan ya como utilidades solo-SQLite
- `scripts/instalar_servicio.sh` solo instala el timer de backup cuando el backend resuelto es `sqlite`
- `scripts/orquesta-backup.service` deja de hardcodear `orquesta.db` y `ORQUESTA_DB` en la unit
- motivo: no dejar utilidades de mantenimiento vendidas como genericas cuando no cubren MySQL/Postgres

## 00:45 aprox. — el modo local de recuperacion deja de poder escribir por error

Hallazgo:

- aunque la doctrina ya decia que `ORQUESTA_FORCE_LOCAL_DB` era solo para recuperacion, el init generico de CLI todavia podia abrir la BD en escritura para comandos mutantes o de negocio no cubiertos por la whitelist de diagnostico
- eso convertia el modo de recuperacion en una puerta trasera de escritura lateral y reabria riesgo de deriva y contencion

Decision:

- el modo local de recuperacion pasa a ser estrictamente de solo lectura
- solo se permiten comandos de diagnostico explicitamente cubiertos (`status` y familia `runtime` de inspeccion)
- el resto de comandos deben fallar con error claro y exigir servidor

Cambios:

- `cmd/local_recovery.go`
  - nueva validacion explicita de comandos permitidos en recuperacion
  - nuevo error canonico para comandos no cubiertos
- `cmd/root.go`
  - `openDBForCommand` ya no puede abrir la BD en escritura cuando hay `ORQUESTA_FORCE_LOCAL_DB` o `--local`
- `cmd/cliente_servidor.go`
  - `ensureLocalDB` aplica la misma restriccion en el fallback local de la capa API
- `cmd/local_recovery_test.go`
  - tests para rechazo de mutaciones o listados no diagnosticos en recuperacion

Validacion:

- `go test ./cmd -run 'Test(ShouldOpenRecoveryReadOnlyDB|OpenDBForCommandRechazaMutacionesEnRecuperacionLocal|EnsureLocalDBRechazaComandosNoDiagnosticosEnRecuperacionLocal|LoadServerInfoWithHealthFallbackUsaHealthzSiFaltaStatefile|LoadServerInfoWithHealthFallbackRechazaScopeAjeno|LoadServerInfoWithHealthFallbackRechazaDBAjena|ShouldDelegateToLocalServer|ShouldDelegateToLocalServerExcluyeComandosDeRecuperacion|CommandNeedsDBWithDelegationCoverage|BuildLocalServerProcessEnvLimpiaRecuperacionLocal|EscucharServidorUnificadoReservaListenerAntesDelArranque|NormalizarAddrServidorLocal)' -count=1` => OK
- `ORQUESTA_FORCE_LOCAL_DB=1 ./orquesta runtime ordenes --agente Codex1` => sigue funcionando
- `ORQUESTA_FORCE_LOCAL_DB=1 ./orquesta tarea listar` => ahora falla con error explicito y no toca la ruta de negocio local

## 22:10 aprox. — alineadas las tareas vivas con la biblia y la arquitectura server-first

Hallazgo:

- varias tareas abiertas seguian demasiado escuetas o no reflejaban todavia la doctrina consolidada en `BIBLIA_APP_ORQUESTA.md`
- eso dejaba margen para que un agente interpretase "auditoria" o "integracion" como permiso para reabrir caminos locales, scripts legacy o semantica SQLite-first

Decision:

- las tareas vivas de este frente deben llevar dentro de Orquesta la misma regla canonica que las OP y la biblia
- si una tarea toca persistencia, web/OpenClaw o git/worktree, la nota debe fijar explicitamente el contrato server-first y prohibir vias paralelas

Cambios:

- se revisan las tareas activas `409-426`
- ya estaban alineadas o suficientemente concretas `409`, `410`, `411`, `412`, `421`, `422`, `423` y `424`
- se añaden notas canonicas nuevas a:
  - `#413` para fijar `git/worktree/merge` solo por `gitoperaciones` y sin limpieza destructiva ni rutas manuales paralelas
  - `#425` para fijar `web/openclaw` solo sobre contratos server-first del daemon, con diagnostico/persistencia/control plane visibles por API real
  - `#426` para fijar la auditoria de `git/worktree/merge` sobre el servicio oficial de Orquesta y exigir eliminacion de caminos legacy superados

Validacion:

- `./orquesta tarea nota 413 ...` => `✓ Nota añadida a tarea #413`
- `./orquesta tarea nota 425 ...` => `✓ Nota añadida a tarea #425`
- `./orquesta tarea nota 426 ...` => `✓ Nota añadida a tarea #426`

## 2026-03-31 00:xx aprox. — fijada en doctrina la semantica correcta de `runtime_mailbox`

Hallazgo:

- `runtime_mailbox` no es una cola accidental ni una mejora optativa; viene de la doctrina de autogestion supervisada y de los repos de referencia como primitive persistente entre agentes, sesiones y handoff
- el riesgo real no estaba en la existencia de la mailbox, sino en interpretar demasiado pronto una entrega como consumida

Decision:

- se fija en la biblia que `runtime_mailbox` se mantiene como primitive persistente
- se fija tambien que un mensaje solo puede darse por consumido cuando Orquesta confirma entrega real por el camino oficial de control
- los mensajes supersedidos si pueden consumirse como tales, pero no el mensaje vigente antes de entrega efectiva

Cambios:

- `docs/BIBLIA_APP_ORQUESTA.md`
  - nueva seccion `Mailbox persistente y acuse real`

Validacion:

- revision doctrinal cruzada con `ARQUITECTURA.md`, `docs/op_087_autogestion_supervisada_agentes.md`, `docs/op_095_orquestacion_mixta.md` y `docs/analisis_repos_control_agentes_2026-03-23.md`

## 2026-03-31 00:xx aprox. — regla de proceso endurecida en la biblia

Hallazgo:

- hacia falta dejar por escrito una regla de trabajo mas estricta para evitar cambios de arquitectura o nucleo hechos antes de revisar toda la documentacion del frente

Decision:

- se fija en la biblia que no se toca arquitectura, control plane, persistencia, mailbox, handoff, runtime ni semantica operativa sin estudiar antes biblia, arquitectura, politicas, OP, runbook y estado vivo

Cambios:

- `docs/BIBLIA_APP_ORQUESTA.md`
  - nueva seccion `Regla previa obligatoria antes de tocar codigo`

## 23:10 aprox. — el supervisor local pasa a latir contra el daemon y deja de ser solo observador

Hallazgo:

- el supervisor local de `internal/controlruntime` ya sabia observar PIDs, detectar `external_session_id` y controlar señales, pero no informaba al daemon del estado del runtime
- eso seguia dejando la continuidad demasiado apoyada en que el propio runtime llamase a `agente tick`, justo lo contrario de la doctrina "el server orquesta a los agentes"
- en la práctica eso facilita sesiones zombis, heartbeats perdidos y handoffs/watchdogs disparados por falta de pulso del proceso real

Decision:

- el supervisor local debe emitir `heartbeat` y `finalizacion` al daemon en nombre del runtime supervisado
- la activacion de esa supervision orquestada no debe ocurrir antes de que la sesion/runtime/handle existan en Orquesta, para no abrir una carrera en el arranque

Cambios:

- `internal/controlruntime/supervisor_local.go`
  - nuevo contrato `SupervisorSignal` + `SetSupervisorSignalHandler`
  - el supervisor residente ya guarda `agente` y `proyecto`
  - nueva activacion explicita `ActivarSupervisionOrquestada(...)`
  - nuevo bucle de señales: heartbeat periodico y señal final al salir el proceso
- `internal/controlruntime/pty_local.go`
  - el arranque local inyecta `agente` y `proyecto` en metadata del supervisor/manifest
- `db/controlplane_entities.go`
  - tras crear sesion/handle/runtime y actualizar el handle del arranque, se activa la supervision orquestada del runtime local
- `cmd/controlruntime_hooks.go`
  - el daemon registra el hook del supervisor y traduce cada señal a `agentesService.ProcessTick(...)`
  - se fija `CuotaPct=100` para no reinterpretar una salida del proceso como auto-pausa por cuota

Validacion:

- `go test ./internal/controlruntime -run 'Test(ArrancarPlanYEnviarInstruccionProceso|ArrancarPlanLocalRespetaOverrideCanSendInput|ConsultarEstadoLocalDetectaSessionIDSinSobrescribirDriver|SupervisorLocalEmiteHeartbeatYFinalizacionAlActivarse|DetectExternalSessionIDUsaSupervisorLocalResidente|EnviarInstruccionSesionResumeUsaCodexPerfil)' -count=1` => OK
- `go test ./db -run 'Test(RuntimeOrderSendInstructionSessionResume|RuntimeOrderSendInstructionSessionResumeRecuperaWorkingDirPreferida)' -count=1` => OK
- `go test ./cmd -run 'TestProcessSupervisorSignalIgnoraSenalesIncompletas' -count=1` => OK
- nota: una batería más amplia `go test ./cmd -run 'Test(ControlPlane.*|Agente.*|Status.*)'` sigue mostrando rojos previos en tests antiguos de recuperación local de `status`; no son consecuencia de este cambio de supervisor

## 2026-03-31 00:24 aprox. — `stop` reconcilia tambien sesiones abiertas stale

Hallazgo:

- un agente podia dejar de verse activo en `/api/agentes` y `orquesta status`, pero conservar una fila `sesiones.activa=1` huérfana
- el caso real salio con `Codex6` y `antigravity`
- la causa era precisa: `ejecutarRuntimeOrderStop()` buscaba la sesion con `GetSesionActiva(...)`, que filtra por heartbeat operativo reciente
- si la sesion seguia abierta pero stale, el stop completaba runtime y handle, pero no llegaba a aparcar/cerrar la sesion

Decision:

- `stop` debe reconciliar cualquier sesion abierta del agente, no solo la sesion que siga operativa
- la regla correcta es: si el runtime se detiene, la sesion abierta correspondiente no puede quedar viva por haber perdido ya el heartbeat

Cambios:

- `db/controlplane_entities.go`
  - `ejecutarRuntimeOrderStop()` pasa de `GetSesionActiva(...)` a `GetSesionAbierta(...)`
- `db/controlplane_entities_test.go`
  - nuevo test para sesion stale con `external_session_id`, verificando que tras `stop` queda `pausada` e `inactiva`

Validacion:

- el caso vivo deja de apoyarse en una "sesion activa fantasma" para agentes fuera de flota
- `orquesta status` vuelve a mostrar solo `Codex1-5` como activos

## 2026-03-31 00:3x aprox. — `runtime purgar-handles` deja de depender de reset roto y clasificacion hibrida

Hallazgo:

- el fallo visible de `runtime purgar-handles` mezclaba dos problemas distintos
- por un lado, `runtime purgar-handles` no estaba marcado como comando server-first en `commandSupportsServerMode`, asi que podia abrir la BD local en el init generico del CLI
- por otro, `resetFlagSet()` reinyectaba `StringSlice` usando `DefValue` textual (`[cerrado,fallido]`), dejando flags corruptas como `"[cerrado"` y `"fallido]"`

Decision:

- los comandos runtime de mutacion soportados por API deben estar clasificados explicitamente como server-first
- el reset generico de flags debe normalizar los tipos slice en vez de reutilizar sin mas el `DefValue` textual
- ademas, el propio comando normaliza defensivamente el slice de `--estado`

Cambios:

- `cmd/cliente_servidor.go`
  - `runtime purgar-handles` pasa a formar parte del conjunto server-first
- `cmd/root.go`
  - nuevo `normalizedFlagResetValue(...)` para resetear correctamente flags slice
- `cmd/runtime.go`
  - `normalizarSliceFlags(...)` aplicado a `--estado` en `runtime purgar-handles`
- tests:
  - `cmd/cliente_servidor_test.go`
- `cmd/root_test.go`
  - y se añade regresion de `shouldBypassLocalDB` / `commandSupportsServerMode` para que `runtime purgar-handles` y mutaciones runtime criticas no vuelvan a abrir BD local

## 2026-03-31 00:5x aprox. — `session_resume` por cuota externa ya no queda como fallo ciego

Hallazgo:

- la reproduccion viva del caso de `Codex1` mostro que el problema actual de `send_instruction -> session_resume` no era `cwd` ni `external_session_id`
- el conector aceptaba `exec resume` correctamente, pero devolvia `You've hit your usage limit ... try again at ...`
- Orquesta estaba tratando ese caso como error bruto del runtime y podia dejar una orden `send_instruction` pendiente o fallida sin semantica operativa suficiente

Decision:

- cuando `session_resume` devuelve error de cuota/usage limit/rate limit del proveedor, Orquesta debe traducirlo a pausa operativa del agente
- si la orden venia de mailbox, se reencola con backoff explicito y largo, no con reintento ciego corto
- el motivo debe quedar reflejado en `estado_cuota`, `reanimar_at`, `motivo_pausa` y en la propia orden

Cambios:

- `db/controlplane_entities.go`
  - deteccion de errores de cuota externa en `send_instruction/session_resume`
  - auto-pausa del agente via `PausarAgente(...)`
  - reencolado explicito con `retry_after` largo cuando la entrega venia de mailbox
  - soporte para inferir backoff desde mensajes tipo `try again at 12:56 AM`
- `db/controlplane_entities_test.go`
  - nuevo test de quota/provider backoff en `session_resume`

Validacion:

- `go test ./db -run 'TestRuntimeOrderSendInstruction(SessionResumeRecuperaWorkingDirPreferida|PausaPorCuotaProveedor|UsaSessionResumeCodexLocal)' -count=1` => OK

## 2026-03-31 00:4x aprox. — `runtime_mailbox` deja de acumular `nudge/watchdog` eternos

Hallazgo:

- el problema visible ya no era una `runtime_order` atascada; la reconciliacion stale y el backoff de proveedor estaban funcionando
- el hueco real quedaba en `runtime_mailbox`
- `nudge` y `watchdog` si llegaban a mailbox persistente, pero la capa `procesarRuntimeMailbox*Batch()` no sabia convertir esos `kind` en `send_instruction`
- resultado: mensajes pendientes viejos durante horas aunque hubiese handle activo, especialmente en `session_resume`
- ademas, `nudge` no estaba marcado como supersedible/coalescible, asi que podia acumular varias copias del mismo empuje mientras habia una `send_instruction` abierta o un backoff largo

Decision:

- `nudge` y `watchdog` pasan a formar parte del mismo contrato de entrega persistente que `instruction` y `autonomia`
- si son coalescibles, el control plane debe consumir los supersedidos aunque todavia no pueda entregar el ultimo por existir una orden abierta
- la semantica correcta es "solo la ultima señal efimera sigue pendiente", no una pila eterna de nudges viejos

Cambios:

- `db/controlplane_entities.go`
  - `nudge` pasa a ser `runtimeMailboxKindSupersedible(...)`
- `cmd/controlplane_support.go`
  - `nudge` y `watchdog` pasan a ser kinds coalescibles y entregables por `interactive`, `session_resume` y `coordinated_restart`
  - nuevo `coalescerRuntimeMailboxPendiente(...)` para consumir supersedidos en el propio batch aunque exista una `send_instruction` abierta
  - `construirInstruccionMailboxInteractivo(...)` ya traduce `nudge/watchdog` a texto entregable
- tests:
  - `db/controlplane_entities_test.go`
  - `cmd/controlplane_support_test.go`

Validacion prevista:

- `runtime_mailbox` no debe seguir creciendo con `nudge/watchdog` antiguos para un mismo agente/proyecto
- con una `send_instruction` abierta, solo debe sobrevivir el mensaje mas nuevo del kind coalescible
- cuando el conector vuelva a poder entregar, `nudge/watchdog` deben convertirse en `send_instruction` por el camino oficial del daemon

## 2026-03-31 01:03 aprox. — respaldo canonico antes de reescribir git

Hallazgo:

- la rama local tenia un commit valido de nucleo (`62f5ad1`) pero el push a remoto fallaba porque `orquesta.db` habia entrado en el commit y GitHub rechazaba el objeto por superar 100 MB
- esa base contiene el estado vivo completo del orquestador: tareas, OP, sesiones, runtime orders, mailbox, checkpoints y auditoria

Decision:

- antes de tocar el historial git, hay que crear respaldos consistentes por el camino oficial de Orquesta
- `orquesta.db` no debe volver a formar parte del historial git; la base viva se conserva fuera del indice y las copias canonicas se guardan fuera del repo

Accion ejecutada:

- respaldo oficial externo:
  - `/home/alberto/Trabajo/backups/orquestador/2026-03-31_01-03-17.186503348_pre_amend_62f5ad1_orquesta.db.bak`
- respaldo oficial local de salvaguarda:
  - `/tmp/orquesta-backups/2026-03-31_01-03-34.704483012_pre_amend_62f5ad1_orquesta.db.bak`
- despues del respaldo, `orquesta.db` se saca del indice git para reamendar el commit sin perder la base local

## 2026-03-31 01:10 aprox. — el supervisor local deja de revivir handles por PID fantasma

Hallazgo:

- el estado vivo mostraba `runtime_handles` de `Codex1` y `Codex2` como `activo` con `last_seen` reciente, pero los PID publicados ya no existian en `/proc`
- la validacion local del supervisor se apoyaba practicamente en `kill(pid, 0)`, lo que es insuficiente para un proceso adjunto o tras reinicios/reutilizacion de PID

Decision:

- un runtime local no se considera el mismo proceso solo porque el PID responda
- el supervisor debe validar tambien identidad minima del proceso:
  - `cwd` esperada
  - firma de comando esperada (`rendered_command` / `wrapped_command`)

Cambio:

- `internal/controlruntime/supervisor_local.go`
  - nuevo endurecimiento de `snapshot()`: antes de revivir un runtime por PID vivo, valida identidad del proceso
  - nuevas ayudas `validarIdentidadProcesoLocal(...)`, `commandIdentityHints(...)` y `samePath(...)`
- `internal/controlruntime/supervisor_local_test.go`
  - cobertura de hints de identidad, comparacion de rutas y mismatch de `cwd`

Validacion:

- `go test ./internal/controlruntime -run 'Test(ConsultarEstadoLocalDetectaSessionIDSinSobrescribirDriver|SupervisorLocalEmiteHeartbeatYFinalizacionAlActivarse|CommandIdentityHintsIncluyeFirmaUtil|SamePathResuelveSymlink|ValidarIdentidadProcesoLocalDetectaMismatchDeCWD)' -count=1` => OK

## 2026-03-31 01:15 aprox. — SQLite vuelve a validar schema real aunque la revision ya exista

Hallazgo:

- tras reiniciar el daemon con el binario nuevo, `orquesta runtime ordenes` fallaba contra la base viva con `SQL logic error: no such column: claimed_by`
- el problema no era el modelo de dominio, sino la puerta de bootstrap SQLite: si la revision `_sqlite_bootstrap_revision` ya estaba marcada, dejaba de comprobar la estructura real y podia convivir con una `runtime_orders` incompleta

Decision:

- en SQLite no basta con “revision marcada”; siempre hay que validar la estructura real del schema si el daemon arranca con bootstrap habilitado
- la revision sigue sirviendo como marca historica, pero no puede anular la verificacion efectiva de columnas y reconstrucciones necesarias

Cambio:

- `db/backend_sqlite.go`
  - `Prepare(...)` vuelve a preguntar `sqliteBootstrapRequired(...)` siempre que `BootstrapSchema=true`
  - `sqliteBootstrapRequired(...)` deja de cortocircuitar por revision marcada y valida tablas, columnas y rebuilds reales

Validacion:

- `go test ./db -run 'TestSQLite(PrepareAplicaPostMigracionesAunqueLaRevisionYaEsteMarcada|BootstrapRequiredEnDBVacia|PrepareOmiteBootstrapConSchemaActualSinRevision)' -count=1` => OK
- `go build -o ./orquesta .` => OK

## 2026-03-31 01:17 aprox. — falso bug de API por slug de proyecto incorrecto

Hallazgo:

- `runtime diagnostico`, `/api/runtimes/tree`, `/api/runtime-orders` y `/api/runtime-checkpoints` devolvian `sql: no rows in result set` cuando se consultaban con `proyecto=orquesta`
- la API no estaba rota: el estado vivo mostraba que el proyecto activo en BD seguia siendo `orquestador`
- el repositorio fisico vive en `/home/alberto/Trabajo/orquesta`, pero ese path no implica que el slug operativo sea `orquesta`

Decision:

- no tocar codigo para esconder un error de filtro
- fijar en doctrina que el slug canónico se consulta en Orquesta y no se infiere del nombre del directorio

Validacion viva:

- `./orquesta proyecto listar` => proyecto activo `orquestador` con ruta `/home/alberto/Trabajo/orquesta`
- `./orquesta runtime diagnostico --agente Codex2 --proyecto orquestador --limit 5` => OK
- `./orquesta runtime ordenes --agente Codex2 --proyecto orquestador --estado pendiente` => OK

## 2026-03-31 01:2x aprox. — un batch colgado ya no congela todo el control plane

Hallazgo:

- el estado vivo mostraba `runtime_orders` y `runtime_mailbox` pendientes durante horas aunque el servidor HTTP seguia sano
- el patron real no era otra vez SQLite: una sola `send_instruction/session_resume` podia quedarse colgada y bloquear el loop entero del control plane
- `planocontrol.Runner.runControlPlane()` ejecutaba todos los batches de forma secuencial en el mismo hilo logico; si uno no devolvia, tampoco corria `runtime_orders_stale`, `runtime_mailbox` ni el resto de reconciliaciones

Decision:

- el `Runner` no puede depender de que cada batch externo devuelva bien
- cada batch del control plane debe tener timeout propio y exclusion mutua por nombre
- si un batch se queda colgado, el resto del plano de control debe seguir avanzando y el batch atascado no debe duplicarse en paralelo

Cambios:

- `planocontrol/runner.go`
  - nuevo timeout por batch (`BatchTimeout`, por defecto `45s`)
  - exclusion por nombre de batch para no lanzar duplicados mientras uno sigue en curso
  - auditoria explicita de timeout (`..._error`) en lugar de congelar el loop completo
- `planocontrol/runner_test.go`
  - nueva regresion `TestRunnerRunControlPlaneTimeoutDeBatchNoCongelaElResto`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./planocontrol -run 'TestRunner(RunControlPlaneTimeoutDeBatchNoCongelaElResto|RunControlPlaneRecuperaPanicDeBatchYSigue|RunControlPlaneAuditaTrabajoProcesado|RunControlPlaneAuditaErrores)' -count=1` => OK

## 2026-03-31 01:3x aprox. — SQLite arrancaba con schema drift si la revision ya estaba marcada

Hallazgo:

- tras reiniciar el daemon con el binario nuevo, `runtime ordenes` fallo con `SQL logic error: no such column: claimed_by (1)`
- la columna y el contrato de lease ya existian en `schema.go` y en `post_migraciones_compat.go`
- el fallo real estaba en la preparacion SQLite: las post-migraciones solo se ejecutaban cuando la revision de bootstrap no estaba marcada
- ademas, la reconstruccion legacy de `runtime_orders` recreaba la tabla sin las nuevas columnas de lease, fiando la correccion a unas post-migraciones que podian no volver a correr

Decision:

- las post-migraciones SQLite deben ejecutarse siempre que no esten desactivadas explicitamente; son idempotentes y forman parte de la compatibilidad viva
- la reconstruccion legacy de `runtime_orders` debe recrear ya la forma completa de la tabla, no una forma intermedia antigua

Cambios:

- `db/backend_sqlite.go`
  - `Prepare(...)` aplica `postMigracionesPorDriver("sqlite")` siempre que `SkipPostMigrations=false`
  - `sqliteRuntimeOrdersNeedsRebuild(...)` ya exige tambien `claimed_by`, `lease_token`, `attempt_count` y `lease_expires_at`
- `db/db.go`
  - `reconstruirRuntimeOrdersLegacy(...)` recrea `runtime_orders` con las columnas de lease y las rellena con defaults seguros durante la copia
- `db/backend_sqlite_test.go`
  - nueva regresion `TestSQLitePrepareAplicaPostMigracionesAunqueLaRevisionYaEsteMarcada`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(SQLitePrepareAplicaPostMigracionesAunqueLaRevisionYaEsteMarcada|OpenMigraRuntimeTablesLegacy|SQLitePrepareOmiteBootstrapConSchemaActualSinRevision|SQLiteBootstrapRequiredEnDBVacia)' -count=1` => OK

## 2026-03-31 01:1x aprox. — `runtime_orders` deja de depender solo de timestamps para stale

Hallazgo:

- el control plane seguia reclamando ordenes solo con `estado + started_at`
- la recuperacion stale se apoyaba sobre todo en `started_at/updated_at`, sin contrato de lease explicita
- eso dejaba la reconciliacion a medio camino: mejor que antes, pero todavia demasiado heuristica para un supervisor server-first duradero

Decision:

- `runtime_orders` pasa a tener lease minima persistente: `claimed_by`, `lease_token`, `attempt_count`, `lease_expires_at`
- toda reclamacion de orden debe materializar esa lease
- toda finalizacion o reencolado debe liberarla
- la reconciliacion stale debe mirar primero `lease_expires_at`

Cambios:

- `db/schema.go`
- `db/post_migraciones_compat.go`
- `db/config_defaults.go`
- `db/controlplane_entities.go`
- tests:
  - `db/runtime_legacy_migration_test.go`
  - `db/controlplane_entities_test.go`

## 2026-03-31 01:2x aprox. — supervisor local y runner endurecidos contra identidades falsas y batches colgados

Hallazgo:

- el supervisor local podia considerar valido cualquier PID vivo aunque ya no correspondiese al runtime esperado
- el runner del control plane podia quedarse colgado entero si un batch no devolvia

Decision:

- el supervisor local debe validar identidad por `cwd` y firma util del comando, no solo por PID
- cada batch del runner necesita aislamiento por nombre y timeout propio para no congelar el resto del plano de control

Validacion:

- `go test ./internal/controlruntime -run 'Test(ConsultarEstadoLocalDetectaSessionIDSinSobrescribirDriver|CommandIdentityHintsIncluyeFirmaUtil|SamePathResuelveSymlink|ValidarIdentidadProcesoLocalDetectaMismatchDeCWD)' -count=1`
- `go test ./planocontrol -run 'TestRunner(RunControlPlaneRecuperaPanicDeBatchYSigue|RunControlPlaneTimeoutDeBatchNoCongelaElResto|SafeLoopCallRecuperaPanic)' -count=1`

## 2026-03-31 01:4x aprox. — `send_instruction` recupera entrega caliente por supervisor local sin romper la politica anti-TTY

Hallazgo:

- la cola `runtime_orders_batch` estaba viva y los handles activos de Codex tenian `stdin_path`, `supervisor_ref` y `external_session_id`
- aun asi, `send_instruction` seguia bloqueado en `bootstrap_only/session_resume` porque la decision solo miraba `RuntimeHandlePermiteSendInputInteractivo`
- eso mezclaba dos conceptos distintos: TTY interactivo generico y canal de entrada ya supervisado por el daemon

Decision:

- mantener `RuntimeHandlePermiteSendInputInteractivo=false` para Codex PTY inestable
- introducir una capacidad separada para el caso seguro: entrega caliente por supervisor local de Orquesta
- si el handle `process_pty_cli` tiene `stdin_path` y `supervisor_ref` validos, el daemon debe intentar `controlruntime.EnviarInstruccionProceso(...)` antes de degradar a `session_resume` o mailbox

Cambios:

- `db/controlplane_entities.go`
  - nuevo helper `RuntimeHandlePermiteEntregaCalienteSupervisada`
  - `ejecutarRuntimeOrderSendInstruction(...)` ya usa esa via de control real y registra `delivery_path`
- `db/controlplane_entities_test.go`
  - regresion para exigir `stdin_path + supervisor_ref`
  - regresion para mantener fallback a mailbox cuando falta `supervisor_ref`
  - regresion para verificar entrega caliente por supervisor local aunque `can_send_input=false`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(RuntimeHandlePermiteSendInputInteractivoRespetaCapacidadesExplicitas|RuntimeHandlePermiteEntregaCalienteSupervisadaExigeSupervisorYStdin|RuntimeOrderSendInstructionHaceFallbackAMailboxCuandoHandleNoAdmiteInputInteractivo|RuntimeOrderSendInstructionEntregaEnCalientePorSupervisorLocalAunqueCanSendInputSeaFalse)' -count=1` => OK

## 2026-03-31 01:5x aprox. — `status` deja de mentir cuando la sesion sigue viva pero el runtime ya murio

Hallazgo:

- `Codex2` seguia apareciendo activo en `status` porque `ListarSesionesActivasOperativas()` daba prioridad al `heartbeat` reciente de la sesion
- el ultimo `runtime_handle` de la misma sesion ya estaba `fallido`, asi que el sistema mostraba como vivo a un agente sin runtime entregable

Decision:

- para presencia visible, el ultimo handle de la misma sesion manda sobre el heartbeat reciente cuando ese handle ya esta `cerrado` o `fallido`
- una sesion reciente no puede sostener por si sola un agente activo si su runtime mas reciente ya cayo

Cambios:

- `db/sesiones.go`
  - mapa del ultimo handle por agente/proyecto
  - invalidacion de sesion operativa por ultimo handle terminal reciente
  - `GetSesionActivaOperativa(...)` y `ListarSesionesActivasOperativas()` ya comparten esa precedencia
- tests:
  - `db/sesiones_test.go`
  - `cmd/api_test.go`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(ListarAgentesIgnoraSesionesConHeartbeatObsoleto|SesionRecienteNoCuentaComoOperativaSiSuUltimoHandleYaFallo)' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(APIAgentesYStatusOcultanSesionZombi|APIStatusNoCuentaSesionConHandleFallidoReciente)' -count=1` => OK
- validacion viva tras reiniciar daemon:
  - `./orquesta status` pasa de `5` a `4` agentes activos
  - `Codex2` ya no sale activo mientras `runtime handle #352` permanece `fallido`

## 2026-03-31 02:0x aprox. — las `send_instruction` stale de mailbox ya no deben reintentarse contra una sesion vieja

Hallazgo:

- tras corregir la entrega segura para Codex, seguian quedando `send_instruction` antiguas (`#80150`, `#80157`) apuntando al mismo `mailbox_id=64424`
- parte de esa deuda arrastraba `external_session_id` de una sesion anterior, aunque el agente ya hubiese arrancado una sesion nueva
- reintentarlas indefinidamente no aportaba nada: la verdad seguia en el mailbox y la orden quedaba haciendo ruido operativo

Decision:

- si una `send_instruction` procedente de mailbox trae `external_session_id` vieja y el agente ya tiene otra `external_session_id` viva, la orden se completa como `superseded`
- el mailbox permanece como verdad para bootstrap/continuidad; la orden stale deja de competir con la sesion actual

Cambios:

- `db/controlplane_entities.go`
  - nuevo helper `completarRuntimeOrderSendInstructionSesionObsoleta(...)`
  - `ejecutarRuntimeOrderSendInstruction(...)` corta antes los reintentos de mailbox stale por cambio de sesion
- `db/controlplane_entities_test.go`
  - nueva regresion `TestRuntimeOrderSendInstructionMailboxSupersedeSesionObsoleta`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(RuntimeOrderSendInstructionMailboxSupersedeSesionObsoleta|GuardarSesionActivaPreservaMetadataRicaDelHandle|RuntimeOrderSendInstructionCodexSupervisadoSigueCayendoAMailbox)' -count=1` => OK

## 2026-03-31 02:0x aprox. — se corrige una falsa buena idea: `stdin` caliente a Codex PTY rompe el TUI

Hallazgo:

- la entrega caliente por supervisor local parecia funcionar porque `#80147` quedo completada y el transcript registraba `stdin`
- la traza raw del handle `352` demostro la causa real del crash:
  - `The application panicked (crashed).`
  - `byte index ... is out of bounds of 'rueba supervisor local 2026-03-31T01:26Z'`
  - origen en `tui_app_server/src/wrapping.rs:52`
- por tanto, inyectar texto directo por PTY a Codex TUI no es un canal seguro aunque exista `stdin_path` y `supervisor_ref`

Decision:

- la entrega caliente por supervisor local queda permitida solo para runtimes locales seguros
- Codex PTY sigue excluido y debe continuar por `mailbox/session_resume/bootstrap`, no por `stdin` directa
- el hallazgo se fija en la biblia para no volver a caer en programacion ciclica

Cambios:

- `db/controlplane_entities.go`
  - `RuntimeHandlePermiteEntregaCalienteSupervisada(...)` ahora excluye explicitamente `runtimeHandleUsaCodexTTYInestable(...)`
- `db/controlplane_entities_test.go`
  - se mantiene la entrega caliente para un runtime local seguro
  - nueva regresion para forzar que Codex supervisado siga cayendo a mailbox segura

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(RuntimeHandlePermiteEntregaCalienteSupervisadaExigeSupervisorYStdin|RuntimeOrderSendInstructionHaceFallbackAMailboxCuandoHandleNoAdmiteInputInteractivo|RuntimeOrderSendInstructionEntregaEnCalientePorSupervisorLocalSeguroAunqueCanSendInputSeaFalse|RuntimeOrderSendInstructionCodexSupervisadoSigueCayendoAMailbox)' -count=1` => OK
- validacion viva:
  - `./orquesta runtime traza --handle-id 352 --bytes 4096` mostro el `runtime_panic` de Codex tras la inyeccion antigua
  - tras la correccion, una nueva orden `#80161` ya no genero un nuevo `stdin` en transcript ni otro `runtime_panic`

## 2026-03-31 02:0x aprox. — el supervisor local rehidrata metadata rica desde el manifest del runtime

Hallazgo:

- tras reiniciar daemon o reanudar un proceso ya vivo, algunos `runtime_handles` activos quedaban con metadata generica de sesion (`cwd`, `external_session_id`, `herramienta`) aunque el runtime real ya tuviese `stdin_path`, `log_path`, `rendered_command`, `supervisor_ref` y `mailbox_delivery_mode`
- eso degradaba la verdad operativa: el proceso seguia activo, pero el control plane lo veia como handle pobre y perdia capacidad de supervision fina

Decision:

- si el supervisor local observa un proceso valido y la metadata del handle sigue pobre, debe buscar el `runtime manifest` del mismo proceso y rehidratar la metadata rica
- la rehidratacion correcta pertenece al camino oficial del daemon (`sync_status` y supervision), no a rescates manuales ni a acceso directo a BD

Cambios:

- `internal/controlruntime/supervisor_local.go`
  - rehidratacion de `descriptorSupervisorLocal` desde `trace_manifest`, `trace_dir/runtime.json` y candidatos bajo `.orquesta-runtime/.../runtime.json`
  - merge conservador contra manifest validando identidad por `pid`
- `internal/controlruntime/supervisor_local_test.go`
  - nueva regresion `TestConsultarEstadoLocalRehidrataMetadataRicaDesdeRuntimeManifest`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./internal/controlruntime -run 'Test(ConsultarEstadoLocalDetectaSessionIDSinSobrescribirDriver|ConsultarEstadoLocalRehidrataMetadataRicaDesdeRuntimeManifest|SupervisorLocalEmiteHeartbeatYFinalizacionAlActivarse|ValidarIdentidadProcesoLocalDetectaMismatchDeCWD)' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(GuardarSesionActivaPreservaMetadataRicaDelHandle)' -count=1` => OK
- validacion viva:
  - `./orquesta runtime orden-nueva Codex2 sync_status --proyecto orquestador --payload '{}'` => encolada `#80221`
  - `curl -sS 'http://127.0.0.1:16543/api/runtime-handles?agente=Codex2'` ya devuelve para el handle activo `#363` metadata rica persistida con `log_path`, `mailbox_delivery_mode`, `rendered_command` y `can_send_input=false`

## 2026-03-31 02:1x aprox. — el daemon deja de rematerializar `send_instruction` nuevas para el mismo mailbox durable

Hallazgo:

- ya habiamos cerrado el bucle dentro de una misma `send_instruction`, pero el control plane seguia fabricando ordenes nuevas para el mismo `mailbox_id`
- el caso vivo estaba en `Codex2` con `mailbox_id=64431`: la orden derivada se completaba como `mailbox_only`, pero en el siguiente ciclo volvia a nacer otra `send_instruction` sobre el mismo `handle_id=363`

Decision:

- para `runtime_mailbox`, un intento de entrega queda identificado por `mailbox_id + handle_id + external_session_id efectiva`
- si ya existe una `send_instruction` abierta para ese mismo intento, o una completada con `mailbox_only=true`, el batch no puede volver a materializar otra orden nueva
- solo se permite un nuevo intento cuando cambia el handle, la sesion efectiva o el contrato de entrega

Cambios:

- `cmd/controlplane_support.go`
  - nueva deduplicacion `existeIntentoSendInstructionMailboxParaHandle(...)`
  - `procesarRuntimeMailboxInteractivoBatch()` y `procesarRuntimeMailboxSessionResumeBatch()` dejan de crear nuevas `send_instruction` si ya hubo intento diferido para el mismo mailbox sobre el mismo handle/sesion
- `cmd/controlplane_support_test.go`
  - nuevas regresiones:
    - `TestProcesarRuntimeMailboxSessionResumeBatchNoRematerializaMailboxOnlyEnMismaSesion`
    - `TestProcesarRuntimeMailboxInteractivoBatchNoRematerializaMailboxOnlyEnMismoHandle`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'TestProcesarRuntimeMailbox(SessionResumeBatchNoRematerializaMailboxOnlyEnMismaSesion|InteractivoBatchNoRematerializaMailboxOnlyEnMismoHandle|SessionResumeBatchRespetaLeaseBootstrapPendiente|InteractivoBatchCoalesceAutonomiaPendiente)' -count=1` => OK
- validacion viva:
  - se recompila `./orquesta`, se reinicia el daemon y `./orquesta server doctor` vuelve a `Health RPC: OK`
  - antes del reinicio ya existia la orden `#80240` para `mailbox_id=64431`; tras arrancar el daemon nuevo, `#80240` se completa a `mailbox_only`
  - despues del reinicio no aparecen ordenes nuevas `#8024x/#8025x` para ese mismo `mailbox_id=64431`; el mailbox durable sigue pendiente y la cascada de rematerializacion queda cortada

## 2026-03-31 02:2x aprox. — Orquesta gana purga segura de runtime orders terminales para pruebas

Hallazgo:

- ya existia `runtime purgar-handles`, pero no habia una via oficial para limpiar `runtime_orders` de pruebas sin tocar la BD a mano
- eso complicaba reproducir y limpiar escenarios, y empujaba a soluciones ad hoc que contradicen la doctrina server-first

Decision:

- se añade purga oficial solo para `runtime_orders` terminales
- la purga exige filtro por `agente` o `proyecto`
- solo admite estados terminales (`completada`, `fallida`, `expirada`, `cancelada`)
- si se borran ordenes, cualquier `runtime_mailbox.runtime_order_id` asociado queda a `NULL`
- no se ejecuta purga real sobre la BD viva durante esta sesión porque sería destructiva sin necesidad inmediata

Cambios:

- `db/controlplane_entities.go`
  - `PurgarRuntimeOrdersTerminales(...)`
  - validaciones y normalizacion de estados/tipos
- `runtimesapp/service.go`
  - servicio `PurgeTerminalRuntimeOrders(...)`
- `cmd/api.go`
  - endpoint `POST /api/runtime-orders/purgar`
- `cmd/runtime.go`
  - comando `runtime purgar-ordenes`
- tests:
  - `db/controlplane_entities_test.go`
  - `runtimesapp/service_test.go`
  - `cmd/api_runtimes_test.go`
  - `cmd/runtime_test.go`
  - `cmd/cliente_servidor_test.go`
  - `cmd/status_test.go`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'TestPurgarRuntimeOrdersTerminales(BorraSoloTerminalesYDesenlazaMailbox|BloqueaEstadosVivos)' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./runtimesapp -run 'TestServiceDelegatesRuntimeQueries' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(APIRuntimeControlPlaneEndpoints|RuntimeControlPlaneUsaAPICuandoHayServidor|RuntimeMutacionesCriticasSoportanServerMode|ShouldBypassLocalDBConServidorDescubierto)' -count=1` => OK
