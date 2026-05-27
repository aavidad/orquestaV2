# orquesta-core-workflow

Responsabilidad: nucleo durable de orquestacion neutral de agentes.

Este mini-proyecto nace para disenar e implementar el nucleo nuevo sin perder el core actual. El core actual queda congelado como referencia y compatibilidad; este modulo construye la maquina de workflow que debe gobernar fases, decisiones, agentes, entregas, revisiones, bloqueos y outbox para composiciones externas. Programacion, OPES u otros dominios consumen el nucleo por puertos y refs opacas; no definen su frontera.

Incluye:

- estado compacto `OrchestrationRunV0`;
- fases canonicas de una app programada por agentes;
- comandos deterministas;
- eventos append-only;
- reducer puro;
- outbox de efectos externos;
- reglas de idempotencia y replay;
- consultas al director cuando un grupo necesita otra decision.

No incluye:

- DB concreta;
- runtime real de agentes;
- Codex, Claude, Ollama, vLLM o proveedor concreto;
- HTTP, CLI, MCP o UI;
- tmux, Docker, Git o sistema operativo;
- algoritmo pesado de scheduler externo.

## Relacion con `orquesta-core`

`modulos/orquesta-core/` conserva el corte actual: registro puro desde `AppSpecV0`, puertos de salida y contratos de microprogramacion. No se borra ni se pisa.

Este modulo nuevo debe absorber solo lo que se demuestre util mediante contratos pequenos. Si una pieza actual se necesita aqui, se copia o adapta por microtarea documentada, nunca por arrastre completo.

Snapshot historico del core anterior: el arbol de reinicio v2 ya no existe en
la foto vigente y no es prerequisito local. Para contexto vivo usar
`../../AGENTS.md`, `../../docs/estado_actual_2026-05-17.md`,
`../../docs/guia_nucleo_orquestacion_2026-05-17.md` y este README.

## Forma del nucleo

La forma aprobada para v0 es workflow durable propio en Go:

```text
Handle(comando, estado_actual) -> eventos + outbox + error_publico
Apply(evento, estado_actual) -> estado_nuevo
Replay(eventos) -> estado_actual
```

El handler y el reducer son puros. No llaman a DB, runtime, MCP, CLI, web, Git ni modelos. Los efectos externos salen como mensajes de outbox para que otros modulos los ejecuten por conectores.

## Arranque de Codex

La ruta vigente para agentes OrquestaV2 es el servidor residente y la cola
gobernada, no el wrapper local. El wrapper `./arrancar_codex.sh` queda como
compatibilidad historica y devuelve un error publico si no hay contrato manual
versionado. Para trabajo operativo usa `go run ./cmd/orquesta-server run` desde
la raiz y APIs/CLI publicas con write-set, ACK, checkpoint y shutdown
gobernados.

## Rails de detalle

Desde la reconciliacion T15 del 2026-05-27, este modulo documenta la politica
comun de `orquesta-rails`: refs opacas y vocabulario operativo son validos,
mientras valores sensibles efectivos, rutas privadas y material crudo se cortan
por frontera/campo. El servidor activa el rail por defecto con scope acotado; el
modo `programming` conserva apertura para autoprogramacion gobernada.
