# Contratos: orquesta-decision-council

## DecisionCouncilPlan v0

Planifica una deliberacion multiagente por fases.

Entrada: `DecisionCouncilPlanInputV0`.

- `run_ref`: run objetivo.
- `decision_topic_ref`: decision o tema.
- `brainstorm_request_ref`: solicitud durable de brainstorming.
- `vote_request_ref`: solicitud durable de votacion.
- `minimum_capacity_level`: `low`, `medium`, `high` o `xhigh`; por defecto `high`.
- `minimum_agents`: minimo de agentes participantes; por defecto `3`.
- `minimum_distinct_families`: minimo de familias/proveedores opacos; por defecto `2`.
- `required_family_refs`: familias requeridas como refs opacas, resueltas fuera del modulo.
- `candidates`: candidatos ya resueltos por capacity/director.
- `evidence_refs`: refs durables de fuentes disponibles para brainstorming, critica y voto.

Salida: `DecisionCouncilPlanV0`.

- asignaciones de propuesta independiente;
- asignaciones de critica cruzada;
- asignaciones de voto;
- gates de sincronizacion por ronda.

Invariantes:

- No arranca agentes ni emite outbox.
- No decide proveedor, modelo, HOME, OAuth, cuenta, cuota ni secreto.
- Las familias son refs opacas; el modulo no interpreta marcas.
- Cada critica evita revisar la propuesta propia y prefiere otra familia.
- El plan exige `evidence_refs`; no se planifican rondas sin fuentes trazables.
- El contexto de las asignaciones debe usar refs pequenas, no prompts ni transcripts completos.
- La implementacion posterior debe materializar contexto pequeno mediante `orquesta-context`.

## DecisionCouncilVote v0

Evalua votos recibidos de agentes independientes.

Entrada: `DecisionCouncilVoteInputV0`.

- `option_refs`: opciones candidatas.
- `votes`: votos de agentes.
- `minimum_votes`: por defecto `3`.
- `minimum_non_author_votes`: por defecto `2`, heredado de la regla v1 de dos opiniones no autoras.
- `minimum_distinct_families`: por defecto `2`; para tres familias externas se configura `3`.
- `approval_threshold_pct`: por defecto `67`.
- Cada voto no abstencion requiere `evidence_refs`.

Salida: `DecisionCouncilVoteResultV0`.

- opcion ganadora si existe;
- votos considerados;
- familias participantes;
- `evidence_refs` agregadas y deduplicadas;
- disensos y bloqueos;
- `accepted=false` si falta quorum, hay bloqueo o no se alcanza umbral.

Invariantes:

- Un voto `block` impide aceptar decision automaticamente.
- Un voto `approve`, `reject` o `block` sin `evidence_refs` es invalido.
- Se conservan disensos para auditoria.
- La salida conserva refs de fuentes; no copia transcripts completos.
- La aceptacion durable final sigue perteneciendo a `AcceptDecision` en `orquesta-core-workflow`.
