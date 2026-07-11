# Resultado de limpieza y autoreparacion BUG-237..240

Fecha: 2026-07-11. Entorno: local aislado; sin OPES, remoto ni conectores.

## Goal real

- run fuente: `request-ref-bug237-daemon-env-20260711`;
- goal fuente: `goal-ref-task-autoprogramming-35461236b435-g01`;
- goal externo: `019f50a4-8850-7f53-9219-d59e6b94a558`;
- rework: `request-ref-bug237-daemon-env-20260711-rework-38a04c4c4f93b543`;
- goal externo rework: `019f50ab-3ce1-7392-bfcd-8de01a2a00ca`.

El fuente consumio 50.115 tokens observados sin diff y pidio replan. El primer
intento fallo porque no transportaba el binder de tests (BUG-239). Tras el fix,
el mismo estado lanzo el rework causal. El hijo consumio 51.121 tokens sin diff
y fue parado; su primera observacion terminal devolvio 500 porque faltaba la
`OrchestrationRunV0` hija (BUG-240). Tras el segundo fix, la reentrada creo la
run sin segundo launch y `observe` devolvio HTTP 200 con goal/run/closure
`blocked`, causa `operator_forced_stop_no_artifacts` y cero procesos residuales.

Orquesta no produjo el parche BUG-237 tras dos presupuestos. Conforme a la
regla de excepcion, el desbloqueo local aplico el diff minimo: una clave padre
solo cruza al daemon si pertenece al registro efectivo. El test negativo, los
ratchets y el paquete servidor completo quedaron verdes.

## Limpieza medida

```text
deadcode_candidates=1177
private_without_text_references=0
helper_duplicate_definitions=0
helper_name_family_overlap_definitions=307
orphan_modules=8 (retain_pending_composition)
```

Los 307 son solapes nominales y no autorizan consolidacion. El auditor publica
la clasificacion `nominal_overlap_not_code_duplication`.

Medicion posterior a los pilotos batch del mismo dia: 26.161 funciones,
1.172 candidatos, 309 solapes nominales, 24 ficheros mayores de 800 lineas y
cero duplicados demostrados. La foto anterior se conserva como recibo del
corte BUG-237..240; no debe usarse como contador vigente.

## Huellas retenidas

- prepare: `95a3a6693124f5efd3955589679a8b12d4214680d99cc2750f05e7159103dc24`;
- supervisor final: `03eaa40955a6d25a04a3c17a77e3bcf8e6238944ef9023b4f6640f4055f4481b`;
- observe final: `7f8365c82b1d460fcdc801a61c60477f57a13efff490aea3dcc46e2a5b711446`;
- progreso fuente: `12176fdc8ad045a87da338c83854e409e5b14f16a0c7829e6038ba667c931242`;
- progreso rework: `dfce981fcea774dafa2fbf4f8b259925aa3cca9982f7b3151149eb1a1dc59777`.

Los directorios temporales, cache de modulos, worktree y binario se eliminan
tras este recibo; las huellas no dependen de conservarlos.
