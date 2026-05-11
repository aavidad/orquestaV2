# Pruebas locales

Comando canonico:

```sh
go test -count=1 ./modulos/orquesta-director-candidates
```

Cobertura v0:

- construccion desde claims compactos con scopes;
- validacion con el scheduler;
- rechazo de idempotency key ausente;
- construccion de varios candidates desde plan compacto;
- rechazo de ids ausentes y scopes invalidos en plan compacto;
- construccion de candidates de consejo por ronda;
- validacion de candidates de consejo contra scheduler;
- voto de consejo en fase `votacion_y_decision`;
- derivacion de plan de equipo/capacidad desde complejidad compacta;
- rechazo de complejidad invalida o evidencias ausentes;
- arquitectura: imports y terminos de infraestructura prohibidos en codigo productivo.
