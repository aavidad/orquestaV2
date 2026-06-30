# Decisiones locales: orquesta-context

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
