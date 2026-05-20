# Decisiones: orquesta-director-tick-input

## DTI-DEC-001: Builder Puro Sin Fuentes Externas

Decision: el builder recibe el run, refs de outbox y candidates ya calculados.

Motivo: listar outbox, leer event-store o evaluar candidates son responsabilidades de adaptadores o modulos vecinos.

Consecuencia: esta pieza es determinista y facil de probar.

## DTI-DEC-002: Proyecciones A Refs Canonicas

Decision: el builder reduce proyecciones compactas del workflow a las refs que espera el scheduler.

Motivo: el workflow guarda informacion suficiente para validar replay, pero el scheduler solo necesita saber si una ref logica ya existe.

Consecuencia: los adaptadores no deben repetir parseos manuales en tests o produccion.

## DTI-DEC-003: Candidates Explicitos

Decision: el builder no inventa work candidates, phase artifact candidates, lease candidates, progress candidates ni replan candidates.

Motivo: inventar candidates fue una fuente de bucles en versiones anteriores. Cada candidate debe venir de una politica probada o de una decision externa.

Consecuencia: si no hay candidates y no hay outbox pendiente, el scheduler devolvera `quiescent`.

## DTI-DEC-004: Phase Artifacts Como Refs, No Payloads

Decision: `OrchestrationRunV0.PhaseArtifacts` se reduce a refs de artefacto para el scheduler.

Motivo: el workflow conserva proyeccion suficiente para replay; el scheduler solo necesita dedupe y no debe parsear documentos, ACKs ni rutas.

Consecuencia: el provider externo puede reenviar el mismo receipt en ticks posteriores y el scheduler quedara `quiescent` si la ref ya esta reflejada.

## DTI-DEC-005: WorkClaims Se Propagan Sin Inventarse

Decision: el builder transporta `work_claims` del caller al scheduler junto a los work candidates.

Motivo: la evaluacion de concurrencia necesita la ola completa, pero el builder no debe reconstruirla ni duplicarla dentro de cada candidate.

Consecuencia: los providers calculan claims; tick-input solo los normaliza a traves del scheduler.

## DTI-DEC-006: ReworkRequests Viajan Como Snapshot Durable

Decision: `OrchestrationRunV0.ReworkRequests` se expone en
`RunSchedulingSnapshotV0.ReworkRequests`, y `ReviewGateCandidates` se propagan
como candidates explicitos del caller.

Motivo: el scheduler necesita saber si `RequestRework` ya fue reflejado para no
duplicar el comando, pero tick-input no debe evaluar si una revision se acepta o
pide retrabajo.

Consecuencia: `accepted`, `changes_requested` y `rejected` se deciden fuera de
tick-input; este modulo solo clona candidates y reduce estado durable a refs
compactas, sin DB, runtime, proveedor, modelo, HOME ni OAuth.

## DTI-DEC-007: Progreso Ajeno No Entra Al Scheduler

Decision: el builder descarta `progress_supervision_candidates` sin
`command_meta.run_id` y `report.run_id` del run actual.

Motivo: las fuentes externas pueden conservar observaciones durables de runs
anteriores; el scheduler debe seguir estricto y no recibir candidates ajenos.

Consecuencia: tick-input no evalua progreso ni corrige candidates; solo aplica
un filtro de frontera por identidad causal suficiente antes de validar el input
compacto.
