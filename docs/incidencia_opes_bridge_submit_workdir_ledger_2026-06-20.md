# Incidencia OPES Bridge: workdir y ledger claimed

Fecha: 2026-06-20.

## Contexto

Prueba OPES para `Oficial de Servicios Múltiples` C2, programa
`f0f9cb912e353b9c7f3459e9e76a949a`, job
`01421971819c7808c847386a3338b8f8`.

El bridge OPES enviaba el `plan_temario` a `/api/v0/external-work/run`, pero
`opes-drain-once` devolvía solo `request_http_400` y el ledger quedaba en
`claimed`, bloqueando reintentos posteriores.

## Comprobación

Se capturó el cuerpo real enviado por `submitOPESExternalWorkRunV0`:

- el sobre `external_work_run_request` era correcto;
- `app_change_request.external_work` estaba presente;
- el rechazo real de Orquesta fue
  `external_work_project_work_dir_mismatch`;
- el servidor Orquesta de la prueba estaba arrancado desde
  `/home/alberto/Trabajo/orquesta` sin `ORQUESTA_CODEX_PROJECT_WORKDIR`;
- para trabajos OPES el guard exige `/home/alberto/Trabajo/OPES`.

## Arreglo aplicado

- `submitOPESExternalWorkRunV0` conserva códigos públicos de errores HTTP no
  2xx, por ejemplo
  `request_http_400:external_work_project_work_dir_mismatch`.
- El ledger registra `submit_failed` con `last_error` si Orquesta rechaza el
  submit.
- `submit_failed` es reintentable para la misma intención; `claimed` y
  `submitted` siguen bloqueando duplicados.
- Prueba de regresión:
  `TestRunOPESDrainOnceV0SubmitErrorGuardaFalloReintentableV0`.

## Regla operativa

Para lanzar trabajos OPES reales o temporales, arrancar el servidor Orquesta
con:

```bash
ORQUESTA_CODEX_PROJECT_WORKDIR=/home/alberto/Trabajo/OPES
ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_WORKDIR=/home/alberto/Trabajo/orquesta
ORQUESTA_OPES_PROJECT_WORKDIR=/home/alberto/Trabajo/OPES
```

Así el stack principal trabaja sobre OPES y la automejora/backlog de Orquesta
puede seguir apuntando al repo de Orquesta sin romper el guard OPES.

## Evidencia local

Prueba focal ejecutada:

```bash
go test -count=1 ./cmd/orquesta-server -run 'TestRunOPESDrainOnceV0SubmitErrorGuardaFalloReintentableV0|TestRunOPESDrainOnceV0Claim|TestExternalBridgeInputLedger|TestReadCommandHTTPResponseBodyV0|TestRunOPESDrainOnceV0PlanTemarioOperadoresEnviaExternalWorkDocumentPlanV0|TestRunOPESDrainOnceV0SecuenciaPases'
```

Resultado: `ok`.
