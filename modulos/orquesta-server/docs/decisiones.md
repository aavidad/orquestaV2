# Decisiones

## SRV-001: server-first residente

El trabajo real no debe depender de `go test`, de una sesion de Codex ni de una
terminal interactiva. Orquesta debe arrancar como servidor residente y aceptar
ordenes por API/MCP.

## SRV-002: statefile de reenganche

El servidor publica un statefile con PID, direccion HTTP, hora de arranque,
estado y ultima supervision. Al reconectar, el operador consulta ese fichero y
despues valida `healthz` o `/api/status`.

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

## SRV-007: autodiagnostico antes de supervisor

El servidor residente ejecuta un `StartupCheckPortV0` opcional antes de escuchar
HTTP y antes de arrancar el supervisor. Si el check no devuelve `ready`, el
servidor publica `startup_blocked` en el statefile y no empieza a drenar runs.

La purga concreta no vive en `orquesta-server`: una composicion puede decidir
si diagnostica, solicita parada logica, limpia procesos de su runtime o bloquea
el arranque. El servidor solo persiste estado, mensaje para el Director y refs
de evidencia compactas.
