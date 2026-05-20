# Pruebas

- `go test -count=1 ./modulos/orquesta-server`
- `go test -count=1 ./cmd/orquesta-server`
- `cmd/orquesta-server` prueba que `ORQUESTA_OPES_BASE_URL` activa un executor
  `domain_work` OPES opt-in y que sin esa variable queda apagado.
- `cmd/orquesta-server` prueba que `ORQUESTA_DOMAIN_WORK_FILE_ENABLED=1`
  activa un creator durable file-based para `create_job`, sin habilitar
  `submit_artifact` ni `DomainDelivery`.
- `cmd/orquesta-server` prueba que OPES y el backend file de `domain_work` no
  pueden activarse a la vez.
- `cmd/orquesta-server` prueba que los umbrales productivos por defecto para
  agentes Codex no vuelven a valores agresivos de debug, y que siguen siendo
  sobreescribibles por entorno.
- `modulos/orquesta-server` prueba que `StartupCheckPortV0` publica
  `startup_ready` con mensaje/evidencias y bloquea el arranque cuando la
  composicion no esta lista.
- Prueba manual recomendada:
  - `go run ./cmd/orquesta-server run`
  - `curl http://127.0.0.1:8787/healthz`
  - `curl http://127.0.0.1:8787/api/status`
