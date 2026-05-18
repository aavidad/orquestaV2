# Decisiones: orquesta-domain-work-file

```text
Fecha: 2026-05-17
Decision: Implementar lectura filtrada de records mediante
`DomainWorkJobRecordSourcePortV0`.
Motivo: el snapshot ya conserva request normalizado y job aceptado. Exponerlos
por un puerto neutral permite inspeccion local y prepara conectores DB/cola sin
forzar al Director ni al servidor a leer snapshots filesystem.
Alternativas: mantener solo `ListDomainWorkJobsV0`; exponer fingerprint en una
API propia del adaptador; ampliar `DomainWorkJobCreatorPortV0`; cablear lectura
directa en el Director Operativo.
Impacto: `FileDomainWorkJobCreatorV0` implementa
`DomainWorkJobRecordStorePortV0`. La lectura filtra por refs y campos compactos,
devuelve copias defensivas y no habilita ejecucion ni `submit_artifact`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-17
Decision: Crear `orquesta-domain-work-file` como adaptador durable file-based
de `DomainWorkJobCreatorPortV0`.
Motivo: `orquesta-domain-work-memory` fija la semantica de idempotencia, pero
no sobrevive a reinicios. El nucleo necesita una referencia durable sin OPES,
Codex, runtime, DB ni red para probar jobs de dominio externos de forma
hexagonal.
Alternativas: usar `orquesta-persistence`; meter filesystem en
`orquesta-domain-work`; crear directamente un conector SQL; acoplarlo al stack
Codex o a un producto concreto.
Impacto: el adaptador guarda snapshot JSON atomico con request normalizado,
job aceptado y fingerprint. Los paquetes puros deben seguir dependiendo del
puerto, no de este adaptador.
Estado: aceptada_local
```

```text
Fecha: 2026-05-17
Decision: Mantener `DomainWorkJobCreatorPortV0` como puerto de comando y dejar
la lectura/listado como API publica del adaptador.
Motivo: la creacion de jobs es el contrato minimo que necesitan los casos de
uso. La inspeccion durable ayuda en pruebas y operacion local, pero no debe
obligar a todos los conectores a exponer semantica de consulta antes de que
haya consumidores claros.
Alternativas: ampliar el puerto puro con `List`; crear un ledger generico
antes del adaptador; no exponer lectura alguna.
Impacto: `ListDomainWorkJobsV0` existe para inspeccion del adaptador, mientras
los casos de uso siguen inyectando `DomainWorkJobCreatorPortV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-17
Decision: Permitir wiring opt-in desde `cmd/orquesta-server`, no desde el
Director Operativo ni paquetes neutrales.
Motivo: el adaptador usa filesystem y debe entrar solo por composition root.
El endpoint generico `/api/v0/domain-work` ya delega por puerto, asi que puede
crear jobs durables sin introducir OPES ni runtime. El ciclo real del Director
Operativo todavia no materializa review/replan/cierre de jobs derivados con
causalidad completa.
Alternativas: importarlo desde `orquesta-orchestration-core`; hacerlo backend
por defecto; esperar a un conector DB; cablearlo directamente en el Director.
Impacto: el servidor puede activar `create_job` con
`ORQUESTA_DOMAIN_WORK_FILE_ENABLED=1` o `ORQUESTA_DOMAIN_WORK_FILE_DIR`.
`submit_artifact` sigue no disponible salvo que se inyecte un submitter real.
Estado: aceptada_local
```
