# Pruebas: orquesta-app-planner

```bash
go test -count=1 ./modulos/orquesta-app-planner
```

Cobertura:

- plan Go API+web dividido en microtareas pequenas;
- plan Go API+web grande dividido en arquitectura, dominio, persistencia por
  puerto, API, web, i18n, deploy, integracion, docs y revision;
- write-set por unidad no vacio y sin rutas operacionales;
- primera ola solo contiene bootstrap;
- tras bootstrap, dominio y web salen en paralelo con `WorkClaims` compartido y
  claims locales por candidato;
- la proyeccion de progreso distingue tareas listas, bloqueadas, pendientes y
  completadas para continuar hasta terminar;
- unidades ya entregadas o arrancadas no se reemiten;
- contratos runtime por tarea conservan write-set, tests y criterios.
- resolutores de contrato, evidencias y contexto producen un
  `RuntimeLaunchRequestV0` y `AgentStartPacketV0` validos.
- la misma frontera produce `ExternalAgentLaunchSpecV0` valido para conectores
  externos inyectados.
- `AppSpecV0` validada por factory se transforma en plan sin crear una entrada
  paralela fuera de REST/MCP/web.
- `AppSpecV0` con persistencia escala a `large` sin introducir proveedor DB
  concreto en el write-set.
- la unidad API standard usa `cmd/server` como entrypoint Go.
- la unidad API large usa `internal/api` + `cmd/server`.
- `go test ./...` se conserva como test obligatorio de contrato runtime.
- las unidades de docs, integracion y revision salen con su `phase_id`
  especifico, no como programacion.
- cada unidad declara `work_profile_kind` neutral y puede convertirse a
  `WorkProfileV0`/`WorkflowTaskV0` sin reimplementar perfiles de programacion.
- las dependencias por `delivery_ref` del planner se convierten a `task_ref` al
  construir `WorkflowTaskV0`.
- el provider de candidatos reutiliza el resolver neutral de perfiles para rol,
  razon de capacidad y resumen de agente.

Validacion 2026-05-11:

```bash
go test -count=1 ./modulos/orquesta-app-planner
```

Resultado: `ok`.
