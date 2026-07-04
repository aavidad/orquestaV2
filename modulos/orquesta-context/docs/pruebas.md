# Pruebas locales: orquesta-context

## CTX-P007 toolbelt MCP local para code context

Tipo: contract | adapter_boundary

Comando: `go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-runtime-codex-appserver ./cmd/orquesta-server -run Toolbelt`

Evidencia esperada:

- `orquesta.codebase.query.v0` aparece en el toolbelt MCP;
- la entrada declara `transport=mcp_local_sin_http_localhost`;
- la entrada declara `contract=CodeContextQueryPortV0`;
- la entrada usa `orquesta://contracts/codebase-query/v0`;
- la entrada MCP no contiene `POST`, `/api/v0/codebase/query`, `http://`,
  `https://` ni `127.0.0.1`;
- los hints del toolbelt no publican HOME, tokens, secretos, prompts completos
  ni transcripts.

Ultima ejecucion: pendiente de este cierre.

## CTX-P008 consultas estructuradas del broker de codigo

Tipo: unit_contract | adapter_boundary

Comando: `go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-mcp ./modulos/orquesta-runtime-codex-goal ./cmd/orquesta-server -run 'Test(ServerRGCodeContextProviderV0|MCPCodebaseQuery|BuildServerAppHandlerV0CodebaseQuery)|Test.*CodeContext.*|TestBuildCodexGoalStartPacketV0.*Analizador'`

Evidencia esperada:

- `callers`, `imports`, `module_exports` y `relevant_snippets` son aceptados por
  `CodeContextQueryPortV0`;
- MCP expone los nuevos valores en el enum de `query_kind`;
- el servidor devuelve hits compactos usando `go/parser` sin arrancar
  `codebase-memory-mcp`;
- el prompt Goal exige consultar `orquesta.codebase.query.v0` antes de leer
  ficheros completos cuando el write-set es de codigo;
- la composicion Codex precarga `repo_map` antes de lanzar goals con write-set
  de codigo y propaga `code_context_prepared:*`;
- la regla no se inyecta en write-sets puramente documentales.

Ultima ejecucion: 2026-07-04, ok en TAREA-2 primer parche y auto-prepare.

## CTX-P006 repo_map compacto del broker de codigo

Tipo: unit_contract | adapter_boundary

Comando: `go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-mcp ./cmd/orquesta-server -run 'Test.*CodeContext.*RepoMap|Test.*CodebaseQuery'`

Evidencia esperada:

- `repo_map` es aceptado por `CodeContextQueryPortV0`;
- cache central, dedupe in-flight y fingerprint/dirty-worktree funcionan igual
  que en `search`;
- el proveedor central del servidor devuelve rutas, tipos/funciones y snippets
  minimos con `rg`;
- si `rg` falta, el fallback Go local produce refs compactas sin indexador;
- MCP expone `repo_map` en el enum de `query_kind`.
  Los modos estructurados quedan cubiertos en CTX-P008.

Ultima ejecucion: 2026-07-03, ok en T278 con:
`go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-mcp ./cmd/orquesta-server -run 'Test.*CodeContext.*RepoMap|Test.*CodebaseQuery'`.

## CTX-P005 broker de codigo central y leases de herramienta

Tipo: unit_contract

Comando: `go test -count=1 ./modulos/orquesta-context`

Evidencia esperada:

- consultas repetidas usan cache central;
- consultas concurrentes iguales se deduplican con un solo proveedor real;
- la cache distingue `worktree_fingerprint` y `dirty_worktree`;
- `codebase-memory-mcp` queda bloqueado sin opt-in y sin lease central;
- con lease central, el broker registra inicio y cierre;
- un lease expirado sin peticiones activas recomienda parada cooperativa;
- leases terminales o con peticiones activas no piden parada;
- completion `stopped` queda aceptado para watchdogs de composicion.

Ultima ejecucion: 2026-06-30, ok.

Riesgos: el test no arranca `codebase-memory-mcp` real ni mata procesos; eso
queda para adaptador opt-in y smoke de servidor.

## CTX-P004 contexto required ref_only en T15

Tipo: contract | regression

Comando: `go test -count=1 ./modulos/orquesta-rails ./modulos/orquesta-core-workflow ./modulos/orquesta-context ./modulos/orquesta-director-agent ./cmd/orquesta-server`

Evidencia esperada:

- cada entrada required `ref_only` conserva razon y accion requerida;
- ACK completed incluye evidencia explicita cuando `required_ref_action` es
  `ack_evidence_required`;
- refs opacas y vocabulario operativo no se tratan como dato sensible;
- vocabulario operativo o local no bloquea el bundle por strings sueltos;
- valores sensibles efectivos o material crudo siguen bloqueando/saneandose;
- sanitizacion de metadata no persiste valores sensibles efectivos en refs de
  evidencia.

Ultima ejecucion documentada: 2026-05-27, ACKs cerrados de T15 con suite
requerida pasada.

Riesgos: la prueba no habilita nuevos scopes de detalle; solo valida el cierre
documental y el contrato de contexto.

## CTX-P001 builder de contexto pequeno

Tipo: unit_contract

Comando: `go test -count=1 ./modulos/orquesta-context`

Evidencia esperada:

- programacion incluye reglas comunes, docs locales minimos, `docs/pruebas.md`, read-set, write-set y contratos;
- brainstorming incluye `docs/decisiones.md`;
- refs cruzadas quedan en `contract_context`;
- si falta contexto externo, la politica obliga a `CONSULTA AL DIRECTOR`;
- vocabulario HOME/OAuth/proveedor/motor/secreto no reactiva veto automatico
  por palabra suelta;
- exceso de entradas falla con error publico.

Ultima ejecucion: 2026-05-05, ejecutada correctamente.

Riesgos: no prueba lectura real de archivos ni arranque runtime; solo fija el contrato puro del manifiesto.

## CTX-P002 materializacion explicita por filesystem

Tipo: unit_contract

Comando: `go test -count=1 ./modulos/orquesta-context`

Evidencia esperada:

- `FileContextRefStoreV0` exige root explicito;
- bloquea parent traversal;
- reporta ref no encontrada;
- materializa `doc_ref` y `read_ref` como contenido;
- mantiene `write_ref`, `contract_ref` y `evidence_ref` como `ref_only`;
- clasifica cada `required ref_only` con razon y accion esperada;
- trunca entradas grandes;
- mantiene `total_bytes` por debajo de `max_total_bytes`;
- propaga la pista `CONSULTA_AL_DIRECTOR`.

Ultima ejecucion: 2026-05-05, ejecutada correctamente.

Riesgos: usa filesystem temporal de test; no genera prompt final ni arranca agente real.

## CTX-P003 saneamiento local de contexto materializado

Tipo: unit_contract

Comando: `go test -count=1 ./modulos/orquesta-context`

Evidencia esperada:

- `MaterializeContextBundleWithSanitizerV0` permite inyectar
  `ContextSanitizerPortV0`;
- conserva `ContextSanitizationEvidenceV0` sin persistir token, secreto ni HOME;
- si el sanitizador devuelve `review_required`, la entrada queda como `ref_only`
  con accion `ask_director` y el bundle exige `CONSULTA_AL_DIRECTOR`.

Riesgos: el sanitizador real es adaptador de composicion; este modulo solo fija
el puerto y el comportamiento neutral del bundle.
