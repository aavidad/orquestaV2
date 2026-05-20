# Paquete de visuales, tablas y esquemas responsivos

Rol: visuales utiles, tablas comparativas y esquemas responsivos.

Estado: material parcial para integracion editorial. No es un tema final y no
debe marcarse como listo. Las piezas de contenido estan redactadas para que el
agente principal pueda integrarlas, adaptar captions y mover los SVG finales a
`assets/` si procede.

## Alcance entregado

- `plan_visuales.md`: mapa de visuales propuesto, ubicacion sugerida en el
  temario, objetivo didactico, texto alternativo y comportamiento responsive.
- `tablas_comparativas.md`: tablas Markdown integrables sobre arquitecturas,
  estilos de integracion, gobierno de APIs, resiliencia, interoperabilidad,
  seguridad y errores de examen.
- `esquemas_responsivos.md`: patron HTML/CSS para publicar SVG locales con
  scroll horizontal en movil, notas de accesibilidad y sincronizacion.
- `assets_borrador/*.svg`: cuatro SVG deterministas, sin dependencias externas,
  preparados como borrador local.

## Fuentes oficiales usadas como marco editorial

Estas referencias se citan por nombre para evitar insertar URLs reales en el
texto visible del temario. Conviene que `fuentes.md` archive la referencia
formal completa:

- Ley 39/2015, de 1 de octubre, del Procedimiento Administrativo Comun de las
  Administraciones Publicas.
- Ley 40/2015, de 1 de octubre, de Regimen Juridico del Sector Publico.
- Real Decreto 203/2021, de 30 de marzo, por el que se aprueba el Reglamento de
  actuacion y funcionamiento del sector publico por medios electronicos.
- Real Decreto 4/2010, de 8 de enero, por el que se regula el Esquema Nacional
  de Interoperabilidad.
- Real Decreto 311/2022, de 3 de mayo, por el que se regula el Esquema Nacional
  de Seguridad.
- Reglamento (UE) 2024/903 del Parlamento Europeo y del Consejo, sobre medidas
  para un alto nivel de interoperabilidad del sector publico en la Union.
- Reglamento (UE) 910/2014, relativo a la identificacion electronica y los
  servicios de confianza.
- Guia de Comunicacion Digital para la Administracion General del Estado,
  componente de accesibilidad y experiencia de usuario.

## Integracion recomendada

1. Usar el primer visual como mapa inicial tras la orientacion de examen.
2. Repartir las tablas en el desarrollo teorico, siempre precedidas y seguidas
   de explicacion. No deben sustituir la teoria.
3. Mantener las notas de test separadas del texto base. En HTML deben poder
   ocultarse.
4. Mover a `assets/` solo los SVG que el tema final use realmente y actualizar
   los nombres en `plan_visuales.md`.
5. Verificar que el HTML no apunta a assets inexistentes y que cada figura tiene
   `alt`, titulo editorial y explicacion de lectura.

## Limitaciones

- El workspace recibido no contenia documentos locales ni tema base ya creado.
  Por eso el paquete no se ha podido contrastar con una estructura existente de
  `tema_a1.md`.
- No se ejecuta validacion de 20.250 palabras porque este subagente no ensambla
  el tema final.
- No se publica nada ni se accede a sistemas OPES.
