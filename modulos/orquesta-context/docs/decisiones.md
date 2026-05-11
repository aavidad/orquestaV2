# Decisiones locales: orquesta-context

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
