# orquesta-decision-council

Responsabilidad: coordinar deliberacion multiagente para decisiones importantes.

Este modulo planifica rondas de propuesta, critica y voto. No lanza agentes, no decide proveedor, no lee DB y no conoce runtime real. Recibe candidatos ya resueltos por capacity/director como referencias opacas.

Uso previsto:

- arquitectura y brainstorming inicial;
- votacion de decisiones;
- consenso con diversidad de familias de agentes;
- registro de disenso antes de aceptar una decision.

Las rondas trabajan con `evidence_refs` y refs durables. No aceptan votos no abstencion sin evidencia y no copian prompts ni transcripts completos en el resultado.

No sustituye a `orquesta-core-workflow`: lo complementa como politica pura alrededor de `RequestBrainstorm`, `RequestVote` y `AcceptDecision`.
