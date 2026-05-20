# Checklist de integracion A1

## Validacion de estructura

- [ ] El tema final tiene titulo completo y no contiene menciones internas de ejecucion.
- [ ] El indice coincide con el orden real del Markdown y del HTML.
- [ ] La primera lectura puede hacerse sin depender de tablas, test ni visuales.
- [ ] Todas las definiciones importantes aparecen antes del desarrollo complejo.
- [ ] Las notas de test estan separadas de la teoria.
- [ ] Los supuestos practicos tienen situacion, pistas, preguntas, resolucion paso a paso, errores y mini comprobacion.
- [ ] El repaso final incluye 5 a 7 ideas, definiciones rapidas, diferencias criticas y preguntas de recuperacion.
- [ ] El fichero de fuentes cita normas y organismos oficiales por identificador editorial.

## Validacion de palabras

- [ ] `tema_a1.md` alcanza al menos 20.250 palabras.
- [ ] `tema_a1.md` no supera 22.500 palabras salvo decision editorial explicita.
- [ ] Si no alcanza 20.250 palabras, existe `INFORME_BLOQUEO.md` con palabras alcanzadas, secciones completas, secciones pendientes y pasos exactos para cerrar.
- [ ] No se marca como listo profesional si no cumple el minimo.

Comando sugerido para el ensamblador:

```bash
wc -w tema_a1.md
```

## Validacion editorial OPES A1

- [ ] Tono adulto, tecnico y pedagogico.
- [ ] Parrafos cortos, una idea principal por parrafo.
- [ ] Tablas usadas para comparar o repasar, no para sustituir teoria.
- [ ] Ejemplos despues de conceptos abstractos.
- [ ] Distractores y trampas solo en notas de test, enfoque o repaso.
- [ ] Test progresivo separado del temario: base, aplicacion y examen real.
- [ ] Banco completo fuera del tema, en `banco_preguntas_i18n/es/`.
- [ ] Visuales utiles, locales y responsivos.
- [ ] No hay placeholders.
- [ ] No hay listas densas sin explicacion.

## Validacion de fuentes

- [ ] BOE y DOUE revisados para normas principales.
- [ ] BOJA/Junta de Andalucia usado solo como fuente autonomica o ejemplo institucional.
- [ ] Las fuentes internacionales no oficiales se usan solo como apoyo.
- [ ] No se muestran direcciones web reales en el Markdown/HTML final.
- [ ] `fuentes.md` permite localizar cada fuente por norma, organismo, identificador y fecha de consulta.

## Validacion de HTML si se genera

- [ ] Hero del tema, barra de modo de estudio, contenido principal y barra lateral plegable.
- [ ] Primera lectura activa por defecto.
- [ ] Tablas, notas, test y visuales pueden ocultarse para lectura continua.
- [ ] Modo tutor separado del texto base.
- [ ] Notas de test con formato unico y ocultable.
- [ ] Supuestos practicos con solucion ocultable.
- [ ] Repaso antes del enfoque de examen y del test progresivo.
- [ ] HTML parsea correctamente.
- [ ] No referencia assets inexistentes.
- [ ] Visuales con contenedor responsivo y scroll horizontal si hace falta.
- [ ] Banco i18n externo enlazado o declarado por contrato local, no incrustado completo.

## Comprobacion de coherencia por secciones

| Seccion | Debe responder | Riesgo a revisar |
| --- | --- | --- |
| Orientacion | Que se pregunta y como se estudia? | Convertirla en introduccion generica. |
| Mapa inicial | Como se conectan gobierno, semantica, calidad y reutilizacion? | Mapa sin explicacion textual. |
| Definiciones | Que significa cada termino y con que se confunde? | Definir tarde o repetir siglas. |
| Normativa | Que aporta cada capa juridica? | Lista de normas sin criterio. |
| Gobierno | Quien decide, mantiene y controla el dato? | Enfoque puramente tecnologico. |
| Interoperabilidad | Como se conserva significado comun? | Reducirlo a API o formato. |
| Calidad | Como se mide y corrige? | Hablar de calidad sin dimensiones. |
| Reutilizacion | Como se abre y usa informacion publica? | Confundir con transparencia. |
| Arquitectura | Como se materializa en servicios y catalogos? | Manual tecnico excesivo. |
| Garantias | Que limites protegen derechos e intereses publicos? | Apertura ingenua. |
| Supuestos | Como se aplica en casos reales? | Casos sin resolucion. |
| Repaso | Que debe recordar antes del examen? | Repetir el indice. |
