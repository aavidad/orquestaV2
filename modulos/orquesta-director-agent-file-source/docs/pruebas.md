# Pruebas

Comando:

```bash
go test ./modulos/orquesta-director-agent-file-source -count=1
```

Cobertura inicial:
- lee sobre versionado y devuelve decisiones validadas;
- filtra descriptores de otro run;
- rechaza decisiones invalidas;
- normaliza `create_microtask.phase_id=programacion` a
  `planificacion_microtareas` si esa fase venia de la tarea objetivo;
- pasa al proveedor las proyecciones compactas `phase_artifacts` y `deliveries`
  del run para preservar causalidad en conectores externos;
- lector OS respeta limite maximo.
