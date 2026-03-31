# Biblia de la App Orquesta

## Estado de este documento

Este archivo es la doctrina canonica de Orquesta.

Su funcion es consolidar en un unico sitio la direccion arquitectonica, operativa y de desarrollo del proyecto para que ningun agente dependa de una lectura parcial o dispersa del repo.

Regla de precedencia:

1. Este archivo manda como doctrina estatica del proyecto.
2. El estado vivo manda para tareas, propuestas, sesiones, runtimes y colas, y se consulta en Orquesta.
3. Si otro documento antiguo contradice esta biblia, prevalece esta biblia hasta que el otro documento se actualice.

## Fuentes consolidadas

Este archivo consolida y resume, entre otros, estos documentos:

- `ARQUITECTURA.md`
- `docs/orquesta_v1_vision.md`
- `docs/orquesta_v1_roadmap.md`
- `docs/uso_actual_app_orquesta.md`
- `docs/manual_programador.md`
- `docs/politica_acceso_persistencia_es.md`
- `docs/politica_arquitectura_tipos_proyecto_es.md`
- `docs/runbook_control_plane_agentes.md`
- `docs/op_087_autogestion_supervisada_agentes.md`
- `docs/op_088_orquesta_servidor_mcp.md`
- `docs/op_089_relevo_agentes_por_token.md`
- `docs/op_090_sondeo_y_nudge_de_agentes.md`
- `docs/op_091_memoria_entidades.md`
- `docs/op_092_time_travel_debugging.md`
- `docs/op_093_refineria_merge_queue.md`
- `docs/op_094_ui_declarativa_agentes.md`
- `docs/op_095_orquestacion_mixta.md`
- `docs/informe_autonomia_orquestador_2026-03-25.md`
- `docs/informe_revision_hexagonal.md`
- `docs/inventario_pendientes_orquestacion_autonoma_2026-03-24.md`

## Lectura obligatoria

Todo agente, humano o autonomo, debe:

1. Leer este archivo al inicio de cada sesion o bootstrap.
2. Consultar despues el estado vivo en Orquesta.
3. Trabajar segun esta doctrina y no segun recuerdos parciales de conversaciones anteriores.

## Que es Orquesta

Orquesta no es un conjunto de scripts ni una base SQLite con CLI encima.
Orquesta es el plano de control de agentes, proyectos, sesiones, tareas, propuestas, runtimes y continuidad del workspace.

Sus principios estructurales son:

- identidad estable del agente
- asignacion explicita a proyecto
- sesion reanudable
- runtime encapsulado por conector
- reglas, skills y workflows servidos por Orquesta
- persistencia como adaptador, no como nucleo
- web, CLI y futuras apps como clientes finos

## Fuentes de verdad

Hay dos y no deben mezclarse:

### 1. Doctrina estatica

La fuente estatica de referencia es este archivo.

Sirve para:

- arquitectura
- reglas de desarrollo
- reglas operativas
- decisiones de producto ya consolidadas

### 2. Estado operativo vivo

La fuente de verdad viva es Orquesta, consultada por daemon, API, web o CLI server-first.

Sirve para:

- tareas activas
- propuestas abiertas o pendientes de voto
- sesiones
- runtime orders
- runtime mailbox
- runtime handles
- checkpoints
- progreso real del proyecto

Regla de precedencia operativa para presencia de agentes:

- una sesion con `heartbeat` reciente no basta para declarar un agente como activo si el ultimo `runtime_handle` de esa misma sesion ya esta `fallido` o `cerrado`
- para presencia visible en `status`, `/api/status` y `/api/agentes`, el runtime terminal reciente invalida el heartbeat reciente de la sesion

La documentacion no sustituye el estado vivo.
Las tareas y propuestas vigentes no se leen de un `.md` si ya existen en Orquesta.

Regla operativa adicional:

- el slug canónico de proyecto no se asume por el nombre del repositorio en disco
- antes de filtrar `runtime`, `mailbox`, `checkpoints`, `tareas` o `propuestas` por proyecto, hay que consultar el estado vivo en Orquesta
- a fecha de este diario, el repositorio vive en `/home/alberto/Trabajo/orquesta`, pero el proyecto activo registrado en la BD sigue siendo `orquestador`; confundir `ruta` con `slug` rompe diagnósticos y puede aparentar falsos bugs de API

## Doctrina operativa obligatoria

### Server-first

El camino operativo normal es siempre:

- `orquesta serve` o `orquesta server run`
- API HTTP/JSON
- web
- CLI como cliente del servidor

Regla dura:

- no hay operacion normal contra la BD local
- el modo local queda solo para recuperacion explicita o diagnostico de bloqueo real
- `ORQUESTA_FORCE_LOCAL_DB` y `--local` no autorizan mutaciones de negocio; solo inspeccion/diagnostico de solo lectura en los comandos explicitamente cubiertos
- las superficies externas nuevas tambien entran por el daemon oficial: si Orquesta expone MCP por HTTP, debe hacerlo como endpoint server-first del mismo servidor (`/api/mcp`), no como proceso lateral con otra verdad operativa
- A2UI no es una excepcion: su superficie canonica por API debe colgar del mismo daemon, reutilizar la proyeccion existente de `serve` y exponer el detalle de runtime en `/api/runtimes/{id}/a2ui` antes de abrir cualquier edicion humana o flujo paralelo
- OpenClaw Gateway y el resto de canales de notificacion tambien son estado operativo del daemon: se exponen por lectura server-first en `/api/notificaciones` y en el dashboard web del mismo servidor, no se infieren solo de claves de configuracion ni de scripts externos

### Single-writer

La arquitectura objetivo de operacion es:

- un daemon oficial
- un unico plano de control
- un unico arbitro de mutaciones
- clientes finos que no compiten por la BD

Consecuencia:

- `SQLITE_BUSY` no se arregla a base de mas reintentos
- se arregla cerrando caminos hibridos, escrituras laterales y segundos planos de control
- el servidor oficial no puede tocar la BD antes de poseer el listener/puerto que lo convierte en autoridad
- `serve` y `server run` no deben abrir persistencia desde el arranque generico de CLI; su ciclo de vida de BD pertenece al propio proceso servidor
- un proceso hijo del daemon no puede heredar flags de recuperacion local ni semantica de solo lectura
- el daemon sano debe publicar `healthz` y `statefile` coherentes del mismo scope; si una de las dos piezas falta, hay que tratarlo como incoherencia operativa y repararlo
- el contrato de autoridad del daemon no se valida contra una ruta SQLite “especial”, sino contra el `storage_driver` y el `storage_target` del backend activo
- `server status` y `server stop` pueden caer a `healthz` si falta el `statefile`, pero solo para el mismo `scope`, el mismo `storage_driver` y el mismo `storage_target`; el fallback no autoriza a cruzar de daemon ni de backend

### Persistencia como adaptador

La app no puede depender semanticamente de SQLite.
El backend de persistencia debe poder ser SQLite, MySQL/MariaDB o el que se soporte por conector de almacenamiento.

Reglas:

- la logica de negocio no se acopla a SQL directo
- `db/` no debe crecer como nucleo de aplicacion
- la persistencia entra por adaptadores y servicios
- si una necesidad de negocio exige SQL directo, hay que cerrar esa brecha en API/servicios

### Nada de segundo plano de control

No se permite crear:

- otra cola paralela fuera de Orquesta
- otra fuente de verdad para tareas o sesiones
- scripts que gobiernen agentes como camino principal
- UI o CLI con logica de negocio duplicada

Los scripts manuales solo valen para:

- rescate
- compatibilidad transitoria
- diagnostico

Regla adicional:

- si un script o unit solo cubre un backend concreto o un flujo legacy, debe declararlo de forma explicita o salir del camino oficial; no se mantiene como plantilla “generica” si ya no lo es
- la observabilidad y verificacion operativa de persistencia se hacen desde Orquesta (`persistencia info`, `persistencia verificar`); los scripts backend-especificos quedan solo para rescate o forense
- si se automatiza backup por systemd y el backend es SQLite, la unit debe nombrarse de forma explicita como SQLite; no se aceptan nombres genericos para unidades que no son multi-backend
- las operaciones de limpieza o purga de pruebas tambien deben entrar por el daemon: `runtime purgar-handles` para handles inactivos y `runtime purgar-ordenes` solo para ordenes terminales; no se autorizan borrados manuales directos en la BD

## Doctrina de agentes y runtimes

### El runtime va detras de un conector

Orquesta debe conocer el contrato, no el proveedor concreto.
Codex, Claude, Gemini u otro runtime son implementaciones de un conector.

### Orquesta debe gobernar el ciclo de vida

El objetivo no es "arrancar agentes", sino gobernarlos:

- start
- stop
- pause
- resume
- handoff
- checkpoints
- mailbox
- continuidad
- observabilidad

### Decision estrategica sobre frameworks de agentes

La base de Orquesta no se va a reescribir sobre LangChain, LangGraph, CrewAI ni otro framework externo de agentes.

Motivo:

- el problema nuclear del proyecto no es conversacional ni de prompting; es de control operativo duradero
- Orquesta ya tiene piezas reales de control plane, sesiones, tareas, runtime orders, mailbox, checkpoints, API y daemon
- cambiar ahora a otro framework destruiria continuidad y retrasaria semanas el cierre del producto

Regla dura:

- se pueden estudiar y copiar patrones de frameworks de referencia
- no se sustituye el nucleo de Orquesta por otro framework
- el servidor de Orquesta sigue siendo el unico plano de control oficial

Patron elegido como referencia:

- arquitectura tipo supervisor/manager event-driven
- agentes tratados como actores o endpoints gobernados por un controlador central
- handoff con contexto filtrado y resumen util
- mensajes y ordenes duraderos con lease, ack, retry y backoff

Regla operativa para `server_autobootstrap`:

- el autobootstrap del servidor solo debe sembrar bootstrap inicial para agentes que todavia no estan operativos
- si un agente ya tiene sesion activa, handle activo o mailbox bootstrap durable pendiente para ese proyecto, el daemon no debe reenviarle otro bootstrap al reiniciar
- reiniciar el servidor no puede equivaler a “volver a arrancar” a todos los workers ni a reinyectar el mismo `esperar_o_pedir_tarea`
- una sesion activa del supervisor no debe recibir `supervisar_proyecto` pasivo en cada tick de autonomia; la supervision rica pertenece al batch de supervision y a las señales de transcript/review, no al pulso generico de sesion viva
- la supervision autonoma periodica tampoco debe reinyectar `supervisar_proyecto` si el supervisor ya sigue operativo en ese proyecto; el primer bootstrap de supervision es valido, pero los ciclos posteriores deben apoyarse en actividad viva y señales reales, no en repetir la misma guidance cada intervalo
- los `upsert` de politica/autonomia no pueden borrar timestamps operativos como `last_supervision_at` o `last_review_at`; reiniciar el daemon no debe reabrir un ciclo de supervision “por olvido” del estado persistido
- un agente en `estado_cuota=enfriamiento` con handle `pausado` no es candidato watchdog ni debe recibir `sync_status/watchdog`; el cooldown es ya una decision explicita del plano de control
- el watchdog de handoff no puede decidir agotamiento solo por `sesiones.heartbeat_at`; si el runtime activo sigue emitiendo `last_event_at` o `last_heartbeat_at` recientes, esa actividad invalida el stale de la sesion y no debe encolarse `sync_status/watchdog`
- por la misma razon, un proceso vivo con runtime activo reciente tampoco puede escalar directamente a `handoff`; antes de relevar a un agente, el plano de control debe distinguir entre “sesion sin latido” y “runtime realmente parado o degradado”
- la recuperacion de runtime local no puede decidir `local_runtime_failed` solo por estados persistidos viejos; antes de reencolar `start/resume`, el daemon debe observar el proceso local y revivir `handle/runtime` si el proceso sigue vivo
- la observabilidad de transcript no puede propagar el preambulo completo de `script(1)` ni comandos bootstrap gigantes dentro de un `runtime_panic`; si la linea mezcla `Script started on ... [COMMAND=...]` con el fallo real, Orquesta debe recortarla al texto util del panic antes de persistir transcript/evento
- la API de observabilidad tambien debe servir proyecciones compactas: `runtime-events` no puede devolver mensajes ni payloads gigantes por defecto; el raw vive en BD/trazas, pero la lectura operativa se trunca a tamaños razonables para dashboard, web y CLI
- la limpieza de transcript debe absorber tambien secuencias CSI privadas del TUI (por ejemplo `ESC[<1u`); no pueden sobrevivir en `runtime_panic` ni contaminar la clasificación o la observabilidad visible
- esa misma higiene aplica a eventos históricos leídos por API: aunque el raw viejo siga en BD, `/api/runtime-events` debe sanear ANSI/CSI residual antes de devolver `message` o `payload`
- si tras limpiar ANSI/CSI queda un prefijo puntual sin semántica (`.`, `:`, `|`) justo antes del marcador de `runtime_panic`, también debe retirarse de la proyección visible
- si un evento histórico heredado arrastra un prefijo corto sin espacios justo antes de `The application panicked (crashed).` (`s`, `[>7u`, etc.), también se considera ruido terminal y debe retirarse en la proyección visible
- un agente no puede figurar `activo ahora` solo porque conserve una sesión abierta o un `runtime_handle.estado='activo'`; si el `runtime principal` ya está terminal o su `última actividad` está stale, la proyección visible debe ocultarlo como agente ocupado
- la capa visible debe diferenciar `conectado` de `trabajando`: un runtime vivo en `esperando_io` no equivale a trabajo real. `status` y `/api/status` deben exponer agentes conectados y, aparte, agentes con tarea `en_progreso`
- la orquestacion profesional exige visibilidad de presupuesto por agente: `status`, `/api/status` y `/api/agentes` deben exponer cuanto queda (`remaining_credits`, segundos/tokens/mensajes cuando existan, y porcentaje restante). Si el proveedor no expone creditos, Orquesta debe caer al mejor porcentaje derivable y nunca dejar al orquestador ciego
- ese porcentaje no puede ser una cifra ciega de una sola ventana: Orquesta debe calcular el presupuesto efectivo mas restrictivo entre sesion, diario y semanal, indicar la `ventana` elegida y mostrar tambien `reset_at` cuando se conozca. Un `90%` diario no autoriza a ignorar un semanal casi agotado ni una ventana real de `5h`
- si el proveedor devuelve `usage limit`, `rate limit`, `purchase more credits` o `try again at`, el daemon debe persistir de inmediato un `presupuesto_sesion` canonico con `budget_source=provider_backoff`, `reset_at` observado y ventana efectiva agotada. Esa telemetria no puede quedar solo en `motivo_pausa` o en texto de stderr
- la identidad de cuenta del agente solo puede salir de artefactos canonicos observados por Orquesta. Si el proveedor incluye `email`, `login` o `perfil activo` en su salida o snapshot de presupuesto, el daemon debe promoverlo a claves canonicas (`account_email`, `account_user`) dentro de `raw_snapshot_json` o `metadata_json`; si no existe dato observado, el campo queda vacio y no se inventa
- la cuota e identidad de cuenta deben convivir con dos fuentes separadas y visibles: `viva` (server/control plane/runtime actual) y `observada` (artefactos persistidos de Codex CLI como `auth.json` o eventos `token_count` en sesiones JSONL). Orquesta puede usar la observada como respaldo o enriquecimiento, pero nunca debe presentarla como si fuese telemetria en caliente sin marcar `fuente` y `observed_at/checked_at`
- para Codex CLI, la telemetria observada canonica entra leyendo `auth.json` y los eventos `token_count` de `sessions/*.jsonl` desde el daemon. La ventana primaria `300m` se proyecta como `5h`; la secundaria `10080m` alimenta el presupuesto semanal observado. Esa ingesta no sustituye al control plane vivo, pero si evita quedarse ciego cuando el proveedor solo publica cuota en los artefactos locales
- la telemetria observada no puede secuestrar la cuota efectiva si esta vieja. El TTL general vive en `pool_budget_snapshot_max_age_seconds`, pero Codex CLI observada debe poder usar un TTL propio mas realista (`pool_budget_snapshot_observed_max_age_seconds`) porque `token_count` suele actualizar por turnos y no cada pocos segundos. Cuando un snapshot observado supera su TTL efectivo, Orquesta debe seguir mostrando `sesión/semanal` observadas para inspeccion, pero la `cuota efectiva` vuelve al derivado seguro y la vista debe marcar `telemetría observada stale`
- la higiene historica de runtime pertenece al control plane del daemon. Orquesta puede purgar automaticamente solo deuda terminal vieja (`runtime_instances` en estados terminales, `runtime_handles` en `cerrado/fallido` y `runtime_orders` en estados terminales) respetando ventanas de retencion configurables. Nunca debe tocar runtimes vivos, handles activos/pausados ni ordenes vivas para "limpiar" el estado
- cuando no exista correo observado pero el runtime viva detras de `codex-perfil`, Orquesta puede exponer el `account_user` observado desde el perfil real del `rendered_command` del handle. Eso sigue siendo observacion canonica del runtime; lo que no se permite es rellenar `account_email` por deduccion

Contrato minimo obligatorio para `runtime_orders`:

- toda orden reclamada por el control plane debe quedar con `lease` explicita
- la lease debe registrar quien la reclama, token de lease, intento y caducidad
- una orden reencolada o finalizada debe liberar siempre su lease
- el mismo principio aplica a los batches del runner: un batch en ejecucion no puede quedar secuestrando su nombre para siempre tras un timeout
- el runner debe modelar cada batch con lease y token de generacion; si la lease expira, el siguiente ciclo puede reclamar de nuevo ese batch sin depender de que el goroutine viejo termine
- el cierre de lease de batch debe ser por token, no por nombre a secas, para que un goroutine viejo no pueda borrar la lease vigente de una ejecucion mas nueva
- `send_instruction` no se gobierna por un unico booleano `can_send_input`; el daemon debe distinguir entre input interactivo generico y entrega caliente por supervisor local de Orquesta
- un runtime `process_pty_cli` local y seguro con `stdin_path` y `supervisor_ref` validos puede recibir entrega caliente por el canal del supervisor aunque `can_send_input=false`
- `session_resume` queda como fallback de continuidad, no como sustituto de la entrega caliente cuando Orquesta ya posee el proceso local y su canal de entrada
- excepcion dura: Codex PTY/TUI no admite `send_input` interactivo generico; `can_send_input=false` sigue siendo obligatorio
- matiz operativo obligatorio: un runtime Codex con supervisor local real (`process_pty_cli` + `stdin_path` + `supervisor_ref`) si puede aceptar entrega caliente por la via del supervisor de Orquesta; lo prohibido es el camino generico/interactivo, no la entrega supervisada y trazable
- para runtimes Codex degradados, `session_resume` no puede depender solo de la metadata persistida del `runtime_handle`; debe aceptar metadata observada por el supervisor local cuando esa observacion sea mas rica y coherente que la fila actual
- el reconocimiento de un runtime Codex local no puede atarse a un unico campo como `driver`; debe poder reconstruirse desde `herramienta`, `conector`, `rendered_command`, `wrapped_command`, `supervisor_driver` y la observacion viva del supervisor
- la reconciliacion stale no se decide solo por `started_at`; debe respetar la caducidad de la lease cuando exista
- una `lease_expires_at` ya vencida se reconcilia contra `now`, no contra otro `cutoff` adicional; la lease no puede caducar dos veces
- si un `mailbox_id` ya forma parte de una `bootstrap lease` pendiente (`lease_state=waiting_for_evidence`), ningun batch (`interactive`, `session_resume`, `coordinated_restart`) puede materializar otra `send_instruction` para ese mismo mensaje
- mientras una `bootstrap lease` siga pendiente, el mailbox incluido en ella sigue siendo verdad de continuidad; cualquier `send_instruction` redundante para ese `mailbox_id` debe cerrarse como `superseded`, no reintentarse
- una `send_instruction` nacida desde `runtime_mailbox` no es una segunda cola durable; es solo un intento de entrega
- si ese intento no encuentra runtime entregable inmediato, la orden se completa y la verdad queda en `runtime_mailbox`; no se permite bucle de reintentos sobre la misma orden derivada
- ademas, mientras sigan iguales el `mailbox_id`, el `handle_id` y la `external_session_id` efectiva, el daemon no puede volver a materializar otra `send_instruction` nueva para el mismo mensaje durable; el siguiente intento solo puede nacer tras un cambio real de handle, sesion o contrato de entrega
- para evitar congelar mailbox legacy tras una mejora real de contrato, cada `send_instruction` derivada desde mailbox debe persistir una `delivery_attempt_signature`
- una orden `mailbox_only` con la misma firma debe deduplicar; una orden legacy sin firma puede permitirse un unico reintento cuando el handle actual ya tiene un contrato de entrega mas rico
- `session_resume` no es un canal de latencia infinita: su timeout por defecto debe ser corto y operativo porque una llamada lenta bloquea `runtime_orders` aunque el daemon siga sano
- si una `send_instruction` derivada de mailbox autonomo (`autonomia`, `nudge`, `watchdog`, `governance_refresh`, `skills_refresh`) falla por `session_resume timeout`, el intento se cierra como `mailbox_only`; el mensaje durable sigue pendiente y no se reencola la misma orden en bucle
- la clasificacion de errores de proveedor en `session_resume` debe leer el error relevante aunque el CLI anteponga banner, perfil o cabeceras; si hay `usage limit` o `rate limit`, Orquesta debe traducirlo a `provider_backoff` y `pausa_por_cuota`
- un agente `pausado` o en `estado_cuota=enfriamiento` no cuenta como “activo ahora” en `status`, `/api/status` ni `/api/agentes`, aunque conserve sesion operativa o handle reciente
- el bloque `tareasActivas` de `status`, `/api/status` y sus clientes no debe mezclar backlog global sin proyecto con el estado del proyecto operativo; las tareas sin `proyecto_id` no se muestran ahi
- la autonomia no puede rematerializar `pause` si la pausa ya esta satisfecha: una orden `pause` pendiente, una `sesion` ya `pausada` o un `runtime_handle` ya `pausado` con el agente en `estado_cuota=enfriamiento` deben tratarse como pausa vigente y cerrar el ciclo sin nuevas ordenes
- si un runtime Codex supervisado emite `runtime_panic` o `runtime_crash` tras entrega caliente por `supervisor_local`, el handle debe degradarse de forma persistente para ese ciclo de vida:
  - `disable_supervisor_hot_input=true` en metadata del handle
  - las entregas futuras para ese handle ya no pueden usar `supervisor_local`
  - el fallback vuelve a `session_resume` o `mailbox` segun contrato
  - la degradacion es por handle, no por agente: un handle nuevo puede recuperar capacidades si nace sano
- la metadata viva de `runtime_handles` no puede arrastrar prompts crudos de arranque o continuidad:
  - `bootstrap_prompt` y `continuity_prompt` completos no deben persistirse en el handle
  - solo se admiten resúmenes compactos o metadatos de diagnóstico
  - el prompt completo puede existir en el arranque efectivo o en trazas de runtime, pero no como lastre permanente en la fila viva del handle
- la coalescencia de `runtime_mailbox` para guia operativa no va por `kind` exacto sino por familia de guidance: `autonomia`, `nudge`, `watchdog`, `governance_refresh` y `skills_refresh` se superseden entre si; `instruction` explicita queda fuera de esa familia y no se borra por una guia generica posterior
- la mailbox pendiente de agentes fuera de vida operativa no puede quedar como ruido eterno: si el destinatario no tiene `runtime_handle` activo, ni sesion activa, ni asignacion activa, ni tareas activas, ni `runtime_orders` abiertas, la deuda se consume como zombi reconciliado
- para esa regla, una asignacion `activa` mantenida solo por `reactivacion_automatica` no protege al agente si ya esta fuera de la flota `server_autobootstrap`; no puede retener mailbox zombie por si sola
- `watchdog` queda fuera de esa reconciliacion generica y conserva su ruta especifica de consumo/auditoria
- el ingestor de transcript no puede clasificar banners de `script(1)` (`Script started on ...`, `Script done on ...`) como `runtime_panic` o `runtime_crash`; esas lineas son sistema/observabilidad, no evidencia semantica de fallo del runtime
- el texto inyectado a runtimes Codex por PTY debe compactarse a instrucciones cortas y ASCII seguras; el bootstrap del supervisor no puede entrar como parrafo largo crudo porque rompe el TUI aguas abajo
- el runner de mailbox no se limita a `interactive`, `session_resume` y `coordinated_restart`: cuando exista un runtime supervisado por Orquesta con `stdin_path/supervisor_ref` reales, debe materializar `send_instruction` por un batch propio de `supervisor_local`
- ese batch no puede decidir con metadata pobre; antes de enrutar debe rehidratar el handle desde el estado observado del supervisor y solo despues evaluar si hay entrega caliente supervisada
- la `delivery_attempt_signature` de esos intentos debe registrar `supervisor_local|handle|session` para no mezclarlos con `bootstrap_only` ni con `session_resume`
- un `watchdog` no puede quedarse pendiente cuando el agente ya está conscientemente aparcado por cuota: si el agente está en `estado_cuota=enfriamiento` y su handle vigente está `pausado`, el runner debe consumir ese watchdog como deuda ya satisfecha, no intentar reanimarlo
- un `runtime_panic` o `runtime_crash` de Codex no puede ir seguido de guidance inmediata al mismo worker: primero se aplica una cuarentena corta con `reanimar_at`, se despierta al supervisor y solo al vencer esa cuarentena vuelve la reanimación automática normal
- esa cuarentena debe reutilizar el mecanismo oficial de `enfriamiento/reanimar_at` con un `motivo_pausa` explícito de `runtime_panic`, no otro subsistema paralelo
- una reanimacion automatica no se da por cerrada cuando vence el tiempo; solo cuando Orquesta ha encolado con exito su `resume` o `start`, o ha confirmado que ya existe una orden abierta equivalente
- por tanto, `reanimar_at`, `estado_cuota` y `motivo_pausa` no se limpian antes de la reactivacion efectiva; si la reactivacion falla, la deuda sigue visible para reintento en el siguiente ciclo
- `send_instruction` no usa la misma `lease` que `start`/`handoff`: su ventana debe ser corta y operativa para que un `session_resume` colgado no secuestre `runtime_orders` durante minutos. La `lease` general sigue siendo larga para ciclo de vida, pero la de `send_instruction` es especifica y mas corta
- si un `start` levanta runtime real con bootstrap inyectado pero sin `handoff/resume` pendiente, ese propio `start` debe hacer `ack` de los mailbox bootstrap incluidos; no se puede dejar esa continuidad pendiente por falta de lease formal

Contrato minimo obligatorio para el supervisor local:

- no basta con ver un PID vivo; el supervisor debe validar identidad del proceso observado
- si cambia el `cwd` o la firma de comando esperada, el proceso no puede seguir contandose como runtime valido
- si un `runtime_handle` activo solo conserva metadata pobre de sesion pero existe `runtime manifest` valido del mismo proceso, el supervisor debe rehidratar `stdin_path`, `log_path`, `rendered_command`, `supervisor_ref` y demas metadata operativa rica antes de reconciliar el handle
- esa rehidratacion no puede depender de SQLite directa ni de rescates manuales; debe ocurrir por el camino oficial del daemon, tipicamente durante `sync_status` o supervisión equivalente
- la emision de `SupervisorSignal` no puede tumbar el runtime local ni el teardown de tests: el handler debe ser panic-safe y el supervisor local debe degradar la señal a error recuperable si el plano de control ya no esta disponible
- un batch del runner no puede congelar el resto del control plane; cada batch necesita aislamiento y timeout propio
- la supervision autonoma periodica no puede resembrar el mismo `supervisar_proyecto` por reloj si ya emitio un nudge reciente para el mismo agente/proyecto; la deduplicacion debe aguantar estados transitorios del runtime y no depender solo de que el handle siga “operativo”
- la observabilidad server-first de handles tampoco puede devolver metadata fosilizada del supervisor local; al listar handles filtrados de un agente, Orquesta debe resincronizar primero el estado supervisado para que `supervision_mode`, `supervisor_owner_pid` y demas metadata operativa reflejen el daemon vivo actual

La referencia conceptual mas cercana para el nucleo es:

- supervisor central al estilo AutoGen Core
- handoff filtrado al estilo OpenAI Agents
- observabilidad y reconciliacion continua al estilo controller/operator

No se adopta:

- un supervisor puramente conversacional sin control real del proceso
- handoffs limitados a una sola run sin continuidad duradera
- una segunda capa de flujo que compita con el daemon de Orquesta

### Contrato objetivo del nucleo de orquestacion

El nucleo de orquestacion de agentes debe rehacerse, si hace falta, hasta dejar este contrato definitivo:

- `orquesta server` como unico supervisor oficial
- `runtime_orders` como cola duradera formal, no como tabla de conveniencia
- `runtime_mailbox` como mensajeria persistente tipada, no como mezcla ambigua de señales y entregas
- `runtime_handles` y `runtime_instances` como estado observado del runtime, no como autoridad de negocio
- un mensaje o una orden no termina hasta `ack` real por evidencia operativa

Semantica objetivo de `runtime_orders`:

- `pending`
- `leased`
- `running`
- `completed`
- `failed`
- `cancelled`
- `expired`

Atributos operativos obligatorios:

- `attempt`
- `lease_token`
- `lease_expires_at`
- `retry_at`
- `last_error`

Reglas duras:

- el worker del daemon reclama por lease, no solo por cambio ingenuo de estado
- si el worker cae, reinicia o pierde el lease, la orden vuelve a cola por reconciliacion
- toda entrega debe ser idempotente y reintentable
- todo retry debe tener backoff
- una orden en ejecucion no puede quedar colgada indefinidamente por ausencia de reconciliacion

Semantica objetivo de `runtime_mailbox`:

- solo contiene mensajes que tengan significado persistente para continuidad o entrega al agente
- las señales internas del sistema no deben quedar pendientes en mailbox como si fuesen entregas al agente si no existe una ruta oficial de entrega
- cada `kind` de mailbox debe estar clasificado de forma univoca:
  - entregable al agente
  - coalescible
  - supersedible
  - señal interna del sistema

Regla adicional:

- si un `kind` puede quedar `pendiente`, debe existir una ruta oficial de entrega o consumo
- si no existe esa ruta, ese `kind` no debe vivir como pendiente duradero en mailbox

### Mailbox persistente y acuse real

`runtime_mailbox` es una primitive deliberada del plano de control.
No es un apaño local ni una cola transitoria de conveniencia.

Reglas duras:

- la mailbox entre agentes, sesiones y handoff se mantiene como fuente persistente de continuidad y mensajeria
- `runtime_mailbox` no se sustituye por inyecciones efimeras en wrappers o terminales
- un mensaje no se considera consumido por el mero hecho de haber preparado una entrega
- el consumo o `ack` final solo ocurre cuando Orquesta confirma la entrega real por su camino oficial de control (`runtime_orders`, `send_instruction`, `resume`, handoff o bootstrap segun el caso)
- si una entrega falla o se reencola, el mensaje debe seguir trazable en Orquesta hasta su resolucion real
- los mensajes supersedidos pueden consumirse como tales, pero no se debe perder el mensaje vigente antes de su entrega efectiva

### Handoff y continuidad

Reglas duras:

- no handoff serio sin checkpoint o resumen de continuidad util
- el relevo debe dejar trazabilidad en Orquesta
- el destino no debe arrancar a ciegas

### Observabilidad

La observabilidad primaria debe salir de Orquesta:

- runtimes
- handles
- orders
- mailbox
- transcript
- checkpoints

No se debe depender de inspeccion manual lateral como via normal.

## Doctrina de desarrollo

### Arquitectura

Para modulos core y APIs de negocio:

- arquitectura hexagonal estricta
- puertos y adaptadores
- servicios por dominio
- persistencia mediada
- tests de contrato

No permitido:

- meter logica de negocio en handlers
- crecer `db/` como capa de aplicacion
- añadir nuevos atajos locales de negocio
- duplicar reglas entre web, CLI y daemon

### Nomenclatura

Los modulos de dominio y aplicacion deben usar nomenclatura en castellano, normalmente con sufijo `app`.

### Cliente fino

Web, CLI y futuras apps de escritorio deben ser clientes finos sobre Orquesta.

No permitido:

- acceso directo a BD desde web o desktop
- segundo plano de control en clientes
- duplicar reglas o politicas de negocio en la interfaz

### Mutaciones

Toda mutacion debe pasar por servicios o API de Orquesta y dejar auditoria.

No se debe escribir la BD a pelo desde herramientas externas ni desde nuevas rutas locales.

### Diario y trazabilidad

Toda decision tecnica relevante, cambio de estrategia o cierre de bloqueo serio debe quedar documentado en el diario operativo correspondiente.

## Doctrina de trabajo para agentes

Al empezar un frente:

1. leer esta biblia
2. consultar `orquesta status`
3. consultar tareas, propuestas y runtime del frente
4. trabajar sobre la tarea o frente real asignado
5. documentar supuestos y cambios relevantes

### Regla previa obligatoria antes de tocar codigo

No se modifica arquitectura, nucleo, contratos de control plane, persistencia, mailbox, handoff, runtime o semantica operativa sin estudiar antes:

- esta biblia
- `ARQUITECTURA.md`
- las politicas y OP del frente afectado
- el runbook o documentacion operativa relevante
- el estado vivo en Orquesta si el frente ya existe alli

Regla dura:

- no se toman decisiones de diseño por intuicion, recuerdo parcial o lectura incompleta del codigo
- si un comportamiento parece incorrecto pero viene de una decision documentada, primero se contrasta con la documentacion antes de cambiarlo
- si falta documentacion suficiente, se para, se aclara y se documenta antes de seguir programando

Si falta contexto:

- pedirlo a Orquesta
- consultar briefing
- consultar propuestas pendientes
- consultar checkpoints y transcript

No hacer:

- inventarse backlog paralelo
- trabajar desde documentos obsoletos ignorando el estado vivo
- escribir SQL directo como atajo normal
- reparar sintomas repetidos sin cuestionar la arquitectura

## Regla anti programacion ciclica

Si aparece el mismo sintoma varias veces y la solucion propuesta es solo:

- subir timeout
- meter otro retry
- reencolar otra vez
- reiniciar otra vez
- añadir otro script auxiliar

entonces hay que parar y revisar la frontera arquitectonica.

Regla practica:

- dos iteraciones sobre el mismo sintoma sin cerrar la causa estructural equivalen a programacion ciclica

La respuesta correcta en ese punto es:

- identificar la frontera equivocada
- simplificar el flujo
- mover el control al servidor o al adaptador correcto

## Doctrina especifica sobre SQLite y SQLITE_BUSY

SQLite es un backend soportado, no la arquitectura.

Por tanto:

- no se culpa al motor por un diseno hibrido
- no se acepta `SQLITE_BUSY` como normalidad operativa
- no se arregla solo con `busy_timeout`
- no se permite que varios caminos de negocio compitan por la misma base

La salida buena es:

- server-first real
- single-writer real
- retirada de caminos locales de negocio
- cierres estructurales de mutacion y observabilidad
- backend desacoplado para que SQLite no sea una dependencia conceptual

Hallazgo operativo consolidado:

- una fuente recurrente de `SQLITE_BUSY` en Orquesta era abrir SQLite y correr bootstrap/post-migraciones antes de confirmar que el daemon habia conseguido su puerto
- eso permitia arranques fantasma o duplicados que tocaban la BD y solo despues fallaban al escuchar
- la doctrina correcta es: primero reservar listener, despues abrir persistencia, despues publicar estado y servir
- otra brecha peligrosa era permitir que `ORQUESTA_FORCE_LOCAL_DB` abriera la BD en escritura para comandos no diagnosticos; la doctrina correcta es que el modo local de recuperacion sea de solo lectura y con whitelist explicita

## Comandos base de consulta viva

Comandos base que todo agente debe conocer:

```bash
./orquesta status
./orquesta tarea listar
./orquesta propuesta listar
./orquesta runtime listar
./orquesta runtime diagnostico --agente Codex1
./orquesta runtime ordenes --agente Codex1
./orquesta sesion inicio <agente>
```

Si estos caminos no bastan para una necesidad operativa real, hay que ampliar Orquesta.
No hay que normalizar el acceso lateral a persistencia.

## Prioridades estructurales vigentes

Mientras no se cierre explicitamente por Orquesta, los frentes estructurales dominantes son:

- consolidar server-first
- consolidar single-writer
- cerrar rutas locales de negocio residuales
- hacer que Orquesta gobierne de verdad los runtimes
- reducir dependencia operativa de scripts manuales
- mantener backend de persistencia desacoplado

## Criterio de disciplina para todos los agentes

Todo agente debe actuar como si este archivo fuese la constitucion del proyecto.

Eso implica:

- leerlo
- obedecerlo
- citarlo cuando haya ambiguedad
- no degradarlo con excepciones oportunistas

Cuando haya que cambiar esta doctrina, se cambia aqui primero y luego se alinean los demas documentos.

## Runtime handles vivos

- `GetRuntimeHandle` y `GetRuntimeHandleBySesionID` pueden reconciliar y persistir compactacion de metadata legacy si el handle concreto aun conserva prompts o comandos crudos.
- `ListarRuntimeHandles` y la API `/api/runtime-handles` no deben escribir sobre la BD para compactar toda la historia; en listado solo se compacta en memoria para mantener la vista operativa rapida.
- `runtime_handles.metadata_json` no debe conservar prompts crudos ni comandos gigantes con el bootstrap completo embebido. Para observabilidad basta con resúmenes y comandos compactados que preserven detección/reanudación (`codex-perfil`, agente, wrapper, log).

## Purga segura del control plane

- `runtime purgar-ordenes` existe para limpiar ruido de pruebas y deuda terminal, pero debe trabajar con corte temporal conservador.
- la purga de órdenes terminales no debe borrar actividad fresca; el camino oficial usa `older-than-minutes` y por defecto opera sobre órdenes con más de `60` minutos.
- la limpieza de ruido se hace por daemon y con estados terminales (`completada`, `fallida`, `expirada`, `cancelada`), nunca sobre órdenes vivas.

## Arranque del daemon

- el listener y la API deben quedar disponibles antes de disparar batches pesados del control plane.
- el `Runner` del servidor arranca con una `startup grace` para evitar que `status` o `/api/runtime-handles` queden bloqueados por trabajo interno nada más levantar el daemon.
- esta gracia pertenece al arranque del daemon en [servidor_unificado.go](/home/alberto/Trabajo/orquesta/cmd/servidor_unificado.go), no al constructor genérico del runner; tests y runners embebidos deben poder ejecutar batches inmediatamente.
- `serve` y `server run` no deben abrir la BD desde el gating genérico del CLI; la apertura pertenece al arranque del servidor una vez reservado el listener.
- los comandos server-first como `status` solo pueden caer a recuperación local cuando esta se ha pedido explícitamente; desactivar el cliente de servidor no convierte el comando en local por defecto.

## Bootstrap runtime

- `agente preparar` y el bootstrap runtime siguen siendo lectura y proyección, no una ruta para mutar órdenes o consumir mailbox.
- cuando exista guidance pendiente modelada como `runtime_order` de tipo `nudge` o `discordia`, el bootstrap debe proyectarla dentro del bundle como mailbox sintética para no perder contexto al reanudar o arrancar.
- esa proyección no debe inventar consumo ni `mailbox_id` persistente; el consumo real sigue ocurriendo solo cuando la orden/mailbox se ejecuta o se acusa por el camino oficial.

## Runtime orders stale

- una orden en `tomada/ejecutando` puede quedar stale por dos causas válidas: lease vencida o edad efectiva del intento.
- la reconciliación stale no puede depender solo de `lease_expires_at`; si `started_at/updated_at` demuestran que el intento es viejo, la orden debe recuperarse aunque la lease siga futura por metadata vieja o inconsistente.

## Review gates y políticas de modelo

- el contexto de un `review gate` debe poder enlazar el worktree activo registrado para la tarea aunque su path aún no haya pasado por validación de coherencia en disco; para gobernanza importa primero la relación registrada proyecto/tarea/worktree.
- en resolución de políticas de modelo, a igualdad de `scope`, `perfil` y `prioridad`, debe ganar la política más reciente. Esto evita que seeds antiguas tapen overrides explícitos guardados después.

## Supervisor local y observación de procesos

- `supervisor local` no es cualquier `PID` ni cualquier handle con `stdin_path`; solo aplica cuando existe identidad de runtime local real: `supervisor_ref`, `supervisor_driver`, `driver=process_pty_cli`, `trace_manifest`, `trace_dir` o un `runtime.json` detectable bajo `.orquesta-runtime`.
- `stdin_path` por sí solo no debe convertir un proceso en supervisor local. Ese dato sirve para entrega, no para observación de identidad.
- cuando un supervisor adjunto nace con `ref=pid:<pid>` y luego aparece una `supervisor_ref` canónica desde `runtime.json`, la identidad canónica debe promocionarse y refrescar los campos ricos (`trace_dir`, `stdin_path`, `log_path`, `working_dir`, `wrapped_command`, `rendered_command`).
- la observación local no debe contaminarse entre tests ni entre runtimes distintos que reutilicen el mismo `PID` del proceso padre.

## Infraestructura de tests de persistencia

- `prepararDBTemporal(...)` ya no representa una BD vacía; hoy siembra pools, modelos y políticas base codificadas por `EnsureCapacidadModeloBaseCodex()`.
- por tanto, los tests que ejercen `pools` no deben asumir `len(resumen)==1` salvo que filtren por el pool que están verificando.
- el comportamiento canónico es: la BD temporal nace con capacidad base operativa y los tests validan el elemento bajo prueba, no la ausencia de seeds.

## E2E estables

- los tests E2E que arrancan runtimes reales no deben depender de árboles de procesos ambiguos del shell.
- cuando un launcher de test solo necesita dejar un proceso vivo, debe hacer `exec` del proceso final para que el cleanup mate exactamente al runtime y no deje hijos residuales.
- los tests que esperan eventos visibles por API bajo carga de suite deben usar ventanas temporales realistas; si pasan en aislado y fallan solo por margen corto, se endurece el timeout del test, no se relaja la semántica del control plane.

## Persistencia SQLite activa

- mientras SQLite siga siendo backend soportado y backend vivo del servidor, el adaptador debe quedar sano por sí mismo; no vale asumir que el DSN ya aplicará siempre todos los pragmas correctos.
- `journal_mode=WAL` es parte del contrato operativo del backend SQLite server-first; la verificación viva de persistencia debe reflejar `wal`, no `delete`, cuando el daemon ha arrancado con el binario correcto.
- `persistencia verificar` debe marcar error si un SQLite writable sale de `WAL`; un estado `delete` o equivalente no es un backend sano aunque la conexión siga respondiendo.
- `runtime_mailbox.from_agente` representa un emisor lógico del control plane y no debe exigir FK a `agentes(nombre)`. El destinatario `to_agente` sí sigue siendo un agente real.
- emisores como `server` u `orquesta` son canónicos en mailbox/runtime orders; el schema no puede romper esa semántica.

## Arranque oficial del daemon

- `server run` sigue siendo la ejecución foreground del servidor.
- la vía oficial para dejar el daemon levantado desde Orquesta es `server start`.
- `server start` debe usar la misma ruta interna que emplea la CLI para levantar localrpc y, si el hijo muere antes de exponer `healthz`, debe devolver un error útil con resumen del log y limpiar estado falso.
- `server start` no debe dar el arranque por bueno solo porque responda `healthz`; el daemon tiene que haber publicado también su `statefile`.
- un `statefile` existente no es fuente de verdad suficiente por si solo. Antes de darlo por valido, Orquesta debe comprobar que el proceso anunciado sigue vivo por `healthz` y que ese `healthz` coincide con el storage/scope esperados.
- no se debe depender de `server run &`, `nohup` o wrappers externos como contrato operativo del orquestador.

## Supervisión autónoma periódica

- la supervisión periódica no debe reinyectar `supervisar_proyecto` solo porque haya vencido el reloj si el supervisor ya tiene trabajo activo real en ese proyecto.
- para este contrato, `trabajo activo real` significa al menos una tarea del supervisor en estado `asignada`, `en_progreso` o `bloqueada` dentro del proyecto.
- el batch periódico de supervisión existe para sembrar o recuperar frente útil, no para interrumpir a un supervisor que ya está ejecutando ese frente.

## Transcript y señales de runtime

- el clasificador de transcript no puede elevar a `runtime_panic` ruido de TUI/terminal que no representa un fallo semántico del runtime.
- las secuencias `OSC` de título de terminal (`ESC ] ... BEL/ST`) y ruido equivalente del spinner deben limpiarse antes de clasificar y, si una línea queda vacía tras esa limpieza, debe descartarse por completo.
- también deben eliminarse los caracteres de control residuales del PTY (`BEL`, `BS`, etc.) antes de decidir si una línea tiene valor semántico.
- fragmentos de un solo carácter o restos de repintado del TUI no cuentan como `runtime output` útil y no deben contaminar transcript ni servir como evidencia para `ack` operativo.
- `transcript_log_pending` tampoco puede conservar basura cruda del PTY. Si queda una línea parcial, se compacta a texto imprimible y acotado; si no tiene contenido semántico, se elimina.
- el banner de arranque de Codex (`Perfil activo`, `CODEX_HOME`, `Credenciales`, URL de releases, consejo de login, localhost del TUI) tampoco es trabajo del agente y debe descartarse del transcript operativo.
- enfriar un agente o avisar al supervisor por transcript solo es válido sobre evidencia semántica real, no sobre artefactos visuales del cliente TUI.

## Presupuesto visible e identidad de cuenta

- la cuota visible de un agente no puede resumirse a un único porcentaje ambiguo. Orquesta debe conservar y exponer, como mínimo, `sesión`, `diario`, `semanal` y la `ventana efectiva` que manda en ese momento.
- cada ventana visible debe incluir su propio `reset_at`; no vale mostrar solo el reset de la ventana ganadora si eso oculta un agotamiento semanal o de sesión.
- la `ventana efectiva` es la más restrictiva entre presupuesto de sesión real y derivadas diaria/semanal. Si faltan snapshots ricos, se puede derivar, pero debe quedar claro qué ventana manda.
- para `codex_token_count_observed`, si falta la clave de configuración específica, el TTL por defecto observado sigue siendo `3600s`; no puede caer silenciosamente al TTL genérico de `300s`, porque eso oculta agotamientos reales de la ventana `5h`.
- un presupuesto fresco y crítico observado debe proyectarse también sobre `estado_cuota` visible del agente (`enfriamiento` o `agotado`) aunque la fila persistida aún no haya sido actualizada por un batch posterior.
- la identidad de cuenta del agente (`usuario` / `correo`) debe salir solo de artefactos ya persistidos del runtime o del presupuesto (`raw_snapshot_json`, `metadata_json`). No se abre una segunda fuente de verdad ni se inventan credenciales.
- si Orquesta no observa identidad fiable, deja el campo vacío. El contrato es `mejor dato observado`, no adivinación.

## Liberación de tareas por cuota

- la redistribución automática no puede depender solo del campo persistido `agentes.estado_cuota`. Si un agente tiene presupuesto fresco observado en estado crítico, el planificador debe tratarlo como no disponible aunque la reconciliación persistente llegue unos segundos después.
- por la misma razón, `ListarAgentesPlanificables()` no puede filtrar solo por SQL sobre `estado_cuota='activo'`; debe validar el estado visible enriquecido del agente antes de devolverlo como candidato.
- la liberación automática por cuota solo aplica a tareas `asignada`. Las tareas `en_progreso` no se sueltan por heurística de presupuesto; requieren relevo o checkpoint explícito.
- los barridos del planificador no pueden abrir consultas adicionales mientras mantienen cursores vivos sobre SQLite con `MaxOpenConns=1`; primero se recopilan candidatos y después se enriquecen o validan.

## Mailbox en enfriamiento

- un agente en `estado_cuota=enfriamiento` no debe arrastrar `runtime_mailbox` pendiente que ya no sea entregable durante ese cooldown.
- `watchdog`, `governance_refresh` y `skills_refresh` deben consumirse como deuda satisfecha si el agente esta pausado o enfriado por una decision explicita del control plane.
- dejar ese mailbox pendiente durante el cooldown ensucia diagnostico, da falsa sensacion de trabajo vivo y reabre bucles de entrega sin valor.
- la reconciliacion correcta no es forzar entrega durante el enfriamiento, sino marcar esos mensajes como consumidos y dejar que el siguiente `start/resume` regenere el contexto canonico necesario.
