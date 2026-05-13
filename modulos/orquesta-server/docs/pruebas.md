# Pruebas

- `go test -count=1 ./modulos/orquesta-server`
- `go test -count=1 ./cmd/orquesta-server`
- `cmd/orquesta-server` prueba que `ORQUESTA_OPES_BASE_URL` activa un executor
  `domain_work` OPES opt-in y que sin esa variable queda apagado.
- Prueba manual recomendada:
  - `go run ./cmd/orquesta-server run`
  - `curl http://127.0.0.1:8787/healthz`
  - `curl http://127.0.0.1:8787/api/status`
