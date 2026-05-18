# Pruebas

Validar con:

```bash
go test -count=1 ./modulos/orquesta-state-file
```

La suite debe cubrir recreacion de instancia y recuperacion de estado.

Casos relevantes:

- recuperar runs, eventos, tasks y agent processes tras recrear `StoreV0`;
- recuperar `WorkflowTaskWaitStateV0` por `run_ref + wait_ref`;
- recuperar y actualizar `OperationalDirectorPlanStateV0` por
  `run_ref + plan_ref`;
- recuperar `RequiredTestEvidenceV0` por `run_ref + evidence_ref` y rechazar la
  misma ref con payload distinto;
- conservar `required_test_evidence_refs` dentro del plan state para que el
  cierre posterior pueda consumir evidencias ya aceptadas;
- rechazar documentos de plan state con refs internas inconsistentes;
- rechazar documentos corruptos o estados que no validen el contrato del nucleo.
