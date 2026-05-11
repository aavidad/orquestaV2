# Tareas

- [x] Definir puerto `RunDrainerPortV0`.
- [x] Definir comando y resultado compacto del tick.
- [x] Leer candidatos desde `orquesta-run-queue`.
- [x] Rankear con `RankRunCandidatesV0`.
- [x] Consultar `orquesta-run-control` por run rankeada.
- [x] Saltar estados no despachables.
- [x] Saltar `ExcludeRunRefs` para que un supervisor pueda evitar repeticion
      dentro de una pasada.
- [x] Ejecutar hasta `MaxRuns`.
- [x] Propagar `DrainLimits` al puerto de drain.
- [x] Tratar control ausente como opcional y bloquear `QueueReader/Drainer`
      ausentes con error, no panic.
- [x] Cubrir contratos principales con tests Go.
