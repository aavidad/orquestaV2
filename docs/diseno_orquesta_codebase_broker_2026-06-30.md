# Diseno Orquesta Codebase Broker - 2026-06-30

Estado: segundo corte parcial implementado. Orquesta expone un broker central
por contrato neutral, tool MCP y endpoint HTTP; el proveedor activo por defecto
es `rg` central y `codebase-memory-mcp` queda reservado para adaptador opt-in
con lease central.

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
  tool/HTTP, gateway, proveedor `rg` con salida acotada y proteccion de
  `CODEX_HOME`.

## Pendientes

- Adaptador real `codebase-memory-mcp` detras del puerto neutral.
- Persistir leases/cache por repo y commit.
- Publicar estado compacto en supervisor/status.
- Cablear watchdog/TTL de herramientas auxiliares al servidor con parada
  cooperativa real y evidencia publica.
- Smoke opt-in con repo real acotado antes de habilitarlo en sesiones de agentes.
