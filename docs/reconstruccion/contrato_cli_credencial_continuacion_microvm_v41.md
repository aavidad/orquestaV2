# Provisionado local de la credencial de continuación microVM V41

## Contrato

`orquesta credentials provision-microvm-continuation-authority` es un comando
local de mantenimiento. No arranca servidor, scheduler, proveedor ni conector
de Agente MicroVM. Solo abre el `CredentialStore` configurado y crea, o vuelve
a consultar, una credencial Ed25519.

```text
orquesta credentials provision-microvm-continuation-authority \
  --config /ruta/absoluta/orquesta.toml \
  --request-ref request:provision-continuation:<identidad-estable>
```

No acepta por flags referencias de credencial, identidad, proyecto, propósito
ni valores de confianza. Proceden exclusivamente de la configuración y de la
identidad local canónicas:

- credencial:
  `runtime.microvm.expired_launch_continuation_authority_signing_credential_ref`;
- propietario: `identity.local_actor`;
- ámbito: `project.default`;
- propósito fijo:
  `orquesta.microvm-expired-launch-continuation-authority.v1`;
- identidad pública: `key_id`, `key_epoch` y `trust_revision` de
  `runtime.microvm.expired_launch_continuation_authority_*`.

La operación es *create-only*: nunca rota, revoca ni sustituye una credencial.
Un reintento describe primero la autoridad existente, no invoca el generador
de claves y deriva la parte pública prestando la versión exacta al callback del
almacén. La privada no se devuelve, serializa ni registra y todas sus copias
temporales se destruyen al salir del callback.

## Bootstrap y fijación final

El perfil de configuración exclusivo de este comando permite omitir una sola
clave que todavía no puede conocerse:

```text
runtime.microvm.expired_launch_continuation_authority_public_key_sha256
```

Todo lo demás, incluido el descriptor físico, la referencia de credencial, el
ID, la época y la revisión de confianza, sigue siendo obligatorio. El flujo es:

1. Ejecutar el comando con el resumen público omitido.
2. Conservar el JSON de salida como recibo material-free.
3. Fijar `public_key_sha256` con el valor devuelto en el TOML final.
4. Repetir exactamente el comando. Debe devolver `credential_state=resumed` y
   `configured_public_key_sha256_matches=true`.
5. Validar el TOML con el resolver normal antes de arrancar Orquesta.

Si el resumen ya está configurado y no coincide, el comando falla cerrado con
`cli.continuation_credential_public_key_digest_mismatch`. La credencial no se
rota: se corrige la configuración o se investiga la divergencia.

## Recibo público y frontera con Agente MicroVM

La salida JSON no incluye rutas, almacenes, propietarios, ámbitos, referencias
internas ni material privado. El único tuple que se entrega al actualizador de
Agente MicroVM es:

```json
{
  "key_id": "continuation-...",
  "key_epoch": 1,
  "trust_revision": 1,
  "public_key_base64": "...",
  "public_key_sha256": "..."
}
```

Agente MicroVM materializa de forma independiente sus configuraciones y el
archivo público crudo de 32 bytes. Su actualizador no abre, migra, copia ni
restaura el `CredentialStore`, el TOML o la SQLite de Orquesta. La clave privada
permanece únicamente bajo autoridad de Orquesta.
