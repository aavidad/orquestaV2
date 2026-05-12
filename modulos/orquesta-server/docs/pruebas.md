# Pruebas

- `go test -count=1 ./modulos/orquesta-server`
- `go test -count=1 ./cmd/orquesta-server`
- Prueba manual recomendada:
  - `go run ./cmd/orquesta-server run`
  - `curl http://127.0.0.1:8787/healthz`
  - `curl http://127.0.0.1:8787/api/status`

