# Pruebas

Comando:

```bash
go test -count=1 ./modulos/orquesta-run-supervisor
```

Cobertura local:

- respeta `MaxTicks`;
- respeta `MaxExecutions`;
- para por tick sin ejecuciones;
- propaga campos al tick;
- excluye runs ya ejecutadas en la siguiente vuelta;
- no muta la entrada;
- propaga error de tick y contexto cancelado;
- bloquea imports de adaptadores en codigo de produccion.
