# Pruebas: orquesta-domain-work

Comando:

```sh
go test -count=1 ./modulos/orquesta-domain-work
```

Cobertura:

- normalizacion de job externo tipo OPES sin importar OPES;
- deduplicacion de refs e idempotency/correlation por defecto;
- rechazo de refs no compactas;
- validacion de entrega de artefacto;
- puertos hexagonales sin adaptadores concretos;
- guard de arquitectura contra DB, red, runtime, filesystem y legacy.
