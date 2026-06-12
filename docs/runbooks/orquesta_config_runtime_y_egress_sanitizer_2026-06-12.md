# Configuracion canonica de runtime y egress sanitizer

Fecha: 2026-06-12.

## Regla

Toda composicion que lance agentes debe tener una unica superficie canonica para
runtime por proveedor y saneamiento de salida. En el stack Codex esa superficie
es `ConfigV0` de `modulos/orquesta-app-codex-stack`:

- `CodexRuntimeConfigV0`, `GeminiRuntimeConfigV0` y `ClaudeRuntimeConfigV0`
  guardan configuracion explicita por proveedor;
- `CanonicalRuntimeProviderConfigsV0` proyecta esa configuracion como lista
  uniforme por proveedor usando flags/refs compactas, no rutas crudas;
- `EgressSanitizerConfigV0` activa el sanitizador de salida de forma opt-in;
- `PrivacyFilterSidecarConfigV0` declara el sidecar local
  `openai_privacy_filter_local` por refs/flags opacos, sin mover proveedor,
  modelo, HOME, endpoint ni comando crudo al nucleo;
- `PrivacyFilterModelConfigV0` queda como metadata local heredada para pruebas
  focales; el camino operativo nuevo es sidecar/puerto opt-in.

No se duplican variables con nombres alternativos en varios sitios. Si una
composicion necesita nuevas variables de entorno, debe mapearlas una sola vez a
esta superficie y documentar si requieren reinicio.

Toda app IA generada o modificada por Orquesta debe salir con un filtro de
egress documentado en su composicion. El default operativo recomendado es un
egress sanitizer/Privacy Filter local, opt-in y sustituible por puerto:

- la app generada concentra su configuracion en una unica superficie propia de
  composicion, no en el core de Orquesta ni en modulos neutrales;
- el filtro se activa para contexto saliente hacia modelos, proveedores,
  sidecars, agentes remotos, logs exportables o paquetes compartidos;
- el filtro sanea secretos, tokens, credenciales, paths privados, HOME real,
  payloads no publicables, endpoints internos y material sensible efectivo;
- las URLs publicas no sensibles se conservan como URLs, porque son evidencia y
  contexto util; solo se sustituyen por ref opaca si incluyen token, query
  sensible, endpoint privado, host interno, path local, credencial embebida o
  contenido no publicable;
- la seleccion de modelo, proveedor, cuota, comando, endpoint o transporte del
  Privacy Filter vive solo en composicion/adaptador opt-in y se expone al core
  como refs, flags o evidencia compacta.

## Frontera

`orquesta-context` solo define el puerto neutral `ContextSanitizerPortV0` y la
evidencia `ContextSanitizationEvidenceV0`. No conoce Codex, OpenAI, proveedor,
modelo, HOME, shell, HTTP, MCP ni DB.

`orquesta-app-codex-stack` vive en composicion exterior. Puede inyectar
`LocalSensitiveDataSanitizerV0` si `EgressSanitizerConfigV0.Enabled=true` y no
hay otro `ContextSanitizerPortV0` explicito. Si ya hay sanitizador inyectado,
manda el puerto explicito.

El contrato del sidecar vive tambien fuera del nucleo:
`PrivacyFilterSidecarPortV0` recibe payload local y devuelve contenido
saneado, categorias y refs de evidencia. La implementacion HTTP, proceso local,
Python o modelo pesado no es obligatoria aqui; debe entrar despues como
adaptador opt-in de composicion y solo publicar flags como
`CommandConfigured`/`LocalEndpointConfigured`, nunca valores crudos.

El sanitizador no es rail de contenido ni clasificador de calidad de agentes.
Sanea valores sensibles, adjunta evidencia y, si duda, degrada la entrada a
`ref_only` para revision por Director. Palabras operativas como provider, model,
runtime o capacity no bloquean una entrega por si solas.

## Operacion

1. Configurar proveedor principal con `CodexRuntimeConfigV0`.
2. Configurar proveedores auxiliares solo si se usan: `GeminiRuntimeConfigV0`
   y `ClaudeRuntimeConfigV0`.
3. Activar `EgressSanitizerConfigV0` solo en composicion opt-in.
4. Si se usa OpenAI Privacy Filter local, marcar
   `Sidecar.Enabled=true` y refs opacas (`SidecarRef`, `AdapterRef`,
   `TransportRef`, `EvidenceRef`). No poner rutas, HOME, tokens, modelos,
   endpoints ni transportes productivos en el core.
5. Verificar que el packet incluye evidencia de saneamiento cuando toca y que no
   conserva el dato sensible original.
6. En apps IA generadas, comprobar antes de entregar que existe una decision
   explicita sobre egress sanitizer/Privacy Filter local: activado con puerto de
   composicion, o pendiente documentado si la app todavia no tiene egress real.
7. Revisar muestras de salida: deben conservar enlaces publicos no sensibles y
   redactar solo URLs privadas o publicas con datos sensibles embebidos.

## Riesgos abiertos

- Este runbook fija la regla operativa; no prueba por si solo que cada app
  generada tenga wiring real de Privacy Filter local.
- Un sanitizer demasiado agresivo puede romper investigacion, citas o contexto
  publico si elimina URLs publicas no sensibles. Ese comportamiento es una
  regresion de adaptador, no una decision del core.
- Un sanitizer demasiado laxo puede filtrar secretos si trata todas las URLs
  como publicas. La composicion debe distinguir URL publica, URL privada y URL
  con datos sensibles antes del egress.

## Validacion focal

```bash
go test -count=1 ./modulos/orquesta-context
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestLocalSensitiveDataSanitizerV0|Test.*Sanitizer|Test.*Egress'
```

En sandbox restringido puede requerirse preparar `GOCACHE` en `/tmp` antes de
ejecutar los comandos, conservando el comando de prueba como contrato.
