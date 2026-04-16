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
