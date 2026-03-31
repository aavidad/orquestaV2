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
- la autonomia no puede rematerializar `pause` si la pausa ya esta satisfecha: una orden `pause` pendiente, una `sesion` ya `pausada` o un `runtime_handle` ya `pausado` con el agente en `estado_cuota=enfriamiento` deben tratarse como pausa vigente y cerrar el ciclo sin nuevas ordenes
- la coalescencia de `runtime_mailbox` para guia operativa no va por `kind` exacto sino por familia de guidance: `autonomia`, `nudge`, `watchdog`, `governance_refresh` y `skills_refresh` se superseden entre si; `instruction` explicita queda fuera de esa familia y no se borra por una guia generica posterior
- el runner de mailbox no se limita a `interactive`, `session_resume` y `coordinated_restart`: cuando exista un runtime supervisado por Orquesta con `stdin_path/supervisor_ref` reales, debe materializar `send_instruction` por un batch propio de `supervisor_local`
- ese batch no puede decidir con metadata pobre; antes de enrutar debe rehidratar el handle desde el estado observado del supervisor y solo despues evaluar si hay entrega caliente supervisada
- la `delivery_attempt_signature` de esos intentos debe registrar `supervisor_local|handle|session` para no mezclarlos con `bootstrap_only` ni con `session_resume`
- un `watchdog` no puede quedarse pendiente cuando el agente ya está conscientemente aparcado por cuota: si el agente está en `estado_cuota=enfriamiento` y su handle vigente está `pausado`, el runner debe consumir ese watchdog como deuda ya satisfecha, no intentar reanimarlo
- un `runtime_panic` o `runtime_crash` de Codex no puede ir seguido de guidance inmediata al mismo worker: primero se aplica una cuarentena corta con `reanimar_at`, se despierta al supervisor y solo al vencer esa cuarentena vuelve la reanimación automática normal
- esa cuarentena debe reutilizar el mecanismo oficial de `enfriamiento/reanimar_at` con un `motivo_pausa` explícito de `runtime_panic`, no otro subsistema paralelo
- `send_instruction` no usa la misma `lease` que `start`/`handoff`: su ventana debe ser corta y operativa para que un `session_resume` colgado no secuestre `runtime_orders` durante minutos. La `lease` general sigue siendo larga para ciclo de vida, pero la de `send_instruction` es especifica y mas corta
- si un `start` levanta runtime real con bootstrap inyectado pero sin `handoff/resume` pendiente, ese propio `start` debe hacer `ack` de los mailbox bootstrap incluidos; no se puede dejar esa continuidad pendiente por falta de lease formal

Contrato minimo obligatorio para el supervisor local:

- no basta con ver un PID vivo; el supervisor debe validar identidad del proceso observado
- si cambia el `cwd` o la firma de comando esperada, el proceso no puede seguir contandose como runtime valido
- si un `runtime_handle` activo solo conserva metadata pobre de sesion pero existe `runtime manifest` valido del mismo proceso, el supervisor debe rehidratar `stdin_path`, `log_path`, `rendered_command`, `supervisor_ref` y demas metadata operativa rica antes de reconciliar el handle
- esa rehidratacion no puede depender de SQLite directa ni de rescates manuales; debe ocurrir por el camino oficial del daemon, tipicamente durante `sync_status` o supervisión equivalente
- un batch del runner no puede congelar el resto del control plane; cada batch necesita aislamiento y timeout propio

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
