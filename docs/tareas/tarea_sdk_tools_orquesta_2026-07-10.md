# Tarea: SDK neutral de tools/capabilities

Fecha: 2026-07-10. Estado: reparacion adversarial local completada; pendiente
composicion/materializador real.

## Integracion en la rama vigente

El SDK y el adaptador file se integraron mecanicamente desde el worktree de
tools despues de reejecutar sus pruebas en la rama vigente. Se excluyo de forma
deliberada el diff contra `docs/inventario_bugs_orquesta_2026-06-30.md`: usaba
`BUG-ORQ-20260710-209`, ID ya asignado a una incidencia distinta. No se pierde
la evidencia tecnica del SDK y no se duplica el bug historico.

Verificacion de integracion:

- `go test -race -count=1 ./modulos/orquesta-tool-capability
  ./modulos/orquesta-tool-capability-file`;
- `go test -race -count=50 ./modulos/orquesta-tool-capability -run
  '^(TestConcurrentClaimsAcrossLogicalStoresMaterializeExactlyOnceV0|TestConcurrentResumeUsesCASAndProducesOneEffectV0)$'`;
- `go test -race -count=10 ./modulos/orquesta-tool-capability-file -run
  '^TestToolOperationFileStoreMultiprocessClaimV0$'`;
- frontera neutral raiz focal.

## Alcance

Crear un SDK provider-neutral y hexagonal para integrar tools en apps generadas
sin meter filesystem, runtime, transporte, proveedor ni permisos del caller en
el core.

## Criterios cerrados

1. El request contiene refs/intencion, no `GeneratedAppToolBindingV0`. Binding,
   manifest y snapshot se resuelven por autoridades independientes.
2. Manifest y bundle resueltos coinciden exactamente con los refs solicitados;
   el snapshot es content-addressed y lleva handle/verification refs opacos.
3. Hash canonico cubre manifest+bundle y rechaza mutacion, orden ambiguo y
   duplicados. El plan tiene fingerprint de todo payload/autoridad inmutable.
4. Misma idempotencia con fingerprint distinto devuelve conflicto. Un lease
   durable app+tool+binding impide operaciones solapadas con otra key.
5. Store persiste plan+receipt atomicamente, completa por CAS y conserva
   tool/capability/manifest/bundle/hash/config/snapshot/fingerprint.
6. Reconcile/Resume cubren crash antes y despues del efecto sin duplicarlo;
   resume adquiere estado `resuming` por CAS. Errores quedan accionables como
   `recovery_required` mientras el store sea escribible.
7. Upgrade valida previous receipt terminal exacto y `UpgradeFromRefs`;
   rollback recibe el snapshot/plan previo exacto y rechaza estados impropios.
8. Status exige app+tool. Write-set rechaza URI, Windows, UNC, tilde, NUL y
   traversal, y compara por segmentos.
9. El core sigue sin filesystem/PDF/runtime/MCP/HTTP. El store file vive en un
   adaptador separado y prueba claim durable entre procesos reales.

## Write-set

- `architecture_boundaries_test.go`
- `docs/patron_sdk_tools_capabilities_orquesta_2026-07-10.md`
- `docs/tareas/tarea_sdk_tools_orquesta_2026-07-10.md`
- `modulos/orquesta-tool-capability/**`
- `modulos/orquesta-tool-capability-file/**`

## Pruebas adversariales

- autoridad de binding y refs manifest/bundle discordantes;
- snapshot/hash/handle, fingerprint mutado e idempotency conflictiva;
- claim concurrente, target lease con keys distintas y CAS stale;
- crash antes/despues del efecto, reconcile error y resume concurrente;
- upgrade-from, previous terminal y rollback de snapshot exacto;
- duplicados, hash golden/orden y paths adversariales;
- store durable reabierto, lease/CAS, symlinks/permisos/trailing state y ocho
  procesos reclamando la misma operacion.

## Semantica de entrega

El core evita despacho concurrente duplicado, pero no promete exactly-once
distribuido. Resume es at-least-once explicito con el mismo `operation_ref` y
requiere adaptador idempotente/reconciliable. El file-store demuestra
atomicidad/durabilidad del puerto, no la del futuro efecto real.

## Evidencia local

- `go test -race -count=1 ./modulos/orquesta-tool-capability
  ./modulos/orquesta-tool-capability-file` verde.
- `go test -race -count=50 ./modulos/orquesta-tool-capability -run
  '^(TestConcurrentClaimsAcrossLogicalStoresMaterializeExactlyOnceV0|TestConcurrentResumeUsesCASAndProducesOneEffectV0)$'`
  verde.
- `go test -race -count=10 ./modulos/orquesta-tool-capability-file -run
  '^TestToolOperationFileStoreMultiprocessClaimV0$'` verde; ocho procesos por
  repeticion y exactamente un claim adquirido.
- Fronteras raiz, `go vet`, `gofmt -d`, `git diff --check` y check de untracked
  verdes.
- No se usa `go test -count=1 ./...` como evidencia de integracion: es una
  bateria global prohibida por coste. Los ratchets citados en el WIP original
  son historicos; el estado vigente de variables vive en el inventario vivo.

## Pendiente fuera del corte

- resolver/materializador real con root confiable y handles no-follow;
- wiring de binding authority y file/SQL store en una composicion concreta;
- transporte MCP/API y smoke de una app consumidora real.
