#!/bin/bash
set -euo pipefail

secrets=/run/v11-secrets
certificates=/run/v11-certificates
admin_password=$(<"${secrets}/administrator_password")
alice_password=$(<"${secrets}/alice_password")
disabled_password=$(<"${secrets}/disabled_password")
dex_password=$(<"${secrets}/dex_password")

rm -f /etc/samba/smb.conf
install -d -m 0755 /run/samba /var/log/samba /var/lib/samba

samba-tool domain provision \
    --realm=V11.EXAMPLE.TEST \
    --domain=V11 \
    --server-role=dc \
    --dns-backend=SAMBA_INTERNAL \
    --use-rfc2307 \
    --adminpass="${admin_password}"

tls_directory=/var/lib/samba/private/tls
install -d -o root -g root -m 0700 "${tls_directory}"
install -o root -g root -m 0600 "${certificates}/samba.key" "${tls_directory}/v11-samba.key"
install -o root -g root -m 0644 "${certificates}/samba.crt" "${tls_directory}/v11-samba.crt"
install -o root -g root -m 0644 "${certificates}/ca.crt" "${tls_directory}/v11-ca.crt"

sed -i "/^\[global\]/a\\
        tls enabled = yes\\
        tls keyfile = ${tls_directory}/v11-samba.key\\
        tls certfile = ${tls_directory}/v11-samba.crt\\
        tls cafile = ${tls_directory}/v11-ca.crt" /etc/samba/smb.conf

samba-tool user create dexbind "${dex_password}" \
    --given-name=Dex --surname=Bridge --mail-address=dexbind@v11.example.test
samba-tool user create alice "${alice_password}" \
    --given-name=Alice --surname=V11 --mail-address=alice@v11.example.test
samba-tool user create disabled "${disabled_password}" \
    --given-name=Disabled --surname=V11 --mail-address=disabled@v11.example.test
samba-tool group add orquesta-users
samba-tool group add project-admins
samba-tool group addmembers orquesta-users alice
samba-tool group addmembers project-admins alice
samba-tool group addmembers orquesta-users disabled
samba-tool user disable disabled

unset admin_password alice_password disabled_password dex_password
exec samba --interactive --no-process-group --debug-stdout
