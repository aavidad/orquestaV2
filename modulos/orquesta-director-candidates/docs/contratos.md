# Contratos locales: orquesta-director-candidates

## `BuildSchedulableWorkCandidateV0`

Tipo: caso de uso puro.

Entrada:

- refs explicitas del candidate, run, fase, tarea, capacidad y agente;
- comandos explicitos para capacidad, gate y agente;
- claims completos o claims compactos con scopes;
- evidencias opacas, incluyendo evidencias separadas para gate.

Salida:

- `orquesta-director-scheduler.SchedulableWorkCandidateV0` validado.

Invariantes:

- No crea refs semanticas nuevas: solo normaliza espacios y propaga lo recibido.
- `run_ref` debe coincidir en comandos y claims.
- Los comandos se validan con constructores publicos de workflow/director.
- El candidate siempre contiene capacidad, gate y agente para el plan work v0.
- No transporta payloads de infraestructura ni proveedor.

## `BuildSchedulableWorkCandidatesFromPlanV0`

Tipo: caso de uso puro.

Entrada:

- `run_ref` y `phase_id` comunes del plan compacto;
- lista de microtareas con `candidate_ref`, `task_ref`, claims por scopes, comandos, idempotency keys, capacidad, agente y evidencias explicitas.

Salida:

- lista ordenada de `orquesta-director-scheduler.SchedulableWorkCandidateV0` validada con el builder unitario.

Invariantes:

- No inventa candidate, task, command, idempotency, capacity, agent ni claim refs.
- Cada microtarea se construye mediante `BuildSchedulableWorkCandidateV0`.
- Si una microtarea trae scopes invalidos o refs obligatorias ausentes, todo el corte falla sin producir candidates parciales.
- Mantiene el orden de entrada para que el scheduler pueda aplicar su prioridad estable.

## `BuildSchedulableWorkCandidatesFromCouncilPlanV0`

Tipo: caso de uso puro.

Entrada:

- `DecisionCouncilPlanV0` producido por `orquesta-decision-council`;
- `role` de ronda: propuesta, critica o voto;
- `phase_id` activa recibida del director;
- metadatos compactos de comando.

Salida:

- lista ordenada de `SchedulableWorkCandidateV0` solo para esa ronda.

Invariantes:

- No lanza agentes ni espera resultados.
- No incluye Codex, Gemini, Claude, HOME, OAuth, cuotas, cuentas ni proveedor en payloads que cruzan al core.
- Deriva refs neutrales por ordinal de ronda, no desde nombres reales de agente o familia.
- La ronda de propuesta y critica se ejecuta desde `brainstorming_arquitectura`; la ronda de voto desde `votacion_y_decision`.
- Las barreras entre rondas pertenecen al director/consejo; este builder solo prepara candidates seguros para el scheduler.

## `BuildDecisionCouncilTeamPlanFromComplexityV0`

Tipo: utilidad pura.

Entrada:

- refs explicitas de run, decision, brainstorming y voto;
- complejidad compacta (`low`, `medium`, `high`, `xhigh` o alias cortos);
- evidencias opacas.

Salida:

- `DecisionCouncilPlanV0` con slots neutrales, capacidad minima, quorum y barreras de consejo.

Invariantes:

- No selecciona familias reales ni cuentas: solo slots derivados de la ref de decision.
- No lee estado durable ni consulta entorno externo.
- La complejidad `low` usa 2 agentes; `medium` usa 3; `high` usa 4; `xhigh` usa 5.
- Toda salida se valida con `BuildDecisionCouncilPlanV0`.
