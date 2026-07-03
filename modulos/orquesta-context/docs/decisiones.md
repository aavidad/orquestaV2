# Decisiones locales: orquesta-context

## CTX-D006 - Benchmark opt-in de compresion documental

Fecha: 2026-07-03

Decision: CTX-TASK-801D queda como benchmark documental opt-in bajo `scripts/`,
no como adaptador productivo. El script exige
`ORQUESTA_CONTEXT_COMPRESSION_BENCHMARK_CONFIRM=CTX_COMPRESSION_BENCHMARK_OPT_IN`,
rechaza material protegido y compara bytes, tokens estimados, refs perdidas y
tiempo. Puede evaluar un comando local externo que lea stdin y escriba stdout,
pero no instala ni exige LLMLingua u otra dependencia runtime.

Evidencia local:

- Corpus: `scripts/docs/context_compression_benchmark_sample_noncritical_t284.md`.
- Resultado: `scripts/docs/context_compression_benchmark_sample_result_t284.json`.
- Compresor: `stdlib_extractive_baseline`, sin comando externo.
- Bytes: 4007 origen -> 1817 comprimido (`byte_ratio=0.4535`).
- Tokens estimados: 720 origen -> 334 comprimido (`token_ratio=0.4639`).
- Refs perdidas: 0 (`lost_refs=[]`).
- Tiempo local medido: 0.057 ms.

Conclusion: no procede adaptador productivo ahora. La evidencia solo demuestra
que el harness opt-in mide una muestra no critica y conserva refs en ese caso.
Antes de cualquier adaptador productivo hace falta una decision nueva con corpus
representativo no critico, 0 refs perdidas, coste local aceptable, diseno por
puerto/adaptador y garantia de que reglas hard, write-set, AGENTS y contratos no
entran nunca en compresion.

Nota de rebase: el goal llego con
`backlog_scan_ref=scan-ref-backlog-d438fd99b7a6` y hash esperado de linea 16
`49ddfa8cf3f609e42a9cec0342e28231a4bce588da6c8af8ba14f046c9e78d7a`; la linea
16 actual del backlog en este worktree hashea
`c294611add48932f0fb6ea5ee3c5f87e89d890d7e6ca20430c7c3cf69653d0c1`. No se
modifica el backlog desde este write-set.

## CTX-D005 - Codebase centralizado con lease

Fecha: 2026-06-30

Decision: las consultas Codebase para agentes pasan por un broker central de
Orquesta con cache, dedupe in-flight y lease de herramienta auxiliar.

Motivo: varias sesiones/subagentes pueden arrancar `codebase-memory-mcp` y dejar
procesos vivos consumiendo CPU si la herramienta queda distribuida en cada
agente. Orquesta debe gobernar el indice por repo y no delegar ese lifecycle a
prompts.

Alternativas:

- permitir `codebase-memory-mcp` en cada `CODEX_HOME`;
- prohibirlo por completo;
- usar solo `rg`.

Consecuencia: `codebase-memory-mcp` queda opt-in y exige lease central. `rg`
central sigue siendo fallback para strings, docs y config. La parada real de
procesos vive en servidor/composicion, no en este modulo neutral.

## CTX-D004 - T15 cerrado no reabre contexto

Fecha: 2026-05-27

Decision: las entradas `required ref_only` se resuelven por
`required_ref_action` y evidencia ACK; no se materializa contexto faltante ni se
relaja el rail para completar una tarea.

Motivo: T15 quedo historico; desde 2026-06-02 los rails de detalle estan
offline y no se reactivan por entorno mientras siga vigente `RAILS-D016`. El
contexto debe mantener refs pequenas y verificables sin usar payloads crudos
como sustituto de conectores.

Consecuencia: `orquesta-context` consume la politica comun por campo y conserva
tolerancia a refs opacas. Los cortes por detalle quedan offline; en runtime
normal, el director/agente decide si una evidencia se aprovecha, repara o
convierte en tarea. El saneamiento de metadata sigue anonimizando valores
sensibles efectivos en evidencias, sin usar palabras genericas como veto.

## CTX-D001 - Builder puro antes que adaptador de filesystem

Fecha: 2026-05-05

Decision: `orquesta-context` empieza con un builder puro de manifiestos y refs, sin leer filesystem productivo.

Motivo:

- el nucleo debe decidir que contexto entra sin depender de rutas reales ni proveedor;
- los agentes deben recibir bundles pequenos y repetibles;
- leer disco, MCP o REST es responsabilidad de conectores versionados.

Consecuencia: `BuildContextBundleV0` produce un manifiesto compacto; la materializacion de refs queda para adaptadores futuros.

## CTX-D002 - Consulta al director como frontera entre grupos

Fecha: 2026-05-05

Decision: si una tarea necesita informacion fuera del bundle o detalles de otro modulo, el agente no amplia contexto por su cuenta; emite `CONSULTA AL DIRECTOR`.

Motivo: evita contextos enormes y cruces internos entre grupos de trabajo.

Consecuencia: el bundle incluye politica explicita de consulta y solo acepta refs cruzadas publicas.

## CTX-D003 - Materializacion como conector explicito

Fecha: 2026-05-05

Decision: materializar refs de ContextBundleV0 mediante `ContextRefReaderV0` y un primer conector `FileContextRefStoreV0` con root explicito.

Motivo:

- el builder debe seguir siendo puro;
- el filesystem es periferico y debe entrar por conector;
- los agentes necesitan contenido local pequeno sin abrir contexto global.

Consecuencia: `MaterializeContextBundleV0` solo materializa reglas compactas, docs locales y read-set. Write-set, contratos cruzados y evidencias siguen como refs.
