#!/bin/bash
# Instalador de Orquesta como Servicio del Sistema (Daemon)
# Autor: OSL - Diputación de Granada

set -e

APP_DIR="/home/alberto/Trabajo/PlataformaMunicipal/orquestador"
BIN_NAME="orquesta"
SERVICE_NAME="orquesta"
USER_NAME="alberto"

echo "⚙ Preparando instalación de Orquesta Daemon..."

cd "$APP_DIR"

# 1. Compilar binario
echo "🔨 Compilando binario..."
go build -o "$BIN_NAME" main.go
chmod +x "$BIN_NAME"

# 2. Crear directorios de logs si no existen
echo "📁 Creando estructura de logs..."
mkdir -p "$APP_DIR/logs"

# 3. Generar el fichero de servicio systemd
echo "📄 Generando fichero de servicio..."
sudo tee /etc/systemd/system/$SERVICE_NAME.service > /dev/null <<EOF
[Unit]
Description=Orquesta Control Plane Daemon
After=network.target

[Service]
Type=simple
User=$USER_NAME
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/$BIN_NAME serve --puerto 8080
Restart=always
RestartSec=5
StandardOutput=append:$APP_DIR/logs/daemon.log
StandardError=append:$APP_DIR/logs/daemon.err

[Install]
WantedBy=multi-user.target
EOF

# 4. Recargar y activar
echo "🚀 Activando servicio..."
sudo systemctl daemon-reload
sudo systemctl enable $SERVICE_NAME
sudo systemctl restart $SERVICE_NAME

echo "✅ Orquesta está corriendo como servicio del sistema."
echo "🔗 Panel web disponible en: http://localhost:8080"
echo "📜 Logs disponibles en: $APP_DIR/logs/daemon.log"
