# Pruebas locales: orquesta-rails

## RAILS-P016 rails offline sin reactivacion por env

Tipo: live-smoke | compile

Comandos vigentes:

- `go build ./cmd/orquesta-server ./modulos/orquesta-rails ./modulos/orquesta-runtime-codex ./modulos/orquesta-director ./modulos/orquesta-app-change-director-source ./modulos/orquesta-orchestration-core`
- `go test -count=1 ./modulos/orquesta-rails`
- prueba viva por API arrancando `orquesta-server` con
  `ORQUESTA_RAILS_MODE=enforced` y `ORQUESTA_DETAIL_PROHIBITED_RAILS=on`, y
  comprobando `/api/v0/server/status`.

Evidencia esperada:

- `ORQUESTA_RAILS_MODE` efectivo queda `offline` incluso si el entorno intenta
  poner `enforced`;
- `ORQUESTA_DETAIL_PROHIBITED_RAILS` efectivo queda `off` incluso si el entorno
  intenta ponerlo `on`;
- los helpers de texto, detalle, arquitectura, filesystem, privacidad y
  taxonomia no bloquean ni ocultan contenido mientras `RailsEnforcedV0()` sea
  falso;
- `RedactOperationalTextForFieldV0` sanitiza valores sensibles efectivos sin
  reactivar deteccion/bloqueo de rails blandos;
- la redaccion no se dispara por marcadores genericos como `prompt`, `payload`,
  HOME, provider/model o vocabulario operativo sin valor secreto efectivo;
- el servidor mantiene default `ORQUESTA_RAILS_MODE=offline` y
  `ORQUESTA_DETAIL_PROHIBITED_RAILS=off`.

Nota: los tests antiguos que esperan bloqueo estricto quedan como inventario
historico. No son la validacion vigente mientras los rails esten offline; no se
deben adaptar para simular una politica que produccion no usa.

Riesgos: al reintroducir un rail habra que crear matriz nueva y probar en vivo
que el Director/orquestador puede reparar o aprovechar salidas imperfectas sin
tirar trabajo util.
