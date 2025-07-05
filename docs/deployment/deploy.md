# AutoCert Production Deployment Guide

This comprehensive guide provides step-by-step instructions for deploying the AutoCert application in a production environment using Ubuntu 24.04.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Initial Setup](#initial-setup)
3. [Database Configuration](#database-configuration)
4. [Message Queue Setup](#message-queue-setup)
5. [Object Storage Configuration](#object-storage-configuration)
6. [Application Deployment](#application-deployment)
7. [Web Server Configuration](#web-server-configuration)
8. [Monitoring and Maintenance](#monitoring-and-maintenance)

## Prerequisites

- Ubuntu 24.04 server with root access
- Domain names configured for the application components
- Basic knowledge of Linux system administration

## Initial Setup

### 1. Clone the Repository

```bash
git clone --recursive https://github.com/SeakMengs/AutoCert.git /var/www/AutoCert
cd /var/www/AutoCert
```

### 2. Install Go Programming Language

```bash
sudo apt update
sudo apt install golang -y
```

### 3. Install Node.js and npm

Install Node Version Manager (nvm) and Node.js:

```bash
# Download and install nvm
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash

# Reload shell environment
source "$HOME/.nvm/nvm.sh"

# Install Node.js version 22
nvm install 22

# Verify installation
node -v  # Should print "v22.16.0"
npm -v   # Should print "10.9.2"
```

Install PM2 process manager:

```bash
npm install -g pm2
```

## Database Configuration

### 1. Install PostgreSQL

```bash
sudo apt update
sudo apt install postgresql postgresql-contrib -y
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

### 2. Configure Database

Switch to PostgreSQL user and create database:

```bash
sudo su - postgres
psql
```

Execute the following SQL commands:

```sql
CREATE DATABASE autocert;
\c autocert
CREATE EXTENSION IF NOT EXISTS citext;

CREATE USER admin WITH LOGIN PASSWORD 'adminadmin';

-- Grant privileges
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO admin;
GRANT USAGE, CREATE ON SCHEMA public TO admin;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO admin;
ALTER SCHEMA public OWNER TO admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO admin;

\q
```

### 3. Configure PostgreSQL Network Access

Edit PostgreSQL configuration:

```bash
sudo vim /etc/postgresql/16/main/postgresql.conf
```

Change the listen_addresses line:
```
listen_addresses = '*'
```

Edit client authentication:

```bash
sudo vim /etc/postgresql/16/main/pg_hba.conf
```

Add or modify the IPv4 connection line:
```
host    all             all             0.0.0.0/0               md5
```

**Note:** Change any `peer` authentication methods to `md5` for the admin user.

Restart PostgreSQL:

```bash
sudo systemctl restart postgresql
```

### 4. Run Database Migration

First, configure the environment file:

```bash
cp /var/www/AutoCert/.env.example /var/www/AutoCert/.env
```

Edit the `.env` file and fill in the database connection details, then run migration:

```bash
go run /var/www/AutoCert/cmd/migrate/main.go
```

## Message Queue Setup

### 1. Install RabbitMQ

Add RabbitMQ repositories and install:

```bash
# Install dependencies
sudo apt-get install curl gnupg apt-transport-https -y

# Add signing keys
curl -1sLf "https://keys.openpgp.org/vks/v1/by-fingerprint/0A9AF2115F4687BD29803A206B73A36E6026DFCA" | sudo gpg --dearmor | sudo tee /usr/share/keyrings/com.rabbitmq.team.gpg > /dev/null
curl -1sLf https://github.com/rabbitmq/signing-keys/releases/download/3.0/cloudsmith.rabbitmq-erlang.E495BB49CC4BBE5B.key | sudo gpg --dearmor | sudo tee /usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg > /dev/null
curl -1sLf https://github.com/rabbitmq/signing-keys/releases/download/3.0/cloudsmith.rabbitmq-server.9F4587F226208342.key | sudo gpg --dearmor | sudo tee /usr/share/keyrings/rabbitmq.9F4587F226208342.gpg > /dev/null

# Add repositories
sudo tee /etc/apt/sources.list.d/rabbitmq.list <<EOF
deb [arch=amd64 signed-by=/usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg] https://ppa1.rabbitmq.com/rabbitmq/rabbitmq-erlang/deb/ubuntu noble main
deb-src [signed-by=/usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg] https://ppa1.rabbitmq.com/rabbitmq/rabbitmq-erlang/deb/ubuntu noble main

deb [arch=amd64 signed-by=/usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg] https://ppa2.rabbitmq.com/rabbitmq/rabbitmq-erlang/deb/ubuntu noble main
deb-src [signed-by=/usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg] https://ppa2.rabbitmq.com/rabbitmq/rabbitmq-erlang/deb/ubuntu noble main

deb [arch=amd64 signed-by=/usr/share/keyrings/rabbitmq.9F4587F226208342.gpg] https://ppa1.rabbitmq.com/rabbitmq/rabbitmq-server/deb/ubuntu noble main
deb-src [signed-by=/usr/share/keyrings/rabbitmq.9F4587F226208342.gpg] https://ppa1.rabbitmq.com/rabbitmq/rabbitmq-server/deb/ubuntu noble main

deb [arch=amd64 signed-by=/usr/share/keyrings/rabbitmq.9F4587F226208342.gpg] https://ppa2.rabbitmq.com/rabbitmq/rabbitmq-server/deb/ubuntu noble main
deb-src [signed-by=/usr/share/keyrings/rabbitmq.9F4587F226208342.gpg] https://ppa2.rabbitmq.com/rabbitmq/rabbitmq-server/deb/ubuntu noble main
EOF

# Update and install
sudo apt-get update -y
sudo apt-get install -y erlang-base erlang-asn1 erlang-crypto erlang-eldap erlang-ftp erlang-inets erlang-mnesia erlang-os-mon erlang-parsetools erlang-public-key erlang-runtime-tools erlang-snmp erlang-ssl erlang-syntax-tools erlang-tftp erlang-tools erlang-xmerl
sudo apt-get install rabbitmq-server -y --fix-missing
```

### 2. Configure RabbitMQ Users

Remove default guest user and create admin user:

```bash
sudo rabbitmqctl delete_user guest
sudo rabbitmqctl add_user admin adminpw
sudo rabbitmqctl set_user_tags admin administrator
sudo rabbitmqctl set_permissions -p / admin ".*" ".*" ".*"
```

Verify configuration:

```bash
sudo rabbitmqctl list_users
sudo rabbitmqctl list_permissions -p /
```

Enable management plugin:

```bash
sudo rabbitmq-plugins enable rabbitmq_management
```

## Object Storage Configuration

### 1. Install MinIO

Download and install MinIO:

```bash
wget https://dl.min.io/server/minio/release/linux-amd64/archive/minio_20250422221226.0.0_amd64.deb -O minio.deb
sudo dpkg -i minio.deb
```

### 2. Configure MinIO

Create MinIO user and group:

```bash
sudo groupadd -r minio-user
sudo useradd -M -r -g minio-user minio-user
sudo mkdir -p /mnt/data
sudo chown minio-user:minio-user /mnt/data
```

Create MinIO configuration file:

```bash
sudo vim /etc/default/minio
```

Add the following content:

```bash
# MinIO root credentials
MINIO_ROOT_USER=myminioadmin
MINIO_ROOT_PASSWORD=minio-secret-key-change-me

# Storage path
MINIO_VOLUMES="/mnt/data"

# Console configuration
MINIO_OPTS="--console-address :9001"
```

### 3. Start MinIO Service

```bash
sudo systemctl enable minio
sudo systemctl start minio
sudo systemctl status minio.service
```

## Application Deployment

### 1. Build Application Components

Build the API server:

```bash
cd /var/www/AutoCert
go build -o autocertapi ./cmd/api/main.go
```

Build the certificate worker:

```bash
go build -o autocertcertworker ./cmd/cert_consumer/main.go
```

Build the mail worker:

```bash
go build -o autocertmailworker ./cmd/mail_consumer/main.go
```

### 2. Configure API Service

Create systemd service file:

```bash
sudo vim /etc/systemd/system/autocertapi.service
```

Add the following content:

```ini
[Unit]
Description=AutoCert Go API
After=network.target

[Service]
WorkingDirectory=/var/www/AutoCert
ExecStart=/var/www/AutoCert/autocertapi
Restart=always
User=root
Group=root
Environment=PORT=8080

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable autocertapi
sudo systemctl start autocertapi
```

### 3. Configure Certificate Worker Service

Create systemd service file:

```bash
sudo vim /etc/systemd/system/autocertcertworker.service
```

Add the following content:

```ini
[Unit]
Description=AutoCert Certificate Worker
After=network.target

[Service]
WorkingDirectory=/var/www/AutoCert
ExecStart=/var/www/AutoCert/autocertcertworker
Restart=always
User=root
Group=root

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable autocertcertworker
sudo systemctl start autocertcertworker
```

### 4. Configure Mail Worker Service

Create systemd service file:

```bash
sudo vim /etc/systemd/system/autocertmailworker.service
```

Add the following content:

```ini
[Unit]
Description=AutoCert Mail Worker
After=network.target

[Service]
WorkingDirectory=/var/www/AutoCert
ExecStart=/var/www/AutoCert/autocertmailworker
Restart=always
User=root
Group=root

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable autocertmailworker
sudo systemctl start autocertmailworker
```

### 5. Configure Frontend Application

Navigate to the web directory and configure environment:

```bash
cd /var/www/AutoCert/web
cp .env.example .env
```

**Important:** Edit the `.env` file and fill in the correct values for your environment.

Install dependencies and build:

```bash
npm install --force
npm run build
```

Start the frontend with PM2:

```bash
pm2 start npm --name "autocert-frontend" -- run start
pm2 save
pm2 startup
```

Follow the instructions provided by `pm2 startup` to ensure PM2 starts on system boot.

## Web Server Configuration

### 1. Install Nginx

```bash
sudo apt update
sudo apt install nginx -y
sudo systemctl enable nginx
sudo systemctl start nginx
```

### 2. Configure Nginx

Create the Nginx configuration file:

```bash
sudo vim /etc/nginx/sites-available/autocert
```

Add the following configuration (replace domain names with your actual domains):

```nginx
# Catch-all server block to reject unknown domains
server {
    listen 80 default_server;
    server_name _;
    return 444;
}

# MinIO Console
server {
    listen 80;
    server_name autocert-console.yourdomain.com;

    location / {
        proxy_pass http://localhost:9001;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}

# MinIO S3 API
server {
    listen 80;
    server_name autocert-storage.yourdomain.com;

    ignore_invalid_headers off;
    client_max_body_size 0;
    proxy_buffering off;
    proxy_request_buffering off;

    location / {
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 300;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        chunked_transfer_encoding off;
        proxy_pass http://localhost:9000;
    }
}

# Backend API
server {
    listen 80;
    server_name autocert-api.yourdomain.com;
    client_max_body_size 50m;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# Frontend Application
server {
    listen 80;
    server_name autocert.yourdomain.com;
    client_max_body_size 50m;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# RabbitMQ Management Interface
server {
    listen 80;
    server_name autocert-queue.yourdomain.com;

    location / {
        proxy_pass http://localhost:15672;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Enable the configuration:

```bash
sudo ln -s /etc/nginx/sites-available/autocert /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## Monitoring and Maintenance

### Service Status Monitoring

Check the status of all services:

```bash
# Check API service
sudo systemctl status autocertapi

# Check certificate worker
sudo systemctl status autocertcertworker

# Check mail worker
sudo systemctl status autocertmailworker

# Check PM2 processes
pm2 status
```

### Live Log Monitoring

Monitor service logs in real-time:

```bash
# API service logs
sudo journalctl -f -u autocertapi.service

# Certificate worker logs
sudo journalctl -f -u autocertcertworker.service

# Mail worker logs
sudo journalctl -f -u autocertmailworker.service

# PM2 logs
pm2 logs autocert-frontend
```

### Important Configuration Notes

1. **Environment Variables**: Ensure all `.env` files are properly configured with production values
2. **Security**: Change all default passwords and credentials
3. **MinIO Credentials**: Use the MinIO root user and password from `/etc/default/minio` as `ACCESS_KEY` and `SECRET_KEY` in your application configuration
4. **Domain Names**: Replace all example domain names with your actual domain names
5. **SSL/TLS**: Consider implementing SSL certificates for production use
6. **Firewall**: Configure appropriate firewall rules for your setup
    - Allow HTTP (80) and HTTPS (443) traffic
    - Allow PostgreSQL (5432) and RabbitMQ (5672, 15672) traffic from trusted sources
    - Allow MinIO (9000, 9001) traffic from trusted sources
    - Allow API (8080) traffic from trusted sources
    - Allow Frontend (3000) traffic from trusted sources
7. **Backups**: Implement regular backup procedures for your database and MinIO data

### Troubleshooting

If services fail to start, check the logs using the journalctl commands provided above. Common issues include:

- Incorrect environment variables
- Database connection problems
- Port conflicts
- Permission issues
- Missing dependencies

For persistent issues, ensure all services are running and accessible on their designated ports before proceeding to the next component in the deployment process.