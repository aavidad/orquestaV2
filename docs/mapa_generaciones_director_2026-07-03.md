# Mapa canonico de generaciones Director

Fecha: 2026-07-03.

Este documento es la referencia unica para clasificar los modulos `*director*`
del repo. La regla de producto vigente es: **goal-first es el unico camino de
produccion para trabajo nuevo**. `legacy_director_loop` queda en sunset:
solo mantenimiento correctivo, sin features nuevas, y candidato a retirada
cuando goal-first cubra Claude y Gemini (T18).

## Tabla

| Modulo | Generacion | Estado | Consumidores actuales por imports | Decision |
| --- | --- | --- | --- | --- |
| `orquesta-app-change-director-source` | soporte V1/legacy | mantenimiento | raiz/tests; `orquesta-app-codex-stack` | Mantener solo como adaptador de compatibilidad para convertir app-change existente en decisiones; no anadir features. |
| `orquesta-app-director-intake` | goal-first/soporte compartido | vivo | `orquesta-app-director-service`; `orquesta-mcp` | Mantener como preparador neutral de intake para start/preview; no meter runtime ni persistencia concreta. |
| `orquesta-app-director-service` | goal-first + compatibilidad legacy | vivo con sunset legacy | `cmd/orquesta-server`; `orquesta-app-codex-stack`; `orquesta-app-gateway`; `orquesta-mcp`; `orquesta-opes-*`; `orquesta-runtime-codex-delivery`; `orquesta-web` | Camino de produccion: goal-first. `legacy_director_loop` solo explicit opt-in y publica `legacy_sunset_notice`. |
| `orquesta-director` | soporte neutral historico/V2 | vivo acotado | raiz/tests; `cmd/orquesta-server`; `orquesta-app-codex-stack`; `orquesta-cli`; `orquesta-director-*`; `orquesta-mcp`; `orquesta-orchestration-core`; `orquesta-web` | Mantener contratos puros reutilizados por scheduler/candidatos; no convertirlo en daemon ni runtime. |
| `orquesta-director-agent` | V1 agentes/workflow | mantenimiento | `orquesta-app-change-director-source`; `orquesta-app-codex-stack`; `orquesta-app-director-service`; `orquesta-director-agent-*`; `orquesta-mcp`; `orquesta-runtime-codex-delivery` | Mantener contratos de agente y refs usados por legacy/cierre; features nuevas deben ir a goal-first o Director V2 aprobado. |
| `orquesta-director-agent-file-source` | soporte V1 | mantenimiento | `cmd/orquesta-server`; `orquesta-app-codex-stack`; `orquesta-app-director-service`; `orquesta-runtime-codex-delivery` | Solo fuente/adaptador de decisiones desde fichero; no ampliar como backend de planificacion. |
| `orquesta-director-agent-workflow` | V1 workflow task | mantenimiento | `orquesta-app-change-director-source`; `orquesta-app-codex-stack`; `orquesta-app-director-service`; `orquesta-director-agent-file-source`; `orquesta-mcp` | Mantener por contratos causales `WorkflowTaskV0`; no usar para nuevas generaciones de loop. |
| `orquesta-director-candidates` | V2 soporte de candidatos | vivo acotado | `orquesta-app-codex-stack`; `orquesta-orchestration-core` | Fuente neutral de candidatos para scheduler; no debe decidir runtime, proveedor ni persistencia. |
| `orquesta-director-cycle` | Director V2 | congelado | raiz/tests; `cmd/orquesta-server`; `orquesta-director-supervised-burst`; `orquesta-director-supervisor`; `orquesta-orchestration-core` | Congelado por T-PER-202: solo fixes correctivos con bug enlazado. |
| `orquesta-director-cycle-outbox` | Director V2 | congelado | raiz/tests; `cmd/orquesta-server`; `orquesta-app-codex-stack`; `orquesta-app-director-service`; `orquesta-app-runner`; `orquesta-director-cycle`; `orquesta-orchestration-core`; `orquesta-persistence`; `orquesta-runtime-codex-delivery` | Congelado por T-PER-202: ledger/outbox de ciclo, sin dispatch ni nuevas features. |
| `orquesta-director-operativo` | Director Operativo V1 | mantenimiento vivo | raiz/tests; `cmd/orquesta-server`; `orquesta-app-codex-stack`; `orquesta-app-director-service`; `orquesta-orchestration-core`; `orquesta-state-file` | Mantener contratos de plan/olas/waits/cierre ya integrados; no reabrir como loop principal frente a goal-first. |
| `orquesta-director-runner` | Director V2 | congelado | raiz/tests; `cmd/orquesta-server`; `orquesta-app-director-service`; `orquesta-director-cycle`; `orquesta-director-cycle-outbox`; `orquesta-director-supervised-burst`; `orquesta-director-supervisor`; `orquesta-director-tick-input`; `orquesta-orchestration-core` | Congelado por T-PER-202: ejecuta un ciclo acotado; no daemon, no runtime. |
| `orquesta-director-scheduler` | Director V2 | congelado | raiz/tests; `cmd/orquesta-server`; `orquesta-app-*`; `orquesta-director-*`; `orquesta-orchestration-core`; `orquesta-runtime-codex-delivery` | Congelado por T-PER-202: planifica ticks puros desde candidatos; no ampliar sin decision documentada. |
| `orquesta-director-supervised-burst` | soporte Director V2 | vivo acotado | raiz/tests; `orquesta-app-codex-stack`; `orquesta-app-director-service`; `orquesta-mcp`; `orquesta-orchestration-core`; `orquesta-runtime-codex-delivery` | Mantener como adaptador de ejecucion acotada sobre V2; si crece, decidir si se congela o se retira. |
| `orquesta-director-supervisor` | soporte Director V2 | vivo acotado | raiz/tests; `cmd/orquesta-server`; `orquesta-app-codex-stack`; `orquesta-director-supervised-burst`; `orquesta-mcp`; `orquesta-orchestration-core` | Mantener para supervision acotada; no debe relanzar trabajo nuevo por loop legacy. |
| `orquesta-director-tick-input` | Director V2 | congelado | raiz/tests; `orquesta-director-cycle`; `orquesta-director-cycle-outbox` | Congelado por T-PER-202: prepara inputs compactos; no calcula candidates ni despacha. |
| `orquesta-opes-director` | adaptador de dominio OPES | soporte consumidor | `cmd/orquesta-server` | OPES es consumidor, no nucleo. Mantener como conector de dominio sobre puertos/refs opacas; produccion nueva debe usar goal-first/DomainWork. |

## Decisiones

- No hay mas de una fuente de verdad generacional: este mapa gobierna cuando
  haya duda entre legacy, V1, V2 y goal-first.
- Goal-first queda como ruta de produccion para apps nuevas, autoprogramacion
  y conectores nuevos.
- V2 congelado no es una nueva plataforma de features; es una espina pura de
  scheduler/workflow/outbox que solo recibe fixes con bug enlazado.
- V1/legacy existe para compatibilidad, cierre causal y smokes historicos; no
  se usa para abrir capacidades nuevas.
- OPES y otros dominios no son nucleo: entran por conectores y refs opacas.
