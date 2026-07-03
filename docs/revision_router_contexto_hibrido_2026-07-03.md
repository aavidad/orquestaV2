# Revision router contexto hibrido

Fecha: 2026-07-03.

Entrada esperada por T-PER-801:
`docs/diseno_router_contexto_hibrido_2026-07-03.md`.

Resultado de busqueda local/remota: el fichero no existe en la rama local ni en
`/srv/orquesta-self/worktrees/orquesta` del servidor remoto. Esta revision cierra
la decision con la evidencia disponible y deja backlog ejecutable sin crear una
nueva fuente de verdad.

## Decision

No implementar un "router de contexto hibrido" como componente nuevo e
independiente.

Implementar solo un recorte sobre el broker central existente:

- `CodeContextQueryPortV0` sigue siendo la unica entrada canonica para contexto
  de codigo de agentes.
- El broker puede ganar estrategias internas (`search`, `symbol`,
  `architecture`, futuro `repo_map`, futuro `compressed_summary`), pero no otro
  store, API o daemon paralelo.
- Los prompts de agentes deben seguir con salida compacta/caveman, refs
  compactas, cache y consultas acotadas por bytes.
- `codebase-memory-mcp` sigue opt-in detras del broker central y con lease/
  watchdog; nunca por subagente.

Motivo: el problema real es consumo de tokens/contexto por agente, pero Orquesta
ya tiene la pieza arquitectonica correcta para gobernarlo: broker central con
cache, dedupe, fingerprint de worktree, fallback `rg`, leases y watchdog.
Duplicarlo como "router" reabriria justo el patron de varias fuentes de verdad
que Claude marco como causa de retraso.

## Estado actual comparado

Ya existe:

- `modulos/orquesta-context.CodeContextQueryPortV0`.
- `CodeContextBrokerV0` con cache, dedupe in-flight, limite de concurrencia y
  cache sensible a `worktree_fingerprint`/`dirty_worktree`.
- MCP/HTTP `orquesta.codebase.query.v0` y `/api/v0/codebase/query`.
- `code_context_tooling_status.v0`, leases y watchdog opt-in para
  `codebase-memory-mcp`.
- prompt hints del servidor que ordenan usar Orquesta como broker central y no
  arrancar indexadores por agente.
- reglas de salida compacta/caveman en `AGENTS.md`, `ARQUITECTURA.md` y
  prompts Codex/Claude/Gemini.

Brecha real:

- No hay medicion estable por goal de tokens de contexto evitados.
- No hay estrategia tipo repo-map dentro del broker.
- No hay selector formal de estrategia por tipo de tarea; hoy se decide por
  `query_kind` simple y por instrucciones.
- No hay evaluacion de compresores de prompt para contenido no-codigo.
- No hay aprovechamiento explicito de prompt caching por estructura estable de
  prompt en el app-server.

## Alternativas revisadas

### LLMLingua / LongLLMLingua

Fuente primaria: https://github.com/microsoft/LLMLingua

LLMLingua comprime prompts con modelos pequenos y declara hasta 20x de
compresion con perdida minima; LongLLMLingua se centra en contexto largo y
mejora RAG con menos tokens. Es maduro como investigacion/herramienta Python,
pero no debe ser dependencia base de Orquesta:

- integra peor en Go puro;
- introduce modelos auxiliares y coste operacional;
- puede degradar instrucciones legales/arquitectonicas si se aplica a reglas;
- encaja mejor como adaptador experimental para resumen de docs o OPES, nunca
  para comprimir contratos duros, AGENTS.md o write-set.

Decision: no implementar ahora. Dejar como opt-in futuro medido por benchmark.

### Repo map tipo Aider

Fuente primaria: https://aider.chat/docs/repomap.html

Aider usa un mapa conciso del repo con clases, funciones, firmas y lineas clave
para que el LLM entienda relacion entre piezas. Tambien selecciona las porciones
mas relevantes segun presupuesto de tokens.

Esto encaja bien con Orquesta, pero debe vivir como estrategia interna del
broker central, no como herramienta suelta:

- `query_kind=repo_map` o `architecture` puede devolver simbolos/rutas/snippets
  compactos;
- puede usar proveedor `rg`/Go fallback primero y tree-sitter/codebase despues;
- el resultado debe respetar `max_results`, `max_bytes`, cache y fingerprint.

Decision: implementar primero un repo-map barato y determinista dentro del
broker, sin dependencia externa obligatoria.

### Tree-sitter

Fuente primaria: https://tree-sitter.github.io/tree-sitter/

Tree-sitter es parser incremental y puede extraer estructura de codigo con
soporte multi-lenguaje. Tiene bindings Go oficiales/terceros y es buena base
para repo-map estructural.

Riesgo: meter parsers/lenguajes como dependencia directa puede complicar build y
portabilidad. Para Orquesta conviene fasear:

1. repo-map barato con `rg`/parser Go simple para funciones/tipos;
2. adapter tree-sitter opt-in si el valor medido supera el coste.

Decision: no meter tree-sitter en el primer corte.

### Prompt caching de proveedor

Fuente primaria: https://developers.openai.com/api/docs/guides/prompt-caching

OpenAI documenta que prompt caching funciona automaticamente en prompts largos
con prefijos identicos, reduce coste/latencia, y exige poner contenido estatico
al principio y variable al final para maximizar cache hits.

Esto es compatible con Codex/app-server sin crear componente nuevo:

- estabilizar prefijo: AGENTS, reglas, toolbelt, politica compacta;
- mover variable: objetivo, write-set, diff, estado vivo y refs al final;
- medir si el proveedor expone cached input tokens en usage/resultados.

Decision: aplicar como regla de construccion de prompt, no como router.

## Recomendacion final

Recortar T-PER-801 a tres entregas pequenas:

1. metrica de contexto por goal;
2. estrategia `repo_map` compacta dentro del broker;
3. orden canonico de prompt para cache de proveedor.

No tocar `codebase-memory-mcp` directo ni abrir un segundo endpoint de contexto.
Todo debe pasar por `CodeContextQueryPortV0` y publicar evidencia compacta.

## Backlog ejecutable

## CTX-TASK-801A: medir presupuesto de contexto por goal

Objetivo: registrar por goal/app-server los bytes/tokens aproximados de prompt
estatico, contexto consultado, contexto materializado y resultado de cache si el
proveedor lo expone.

Estado: pendiente.

Alcance:
- `modulos/orquesta-goal`
- `modulos/orquesta-runtime-codex-goal`
- `cmd/orquesta-server`
- `modulos/orquesta-mcp`
- `docs/inventario_bugs_orquesta_2026-06-30.md`

Criterios:
- no crea store nuevo;
- cuelga la metrica de `GoalWorkStateV0`, estado vivo o diagnostico MCP
  existente;
- no publica prompt completo ni secretos;
- muestra `context_budget_total_bytes`, `static_prompt_bytes`,
  `dynamic_context_bytes`, `code_context_cache_status` si aplica.

Tests:
- `go test -count=1 ./modulos/orquesta-goal ./modulos/orquesta-runtime-codex-goal ./modulos/orquesta-mcp ./cmd/orquesta-server`
- `git diff --check`

Dependencias:
- ninguna.

## CTX-TASK-801B: repo-map compacto como estrategia del broker

Objetivo: anadir `query_kind=repo_map` al broker central para devolver rutas,
tipos/funciones y snippets minimos dentro de `max_results/max_bytes`.

Estado: pendiente.

Alcance:
- `modulos/orquesta-context`
- `modulos/orquesta-mcp`
- `cmd/orquesta-server`
- `modulos/orquesta-context/docs/contratos.md`
- `docs/diseno_orquesta_codebase_broker_2026-06-30.md`

Criterios:
- `CodeContextQueryPortV0` sigue siendo la unica entrada;
- usa proveedor central existente; primer corte puede ser `rg`/Go fallback;
- cache/dedupe/fingerprint funcionan igual que en `search`;
- no arranca tree-sitter ni codebase-memory-mcp directo por subagente.

Tests:
- `go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-mcp ./cmd/orquesta-server -run 'Test.*CodeContext.*RepoMap|Test.*CodebaseQuery'`
- `go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-mcp ./cmd/orquesta-server`
- `git diff --check`

Dependencias:
- CTX-TASK-801A recomendable para medir impacto, no bloqueante.

## CTX-TASK-801C: prompt estable para cache del proveedor

Objetivo: separar en el app-server/Codex Goal el prefijo estatico cacheable
de la parte dinamica del trabajo, conservando exactamente las mismas reglas.

Estado: pendiente.

Alcance:
- `cmd/orquesta-server`
- `modulos/orquesta-runtime-codex-goal`
- `modulos/orquesta-runtime-codex`
- `docs/orquesta_goal_first_codex_2026-06-25.md`

Criterios:
- reglas estables antes; objetivo, write-set, estado vivo y refs al final;
- tests demuestran que el prompt conserva AGENTS/toolbelt/reglas hard;
- si hay `prompt_cache_key` o metrica equivalente disponible, se proyecta sin
  acoplar core a OpenAI.

Tests:
- `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex-goal ./modulos/orquesta-runtime-codex`
- `git diff --check`

Dependencias:
- ninguna.

## CTX-TASK-801D: evaluacion opt-in de compresion LLMLingua

Objetivo: hacer benchmark documental opt-in de compresion de contexto no
critico, sin comprimir reglas hard, write-set, AGENTS ni contratos.

Estado: pendiente.

Alcance:
- `scripts/`
- `docs/runbooks/`
- `modulos/orquesta-context/docs/decisiones.md`

Criterios:
- no anade dependencia runtime obligatoria;
- solo scripts opt-in;
- compara tokens/bytes, perdida de refs y tiempo;
- decision posterior antes de meter adaptador productivo.

Tests:
- `bash -n scripts/<script_nuevo>.sh`
- test shell focal si se crea;
- `git diff --check`

Dependencias:
- CTX-TASK-801A.

## Fuentes externas

- Microsoft LLMLingua: https://github.com/microsoft/LLMLingua
- Aider Repository Map: https://aider.chat/docs/repomap.html
- Tree-sitter: https://tree-sitter.github.io/tree-sitter/
- OpenAI Prompt Caching: https://developers.openai.com/api/docs/guides/prompt-caching
