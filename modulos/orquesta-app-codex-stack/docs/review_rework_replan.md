# Review rework replan

`ReviewReworkReplanSourceV0` convierte una revision no aceptada en un plan
compacto `retry_task` para el puerto
`ReviewReworkReplanPlanProviderPortV0`.

Derivacion de tarea:

1. lee `run.ReworkRequests`;
2. enlaza `rework_request -> review_result -> delivery_ref`;
3. busca en `ReceiptStore` el descriptor cuyo
   `Spec.AgentPacket.DeliveryRefs.AckRef` coincide con ese `delivery_ref`;
4. usa `descriptor.Spec.AgentPacket.Task.TaskRef`.

La primera tarea del run solo es fallback cuando no existe descriptor usable
para la entrega. Ese fallback anade
`evidence-ref-review-rework-task-fallback` para que no parezca una decision
preferente.

El source mantiene el mismo plan disponible aunque `RecordReplanDecision` ya
este proyectado. Esto es necesario porque el scheduler usa el mismo plan para
continuar en `programacion` hasta `RequestCapacity` y `RequestAgent`. El plan se
deja de emitir cuando el agente de retry ya aparece en `agents` o
`started_agents`.

La capacidad minima sale de `CapacityConfig.Tier`. Si no viene configurada, el
plan usa `high`.

Integracion con review gate:

- el source de review gate no se apaga al proyectar `RequestReview`;
- para accepted se considera terminal cuando existe `AcceptedReviews`;
- para changes_requested/rejected se considera terminal cuando existe
  `ReworkRequests` enlazado a la entrega;
- esto permite que `DrainRunV0` avance por pasos reales del scheduler hasta
  reabrir `programacion` y lanzar el agente de retry.
