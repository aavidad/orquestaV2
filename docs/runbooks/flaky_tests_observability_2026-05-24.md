# Observabilidad de tests intermitentes

Fecha: 2026-05-24.

## Caso T10

- Fecha observada: 2026-05-23.
- Comando: `go test -count=1 ./...`.
- Paquete: `cmd/orquesta-server`.
- Test: `TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje`.
- Fallo exacto: stdout de un agente fake no contenia el prompt esperado.
- Reintento posterior observado: el test focal y `go test -count=1 ./cmd/orquesta-server`
  pasaron.

## Causa

El fake runtime escribia `codex_last_message.txt` antes de completar stdout.
El test esperaba solo la existencia de `last_message` y leia stdout
inmediatamente despues. Esa secuencia creaba una carrera entre evidencia de
entrega y flush de stdout.

## Correccion

- El test espera ahora contenido esperado en `codex_stdout.log`, no solo la
  presencia de `codex_last_message.txt`.
- Se anade un harness acotado que repite tres veces el caso focal observado,
  con timeout por intento y timeout interno de `go test` para que una regresion
  bloqueante quede visible como fallo de fiabilidad:

```bash
go test -count=1 ./cmd/orquesta-server -run TestFlakyHarnessV0RepiteCasoDirectorRecursiveFakeRuntimeV0
```

## Regla

Antes de cerrar un flake como fiable:

- registrar fecha, comando, test y fallo exacto;
- aislar fuente probable: orden de mapas, tiempos, concurrencia, ficheros
  temporales, stdout/stderr o estado compartido;
- repetir el caso focal con limite bajo;
- ejecutar despues la bateria obligatoria del write-set.

## Evidencia focal

- 2026-05-24: `go test -count=1 ./cmd/orquesta-server -run TestFlakyHarnessV0RepiteCasoDirectorRecursiveFakeRuntimeV0` paso con tres repeticiones del caso observado.
- 2026-05-24: `go test -count=1 ./cmd/orquesta-server` paso despues del harness acotado.

## Entorno restringido

- Los tests del servidor que necesitan HTTP local verifican antes si `127.0.0.1:0`
  puede abrir socket. Si el sandbox no permite sockets, se registran como
  skip explicito en vez de panico de `httptest` o de fake servers Python.
