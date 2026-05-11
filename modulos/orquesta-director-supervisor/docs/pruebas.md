# Pruebas: orquesta-director-supervisor

## Suite local

Comando:

```sh
go test -count=1 ./modulos/orquesta-director-supervisor
```

Cobertura esperada:

- outbox pendiente produce `wait_outbox`;
- waiting por outbox produce `wait_outbox`;
- waiting por capacidad externa o entrega de agente pendiente produce
  `wait_external`;
- `needs_director` con `candidate_missing` produce `wait_external` y recomendacion
  autonoma `wait`;
- las decisiones `continue`, `wait_*` y terminales exponen recomendacion autonoma conocida;
- comandos aplicados con presupuesto disponible producen `continue`;
- comandos aplicados sin presupuesto producen `stop_max_steps`;
- `needs_director`, `blocked` y `quiescent` se conservan como decisiones terminales;
- error del ultimo paso produce `stop_error`;
- inputs invalidos devuelven error publico.

Resultado:

- ok el 2026-05-06.
