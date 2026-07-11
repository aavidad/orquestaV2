# Incidencia: invariantes causales del nucleo

Fecha: 2026-07-11.
Estado: cerrada localmente, revision independiente R4 sin bloqueantes.
Alcance: nucleo y composicion local; sin remoto, OPES ni procesos reales.

## Hallazgos

Una auditoria con propiedades encontro cuatro defectos distintos que compartian
la misma causa: una proyeccion podia inferir mas estado del que demostraban sus
referencias causales.

| ID | Defecto | Riesgo |
| --- | --- | --- |
| BUG-ORQ-20260711-211 | `ReconcileExternalWorkPublicStatusV0` decidia `running` y `completed` con logica paralela a `DerivarVeredictoCausalV0` | estados publicos incompatibles entre superficies |
| BUG-ORQ-20260711-212 | `EvidenciaEstadoV0{Terminal:true}` era suficiente aun sin fuente o `evidence_refs` durable | falso cierre terminal sin prueba auditable |
| BUG-ORQ-20260711-213 | `ProjectRunQueueAttemptsV0` elegia el intento activo antes de conocer todas las relaciones `supersedes_run_ref` | el intento supersedido podia ganar segun el orden de lectura |
| BUG-ORQ-20260711-214 | el fallback streaming con `TaskRefs` vacio incorporaba todas las `DeliveredTasks` del run | review de tareas pertenecientes a otra ola o wait |

## Correccion

- El reconciliador external-work delega su autoridad en
  `DerivarVeredictoCausalV0`. Un outbox o una task abierta demuestran trabajo
  `pending`, no un proceso `running`; este ultimo exige observacion e identidad
  runtime atribuible.
- Un terminal sin fuente y al menos una referencia durable queda
  `indeterminate`, con `durable_terminal_evidence_missing` y reparacion
  requerida. Los adaptadores reales de receipts ya derivan referencias desde
  receipts, artefactos, checklist, tests y cierres.
- La cola recopila primero todos los intentos y supersesiones y elige despues
  el activo. La propiedad prueba ambas permutaciones.
- El substream reconstruye `TaskRef -> AgentRequestRef` cuando falta
  `TaskRefs`; si no existe mapping causal o dos tasks normalizan al mismo agent
  ref, conserva el wait y no abre review.
- El coordinador excluye antes del ranking cualquier `run_ref` citado por un
  `supersedes_run_ref`; no depende de que el intento activo conserve esa ref.

La autoridad causal sigue en el core puro. El reconciliador solo adapta sus
señales tipadas y queda incluido en el guard de llamadores autorizados; no se
introducen dependencias de runtime en `orquesta-estado-vivo`.

## Evidencia local

Verdes:

```text
go test -count=1 ./modulos/orquesta-estado-vivo
go test -count=1 ./modulos/orquesta-run-coordinator
go test -count=1 ./modulos/orquesta-run-queue
go test -count=1 ./modulos/orquesta-run-control
go test -count=1 ./modulos/orquesta-app-director-service
go test -count=1 ./modulos/orquesta-app-codex-stack
go test -count=1 .
git diff --check
```

Las propiedades nuevas cubren determinismo/inmutabilidad de run-control,
permutacion y supersesion de run-queue, y ausencia de cierre/running sin
evidencia causal. Las pruebas del substream cubren entrega ajena y mapping
ausente.

## Criterio de reapertura

Reabrir el ID concreto si una superficie publica `running` sin identidad viva,
acepta terminal sin referencia durable, cambia el intento activo al permutar la
entrada o revisa tareas fuera del scope causal del wait.

## Revision adversarial R1

La primera revision independiente rechazo correctamente el falso cierre
inicial. Detecto que la proyeccion de intentos ya era determinista pero el
coordinador aun podia ejecutar el supersedido; que el mapping task/agente no es
inyectivo; que un terminal malformado podia ocultar otro conflicto valido; y
que `pending` podia ocultar `RequiereReparacion`.

El segundo parche añade casos adversos para los cuatro huecos. La reparacion
causal gana a outbox/dominio pending, un terminal valido y un proceso vivo
conservan el conflicto aunque coexista evidencia malformada, y el coordinador
prueba prioridad mayor del intento supersedido y un tercer intento activo sin
`SupersedesRunRef`.

R2/R3 detectaron ademas paginacion antes de supersesion y un enlace cruzado
entre grupos. El coordinador lee la cola completa, resuelve supersesion y
ranking y aplica despues `QueueLimit`. Una supersesion exige grupo causal
normalizado coincidente; el fallback legacy exige `ParentRunRef` exacto. R4
reviso grupos distintos, parent legacy inexacto y limite posterior y declaro
que no quedan bloqueantes locales.
