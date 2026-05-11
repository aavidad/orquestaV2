# Pruebas locales: orquesta-agent-process-registry

```text
Caso: Validacion neutral de AgentProcessRegistryRecordV0
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-agent-process-registry`
Evidencia esperada: Normaliza espacios, compacta evidencias, acepta refs opacas y rechaza detalles operativos.
Ultima ejecucion: 2026-05-10; ejecutada correctamente.
Riesgos: No prueba adaptadores durables; eso pertenece a `orquesta-persistence`.
```
