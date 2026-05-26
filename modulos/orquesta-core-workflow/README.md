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

Snapshot del core actual:

- `../../docs/reinicio_orquesta_v2/snapshots/orquesta-core-actual-2026-05-04.tgz`

## Forma del nucleo

La forma aprobada para v0 es workflow durable propio en Go:

```text
Handle(comando, estado_actual) -> eventos + outbox + error_publico
Apply(evento, estado_actual) -> estado_nuevo
Replay(eventos) -> estado_actual
```

El handler y el reducer son puros. No llaman a DB, runtime, MCP, CLI, web, Git ni modelos. Los efectos externos salen como mensajes de outbox para que otros modulos los ejecuten por conectores.

## Arranque de Codex

Desde este directorio:

```bash
./arrancar_codex.sh "microtarea concreta"
```

Al arrancar, Codex debe leer `AGENTS.md`, este `README.md` y los docs locales en `docs/`.
