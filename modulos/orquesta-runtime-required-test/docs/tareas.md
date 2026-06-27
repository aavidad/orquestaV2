# Tareas: orquesta-runtime-required-test

## RTRT-001: executor local de tests requeridos

Estado: hecho primer corte y cableado opt-in desde `cmd/orquesta-server`.

Objetivo: implementar `RequiredTestCommandExecutorPortV0` fuera del nucleo para
ejecutar comandos reales con binarios permitidos por allowlist, guardar salida
en artefacto local y devolver refs relativas.

Decision: no vive en `orquesta-runtime` base porque ese paquete no puede
importar `orquesta-orchestration-core` sin crear ciclos. Este modulo separado es
el adaptador valido para implementar el puerto del nucleo.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-runtime-required-test
```

Prueba integrada local:

- `TestRequiredTestRunnerV0ConLocalCommandExecutorEjecutaGoTestReal` crea un
  modulo Go temporal, ejecuta `go test ./...` mediante allowlist explicita y
  guarda `RequiredTestEvidenceV0` causal con artefacto relativo.
- `TestLocalCommandExecutorV0NoHeredaEntornoPadre` prueba que el proceso de test
  requerido no recibe variables del entorno padre cuando `Env` esta vacio.
- `TestLocalCommandExecutorV0ValidacionSinFicherosEscaneadosEsFailed` prueba que
  un validador que sale con codigo cero pero reporta `status=pass` y
  `files_scanned=0` no se acepta como evidencia de cierre.

Pendiente:

- prueba real de programacion con agentes/subagentes usando el runner activado;
- replan negativo automatico si el runner devuelve evidencia `failed`.
