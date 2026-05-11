# Pruebas

Comando:

```bash
go test ./modulos/orquesta-director-agent-file-source -count=1
```

Cobertura inicial:
- lee sobre versionado y devuelve decisiones validadas;
- filtra descriptores de otro run;
- rechaza decisiones invalidas;
- lector OS respeta limite maximo.
