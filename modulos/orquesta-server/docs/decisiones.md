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
