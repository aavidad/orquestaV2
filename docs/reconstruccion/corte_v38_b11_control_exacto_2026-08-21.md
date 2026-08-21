# Corte V38 B11: control Stop exacto offline

Fecha: 2026-08-21.

Estado: incremento seguro sin acreditación. `ORC-28`, AC-V38 y V38 siguen
abiertos; no hubo commit, push ni ejecución física de MicroVM.

## Alcance y autoridad

Write-set B11: `control.go` y sus tests, hunks Stop de `adapter.go` y tests,
wrapper/factoría/tests bootstrap, aceptación B11 y este corte. No se modificó
application, SQLite, retry durable, vendor, el repositorio hermano, VEC ni
CODEX_HOME. El WIP concurrente se preservó.

El preflight `ORC-28` sobre `Adapter.Stop` devolvió `characterize`: no halló
implementación exacta reutilizable ni autorizó copiar legacy. Agente MicroVM
conserva la autoridad física; el adaptador traduce exclusivamente su API
pública vendorizada. Bootstrap solo compone y delega. Application/SQLite y el
retry durable pertenecen a Task04 y son un gate causal externo a B11.

## Contrato implementado

- La negociación exige `consultar_ejecucion` y `detener_ejecucion` y publica
  los modos cooperativo y forzado que declara el contrato estable.
- `Stop` valida identidad causal completa, observa una ejecución
  `disponible`, captura revisión/cerca y emite una sola llamada `Detener` con
  referencia, modo e idempotencia exactos.
- Pendiente no acredita parada. Confirmada conserva identidad, modo, receipt y
  fecha; cruces, campos parciales y estados imposibles fallan cerrados.
- La API vendorizada exige que el modo efectivo coincida con el solicitado,
  salvo `ya_ausente`. Cooperativa→forzada queda post-llamada, ambigua y sin
  receipt: B11 no anuncia escalado automático.
- La API pública no ofrece `ReconcileStop`. Adapter y bootstrap no reenvían,
  consultan privados ni inventan reconciliación; el resultado ambiguo queda en
  cuarentena para la autoridad de application.
- `Shutdown` solo cierra recursos locales de composición. No descubre ni
  sustituye una parada física individual.

La cobertura de composición atraviesa
`Build → agenteMicroVM → Adapter → Observar/Detener` y ratchea referencia,
revisión, cerca, modo, clave y receipt. La prueba UDS usa el cliente público
real para el negativo de escalado incompatible. La aceptación impide publicar
`ReconcileStop` inexistente.

## Presupuesto y gates

Medición del slice B11 contra HEAD, excluyendo hunks B12 y Task04:

```text
producto P <= 300
verificación V <= 350
documento <= 140 líneas
disco adicional observado < 150 MiB
```

Gates exigidos: focal adapter/bootstrap/acceptance normal y `-race`, paquetes
application/acceptance, vet, arquitectura, compilación completa y
`git diff --check`. Un fallo concurrente ajeno se atribuye y no se corrige
desde este write-set.

No se ejecutaron KVM, Firecracker, huésped Codex, caída/reinicio de host ni
parada física. No existe receipt de candidato ni revisión acreditante.

## Cierre contractual

```text
hecho: Stop público exacto compuesto offline, sin reconciliación inventada
invariante: una llamada física; pendiente/error post-llamada no acreditan
autoridad final: hermano físico; application lifecycle/ledger (fuera de B11)
tests: unitarios, UDS público, composición Build y aceptación negativa
receipts/revisión: ninguno; ORC-28/AC-V38/V38 abiertos
retirada: ninguna; no se copió legacy
P0: ninguno observado
P1: retry/cuarentena durable Task04; E2E físico y restart pendientes
siguiente gate: integrar Task04 y acreditar UDS/físico/restart sobre candidato
```
