# Pruebas: orquesta-app-change

Comando local:

```bash
go test -count=1 ./modulos/orquesta-app-change
go test -count=1 ./modulos/orquesta-app-change-director-source
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-web ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway
go test -count=1 ./modulos/orquesta-app-codex-stack -run TestCodexStackV0AppChangeNotificaDirectorPorWorkflow -v
```

Cobertura esperada:

- solicitud valida se persiste y notifica al director;
- solicitud invalida no toca puertos;
- evento de cambio se normaliza a solicitud, se persiste y notifica al
  director;
- write-set inseguro se rechaza;
- puertos ausentes no producen aceptacion falsa.
- `external_work.input_fields` se normaliza, rechaza nombres no compactos y se
  preserva como `DomainWorkJobRequestV0.InputFields`.
- MCP/REST/web delegan sin `cmd`, DB ni runtime.
- El stack registra el cambio como pregunta del director, outbox durable y, tras
  `DrainRunV0`, microtarea programable cuando el cambio es concreto.
