# Tareas: orquesta-runtime-required-test

## RTRT-001: executor local de tests requeridos

Estado: hecho primer corte.

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

Pendiente:

- wiring opt-in desde la composicion real del servidor/director;
- politica de comandos permitidos para el repo actual;
- replan negativo automatico si el runner devuelve evidencia `failed`.
