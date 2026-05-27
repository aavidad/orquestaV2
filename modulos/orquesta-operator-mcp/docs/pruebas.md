# Pruebas: orquesta-operator-mcp

## Contrato local

- `OPMCP-CT-001`: capabilities no expone terminos prohibidos.
- `OPMCP-CT-002`: status query exige `request_ref`, `subject_ref` y conector.
- `OPMCP-CT-003`: burst supervisado exige `run_ref`, conector y `max_steps` valido.
- `OPMCP-CT-004`: directed query exige refs opacas y pregunta compacta.
- `OPMCP-CT-005`: outbox pendiente exige refs opacas y `limit` valido.
- `OPMCP-CT-006`: `orquesta-mcp` publica resource/tools operativas y delega solo por puertos inyectados.
- `OPMCP-CT-007`: `TransportPortV0` fake recibe el registro opt-in y un tool operativo se sirve solo mediante puerto fake.
- `OPMCP-CT-008`: conector simulado cubre estado, burst, outbox y consulta dirigida con normalizacion de alias segura.
- `OPMCP-CT-009`: transporte sin conector devuelve `operator_mcp_port_unavailable` y con conector agregado simulado delega correctamente.
- `OPMCP-CT-010`: transporte propaga errores publicos del conector y cubre
  estado, outbox y consulta dirigida mediante conector agregado simulado.
- `OPMCP-CT-011`: T198 conserva paridad entre `OperatorMCPCapabilitiesV0` y el
  `descriptor_source` MCP, sin exponer internos ni transporte real.

## Comando

```bash
go test -count=1 ./modulos/orquesta-operator-mcp ./modulos/orquesta-mcp
```

Ultima ejecucion: 2026-05-23, ok.

Reconciliacion T198: ejecutada el 2026-05-27 en el paquete
`agent-ref-task-autoprogramming-c3678e9bc306-g01` con el comando obligatorio
ampliado de T198.

Revalidacion T198 adicional: `agent-ref-task-autoprogramming-c3678e9bc306-g01`
usa el mismo comando obligatorio para conservar el cierre documental.
