# Pruebas

Comando:

```bash
go test -count=1 ./modulos/orquesta-run-coordinator
```

Cobertura local:

- elige la run con mayor prioridad;
- salta una run pausada y continua con la siguiente;
- trata estado de control no encontrado como `running`;
- permite `ControlReader` ausente;
- devuelve error claro si faltan `QueueReader` o `Drainer`;
- salta runs excluidas por `ExcludeRunRefs`;
- ejecuta hasta `MaxRuns`;
- propaga limite de cola, `occurred_at`, `correlation_id` y `DrainLimits`;
- no muta los candidatos recibidos de la cola;
- bloquea imports de adaptadores o stack en codigo de produccion.
