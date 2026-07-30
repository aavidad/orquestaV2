# Diseño condicionado de una instantánea Git histórica segura

Fecha: 2026-07-30.

Estado: estudio aceptado; implementación no autorizada todavía. Este documento
no acredita un copiador, no permite censar raíces reales y no crea una
capacidad paralela.

## Encargo y conclusión

```text
capability IDs: GOV-16
invariante: un inventario no modifica la fuente que pretende conservar
autoridad que escribe: ninguna en este estudio
puertos afectados: futura lectura confinada y almacén de instantáneas
adaptadores afectados: futuro adaptador Linux local
write-set: este documento
dependencias causales: censador físico acreditado y fuente congelada
código antiguo que permitirá retirar: recorridos Git directos sobre raíces vivas
test de contrato: copia exacta, repetible y utilizable sin la fuente original
negativo/mutación: fecha de acceso, carrera, enlace, montaje, alternates y caída
E2E/gate: censo -> copia -> retirar temporal fuente -> censos Git sobre copia
presupuesto: se fijará por raíz; ningún valor implícito
```

La consulta local de lecciones `GOV-16` no devolvió patrones. El registro de
adopción de herramientas y su inventario transversal tampoco contienen una
entrada para instantáneas, `openat2` o lectura sin fecha de acceso. Son huecos
explícitos; antes de programar habrá que registrar la decisión y su retirada.

No es posible prometer una copia exacta y coherente de una fuente viva usando
solamente `openat2` y `O_NOATIME`:

- `O_NOATIME` puede fallar sin propietario o permiso suficiente y no admite una
  ruta alternativa silenciosa;
- leer un enlace puede actualizar su fecha de acceso;
- copiar el árbol y el directorio Git común no forma un instante indivisible si
  existen escritores;
- VEC cambió de 88 a 89 espacios durante la propia auditoría;
- una lectura Git ordinaria puede consultar objetos, configuración o rutas que
  queden fuera de la raíz aparente.

Por ello el copiador rechazará fuentes vivas. Necesita simultáneamente:

1. un censo físico completo, privado y sellado que actúe como lista exacta;
2. un justificante de congelación o una instantánea ya montada en solo lectura
   y sin actualización de fecha de acceso.

Sin ambos hechos la operación no empieza.

## Aplicación propuesta

Nombre provisional descriptivo:

```text
copia_instantanea_git_historica
```

Será una orden local pequeña para análisis, no un clonador general, un servicio
residente ni otro almacén de producto. No descubre rutas mediante recorrido
libre y no ejecuta Git contra la fuente.

### Entradas

- resumen y ubicación privada del censo físico completo;
- correspondencia privada de alias lógico a descriptor de raíz;
- justificante de congelación o instantánea;
- destino privado existente;
- clave de idempotencia;
- límites explícitos;
- identidad y autorización del operador local.

La correspondencia física no se versiona. Los justificantes conservan sujeto,
instante, propietario, montaje, revisión y algoritmo.

### Contenido

Para cada repositorio se copiarán:

- el árbol físico de cada espacio de trabajo, incluidos ignorados y no seguidos;
- el directorio Git común completo;
- objetos y paquetes;
- referencias y cabeceras por espacio;
- índices, registros históricos, configuración y ganchos Git;
- almacenes LFS y metadatos de espacios cuando existan;
- repositorios anidados declarados.

`alternates`, repositorios promisor y configuración con rutas o procesos
externos bloquean la copia hasta que sus destinos sean otras raíces explícitas.
No se descarga ningún objeto.

### Apertura y equivalencia

Cada ruta del censo se reabre desde un descriptor estable mediante:

```text
RESOLVE_BENEATH
RESOLVE_NO_MAGICLINKS
RESOLVE_NO_SYMLINKS
RESOLVE_NO_XDEV
O_NOATIME
```

Se compara `statx` con el censo antes y después. Un permiso insuficiente, cambio,
entrada nueva o montaje distinto aborta la generación completa. No hay ruta
alternativa que vuelva a una lectura ordinaria.

Los enlaces solo pueden copiarse si el justificante acredita un montaje en solo
lectura y sin fecha de acceso. Los montajes hijos se rechazan y, si son
necesarios, entran como otra raíz.

Sockets, colas, dispositivos y demás especiales no se abren ni recrean. Se
conserva el hecho de metadatos y la generación queda incompleta cuando formen
parte del sujeto necesario.

### Enlaces físicos y metadatos

La primera versión copia cada fichero regular de forma independiente. Conserva
un grupo local seudonimizado derivado de dispositivo, inodo y número de enlaces,
pero no intenta recrear enlaces físicos ni reflinks. Esa optimización no
condiciona la equivalencia de bytes.

Se conservan nombres, bytes, modo, fecha de modificación y resúmenes. Propietario
y grupo se registran, pero no se intenta elevar permisos ni cambiar propietario.
Atributos extendidos, listas de control y capacidades deben estar en el censo o
bloquear la completitud.

## Límites obligatorios

La orden recibe y aplica límites de:

- entradas;
- profundidad;
- componentes y bytes de ruta;
- bytes por fichero;
- bytes lógicos y realmente asignados;
- atributos extendidos;
- tiempo;
- descriptores abiertos;
- memoria y grupos de enlaces físicos;
- espacio libre mínimo y reserva de disco.

El tiempo y los bytes se comprueban por bloque. Un límite agotado produce una
generación incompleta y no utilizable, nunca una copia presentada como total.

## Publicación y recuperación

- exclusión mutua por destino;
- preparación en directorio privado `0700`;
- ficheros privados `0600` y creación exclusiva;
- sincronización de contenido y directorios;
- una generación inmutable renombrada una sola vez;
- índice atómico que apunta a la generación;
- justificante posterior externo que sella árbol, binario y configuración sin
  autorreferencia.

Una caída conserva preparación y respaldos para estudio. No borra
automáticamente ningún material. La recuperación usa identidad de generación,
diario sincronizado e idempotencia; nunca adivina qué pareja era válida.

## División interna

Cada responsabilidad se mantendrá por debajo del presupuesto ordinario de
300–350 líneas:

```text
opciones
justificante
apertura_linux
copia_regular
metadatos
limites
publicacion
manifiesto
catalogo
entrada
```

El recorrido seguro solo se extraerá del censador físico cuando existan dos
consumidores reales y el contrato de aquel esté acreditado. No se copiará el
código ni se creará un segundo recorrido con semántica distinta.

## Prueba necesaria

Los censadores de funciones y superficies exigirán un justificante de
instantánea. Rechazarán raíz directa, enlace y justificante cuyo sujeto no esté
dentro de la generación.

La prueba completa:

1. crea un repositorio temporal con directorio Git común, espacio enlazado,
   ignorados y objetos;
2. lo congela, censa y copia;
3. retira la fuente temporal del alcance;
4. ejecuta funciones y superficies únicamente sobre la copia;
5. repite y compara bytes, recuentos y resúmenes;
6. ejecuta `git fsck` sin red;
7. demuestra cero procesos y recursos residuales.

Negativos obligatorios:

- fuente que cambia durante la copia;
- permiso que impide `O_NOATIME`;
- enlace o montaje sustituido;
- atributo o entrada no presentes en el censo;
- `alternates` absoluto con fichero canario exterior;
- repositorio promisor con objeto ausente;
- agotamiento de cada límite;
- dos copias concurrentes;
- caída antes y después de cada punto de publicación;
- justificante, árbol o configuración manipulados.

En composición real, la máquina o recinto de análisis verá solo la generación.
No se ampliará Bubblewrap: la decisión vigente lo limita al atestador de
pruebas. Firecracker tampoco se convierte en requisito de este inventario.

## Dependencias y decisión de programación

No se programa todavía. El orden causal es:

1. corregir y acreditar el censador físico;
2. producir el primer censo físico completo sobre una fuente realmente
   congelada;
3. registrar la adopción y el contrato del copiador;
4. demostrar la congelación o instantánea de entrada;
5. implementar la aplicación pequeña;
6. contrarrevisarla antes de copiar una fuente real;
7. ejecutar los censos Git solo sobre la copia acreditada.

## Cierre de este estudio

```text
hecho: límites y arquitectura mínima definidos
invariante restaurado: una lectura cooperativa no se llama instantánea
autoridad final: futura evidencia del censo, congelación y copia
tests/negativos/mutaciones/E2E: definidos, no ejecutados
receipts y revisión acreditada: ninguno; implementación bloqueada
código o decisión retirados: censos Git directos quedan desautorizados
legacy retirado o bloqueo de retirada: ninguna fuente se retira
LOC netas y complejidad: solo documentación
riesgos/P0/P1: coherencia viva, fecha de acceso y referencias externas
siguiente dependencia causal: reauditar el censador físico corregido
```
