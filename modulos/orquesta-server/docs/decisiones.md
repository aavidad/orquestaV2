# Decisiones

## SRV-001: server-first residente

El trabajo real no debe depender de `go test`, de una sesion de Codex ni de una
terminal interactiva. Orquesta debe arrancar como servidor residente y aceptar
ordenes por API/MCP.

## SRV-002: statefile de reenganche

El servidor publica un statefile con PID, direccion HTTP, hora de arranque,
estado y ultima supervision. Al reconectar, el operador consulta ese fichero y
despues valida `/api/v0/server/readiness` o `/api/v0/server/status`; `/api/status`
queda solo como alias legacy.

## SRV-003: supervisor por puerto

El daemon no contiene logica del nucleo de orquestacion. Solo llama a un puerto
`RunGlobalSupervisorV0` con una orden acotada. El stack concreto decide que
runs drenar.

## SRV-004: estado operativo durable por conectores

El statefile del proceso no basta para autoprogramacion larga. Runs, eventos,
outbox, tareas, cambios solicitados, cola, control y registro de procesos deben
entrar por puertos persistentes reemplazables.

La implementacion local inicial puede ser file-based con JSON atomico. No se
autoriza acoplar el servidor a SQLite, Postgres ni otra base concreta. Si en el
futuro se usa una base de datos, sera otro conector con los mismos contratos.

## SRV-005: umbrales reales para agentes frontera

El servidor productivo no debe marcar como perdido a un agente de razonamiento
alto solo porque pase unos minutos pensando sin tocar ficheros. En la prueba
real OPES -> Orquesta -> Codex `gpt-5.5 xhigh`, el agente entrego resultado
valido despues de varios minutos, pero los defaults antiguos del comando eran
8 ticks de progreso con intervalo de 2s.

La composicion productiva usa ahora defaults conservadores: 300 ticks para
`stalled`, 300 ticks para posible bucle, 10 minutos sin actividad y 20 minutos
como presupuesto esperado. El nucleo no conoce esos numeros; siguen entrando
por configuracion y pueden ajustarse por entorno.

## SRV-006: backend file opt-in para domain_work generico

`cmd/orquesta-server` puede inyectar `orquesta-domain-work-file` como backend
de `DomainWorkJobCreatorPortV0` cuando se configura
`ORQUESTA_DOMAIN_WORK_FILE_ENABLED=1` o `ORQUESTA_DOMAIN_WORK_FILE_DIR`.

Esto cablea `/api/v0/domain-work` para `create_job` sin depender de OPES y sin
importar filesystem desde paquetes puros. No habilita `submit_artifact` ni el
bridge de entrega, porque el adaptador file solo crea jobs. Si tambien existe
`ORQUESTA_OPES_BASE_URL`, el servidor falla por backend ambiguo.

## SRV-008: backend HTTP neutral opt-in para domain_work generico

`cmd/orquesta-server` puede inyectar `orquesta-domain-work-http` cuando se
configura `ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL`. El adaptador envia
`DomainWorkJobRequestV0` y `DomainWorkArtifactSubmissionV0` como JSON neutral a
la app externa propietaria y espera `DomainWorkJobV0`/`DomainWorkArtifactReceiptV0`.

Esto habilita `create_job`, `submit_artifact` y `DomainDelivery` sin OPES. El
backend HTTP es excluyente con OPES y con el backend file para evitar mezclar
propietarios de jobs. La app externa conserva persistencia, validacion y
ensamblado; Orquesta solo conserva refs opacas y evidencia causal.

## SRV-007: autodiagnostico antes de supervisor

El servidor residente ejecuta un `StartupCheckPortV0` opcional antes de escuchar
HTTP y antes de arrancar el supervisor. Si el check no devuelve `ready`, el
servidor publica `startup_blocked` en el statefile y no empieza a drenar runs.

La purga concreta no vive en `orquesta-server`: una composicion puede decidir
si diagnostica, solicita parada logica, limpia procesos de su runtime o bloquea
el arranque. El servidor solo persiste estado, mensaje para el Director y refs
de evidencia compactas.

## SRV-009: el supervisor residente no espera agentes largos en linea

El servidor residente debe lanzar u observar trabajo y volver pronto al bucle
global. La manga ancha de los agentes se gestiona por runtime, heartbeats,
progreso, leases y reentradas posteriores, no dejando bloqueado el tick del
supervisor durante minutos.

Por defecto `ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS` usa una espera externa
en linea. Esto permite materializar el arranque y una observacion inicial sin
impedir que otros runs entren en ticks posteriores. El modo residente acepta
subir ese margen hasta el maximo publicado de `70` cuando la tanda real lo
necesita; valores por encima siguen bloqueados para detectar configuraciones
descontroladas en vez de dejarlas ambiguas.

El pulso residente se ejecuta de forma asincrona con una guarda atomica de
actividad: un segundo tick no se solapa con el primero, pero la siguiente
iteracion puede avanzar en cuanto el pulso anterior libera el slot. La
preparacion de automejora se lanza aparte para no mezclar trabajo secundario con
el tick principal.

## SRV-010: stats T210 son contrato de transporte

El servidor no recalcula la semantica de progreso vivo ni cierre. Transporta la
proyeccion de `DirectorRunStatsV0` y conserva `include_agent_progress` como
opcion publica. Si una senal de progreso llega degradada, se publica con reason
code; no se rellena con 0% silencioso ni se convierte entrega en task cerrada.

## SRV-011: T208 reconciliado por resultado estructurado

El servidor no debe inferir exito del guardian por exit code ni tratar cualquier
fallo break-glass como cierre de promocion. La frontera vigente consume
`orquesta_guardian_result.v0`, reason codes y evidence refs compactas; solo
`candidate_promoted` cierra el efecto cuando la promocion real esta habilitada.

T208 queda como umbrella historico porque los huecos concretos se cerraron en
owners focales. Nuevos blockers deben entrar como tareas separadas, manteniendo
guardian, Codex, filesystem, proveedor y configuracion operacional fuera del
modulo residente puro.

## SRV-012: Director residente opt-in por puerto

El servidor puede alojar un pulso residente del Director, pero no debe construir
briefings ni conocer Codex, OPES, MCP, modelos, HOME o persistencia concreta.
Por eso el contrato es `ResidentDirectorPortV0` y se activa solo con
`ConfigV0.ResidentDirectorEnabled=true`.

El pulso usa el mismo patron operativo que el supervisor: tick inicial, ticker,
wakeup no bloqueante por progreso durable, guarda atomica de anti-solape,
coalescing de un pulso pendiente, panic convertido en error durable y estado
publico `resident_director_*`. El self-watchdog trata actividad y progreso del
Director residente como causa operativa observable.

Composicion cerrada: `cmd/orquesta-server` inyecta un adaptador real opt-in que
usa `RunResidentDirectorBriefingLoopV0` con fuente de briefing reentrable desde
stores vivos. El modulo `orquesta-server` sigue puro: no construye briefings ni
conoce Codex, OPES, MCP, modelos, HOME ni persistencia concreta.
