#!/bin/sh

set -e

echo "Building configuration from template..."
envsubst < "./config.template" > "./saiwebsocket.config"

echo "Configuration built successfully"
echo "Starting application..."

exec "$@"
