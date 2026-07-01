# Incidencia: wizard Nueva App no mostraba contrato guiado cerrado

Fecha: 2026-07-02.

ID inventario: `BUG-ORQ-20260702-098`.

## Sintoma

El wizard de `/nueva-app` ya tenia sesion guiada y handoff parcial, pero la UI
no mostraba con claridad las preguntas pendientes ni el estado terminal
`lista_para_solicitar`. El operador podia llegar a la pantalla final sin una
senal visual fuerte de que el contrato conversacional seguia incompleto.

Ademas, `onion` existia como arquitectura soportada en select/normalizador, pero
no como accion guiada visible.

## Causa

El backend guiado devolvia `pending_questions`, pero el render JS solo mostraba
el campo activo. La ayuda larga existia como guia global, pero no habia enlaces
contextuales por campo. El set de acciones guiadas de arquitectura no estaba
alineado con todas las opciones visibles.

## Cierre

La UI de Nueva App ahora:

- muestra estado de sesion y lista de preguntas pendientes;
- muestra `lista_para_solicitar` cuando no quedan pendientes;
- deshabilita acciones finales mientras la sesion guiada sigue incompleta;
- expone accion guiada `architecture_onion`;
- genera anchors en la guia larga y enlaces contextuales desde campos
  principales.

## Evidencia

Pruebas locales:

```bash
go test -count=1 ./modulos/orquesta-web
```

No se ejecuto browser externo; queda como verificacion opt-in de accesibilidad
runtime.
