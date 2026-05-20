# Patron de esquemas responsivos

Los SVG incluidos son borradores locales. Antes de publicar el tema, el agente
principal debe mover a `assets/` solo los visuales finalmente usados y actualizar
las referencias desde Markdown y HTML.

## Patron Markdown

```markdown
![Mapa de capas de arquitectura e integracion](assets/mapa_capas_integracion.svg)

Figura. Mapa de lectura del tema: gobierno, canales, APIs, microservicios,
integracion, datos, seguridad y operacion.

> Modo tutor: usa el mapa para ordenar una pregunta larga. Primero localiza el
> servicio publico y el dato; despues identifica el contrato, la tecnologia y
> las garantias.
```

## Patron HTML

```html
<figure class="visual visual--scroll" data-study-layer="visual">
  <div class="visual__canvas" tabindex="0" aria-label="Esquema con desplazamiento horizontal">
    <img src="assets/mapa_capas_integracion.svg"
         alt="Mapa de capas que relaciona gobierno, canales, APIs, microservicios, integracion, datos, seguridad y operacion."
         loading="lazy">
  </div>
  <figcaption>
    Mapa de lectura del tema: de la decision publica al contrato tecnico y a la operacion.
  </figcaption>
</figure>
```

## CSS recomendado

```css
.visual {
  margin: 1.5rem 0;
}

.visual__canvas {
  overflow-x: auto;
  overscroll-behavior-x: contain;
  border: 1px solid var(--border-subtle, #d7dde8);
  border-radius: 8px;
  background: #ffffff;
}

.visual__canvas img,
.visual__canvas svg {
  display: block;
  width: 100%;
  min-width: 720px;
  height: auto;
}

.visual figcaption {
  margin-top: .55rem;
  color: var(--text-muted, #475569);
  font-size: .95rem;
  line-height: 1.45;
}

@media (max-width: 760px) {
  .visual__canvas {
    border-radius: 6px;
  }

  .visual__canvas img,
  .visual__canvas svg {
    min-width: 840px;
  }
}
```

## Reglas editoriales de figura

| Regla | Aplicacion |
| --- | --- |
| No decorativo | Cada visual debe resolver una confusion o explicar una relacion estructural |
| Local | No referenciar assets remotos |
| Accesible | Incluir `alt`, caption y explicacion de lectura |
| Responsive | Conservar `viewBox` y permitir scroll horizontal si la figura contiene texto |
| Sin URLs visibles | Las fuentes se archivan en `fuentes.md`, no dentro del SVG |
| Sin sobrecarga | No introducir mas de 6-8 nodos principales por figura |
| Sin sustitucion de teoria | La teoria debe explicar lo que la figura resume |

## Manifest de assets borrador

| Archivo | Uso | Revision pendiente |
| --- | --- | --- |
| `assets_borrador/mapa_capas_integracion.svg` | Mapa inicial del tema | Ajustar paleta al HTML final |
| `assets_borrador/flujo_api_gateway_microservicios.svg` | Bloque APIs y microservicios | Revisar nombres si el tema adopta otro vocabulario |
| `assets_borrador/eventos_observabilidad_resiliencia.svg` | Bloque eventos y resiliencia | Confirmar que no duplica otro esquema del tema |
| `assets_borrador/intermediacion_datos_aapp.svg` | Bloque interoperabilidad tecnica | Coordinar con el tema de gobierno del dato para no solapar |

## Checklist tecnico antes de publicar

- El HTML parsea sin errores y no referencia assets inexistentes.
- Cada SVG usado existe en `assets/` y no se enlaza desde `subagentes/`.
- El modo primera lectura puede ocultar visuales sin romper continuidad.
- Las notas de test asociadas se pueden ocultar.
- En movil, los esquemas no se recortan; si hay mucho texto, se desplazan en
  horizontal.
- La paleta no depende solo del color: las agrupaciones se distinguen por texto,
  posicion y borde.
