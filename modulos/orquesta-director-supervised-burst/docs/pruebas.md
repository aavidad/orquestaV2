# Pruebas: orquesta-director-supervised-burst

## Suite local

Comando:

```sh
go test -count=1 ./modulos/orquesta-director-supervised-burst
```

Cobertura esperada:

- continua mientras la politica devuelve `continue`;
- corta en `wait_outbox`;
- corta al llegar a `max_steps`;
- expone `final_briefing` y briefing por paso con `next_action` canonico;
- propaga error de builder;
- registra `stop_error` cuando falla el paso;
- no importa adaptadores operativos.

Resultado:

- ok el 2026-05-06.
