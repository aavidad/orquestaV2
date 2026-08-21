# ADR V38: parada pública y retry durable B11

Fecha: 2026-08-22.

## Contexto

El techo prospectivo de B11 (`P=300,V=350`) asumía que `Stop` podía reutilizar
la liquidación de presupuesto de `Launch` y que un error local bastaba para
decidir el reintento. Esa simplificación era incorrecta: una parada no reserva
presupuesto y un texto, una cuarentena o un `BudgetSettlement` no prueban que
el efecto externo no se aplicó.

El corte correcto necesita tres piezas causales: traducción del `Stop` público
versionado por UDS y un hecho neutral, inmutable y durable que permita reencolar
solo el intento exacto demostrado como `definitely_not_applied`, más recuperación
durable del único ACK ambiguo mediante consulta idempotente. Esa recuperación
nunca reenvía `Stop` y falla cerrada ante receipt, lease, fence o CAS divergentes.

## Medición reproducible

La métrica normativa cuenta líneas brutas añadidas por clase y deja la
documentación fuera de `P/V`. B11a es el commit publicado `9f933d99`. B11b se
mide sobre el índice curado y validado contra ese commit, excluyendo
preservación B12, optimizaciones SQLite y ToolExecutor V26. B11c se mide sobre
`e0b6a5ca` en un worktree aislado sin la migración 039 ni WIP posterior:

| Corte | Producto `P` | Verificación `V` | Retiradas `D` |
|---|---:|---:|---:|
| B11a — conector público Stop y composición | 203 | 273 | 20 |
| B11b — evidencia neutral, schema 037, CAS/restart | 331 | 669 | 41 |
| B11c — recovery de ACK ambiguo, schema 038, restart/CAS | 505 | 247 | 22 |
| **B11 real** | **1039** | **1189** | **83** |

La cifra incluye la carrera real de dos writers, duplicado, receipt y
settlement terminales, rollback íntegro, corrupción de recovery, reinicio y el
bloqueo de un tercer intento tras un segundo resultado ambiguo. Reducirla al
techo histórico eliminaría precisamente los negativos que distinguen prueba
neutral de inferencia.

## Decisión y retirada compensatoria

1. B11 sustituye su estimación por `P=1039,V=1189`, igual a la superficie real
   añadida de los tres cortes. No crea una bolsa para trabajo posterior. La
   reserva prospectiva B12.3 `P=100,V=130` permanece íntegra y no se descuenta.
2. Se retiran como autoridades para retry de `Stop` la cuarentena textual, el
   `ActionConsumptionReceipt` terminal y el `BudgetSettlement` de liberación
   cero. Permanecen disponibles para sus acciones históricas, pero ninguno
   autoriza ya un nuevo intento de parada.
3. También se retira el catálogo privado duplicado de operaciones remotas del
   adaptador: la negociación usa una única lista requerida y el contrato
   público versionado. Las 83 líneas no documentales retiradas se registran, pero no se
   convierten en crédito ni se restan de `P/V`.
4. La única autoridad nueva de no aplicación es
   `EffectAttemptOutcome(definitely_not_applied)`, ligada a intento, intent,
   approval, sujeto, acción, fence, idempotencia y ventana temporal. Se inserta
   en la misma transacción que libera el claim para retry y no liquida
   presupuesto.
5. Tras B11c, el total V38 pasa de `P=9517,V=11304` a `P=10022,V=11551`.
   Es regularización contable, no promoción
   de `ORC-28`, B11, Gate B ni V38.

## Evidencia y límites

El candidato B11b pasó application y SQLite completos, aceptación focal,
focales `-race`, vet, compilación de todos los paquetes y `diff-check` en un
worktree aislado. B11a pasó además adaptador y bootstrap normales y `-race`.
B11c pasó application focal, SQLite focal normal y `-race`, compatibilidad de
migraciones y SQLite completo en un worktree aislado con schema latest 038.

Todo ello es evidencia offline. No demuestra una MicroVM, UDS físico, parada
cooperativa→forzada del proceso real, preservación, cierre, limpieza ni los
digests comunes de Gate B/C. B11 sigue sin acreditarse hasta ese E2E.

## Siguiente dependencia

B12 debe persistir y consultar la autoridad histórica exacta, consumir
`Preserve/Recover` por la interfaz pública y resolver `Quiesce/Close` sin
inventar receipts. Después se sella Gate B y recién entonces se habilita C01.
