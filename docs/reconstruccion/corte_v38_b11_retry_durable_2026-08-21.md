# Corte V38 B11: reintento durable de Stop no aplicado

Fecha: 2026-08-21.

Estado: incremento correctivo parcial. B11, `ORC-28` y V38 permanecen abiertos;
este corte no crea receipt de release ni acredita una parada física.

## Invariante

Un error estructural `DefinitelyNotApplied` observado después de persistir el
`EffectAttempt` no es un efecto, un receipt ni consumo de presupuesto. La
aplicación conserva una prueba neutral e inmutable ligada al intento exacto y,
en la misma transacción, libera el claim para reintentar la misma acción.

El siguiente claim conserva la identidad e idempotencia del `EffectIntent`,
pero obtiene delivery attempt, token y fence nuevos. El intento anterior deja
de bloquear únicamente por su prueba exacta. Un resultado ambiguo,
`unknown_applied`, un receipt existente, una prueba ausente o una prueba
cruzada nunca autorizan otra invocación física.

La mutación atómica no escribe `BudgetSettlement`: Stop no reserva presupuesto.
Tampoco crea `EffectReceipt` ni `ActionConsumptionReceipt`, porque la acción
sigue pendiente y el efecto quedó probado como no aplicado.

## Autoridad y recuperación

`internal/application` continúa como único escritor. `StateRepository`
persiste prueba y requeue mediante CAS sobre el claim vigente; SQLite los
confirma en una única transacción. El ledger de intent, aprobación e intentos
permanece append-only.

Los gates de restart deben distinguir tres cortes:

1. antes del commit no existen prueba ni requeue durable;
2. tras el commit ambos existen y el Stop puede reclamarse de nuevo;
3. si se pierde el ACK del commit, repetir la mutación no duplica la prueba ni
   el efecto y la acción ya reencolada sigue siendo la autoridad.

Dos writers con el mismo claim no pueden ganar. Un receipt concurrente, un
attempt distinto o drift de action, intent o fence cierran la mutación.

## Gates exigidos

- aplicación: retry positivo con prueba durable, misma idempotency key y fence
  nuevo; negativos sin prueba y `unknown_applied`;
- SQLite: migración progresiva, inmutabilidad, CAS, replay, carrera y restart;
- contabilidad: cero `BudgetSettlement` en todos los caminos de Stop;
- efecto: ningún intento ambiguo o ya recibido se reenvía;
- aceptación: prueba ejecutable de los invariantes anteriores;
- verificación: focales normales y `-race`, application y SQLite completos,
  acceptance, vet, arquitectura, compilación global y `git diff --check`.

## Límite honesto

Estas pruebas acreditan solo la causalidad local y durable del reintento seguro.
No demuestran que Agente MicroVM haya detenido un proceso, que el protocolo
público soporte reconciliación, ni los gates A+B+C de V38. No se modifican
roadmap, capabilities, evidence, Agente MicroVM, VEC ni `CODEX_HOME`.

## Cierre pendiente

```text
hecho: prueba neutral definitely_not_applied y requeue atómico
autoridad final: application sobre StateRepository; SQLite como adaptador local
receipt acreditante: ninguno
P0/P1: ningún unknown_applied puede reintentarse; E2E físico sigue pendiente
siguiente dependencia: gates restart/race y composición candidata V38
```
