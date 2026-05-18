# Decisiones: orquesta-app-runner

```text
Fecha: 2026-05-14
Decision: Ejecutar el plan preparado por fases reales, tambien con director
autonomo.
Motivo: `request_kind` describe el objetivo global del run, pero cada unidad
tiene su propia `phase_id`. Mantener documentacion, integracion o revision como
programacion duplicaba semantica y rompia la lectura de progreso del director.
Impacto: el runner abre la siguiente fase con unidades listas cuando el loop
queda quiescent y el plan no esta completo. El modo `UseAutonomousDirectorLoop`
usa el mismo avance multi-fase que el runner normal. Las entregas de cualquier
fase son `DeliveryRegistered` solo si corresponden a una microtarea durable.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Integrar `RunAutonomousDirectorLoopV0` solo por opt-in.
Motivo: el loop autonomo decide limites y concurrencia antes de ejecutar; usarlo
por defecto cambiaria el comportamiento observado de callers existentes.
Impacto: `RunPreparedAppOrchestrationV0` acepta `UseAutonomousDirectorLoop`,
permite inyectar `AutonomousDirectorPolicy` y propaga `director_loop_stats`
cuando ese camino se usa. Sin opt-in conserva el loop progresivo/gestionado
existente.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Reconstruir el provider de candidatos desde `Prepared.Plan` al ejecutar.
Motivo: una app preparada puede cruzar REST, MCP o persistencia; no debe
depender de transportar una pieza logica serializada ni aceptar un provider
desfasado respecto al plan.
Impacto: `RunPreparedAppOrchestrationV0` ignora el provider embebido y crea un
`AppPlanCandidateProviderV0` desde el plan preparado y `RequestedBy`.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Guardar el run inicial solo si el store responde `RunNotFoundErrorV0`.
Motivo: tratar cualquier error de `LoadRunV0` como ausencia puede resetear una
app grande ya avanzada si hay un fallo transitorio de infraestructura.
Impacto: el runner propaga errores de store distintos a no encontrado y no llama
a `SaveRunV0`; el run inicial solo se persiste en arranque real.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Normalizar limites de ejecucion antes de arrancar el loop de app.
Motivo: una app grande puede necesitar muchas rafagas, pero el director-cycle
tiene limites compactos por ciclo. Si el caller pide un maximo superior, el
runner debe hacer mas ciclos seguros en vez de fallar por rango.
Impacto: `RunPreparedAppOrchestrationV0` aplica defaults y caps locales antes
de construir el servicio del nucleo. Esto conserva ciclos pequenos y permite
completar planes grandes sin subir los limites internos.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Si existe `ExternalWaiter`, el runner espera por defecto con presupuesto acotado.
Motivo: una app grande real avanza por entregas externas de agentes; con
`MaxExternalWaits=0` el runner se paraba tras el primer `wait_external` aunque
hubiese un waiter inyectado capaz de continuar.
Impacto: `MaxExternalWaits` se normaliza con default y cap local. Sin
`ExternalWaiter` no se espera; con waiter, el runner puede completar varias
olas sin bucle infinito ni sleeps dentro del nucleo.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Un bootstrap con agente arrancado queda en wait_external.
Motivo: un agente vivo sin entrega registrada no es quiescent; el nucleo debe esperar su salida externa.
Impacto: las pruebas verifican agente arrancado, outbox vacio y estado wait_external.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Materializar microtareas del plan antes de abrir programacion.
Motivo: el core rechaza entregas de tareas no registradas; arrancar agentes desde el plan sin `MicrotaskCreated` produciria tareas fantasma y bloquearia cierre real.
Impacto: `PrepareAppOrchestrationV0` encadena vote, decision, contrato funcional y `CreateMicrotask` por cada unidad antes de `OpenPhase(programacion)`.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Ejecutar app preparada mediante un use case propio del runner.
Motivo: web/MCP necesitan pedir avance de una app grande sin conocer scheduler, outbox ni providers internos.
Impacto: `RunPreparedAppOrchestrationV0` compone providers opcionales, guarda el run inicial solo si falta y delega el avance al loop gestionado del nucleo.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Conservar el campo exacto del planner en errores de AppSpec.
Motivo: web/MCP necesitan saber que corregir sin parsear strings ni recibir un `app_spec` generico.
Impacto: `validatePrepareAppOrchestrationRequestV0` mapea `AppPlannerIssueV0` a `AppRunnerIssueV0` preservando prefijos canonicos.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Separar runner de app del planner.
Motivo: el planner solo debe partir trabajo. Preparar un run y conectar el
provider al nucleo es otra responsabilidad.
Impacto: REST/MCP/web podran invocar este adaptador sin conocer scheduler,
outbox ni olas internas.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Crear el run mediante comandos del workflow.
Motivo: construir OrchestrationRunV0 a mano duplicaria invariantes y repetiria
el error de v1/v2 de tener caminos paralelos.
Impacto: StartRun y OpenPhase son la unica forma de preparar el estado inicial.
Estado: aceptada.
```
