# Diseno Orquesta Codebase Broker - 2026-06-30

Estado: cuarto corte parcial implementado. Orquesta expone un broker central
por contrato neutral, tool MCP y endpoint HTTP; el proveedor activo por defecto
es `rg` central y `codebase-memory-mcp` queda reservado para adaptador opt-in
con lease central. La composicion del servidor ya puede persistir cache/leases
en un directorio opt-in y dispone de un watchdog invocable por puerto de parada
cooperativa. El watchdog ya puede observar owner markers file-based y solicitar
SIGTERM solo al PID declarado por un marker valido de `codebase-memory-mcp`;
todavia no arranca el adaptador MCP real ni el loop residente opt-in.

## Decision

Orquesta debe gestionar Codebase de forma centralizada mediante un broker por
repo. Los agentes no llaman directamente a `codebase-memory-mcp`: piden a
Orquesta una consulta de codigo con repo, commit, scope y proposito. Orquesta
decide si usar indice Codebase, cache o `rg` local.

Esto mantiene Codebase como herramienta opt-in de arquitectura/callgraph, no
como dependencia normal de cada subagente. Para strings exactos, Markdown,
config, logs, incidencias y docs, el camino canonico sigue siendo `rg`/lectura
local acotada.

## Broker Por Repo

- Una instancia logica de broker por repo normalizado.
- Indexacion singleton por repo: solo un indexado activo por `repo_ref`.
- Lease visible para indexacion: owner, inicio, commit objetivo, TTL y estado.
- Si ya hay indexacion activa, nuevas peticiones se unen, esperan o degradan a
  fallback segun deadline; no arrancan otro indexador.
- El broker no vive en core puro: pertenece a servidor/composicion por puertos
  de tooling, telemetria y shutdown cooperativo.

## Consultas

Entrada minima implementada:

- `repository_ref`
- `commit_ref` o `worktree_ref`
- `query_kind`: `search`, `symbol`, `architecture`
- `query`
- `scope`: rutas permitidas, modulo o paquete
- `max_results` y `max_bytes`

Regla de enrutado:

- `symbol`, hotspots y arquitectura: Codebase via broker cuando exista el
  adaptador opt-in y el indice este listo o pueda prepararse dentro del
  deadline.
- `search`, docs, configs, errores, mensajes y rutas: `rg` central.
- Si Codebase falla, expira o no esta indexado, responder con fallback o bloqueo
  operativo explicito; no lanzar instancias paralelas desde agentes.

## Concurrencia Y Cache

- Limite de consultas concurrentes en el broker central.
- Cola con dedupe por `repo_ref + commit_ref + query_kind + query + scope`.
  Implementado como dedupe in-flight dentro del broker para consultas
  concurrentes iguales.
- Cache en memoria por `repo_ref + commit_ref + worktree_fingerprint +
  dirty_worktree + query + scope`; el resultado incluye fuente usada y estado
  de cache.
- El cache nunca oculta cambios del worktree: si no hay commit limpio se usa
  `worktree_fingerprint` o se marca resultado como `dirty_worktree`.
- Resultados compactos: refs, snippets pequenos y rutas; no volcar contexto
  bruto a subagentes.

## Watchdog Y TTL

Orquesta debe observar procesos auxiliares de Codebase/indexadores con TTL,
heartbeat y uso de CPU. Un proceso sin lease vigente, sin consultas activas o
con CPU sostenida sin progreso debe cerrarse cooperativamente o escalar alerta
operativa. Esto es supervision de tooling, no rail de contenido de agentes.

El contrato puro ya existe para registrar leases en memoria y evaluar si un
lease expirado sin peticiones activas debe pedir parada cooperativa. La parada
real y la publicacion en status pertenecen al servidor/composicion.

## Implementado

- Puerto neutral `CodeContextQueryPortV0` y broker `CodeContextBrokerV0` en
  `modulos/orquesta-context`.
- Tool MCP `orquesta.codebase.query.v0` y HTTP `POST /api/v0/codebase/query`
  en `modulos/orquesta-mcp`.
- Proyeccion publica `code_context_tooling_status.v0`, tool MCP
  `orquesta.codebase.status.v0` y HTTP `POST /api/v0/codebase/status` para
  listar leases activos/terminales, decisiones `request_stop`, CPU observado,
  acciones recomendadas y evidencias compactas sin PID, HOME ni command line.
- Ruta publicada por gateway, stack Codex y toolbelt de agentes.
- Proveedor `rg` central en `cmd/orquesta-server`.
- Cache en memoria, limite de concurrencia, dedupe in-flight y bloqueo de
  `codebase-memory-mcp` salvo opt-in central con lease.
- Cache distingue `worktree_fingerprint` y `dirty_worktree` para no ocultar
  cambios sin commit limpio.
- Contrato de leases de herramienta auxiliar: begin/finish en memoria y
  evaluacion TTL/CPU sin proceso real ni `kill`.
- Proveedor `rg` central conserva solo una salida acotada aunque lea todo el
  stdout del proceso, evitando acumular resultados gigantes antes de truncar.
- Proyeccion de `CODEX_HOME` sanea `codebase-memory-mcp` en `config.toml` salvo
  ledger opt-in y añade guarda en `AGENTS.md`.
- Tests offline de cache hit, dedupe concurrente, fingerprint de worktree,
  bloqueo de Codebase MCP sin opt-in, lease requerido, evaluador TTL/CPU,
  status publico de leases, tool/HTTP, gateway, proveedor `rg` con salida
  acotada y proteccion de `CODEX_HOME`.
- Persistencia file-based opt-in en `cmd/orquesta-server` para cache y leases
  mediante `ORQUESTA_CODEBASE_BROKER_STATE_DIR`.
- Watchdog invocable en composicion: observa leases activos, evalua TTL/CPU sin
  peticiones activas, llama a un puerto de parada cooperativa por `owner_ref` y
  marca el lease como `stopped` con evidencia.
- Owner marker file-based para herramientas de contexto: `owner_ref`, PID,
  heartbeat, CPU observado, peticiones activas y evidencias. El watchdog puede
  usar esa observacion para no parar un indexador con peticiones activas y el
  stopper concreto envia SIGTERM solo al PID de un marker valido de
  `codebase-memory-mcp`.

## Estado operativo y residuales

- El adaptador real `codebase-memory-mcp` queda detras del puerto neutral del
  broker central y no debe configurarse directamente en subagentes.
- El watchdog/TTL de herramientas auxiliares queda cableado como loop residente
  opt-in con evidencia publica de parada ejecutada/fallida.
- El smoke real sigue siendo opt-in y acotado; no habilita MCP directo por
  defecto en sesiones de agentes.
- Residual operativo: procesos nacidos fuera de Orquesta por sesiones antiguas
  solo se detectan como huerfanos hasta que esas sesiones usen owner marker o el
  broker central.
