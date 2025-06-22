#!/bin/bash

# Restart systemd services
for service in autocertapi autocertcertworker autocertmailworker; do
    echo "Restarting $service..."
    sudo systemctl restart "$service"
done

# Restart pm2 process
echo "Restarting pm2 process: autocert-front..."
pm2 restart autocert-frontend

echo "All services restarted."