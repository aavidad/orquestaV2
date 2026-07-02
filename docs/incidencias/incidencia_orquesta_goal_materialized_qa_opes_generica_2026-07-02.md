# Incidencia: QA generica materializada aceptada como QA OPES

Fecha: 2026-07-02

## Resumen

La reconciliacion goal-first de artefactos materializados aceptaba JSON generico
de QA como `{"passed": true}`, `{"ok": true}` o `{"status": "ok"}`. En una app
generica esto es recuperable, pero en OPES podia activar
`missing_terminal_receipt_after_artifacts_pass` aunque no existiera la terna
editorial exigida para temarios.

## Causa

El detector `goalMaterializedJSONLooksLikeQAPassV0` no diferenciaba contexto
OPES de otros dominios. El contrato OPES ya exige separar:

- `extension_pass`
- `official_text_qa_pass`
- `strict_editorial_qa_pass`

Aceptar un `passed:true` generico reintroducia falsos verdes de QA y podia
recomendar `repair_receipt` sobre evidencias insuficientes.

## Cierre aplicado

- El scanner conserva el comportamiento generico para apps no OPES.
- Si el path, payload o `GoalWorkStateV0` indican OPES, solo se considera QA pass
  cuando aparece la terna canonica o aliases estructurados equivalentes.
- Se mantiene el escaneo acotado al `write_set` y a JSON pequeno.

## Evidencia

Tests:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'Test(StackGoalMaterializedRefsSourceV0|CodexStackObserveAppDirectorGoalExecutorV0TimeoutSnapshot)'
```

Casos cubiertos:

- OPES con `passed/ok/status` generico no cuenta como QA pass.
- App no OPES con QA generica sigue siendo recuperable.
- OPES con `qa_passes.extension_pass`, `official_text_qa_pass` y
  `strict_editorial_qa_pass` cuenta como QA pass.

## Estado

Cerrado en codigo local. Pendiente de commit/push y sincronizacion remota junto
al lote verificado.
