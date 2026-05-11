# Pruebas: orquesta-operator-mcp

## Contrato local

- `OPMCP-CT-001`: capabilities no expone terminos prohibidos.
- `OPMCP-CT-002`: status query exige `request_ref`, `subject_ref` y conector.
- `OPMCP-CT-003`: burst supervisado exige `run_ref`, conector y `max_steps` valido.
- `OPMCP-CT-004`: directed query exige refs opacas y pregunta compacta.
- `OPMCP-CT-005`: outbox pendiente exige refs opacas y `limit` valido.
- `OPMCP-CT-006`: `orquesta-mcp` publica resource/tools operativas y delega solo por puertos inyectados.
- `OPMCP-CT-007`: `TransportPortV0` fake recibe el registro opt-in y un tool operativo se sirve solo mediante puerto fake.

## Comando

```bash
go test -count=1 ./modulos/orquesta-operator-mcp ./modulos/orquesta-mcp
```

Ultima ejecucion: 2026-05-07, ok.
