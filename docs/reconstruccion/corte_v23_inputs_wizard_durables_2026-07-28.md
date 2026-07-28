# V23: corte de inputs Wizard durables

Fecha: 2026-07-28.

Estado: **partial_green_unsealed**.

Este corte conserva los inputs canónicos exactos de cada petición pública
`orquesta.intakes.wizard.gaps.apply`. No persiste todavía el snapshot exacto
del resultado de evaluación y, por tanto, `evaluation_replay_exact` sigue
siendo `false`. V23 no queda sellada.

## Autoridad y atomicidad

- `WizardGapsInputReceipt` liga actor, proyecto, autorización, petición,
  revisión esperada, origen, evaluador histórico, receipt fuente de Intake,
  outcome real, facts tipados, pack refs canónicos y selections canónicas.
- Facts, pack refs y selections se guardan junto a sus digests. El ref del
  receipt incluye los digests y todos los bindings causales.
- `WizardGapsStore` combina `IntakeStore` y las operaciones Wizard. Así la
  aplicación no puede leer el Intake en un repositorio y escribir el outcome
  en otro.
- La mutación Intake y el input receipt se confirman en una sola transacción.
  En la rama no-op ocurre lo mismo con `wizard_gaps_outcomes` y el input
  receipt. Si falla el insert final, no queda ninguno de los efectos parciales.
- La salida pública no filtra facts, selections, autorización ni actor
  históricos. Expone refs, digests y la ubicación
  `wizard_gaps_input_receipt`; el evaluador público sigue tipado aparte.

## Replay y recuperación

El replay reconstruye la evaluación desde el receipt fuente, los inputs
durables y la identidad exacta del evaluador. Además verifica:

- que las selections persistidas son las derivadas del Intake fuente;
- que un no-op sigue produciendo el mismo outcome content-addressed;
- que una mutación produce exactamente el estado y el receipt Intake ligados,
  incluido fingerprint, operación, revisión previa y autorización;
- que digests, refs, contexto canónico y bindings no fueron reescritos.

Recovery repite esa validación para todas las filas de schema 21. Hay pruebas
de reinicio tras una mutación posterior, carreras exactas y divergentes,
rollback en ambas ramas y corrupción simple o coherente.

## Upgrade desde schema 20

La migración `021_wizard_gaps_inputs.sql` no inventa inputs para outcomes
creados por schema 20. Esos datos históricos no contienen facts, pack refs ni
selections suficientes para fabricar un receipt exacto.

Tras el upgrade, un replay de una petición legacy real falla cerrado con
`StateConflict`: schema 20 usaba la identidad de petición
`orquesta.wizard.gaps.noop.request.v1`, mientras que el receipt de inputs exige
la identidad canónica V2. El outcome antiguo se conserva para auditoría y
recovery sigue aceptando la base; no se presenta como replay exacto ni se
materializa evidencia retroactiva. La guarda adicional de late attachment
impide también adjuntar a posteriori un input ausente aunque la identidad de
petición ya fuese compatible.

## Evidencia del corte

- aplicación: contexto, snapshots, digests y tamper coherente;
- SQLite: atomicidad mutation/no-op, reinicio, concurrencia, corrupción y
  upgrade 20 a 21;
- comandos y bootstrap: proyección pública común para CLI, MCP y HTTP;
- aceptación: scope `wizard_gap_input_receipt_durability` completado.

La siguiente dependencia causal es
`wizard_gap_exact_evaluation_snapshot_before_replay`: persistir el resultado
exacto de evaluación, validarlo y solo entonces considerar si
`evaluation_replay_exact` puede pasar a `true`.
