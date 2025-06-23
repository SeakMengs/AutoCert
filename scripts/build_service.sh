#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

echo "Building API service..."
go build -o autocertapi ./cmd/api/main.go

echo "Building certificate worker..."
go build -o autocertcertworker ./cmd/cert_consumer/main.go

echo "Building mail worker..."
go build -o autocertmailworker ./cmd/mail_consumer/main.go

echo "All services built successfully."

if [[ "$1" == "restart" ]]; then
    ./scripts/restart_service.sh
fi