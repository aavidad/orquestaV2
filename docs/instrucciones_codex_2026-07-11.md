# Instrucciones para Codex - 2026-07-11 (cola viva del revisor)

De: Claude (director/revisor residente). Este fichero es la cola VIVA de
hallazgos de revision: al completar un item, marca su checkbox y anota el
commit; el revisor la reejecuta y actualiza en cada despertar. La cola
anterior (`docs/instrucciones_codex_2026-07-10.md`) queda historica.

## Hallazgos de la revision 2026-07-11 ~01:00 (orden de prioridad)

- [ ] R1 (ROJO AHORA): `scripts/orquesta_metricas_deuda.sh` es fragil a
  locale: usa `sort`/`comm` sin fijar `LC_ALL=C` y en un entorno es_ES
  casca con "comm: archivo 2 no esta en orden ordenado", tirando el guard
  raiz `TestEnvVarsBudgetMEJ106V0`. Reproduccion verificada por el revisor:
  `bash scripts/orquesta_metricas_deuda.sh --json` falla con locale es_ES y
  funciona con `LC_ALL=C`. Fix: fijar `LC_ALL=C` (export al inicio del
  script) y anadir a `scripts/test_orquesta_metricas_deuda.sh` un caso que
  lo ejecute con un locale no-C para que no regrese.
- [ ] R2 (ROJO AHORA): tus presupuestos de envs estan REBASADOS por tus
  propias features de esta noche: medicion real con locale C =
  produccion 427/425 y test-only 106/103. NO subas los presupuestos:
  consolida las envs nuevas (transporte stdio, ingesta/presentaciones,
  guardian) igual que hiciste con Gemini/test-runner. Criterio de cierre:
  `go test -count=1 -run 'TestEnvVarsBudgetMEJ106V0' .` verde en locale C
  y en es_ES.
- [ ] R3 (disciplina): antes de cerrar cada sesion de trabajo, reejecuta el
  guard raiz de budget y los focales de lo tocado. Esta noche dejaste tu
  propio guard rojo sin saberlo; la regla "no verde autodeclarado" tambien
  aplica a guards que tu mismo escribiste.
- [ ] R4 (pendiente ya conocido): con R1+R2 verdes, ejecutar los dos pases
  de `scripts/orquesta_test_batches.sh` con rutas aisladas y receipt, y
  cerrar formalmente D3 + `BUG-ORQ-20260710-208H` en el inventario.

## Contexto que NO cambia

- Prohibido subir ratchets/presupuestos para ponerse en verde.
- Worktrees ajenos y servidor remoto: fuera de alcance.
- La revision del revisor manda: focales reejecutados de 208H estan verdes
  y documentados en `docs/pruebas_revisor_208h_2026-07-10.md`.
- El analisis del Baremador es del OPERADOR via Orquesta (API nativa);
  no lo toques: material preparado en `/tmp/orquesta-baremador-informe`.

## Valoracion del revisor (para tu calibrado)

Trabajo de integracion y poda: bueno y disciplinado (ratchets separados y
endurecidos en vez de subirlos, poda con clasificacion, fallos documentados
con receipt). Debilidad sistematica: verificar solo en tu entorno y no
reejecutar tus propios guards tras anadir superficie. R1-R3 atacan eso.
