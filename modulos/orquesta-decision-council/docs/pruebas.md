# Pruebas: orquesta-decision-council

Comando local:

```bash
go test -count=1 ./modulos/orquesta-decision-council
```

Cobertura minima:

- plan con tres familias opacas;
- rechazo de brainstorming sin `evidence_refs`;
- rechazo por falta de diversidad;
- critica cruzada sin auto-revision;
- voto aceptado con quorum;
- voto bloqueado;
- rechazo de voto no abstencion sin `evidence_refs`;
- resultado de voto conserva source refs deduplicadas sin transcripts;
- imports sin DB, runtime, web ni proveedores concretos.
