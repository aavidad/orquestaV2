# Pruebas

Suite local:

```sh
go test -count=1 ./modulos/orquesta-run-control
```

Cobertura esperada:

- `running` permite scheduling y dispatch.
- `paused` bloquea scheduling y dispatch sin ser terminal.
- `stop_requested` y `cancel_requested` requieren checkpoint antes de parar agentes.
- `forced=true` permite parar agentes sin checkpoint previo.
- checkpoint registrado permite parar agentes.
- `stopped` y `canceled` son terminales.
- la arquitectura de produccion no importa paquetes de adaptadores ni contiene terminos de persistencia, red, MCP o runtime.
