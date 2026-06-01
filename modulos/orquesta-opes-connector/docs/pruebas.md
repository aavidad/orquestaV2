# Pruebas

Comando focal del conector:

```sh
go test -count=1 ./modulos/orquesta-opes-connector
```

Comando obligatorio T12 para revalidar frontera bridge/conector:

```sh
go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector
```

Evidencia reconciliada 2026-05-27:

- los intentos T12 `agent-ref-task-autoprogramming-5373ad36695c-g01` y
  `agent-ref-task-autoprogramming-51f9a01810a0-g01` pasaron los tests
  obligatorios;
- el fake `run-until-assemble` paso historicamente hasta
  `assemble_topic -> assembled_topic`; la revalidacion vigente debe cubrir
  tambien `generate_audio_asset -> audio_asset`;
- el smoke real OPES sigue bloqueado sin OPES temporal, Orquesta temporal,
  confirmacion de efectos y cuota/modelo confirmados.
