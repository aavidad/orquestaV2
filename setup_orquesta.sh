#!/bin/bash
# setup_orquesta.sh - Instalador completo de Orquesta Daemon
set -e

echo "🚀 Iniciando preparación de Orquesta..."

# 1. Compilar el motor
echo "🔨 Compilando binario de Go..."
go build -o orquesta main.go

# 2. Configurar estructura de logs
echo "📁 Preparando carpetas de logs..."
mkdir -p logs

if [ ! -f orquesta.env ]; then
cat <<EOF > orquesta.env
# Persistencia explicita obligatoria para el daemon.
# Rellena estos valores antes de activar el servicio.
ORQUESTA_DB_DRIVER=postgres
ORQUESTA_DB_DSN=postgres://usuario:password@localhost/orquesta?sslmode=disable
EOF
fi

# 2.1. Provisionar skills por defecto para Codex
echo "🪓 Provisionando skills de Codex..."
bash scripts/provision_codex_skills.sh

# 3. Crear el archivo de servicio para systemd locally
echo "📄 Generando fichero orquesta.service..."
cat <<EOF > orquesta.service
[Unit]
Description=Orquesta Control Plane Daemon
After=network.target

[Service]
Type=simple
User=$(whoami)
WorkingDirectory=$(pwd)
Environment=ORQUESTA_REQUIRE_EXPLICIT_PERSISTENCE=1
EnvironmentFile=-$(pwd)/orquesta.env
ExecStart=$(pwd)/orquesta serve --puerto 16543
Restart=always
RestartSec=10
StandardOutput=append:$(pwd)/logs/daemon.log
StandardError=append:$(pwd)/logs/daemon.err

[Install]
WantedBy=multi-user.target
EOF

# 4. Instrucciones finales para el usuario
echo "------------------------------------------------"
echo "✅ Ficheros preparados con éxito."
echo "Para activar el servicio en el sistema, ejecuta:"
echo ""
echo "  sudo cp orquesta.service /etc/systemd/system/"
echo "  sudo systemctl daemon-reload"
echo "  sudo systemctl enable --now orquesta"
echo ""
echo "------------------------------------------------"
echo "🔗 El panel estará en: http://localhost:16543"
