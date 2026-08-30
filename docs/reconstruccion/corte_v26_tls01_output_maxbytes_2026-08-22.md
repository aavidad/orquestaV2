# Corte TLS-01: límite exacto de salida de tools

Fecha: 2026-08-22.

## Alcance

Este incremento acotado hace efectivo `CapabilitySpec.Output.MaxBytes` en el
registro neutral de tools. El límite se comprueba sobre los bytes crudos de la
salida antes de validar el schema y canonicalizar JSON. Así, espacios u otros
bytes descartables durante la canonicalización no pueden eludir el presupuesto
de salida declarado.

El rechazo usa el código máquina estable `tooling.payload_invalid` y no entrega
una salida parcial. Una salida cuyo tamaño crudo coincide exactamente con el
límite continúa admitida.

## Fronteras preservadas

- `Output.MaxBytes` limita solo la salida; no introduce un límite implícito para
  el input.
- La validación de input y output conserva el schema cerrado registrado y la
  canonicalización determinista existentes.
- La especificación sigue siendo la única fuente de versión, permisos, coste,
  entrega, idempotencia y política de receipt de cada tool.
- No se añade configuración, transporte, lifecycle, store ni writer.

## Gate focal

```bash
go test -mod=vendor -count=1 ./internal/tooling ./acceptance \
  -run 'TestRegistry|TestAcceptanceV26ToolRegistryOutputMaxBytes'
```

El gate cubre aceptación exacta en el límite, rechazo por whitespace que excede
el tamaño crudo, código máquina, schema adversarial y preservación de la
semántica de input.

## Estado honesto

Este corte implementa y ejercita únicamente el límite de salida de TLS-01. No
acredita por sí solo TLS-01, la vertical V26 ni
`AC-V26-TOOLS-SKILLS-SDK`; tampoco demuestra artifact spill, autorización,
efectos, receipts, SDK, skills, plugins o wiring de transportes.
