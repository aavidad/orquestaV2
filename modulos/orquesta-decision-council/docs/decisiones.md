# Decisiones: orquesta-decision-council

## DC-001: consejo como politica pura

Decision: crear un modulo separado para planificar propuesta, critica y voto multiagente.

Motivo: v1 ya tenia propuestas/votos y v2 tiene comandos durables, pero faltaba la pieza que sincroniza varios agentes sin convertir el core en director monolitico.

Alternativas: meter la logica en `orquesta-core-workflow`; dejarlo en prompts; crear un scheduler que hable con proveedores reales.

Impacto: este modulo solo produce planes/evaluaciones. Runtime, capacity, context, persistence y MCP quedan como conectores.

Estado: aceptada_director.

## DC-002: diversidad por familias opacas

Decision: exigir quorum por `family_ref` opaca en vez de enumerar marcas.

Motivo: permite que voten varias familias externas sin acoplar el nucleo a esos nombres. El adaptador/capacity asigna las familias y este modulo solo valida diversidad.

Alternativas: hardcodear proveedores; permitir cualquier numero de agentes del mismo proveedor; votar solo por agente director.

Impacto: para una decision critica se configura `required_family_refs` con las familias deseadas y `minimum_distinct_families` acorde.

Estado: aceptada_director.

## DC-003: critica cruzada antes de votar

Decision: una propuesta no pasa directamente a voto; antes hay critica cruzada.

Motivo: reduce la probabilidad de que una solucion mala gane por convergencia superficial. Tambien fuerza evidencia y expone disensos.

Alternativas: votar propuestas directamente; sintetizar con un solo agente; pedir revision humana siempre.

Impacto: el plan genera rondas separadas y gates: propuestas, criticas y votos.

Estado: aceptada_local.

## DC-004: evidencia obligatoria por refs

Decision: brainstorming, critica y voto solo operan con `evidence_refs` y refs durables; los votos no abstencion sin evidencia son invalidos.

Motivo: evita aprobaciones baratas y mantiene trazabilidad sin guardar prompts ni transcripts completos.

Alternativas: aceptar votos solo con posicion; copiar el transcript completo al resultado; delegar la barrera al runtime.

Impacto: `BuildDecisionCouncilPlanV0` exige fuentes iniciales y `EvaluateDecisionCouncilVotesV0` agrega refs de evidencia deduplicadas en la salida.

Estado: aceptada_local.
